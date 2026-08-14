#!/usr/bin/env bash
# IMM-OPS-1 — smoke / unit checks for explore no-deployable-candidate observability.
#
# Usage:
#   ./scripts/test-imm-ops-1.sh unit          # go test (default)
#   ./scripts/test-imm-ops-1.sh smoke        # HTTP checks against a running CPM
#   ./scripts/test-imm-ops-1.sh smoke -v     # smoke with HTTP / metrics / log output
#   ./scripts/test-imm-ops-1.sh local -v     # start CPM, verbose smoke, stop
#   ./scripts/test-imm-ops-1.sh all           # unit + local
#
# Env (smoke / local):
#   CPM_BASE_URL          target for smoke (default http://127.0.0.1:8082)
#   CPM_HTTP_ADDR         listen addr for local mode (default :0 = random free port)
#   CPM_LOG_FILE          server log path for local mode (default: timestamped file in $TMPDIR)
#   DISCOVERY_BASE        Discovery root for JWT when smoking against cafe-deploy (default http://localhost:8080)
#   DISCOVERY_TOKEN       reuse existing Bearer (optional; else signup/signin via cafe-deploy lib)
#   CPM_AUTH_TOKEN        alias for DISCOVERY_TOKEN
#   CPM_SKIP_AUTH=1       omit Authorization (only when CPM_AUTH_REQUIRED=false on target)
#   SCAN_ID               optional real wallet scan_id for explore binding=discovery (else omitted → binding=unknown)
#   CAFE_DEPLOY_ROOT      path to cafe-deploy if not ../cafe-deploy (for discovery-test-user.sh)
#   VERBOSE=1             show explore JSON, metrics lines, structured log (-v)
#   SKIP_UNIT=1           skip unit tests in "all"
#   SKIP_SMOKE=1          skip smoke in "all"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CPM_HTTP_ADDR="${CPM_HTTP_ADDR:-:0}"
VERBOSE="${VERBOSE:-0}"
MODE=""
CLI_ARGS=("$@")

base_url_from_addr() {
  local addr="$1"
  local port="${addr##*:}"
  if [[ "$port" == "0" || -z "$port" ]]; then
    fail "CPM listen port unknown (use CPM_HTTP_ADDR=:PORT with port > 0 for smoke, or local mode)"
  fi
  printf 'http://127.0.0.1:%s' "$port"
}

EXPLORE_PATH="/api/cpm/v1/policies/decisions/explore"
INSTANCES_PATH="/api/cpm/v1/policies/instances"
METRIC_NAME="cpm_explore_no_deployable_candidate_total"
LOG_EVENT="cpm.explore.no_deployable_candidate"
WALLET_RAW="0x742d35cc6634c0532925a3b844bc454e4438f44e"

SERVER_PID=""
CPM_LOG_FILE="${CPM_LOG_FILE:-}"
CPM_HEADERS=()
SMOKE_INCLUDE_SCAN_ID=1
SMOKE_EXPECT_BINDING=discovery

info()  { printf '→ %s\n' "$*"; }
warn()  { printf '⚠ %s\n' "$*" >&2; }
fail()  { printf '✗ %s\n' "$*" >&2; exit 1; }
pass()  { printf '✓ %s\n' "$*"; }
have()  { command -v "$1" >/dev/null 2>&1; }

verbose_enabled() { [[ "$VERBOSE" == "1" ]]; }

verbose_block() {
  verbose_enabled || return 0
  printf '\n── %s ──\n' "$1"
}

verbose_line() {
  verbose_enabled || return 0
  printf '%s\n' "$*"
}

verbose_json() {
  local title="$1" body="$2"
  verbose_block "$title"
  if have jq; then
    printf '%s' "$body" | jq .
  else
    printf '%s\n' "$body"
  fi
}

verbose_metrics_snippet() {
  local title="$1"
  verbose_block "$title"
  curl -fsS "$CPM_BASE_URL/metrics" | grep -E "^${METRIC_NAME}(\{| )" || verbose_line "(metric not present yet)"
}

cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    info "stopping CPM (pid $SERVER_PID)"
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

fixture_env() {
  export CPM_AUTH_REQUIRED=false
  export CPM_HTTP_ADDR="$CPM_HTTP_ADDR"
  export CPM_POLICY_TEMPLATE_PATHS="$REPO_ROOT/internal/domain/policy/testdata/crypto_policy_template_pq_account_validation_v1.json"
  export CPM_POLICY_INSTANCE_PATHS="$REPO_ROOT/internal/domain/policy/testdata/crypto_policy_instance_pq_account_validation_v1.json"
}

run_unit() {
  info "unit tests (IMM-OPS-1)"
  (
    cd "$REPO_ROOT"
    go test ./internal/api/... ./internal/metrics/... ./internal/app/... ./internal/domain/policy/... \
      -count=1 \
      -run 'Explore|Metrics|Dominant|Bucket|chain_scope|HandlerMetrics'
  )
  pass "unit tests"
}

wait_for_cpm() {
  local url="$1" i
  for i in $(seq 1 40); do
    if curl -fsS "$url/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

pick_free_port() {
  if have python3; then
    python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()'
    return
  fi
  fail "python3 required to pick a free port for local mode (or set CPM_HTTP_ADDR=:PORT)"
}

default_log_file() {
  local ts tmpdir
  ts="$(date +%Y%m%d-%H%M%S)"
  tmpdir="${TMPDIR:-/tmp}"
  tmpdir="${tmpdir%/}"
  printf '%s/cpm-imm-ops-1-%s-%s.log' "$tmpdir" "$ts" "$$"
}

start_local_cpm() {
  fixture_env
  if [[ "$CPM_HTTP_ADDR" == ":0" ]]; then
    CPM_HTTP_ADDR=":$(pick_free_port)"
  fi
  export CPM_HTTP_ADDR
  if [[ -z "$CPM_LOG_FILE" ]]; then
    CPM_LOG_FILE="$(default_log_file)"
  fi
  info "starting CPM on $CPM_HTTP_ADDR (log: $CPM_LOG_FILE)"
  (
    cd "$REPO_ROOT"
    go run ./cmd/cafe-cpm
  ) >"$CPM_LOG_FILE" 2>&1 &
  SERVER_PID=$!
  CPM_BASE_URL="$(base_url_from_addr "$CPM_HTTP_ADDR")"
  if ! wait_for_cpm "$CPM_BASE_URL"; then
    warn "CPM did not become ready; last log lines:"
    tail -30 "$CPM_LOG_FILE" >&2 || true
    fail "CPM not ready at $CPM_BASE_URL"
  fi
  pass "CPM ready at $CPM_BASE_URL"
}

metric_value() {
  curl -fsS "$CPM_BASE_URL/metrics" | awk -v m="$METRIC_NAME" '
    $0 ~ "^" m "{" { sub(/.*} /, ""); print $1; found=1; exit }
    END { if (!found) print "0" }
  '
}

json_get() {
  local expr="$1" body="$2"
  if have jq; then
    printf '%s' "$body" | jq -r "$expr"
  elif have python3; then
    printf '%s' "$body" | python3 -c "import json,sys; d=json.load(sys.stdin); print($expr)"
  else
    fail "jq or python3 required for JSON assertions"
  fi
}

find_discovery_test_user_lib() {
  local candidates=()
  [[ -n "${CAFE_DEPLOY_ROOT:-}" ]] && candidates+=("${CAFE_DEPLOY_ROOT}/scripts/lib/discovery-test-user.sh")
  candidates+=("$REPO_ROOT/../cafe-deploy/scripts/lib/discovery-test-user.sh")
  local c
  for c in "${candidates[@]}"; do
    [[ -f "$c" ]] && { printf '%s\n' "$c"; return 0; }
  done
  return 1
}

setup_smoke_auth() {
  CPM_HEADERS=()
  SMOKE_INCLUDE_SCAN_ID=1
  SMOKE_EXPECT_BINDING=discovery

  if [[ "${CPM_SKIP_AUTH:-}" == "1" ]]; then
    info "CPM_SKIP_AUTH=1 — no Authorization header (target must have CPM_AUTH_REQUIRED=false)"
    return 0
  fi

  local token="${CPM_AUTH_TOKEN:-${DISCOVERY_TOKEN:-}}"
  if [[ -z "$token" ]]; then
    local lib
    lib="$(find_discovery_test_user_lib)" || true
    DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
    if [[ -n "$lib" ]]; then
      # shellcheck disable=SC1090
      . "$lib"
      discovery_test_user_init "imm-ops-1"
      discovery_test_user_signup_signin || fail "Discovery auth failed (DISCOVERY_BASE=${DISCOVERY_BASE})"
      token="$DISCOVERY_TOKEN"
      info "authenticated via Discovery (${EMAIL})"
    else
      fail "CPM explore requires auth on cafe-deploy stacks. Set DISCOVERY_BASE (cafe-deploy sibling), export DISCOVERY_TOKEN, CPM_SKIP_AUTH=1, or run: ./scripts/test-imm-ops-1.sh local"
    fi
  fi
  CPM_HEADERS=( -H "Authorization: Bearer ${token}" )

  if [[ -n "${SCAN_ID:-}" ]]; then
    SMOKE_INCLUDE_SCAN_ID=1
    SMOKE_EXPECT_BINDING=discovery
    info "SCAN_ID set — explore payloads include scan_id (requires scan authorization)"
  else
    SMOKE_INCLUDE_SCAN_ID=0
    SMOKE_EXPECT_BINDING=unknown
    warn "no SCAN_ID — omitting scan_id from explore payloads (binding=unknown; avoids AUTH-02 403 on fake scan)"
  fi
}

post_explore() {
  local payload="$1" tmp code body
  tmp="$(mktemp)"
  # shellcheck disable=SC2068
  code="$(curl -sS -o "$tmp" -w '%{http_code}' -X POST "$CPM_BASE_URL$EXPLORE_PATH" \
    -H 'Content-Type: application/json' \
    -H 'X-Request-Id: imm-ops-1-smoke' \
    "${CPM_HEADERS[@]}" \
    -d "$payload")"
  body="$(cat "$tmp")"
  rm -f "$tmp"
  [[ "$code" == "200" ]] || fail "POST explore HTTP ${code}: ${body} (auth? try DISCOVERY_BASE=http://localhost:8080 or ./scripts/test-imm-ops-1.sh local)"
  printf '%s' "$body"
}

fetch_policy_instances() {
  # shellcheck disable=SC2068
  curl -fsS "${CPM_HEADERS[@]}" "$CPM_BASE_URL$INSTANCES_PATH"
}

# Explains why target_chain_ids were rejected: catalog scope.chain_ids must cover the full set (all-or-nothing).
print_scope_gap_analysis() {
  local explore_body="$1"
  local instances

  instances="$(fetch_policy_instances)" || {
    warn "could not GET $INSTANCES_PATH — skipping CP scope details"
    return 0
  }

  if have jq; then
  jq -r -n \
    --argjson explore "$explore_body" \
    --argjson items "$(printf '%s' "$instances" | jq '.items')" '
    $explore.decision as $d
    | ($d.request_summary.target_chain_ids // []) as $targets
    | ($d.observed_wallet_summary.chain_ids // []) as $observed
    | [
        "CPM rule: every selection_request.target_chain_id must appear in the Crypto Policy instance scope.chain_ids (all-or-nothing).",
        "Observed wallet chains and requested targets may match each other — the blocker is target ⊆ catalog scope.",
        "",
        ("requested target_chain_ids: " + ($targets | tostring)),
        ("observed chain_ids (wallet): " + ($observed | tostring)),
        ""
      ]
    + (
        if ($d.rejected_candidates | length) == 0 then
          ["(no rejected_candidates in explore response)"]
        else
          $d.rejected_candidates[]
          | .crypto_policy_instance_id as $iid
          | .template_id as $rej_tpl
          | ($items[] | select(.id == $iid)) as $inst
          | if $inst == null then
              ["instance \($iid): not found in GET /policies/instances"]
            else
              ($inst.scope.chain_ids // []) as $scope
              | [$targets[] | select(. as $c | ($scope | index($c)) | not)] as $missing
              | [$targets[] | select(. as $c | ($scope | index($c)))] as $covered
              | [
                  "── Crypto Policy instance: \($iid) ──",
                  "  template_id:         \($rej_tpl // $inst.template_id // "n/a")",
                  "  scope.chain_ids:     \($scope | tostring)",
                  "  targets in scope:    \($covered | tostring)",
                  "  missing from scope:  \($missing | tostring)  → incompatible.chain_scope",
                  "  rejection codes:     \([.rejection_reasons[]?.code] | unique | join(", "))",
                  ""
                ]
            end
        end
      )
    | .[]
    ' | while IFS= read -r line; do
      if [[ -n "$line" ]]; then
        info "$line"
      else
        printf '\n'
      fi
    done
  elif have python3; then
    EXPLORE_JSON="$explore_body" INSTANCES_JSON="$instances" python3 <<'PY'
import json, os
explore = json.loads(os.environ["EXPLORE_JSON"])
items = json.loads(os.environ["INSTANCES_JSON"]).get("items", [])
d = explore.get("decision", {})
targets = d.get("request_summary", {}).get("target_chain_ids") or []
observed = d.get("observed_wallet_summary", {}).get("chain_ids") or []
by_id = {i["id"]: i for i in items}
print("→ CPM rule: every target_chain_id must be in instance scope.chain_ids (all-or-nothing).")
print(f"→ requested target_chain_ids: {targets}")
print(f"→ observed chain_ids (wallet): {observed}")
print()
for rej in d.get("rejected_candidates") or []:
    iid = rej.get("crypto_policy_instance_id")
    inst = by_id.get(iid)
    print(f"→ ── Crypto Policy instance: {iid} ──")
    if not inst:
        print("→   (not found in GET /policies/instances)")
        continue
    scope = (inst.get("scope") or {}).get("chain_ids") or []
    missing = [c for c in targets if c not in scope]
    covered = [c for c in targets if c in scope]
    print(f"→   template_id:         {rej.get('template_id') or inst.get('template_id')}")
    print(f"→   scope.chain_ids:     {scope}")
    print(f"→   targets in scope:    {covered}")
    print(f"→   missing from scope:  {missing}  → incompatible.chain_scope")
    codes = sorted({f.get("code") for f in rej.get("rejection_reasons") or [] if f.get("code")})
    print(f"→   rejection codes:     {', '.join(codes)}")
    print()
PY
  else
    warn "install jq or python3 to display CP scope.chain_ids vs target_chain_ids"
    return 0
  fi

  if verbose_enabled && have jq; then
    verbose_block "GET /policies/instances (catalog snapshot)"
    printf '%s' "$instances" | jq '[.items[] | {id, template_id, scope: .scope.chain_ids}]'
  fi
}

print_scope_match_analysis() {
  local explore_body="$1"
  verbose_enabled || return 0
  local instances
  instances="$(fetch_policy_instances)" || return 0
  if ! have jq; then
    return 0
  fi
  verbose_block "CP scope vs target_chain_ids (deployable — all targets covered)"
  jq -r -n \
    --argjson explore "$explore_body" \
    --argjson items "$(printf '%s' "$instances" | jq '.items')" '
    $explore.decision as $d
    | ($d.request_summary.target_chain_ids // []) as $targets
    | ($d.selected_policy_instance_id // $d.ranked_candidates[0].crypto_policy_instance_id // "") as $iid
    | ($items[] | select(.id == $iid)) as $inst
    | if $inst == null then
        "instance \($iid): not found in catalog"
      else
        ($inst.scope.chain_ids // []) as $scope
        | [
            ("requested target_chain_ids: " + ($targets | tostring)),
            ("instance scope.chain_ids:   " + ($scope | tostring)),
            ("all targets in scope:       " + (if ($targets | all(. as $c | ($scope | index($c)))) then "yes ✓" else "no" end))
          ]
        | .[]
      end
    ' | while IFS= read -r line; do verbose_line "$line"; done
}

no_candidate_payload() {
  local scan_json=""
  if [[ "${SMOKE_INCLUDE_SCAN_ID:-1}" == "1" ]]; then
    scan_json='"scan_id": "705c9704-9428-45e0-882d-fae4cb9d2a0b",
  '
  fi
  cat <<EOF
{
  ${scan_json}  "policy_context": {
    "wallet_address": "0x742d35cc6634c0532925a3b844bc454e4438f44e",
    "wallet_type": "eoa",
    "chain_ids": [1, 2, 5],
    "current_algorithm": "secp256k1_ecrecover",
    "current_pq_posture": "classical_only",
    "scanned_at": "2026-04-17T09:59:58Z"
  },
  "selection_request": {
    "target_posture": "hybrid",
    "target_chain_ids": [1, 2, 5],
    "require_multichain": true,
    "allow_new_wallet": false,
    "address_continuity_required": true,
    "key_rotation_model": "per_userop",
    "recovery_required": true,
    "minimum_maturity": 1,
    "approval_mode": "manual"
  }
}
EOF
}

candidate_found_payload() {
  local scan_json=""
  if [[ "${SMOKE_INCLUDE_SCAN_ID:-1}" == "1" ]]; then
    scan_json='"scan_id": "705c9704-9428-45e0-882d-fae4cb9d2a0b",
  '
  fi
  cat <<EOF
{
  ${scan_json}  "policy_context": {
    "wallet_address": "0x742d35cc6634c0532925a3b844bc454e4438f44e",
    "wallet_type": "eoa",
    "chain_ids": [1, 8453],
    "current_algorithm": "secp256k1_ecrecover",
    "current_pq_posture": "classical_only",
    "scanned_at": "2026-04-17T09:59:58Z"
  },
  "selection_request": {
    "target_posture": "hybrid",
    "target_chain_ids": [1, 8453],
    "require_multichain": true,
    "allow_new_wallet": false,
    "address_continuity_required": true,
    "key_rotation_model": "per_userop",
    "recovery_required": true,
    "minimum_maturity": 1,
    "approval_mode": "manual"
  }
}
EOF
}

assert_no_candidate_response() {
  local body="$1"
  local ranked rejected selected has_chain_scope

  verbose_json "POST explore (no deployable candidate)" "$body"
  print_scope_gap_analysis "$body"

  ranked="$(json_get '(.decision.ranked_candidates // []) | length' "$body")"
  rejected="$(json_get '(.decision.rejected_candidates // []) | length' "$body")"
  selected="$(json_get '.decision.selected_policy_id // ""' "$body")"
  has_chain_scope="$(json_get '[.decision.rejected_candidates[]?.rejection_reasons[]?.code] | any(. == "incompatible.chain_scope")' "$body")"

  [[ "$ranked" == "0" ]] || fail "expected ranked_candidates empty, got $ranked"
  [[ "$rejected" != "0" ]] || fail "expected rejected_candidates non-empty"
  [[ -z "$selected" ]] || fail "expected selected_policy_id empty, got $selected"
  [[ "$has_chain_scope" == "true" ]] || fail "expected incompatible.chain_scope in rejected_candidates"
  pass "explore response: no deployable candidate (chain_scope rejection)"
}

assert_candidate_found_response() {
  local body="$1"
  local ranked selected

  verbose_json "POST explore (deployable candidate — negative case)" "$body"
  print_scope_match_analysis "$body"

  ranked="$(json_get '(.decision.ranked_candidates // []) | length' "$body")"
  selected="$(json_get '.decision.selected_policy_id // ""' "$body")"

  [[ "$ranked" != "0" ]] || fail "expected ranked_candidates non-empty (negative case)"
  [[ -n "$selected" ]] || fail "expected selected_policy_id set (negative case)"
  pass "explore response: deployable candidate found (no IMM-OPS-1 event)"
}

assert_metrics_increment() {
  local before="$1" after="$2"
  awk -v b="$before" -v a="$after" 'BEGIN {
    if (a > b) exit 0
    printf "metric %s: before=%s after=%s (expected increment)\n", "'"$METRIC_NAME"'", b, a > "/dev/stderr"
    exit 1
  }'
  pass "metric $METRIC_NAME incremented ($before → $after)"
}

assert_metrics_unchanged() {
  local before="$1" after="$2"
  awk -v b="$before" -v a="$after" 'BEGIN {
    if (a == b) exit 0
    printf "metric %s: before=%s after=%s (expected no change)\n", "'"$METRIC_NAME"'", b, a > "/dev/stderr"
    exit 1
  }'
  pass "metric $METRIC_NAME unchanged ($before)"
}

assert_metric_labels() {
  local line
  line="$(curl -fsS "$CPM_BASE_URL/metrics" | grep "^${METRIC_NAME}{")" || fail "metric line not found after increment"
  verbose_block "Prometheus metric line"
  verbose_line "$line"
  grep -q 'rejection_code="incompatible.chain_scope"' <<<"$line" || fail "missing rejection_code label"
  grep -q "binding=\"${SMOKE_EXPECT_BINDING}\"" <<<"$line" || fail "missing binding=${SMOKE_EXPECT_BINDING} label"
  grep -q 'wallet_type="eoa"' <<<"$line" || fail "missing wallet_type label"
  grep -q 'missing_chain_count=' <<<"$line" || fail "missing missing_chain_count label"
  grep -q 'scan_id' <<<"$line" && fail "scan_id must not appear in Prometheus labels"
  pass "metric labels look valid (low cardinality)"
}

assert_structured_log() {
  [[ -n "$CPM_LOG_FILE" && -f "$CPM_LOG_FILE" ]] || {
    warn "CPM_LOG_FILE not set — skipping log assertion (use ./scripts/test-imm-ops-1.sh local)"
    return 0
  }
  local log_line
  log_line="$(grep "$LOG_EVENT" "$CPM_LOG_FILE" | tail -1)" || fail "log event $LOG_EVENT not found in $CPM_LOG_FILE"
  verbose_block "structured log ($LOG_EVENT)"
  verbose_line "$log_line"
  grep -q 'dominant_rejection_code="incompatible.chain_scope"' <<<"$log_line" || fail "log missing dominant_rejection_code"
  grep -q 'wallet_address_hash=' <<<"$log_line" || fail "log missing wallet_address_hash"
  grep -q "$WALLET_RAW" <<<"$log_line" && fail "log must not contain raw wallet address"
  pass "structured log $LOG_EVENT present (no raw address)"
}

run_smoke() {
  CPM_BASE_URL="${CPM_BASE_URL:-http://127.0.0.1:8082}"
  info "smoke against $CPM_BASE_URL"
  verbose_enabled && info "verbose mode enabled"

  if [[ -n "$SERVER_PID" ]]; then
    CPM_HEADERS=()
    SMOKE_INCLUDE_SCAN_ID=1
    SMOKE_EXPECT_BINDING=discovery
  else
    setup_smoke_auth
  fi

  local health_body
  health_body="$(curl -fsS "$CPM_BASE_URL/healthz")" || fail "GET /healthz failed"
  verbose_block "GET /healthz"
  verbose_line "$health_body"
  pass "GET /healthz"

  curl -fsS "$CPM_BASE_URL/metrics" >/dev/null || fail "GET /metrics failed"
  verbose_metrics_snippet "GET /metrics (before explore)"
  pass "GET /metrics"

  local before after body

  before="$(metric_value)"
  info "metric $METRIC_NAME before explore = $before"
  verbose_enabled && verbose_line "counter value = $before"

  body="$(post_explore "$(no_candidate_payload)")"
  assert_no_candidate_response "$body"

  after="$(metric_value)"
  assert_metrics_increment "$before" "$after"
  verbose_metrics_snippet "GET /metrics (after no-candidate explore)"
  assert_metric_labels
  assert_structured_log

  before="$after"
  body="$(post_explore "$(candidate_found_payload)")"
  assert_candidate_found_response "$body"
  after="$(metric_value)"
  assert_metrics_unchanged "$before" "$after"
  verbose_metrics_snippet "GET /metrics (after deployable-candidate explore)"

  if verbose_enabled && [[ -n "$CPM_LOG_FILE" && -f "$CPM_LOG_FILE" ]]; then
    verbose_block "CPM server log tail ($CPM_LOG_FILE)"
    tail -20 "$CPM_LOG_FILE" || true
  fi

  pass "smoke IMM-OPS-1 complete"
}

parse_args() {
  set -- "${CLI_ARGS[@]}"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -v|--verbose) VERBOSE=1; shift ;;
      unit|smoke|local|all) MODE="$1"; shift ;;
      help|-h|--help) MODE="help"; shift ;;
      *) fail "unknown argument: $1 (use unit|smoke|local|all [-v])" ;;
    esac
  done
  MODE="${MODE:-unit}"
}

parse_args

case "$MODE" in
  unit)
    run_unit
    ;;
  smoke)
    run_smoke
    ;;
  local)
    start_local_cpm
    run_smoke
    ;;
  all)
    [[ "${SKIP_UNIT:-0}" == "1" ]] || run_unit
    [[ "${SKIP_SMOKE:-0}" == "1" ]] || { start_local_cpm; run_smoke; }
    ;;
  help|-h|--help)
    sed -n '2,16p' "$0"
    ;;
  *)
    fail "unknown mode: $MODE (use unit|smoke|local|all)"
    ;;
esac
