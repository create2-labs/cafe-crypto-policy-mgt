#!/usr/bin/env bash
# Smoke PR13g: Discovery wallet scan -> CPM assessment request -> NATS publication,
# plus TLS/foreign scan_id 404 checks.
#
# Requires: curl, jq. NATS capture requires Docker access to the compose network.

set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/cpm-route-paths.sh
source "${_SCRIPT_DIR}/lib/cpm-route-paths.sh"
# shellcheck source=lib/discovery-route-paths.sh
source "${_SCRIPT_DIR}/lib/discovery-route-paths.sh"

show_help() {
  local self
  self="$(basename "$0")"
  cat <<EOF
Usage: ${self} [--help|-h]

Runs the PR13g smoke checks:
  1) POST ${CPM_POLICIES_ASSESSMENT_REQUEST} with a completed wallet scan_id -> 202
  2) Capture NATS subject cafe.policy.events.policy.assessment.requested.v0_1
  3) Verify TLS scan_id -> 404, not 403
  4) Verify foreign wallet scan_id -> 404, not 403
  5) Verify policy_context on assessment request -> 400

Required:
  DISCOVERY_EMAIL_A       Owner A Discovery email
  DISCOVERY_PASSWORD_A    Owner A password

Optional:
  DISCOVERY_EMAIL_B       Foreign owner email
                           default: cpm.assessment.foreign@cafe-e2e.invalid
  DISCOVERY_PASSWORD_B    Foreign owner password
                           default: DISCOVERY_PASSWORD_A
  DISCOVERY_BASE          Direct Discovery root URL
                           default: http://localhost:8080
  CPM_BASE                Direct CPM root URL
                           default: http://localhost:8082
  WALLET_ADDRESS          Wallet address scanned as owner A
                           default: 0x742d35Cc6634C0532925a3b844Bc454e4438f44e
  TLS_URL                 TLS endpoint scanned as owner A
                           default: https://example.com
  TURNSTILE_TOKEN         Discovery signin/signup turnstile token
                           default: auth06-dev-turnstile-placeholder
  SIGNUP_IF_NEEDED=0      Do not create users if signin fails
  POLL_INTERVAL_SEC       Delay between scan detail polls
                           default: 3
  POLL_MAX_ATTEMPTS       Max scan detail polling attempts
                           default: 40
  CURL_INSECURE=1         curl -k
  CURL_REDIRECT=0         Disable curl redirect following (-L off)

NATS capture:
  SKIP_NATS=1             Skip live NATS subject capture
  NATS_DOCKER_NETWORK     Docker network for nats-box; auto-detected if empty
  NATS_IMAGE              Docker image used for NATS CLI
                           default: natsio/nats-box:latest
  NATS_URL_IN_DOCKER      NATS URL from inside the Docker network
                           default: nats://nats:4222
  NATS_SUBJECT            Subject to capture
                           default: cafe.policy.events.policy.assessment.requested.v0_1
  NATS_TIMEOUT_SEC        Wait for one message
                           default: 15

Consumer contract:
  RUN_CONSUMER_UNIT_TEST=0  Skip go test ./internal/integration/nats

Example:
  DISCOVERY_EMAIL_A=auth06.user.a@cafe-e2e.invalid \\
  DISCOVERY_PASSWORD_A='Auth06DevStaticPass!' \\
  ./${self}

If the wallet assessment returns 503, check CPM env:
  CPM_NATS_URL, CAFE_DISCOVERY_HTTP_BASE, CAFE_SCAN_AUTHORIZATION_URL.
EOF
}

case "${1:-}" in
  -h|--help)
    show_help
    exit 0
    ;;
  "")
    ;;
  *)
    echo "$(basename "$0"): unexpected argument '$1' (use --help)" >&2
    exit 2
    ;;
esac

: "${DISCOVERY_EMAIL_A:?}"
: "${DISCOVERY_PASSWORD_A:?}"

DISCOVERY_EMAIL_B="${DISCOVERY_EMAIL_B:-cpm.assessment.foreign@cafe-e2e.invalid}"
DISCOVERY_PASSWORD_B="${DISCOVERY_PASSWORD_B:-$DISCOVERY_PASSWORD_A}"
DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
CPM_BASE="${CPM_BASE:-http://localhost:8082}"
WALLET_ADDRESS="${WALLET_ADDRESS:-0x742d35Cc6634C0532925a3b844Bc454e4438f44e}"
TLS_URL="${TLS_URL:-https://example.com}"
TURNSTILE_TOKEN="${TURNSTILE_TOKEN:-auth06-dev-turnstile-placeholder}"
SIGNUP_IF_NEEDED="${SIGNUP_IF_NEEDED:-1}"
POLL_INTERVAL_SEC="${POLL_INTERVAL_SEC:-3}"
POLL_MAX_ATTEMPTS="${POLL_MAX_ATTEMPTS:-40}"
SKIP_NATS="${SKIP_NATS:-0}"
NATS_IMAGE="${NATS_IMAGE:-natsio/nats-box:latest}"
NATS_URL_IN_DOCKER="${NATS_URL_IN_DOCKER:-nats://nats:4222}"
NATS_SUBJECT="${NATS_SUBJECT:-cafe.policy.events.policy.assessment.requested.v0_1}"
NATS_TIMEOUT_SEC="${NATS_TIMEOUT_SEC:-15}"
NATS_SUBSCRIBE_DELAY_SEC="${NATS_SUBSCRIBE_DELAY_SEC:-3}"
RUN_CONSUMER_UNIT_TEST="${RUN_CONSUMER_UNIT_TEST:-1}"

DISCOVERY_BASE="${DISCOVERY_BASE%/}"
CPM_BASE="${CPM_BASE%/}"

command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

CURL_OPTS=( -sS )
[[ "${CURL_INSECURE:-}" == "1" ]] && CURL_OPTS+=( -k )
if [[ "${CURL_REDIRECT:-1}" != "0" ]]; then
  CURL_OPTS+=( -L )
fi

TMPDIR_SMOKE="$(mktemp -d)"
NATS_PID=""
cleanup() {
  if [[ -n "${NATS_PID:-}" ]] && kill -0 "$NATS_PID" >/dev/null 2>&1; then
    kill "$NATS_PID" >/dev/null 2>&1 || true
    wait "$NATS_PID" >/dev/null 2>&1 || true
  fi
  rm -rf "$TMPDIR_SMOKE"
}
trap cleanup EXIT

die() {
  echo "error: $*" >&2
  exit 1
}

log() {
  printf '%s\n' "$*"
}

http_post_json() {
  local out="$1"
  local url="$2"
  local body="$3"
  shift 3
  curl "${CURL_OPTS[@]}" "$@" \
    -o "$out" \
    -w "%{http_code}" \
    -X POST \
    -H 'Content-Type: application/json' \
    --data-binary "$body" \
    "$url" || true
}

http_get() {
  local out="$1"
  local url="$2"
  shift 2
  curl "${CURL_OPTS[@]}" "$@" \
    -o "$out" \
    -w "%{http_code}" \
    "$url" || true
}

expect_status() {
  local got="$1"
  local want="$2"
  local label="$3"
  local body_file="$4"
  if [[ "$got" != "$want" ]]; then
    echo "---- ${label} response body ----" >&2
    cat "$body_file" >&2 || true
    echo >&2
    die "${label}: HTTP ${got}, want ${want}"
  fi
}

expect_2xx() {
  local got="$1"
  local label="$2"
  local body_file="$3"
  case "$got" in
    2??) return 0 ;;
  esac
  echo "---- ${label} response body ----" >&2
  cat "$body_file" >&2 || true
  echo >&2
  die "${label}: HTTP ${got}, want 2xx"
}

signin_once() {
  local email="$1"
  local password="$2"
  local out="$3"
  local body code
  body=$(jq -nc \
    --arg email "$email" \
    --arg password "$password" \
    --arg tt "$TURNSTILE_TOKEN" \
    '{email:$email, password:$password, turnstile_token:$tt}')
  code=$(http_post_json "$out" "${DISCOVERY_BASE}/auth/signin" "$body")
  [[ "$code" == "200" ]]
}

signup_once() {
  local email="$1"
  local password="$2"
  local out="$3"
  local body code
  body=$(jq -nc \
    --arg email "$email" \
    --arg password "$password" \
    --arg tt "$TURNSTILE_TOKEN" \
    '{email:$email, password:$password, confirm_password:$password, turnstile_token:$tt}')
  code=$(http_post_json "$out" "${DISCOVERY_BASE}/auth/signup" "$body")
  [[ "$code" == "200" || "$code" == "201" ]]
}

signin_or_signup() {
  local email="$1"
  local password="$2"
  local out_file="$TMPDIR_SMOKE/auth-${email//[^a-zA-Z0-9]/_}.json"
  local token

  if ! signin_once "$email" "$password" "$out_file"; then
    if [[ "$SIGNUP_IF_NEEDED" != "1" ]]; then
      cat "$out_file" >&2 || true
      die "signin failed for ${email}"
    fi
    printf '%s\n' "Signin failed for ${email}; trying signup..." >&2
    signup_once "$email" "$password" "$out_file" || {
      cat "$out_file" >&2 || true
      die "signup failed for ${email}"
    }
    signin_once "$email" "$password" "$out_file" || {
      cat "$out_file" >&2 || true
      die "signin after signup failed for ${email}"
    }
  fi

  token="$(jq -r '.token // empty' "$out_file")"
  [[ -n "$token" && "$token" != "null" ]] || die "no token in auth response for ${email}"
  printf '%s' "$token"
}

url_escape() {
  jq -rn --arg v "$1" '$v|@uri'
}

wait_for_scan_detail() {
  local token="$1"
  local collection_path="$2"
  local scan_id="$3"
  local label="$4"
  local escaped out code status
  escaped="$(url_escape "$scan_id")"
  out="$TMPDIR_SMOKE/${label}-detail.json"

  for attempt in $(seq 1 "$POLL_MAX_ATTEMPTS"); do
    code=$(http_get "$out" "${DISCOVERY_BASE}${collection_path}/${escaped}" \
      -H "Authorization: Bearer ${token}")
    if [[ "$code" == "200" ]]; then
      status="$(jq -r '.status // empty' "$out")"
      if jq -e '.result != null' "$out" >/dev/null || [[ "$status" == "completed" ]]; then
        cp "$out" "$TMPDIR_SMOKE/${label}-completed.json"
        log "${label} detail ready: scan_id=${scan_id}"
        return 0
      fi
      if [[ "$status" == "failed" ]]; then
        cat "$out" >&2
        die "${label} scan failed"
      fi
      log "  ${label} poll ${attempt}/${POLL_MAX_ATTEMPTS}: status=${status:-unknown}"
    elif [[ "$code" == "404" ]]; then
      log "  ${label} poll ${attempt}/${POLL_MAX_ATTEMPTS}: detail not found yet"
    else
      cat "$out" >&2 || true
      die "${label} detail returned HTTP ${code}"
    fi
    sleep "$POLL_INTERVAL_SEC"
  done
  die "timed out waiting for ${label} detail (${scan_id})"
}

build_selection_request() {
  local detail_file="$1"
  local chains
  chains="$(jq -c '.result.chain_ids // [1]' "$detail_file")"
  jq -nc --argjson chains "$chains" '{
    target_posture: "hybrid",
    target_chain_ids: $chains,
    require_multichain: (($chains | length) > 1),
    allow_new_wallet: false,
    address_continuity_required: true,
    key_rotation_required: true,
    recovery_required: true,
    minimum_maturity: 1,
    approval_mode: "manual"
  }'
}

assessment_body() {
  local scan_id="$1"
  local selection="$2"
  local client_request_id="$3"
  jq -nc \
    --arg scan_id "$scan_id" \
    --arg client_request_id "$client_request_id" \
    --argjson selection "$selection" \
    '{scan_id:$scan_id, client_request_id:$client_request_id, selection_request:$selection}'
}

post_assessment_expect() {
  local token="$1"
  local body="$2"
  local want="$3"
  local label="$4"
  local out="$TMPDIR_SMOKE/${label}.json"
  local code
  code=$(http_post_json "$out" "${CPM_BASE}${CPM_POLICIES_ASSESSMENT_REQUEST}" "$body" \
    -H "Authorization: Bearer ${token}")
  expect_status "$code" "$want" "$label" "$out"
  printf '%s\n' "${label}: HTTP ${code}" >&2
  cat "$out"
}

auto_detect_nats_network() {
  docker network ls --format '{{.Name}}' \
    | awk '/(^|_)cafenetwork$/ { print; exit }'
}

start_nats_capture() {
  [[ "$SKIP_NATS" == "1" ]] && return 0
  command -v docker >/dev/null || die "docker is required for live NATS capture (or set SKIP_NATS=1)"

  local network="${NATS_DOCKER_NETWORK:-}"
  if [[ -z "$network" ]]; then
    network="$(auto_detect_nats_network)"
  fi
  [[ -n "$network" ]] || die "could not detect cafenetwork Docker network; set NATS_DOCKER_NETWORK or SKIP_NATS=1"

  NATS_CAPTURE_FILE="$TMPDIR_SMOKE/nats-message.json"
  NATS_CAPTURE_ERR="$TMPDIR_SMOKE/nats-sub.err"

  if ! docker image inspect "$NATS_IMAGE" >/dev/null 2>&1; then
    log "Pulling ${NATS_IMAGE} before subscribing (prevents missing the one-shot event) ..."
    docker pull "$NATS_IMAGE" >/dev/null
  fi

  log "Starting NATS capture on ${NATS_SUBJECT} via network ${network} ..."
  docker run --rm --network "$network" "$NATS_IMAGE" \
    nats -s "$NATS_URL_IN_DOCKER" sub "$NATS_SUBJECT" --raw --count 1 \
    >"$NATS_CAPTURE_FILE" 2>"$NATS_CAPTURE_ERR" &
  NATS_PID="$!"
  sleep "$NATS_SUBSCRIBE_DELAY_SEC"
}

assert_nats_capture() {
  [[ "$SKIP_NATS" == "1" ]] && {
    log "SKIP_NATS=1: live NATS capture skipped."
    return 0
  }

  for _ in $(seq 1 "$NATS_TIMEOUT_SEC"); do
    if [[ -s "$NATS_CAPTURE_FILE" ]]; then
      break
    fi
    if ! kill -0 "$NATS_PID" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if [[ ! -s "$NATS_CAPTURE_FILE" ]]; then
    echo "---- nats subscriber stderr ----" >&2
    cat "$NATS_CAPTURE_ERR" >&2 || true
    die "no NATS message captured on ${NATS_SUBJECT}"
  fi

  log "NATS message captured on ${NATS_SUBJECT}:"
  jq . "$NATS_CAPTURE_FILE"
  jq -e '.event_id and .event_type and .event_version and .payload.observation' "$NATS_CAPTURE_FILE" >/dev/null \
    || die "captured NATS payload is missing expected event fields"
  if grep -q 'policy_context' "$NATS_CAPTURE_FILE"; then
    die "captured NATS payload contains forbidden policy_context"
  fi
}

run_consumer_unit_test() {
  [[ "$RUN_CONSUMER_UNIT_TEST" == "1" ]] || {
    log "RUN_CONSUMER_UNIT_TEST=0: consumer unit test skipped."
    return 0
  }
  command -v go >/dev/null || die "go is required for consumer unit test (or set RUN_CONSUMER_UNIT_TEST=0)"
  log "Running consumer contract test..."
  (
    cd "${_SCRIPT_DIR}/.."
    go test ./internal/integration/nats -run TestAssessmentRequestConsumer -v
  )
}

log "Signing in owner A (${DISCOVERY_EMAIL_A})..."
TOKEN_A="$(signin_or_signup "$DISCOVERY_EMAIL_A" "$DISCOVERY_PASSWORD_A")"
log "Signing in owner B (${DISCOVERY_EMAIL_B})..."
TOKEN_B="$(signin_or_signup "$DISCOVERY_EMAIL_B" "$DISCOVERY_PASSWORD_B")"

log "Creating wallet v1 scan for owner A..."
wallet_scan_payload="$(jq -nc --arg address "$WALLET_ADDRESS" '{address:$address}')"
wallet_scan_out="$TMPDIR_SMOKE/wallet-scan.json"
wallet_scan_code=$(http_post_json "$wallet_scan_out" "${DISCOVERY_BASE}${DISCOVERY_V1_SCAN}" "$wallet_scan_payload" \
  -H "Authorization: Bearer ${TOKEN_A}")
expect_2xx "$wallet_scan_code" "wallet scan request" "$wallet_scan_out"
WALLET_SCAN_ID="$(jq -r '.scan_id // empty' "$wallet_scan_out")"
[[ -n "$WALLET_SCAN_ID" && "$WALLET_SCAN_ID" != "null" ]] || die "wallet scan response missing scan_id"
log "Wallet scan_id: ${WALLET_SCAN_ID}"

wait_for_scan_detail "$TOKEN_A" "$DISCOVERY_V1_WALLET_SCANS" "$WALLET_SCAN_ID" "wallet"
wallet_detail="$TMPDIR_SMOKE/wallet-completed.json"
selection_json="$(build_selection_request "$wallet_detail")"
wallet_assessment_body="$(assessment_body "$WALLET_SCAN_ID" "$selection_json" "pr13g-wallet-$(date +%s)")"

start_nats_capture
log "Posting wallet assessment request (expect 202)..."
wallet_assessment_response="$(post_assessment_expect "$TOKEN_A" "$wallet_assessment_body" "202" "wallet-assessment")"
wallet_event_id="$(echo "$wallet_assessment_response" | jq -r '.event_id // empty')"
wallet_correlation_id="$(echo "$wallet_assessment_response" | jq -r '.correlation_id // empty')"
[[ -n "$wallet_event_id" && "$wallet_event_id" != "null" ]] || die "assessment response missing event_id"
[[ "$wallet_correlation_id" == "$WALLET_SCAN_ID" ]] || die "assessment correlation_id=${wallet_correlation_id}, want ${WALLET_SCAN_ID}"
assert_nats_capture

log "Checking forbidden policy_context returns 400..."
forbidden_body="$(echo "$wallet_assessment_body" | jq -c '. + {policy_context: {wallet_address: "0x1"}}')"
post_assessment_expect "$TOKEN_A" "$forbidden_body" "400" "policy-context-forbidden" >/dev/null

log "Creating TLS v1 scan for owner A..."
tls_scan_payload="$(jq -nc --arg url "$TLS_URL" '{url:$url}')"
tls_scan_out="$TMPDIR_SMOKE/tls-scan.json"
tls_scan_code=$(http_post_json "$tls_scan_out" "${DISCOVERY_BASE}${DISCOVERY_V1_SCAN}" "$tls_scan_payload" \
  -H "Authorization: Bearer ${TOKEN_A}")
expect_2xx "$tls_scan_code" "tls scan request" "$tls_scan_out"
TLS_SCAN_ID="$(jq -r '.scan_id // empty' "$tls_scan_out")"
[[ -n "$TLS_SCAN_ID" && "$TLS_SCAN_ID" != "null" ]] || die "TLS scan response missing scan_id"
log "TLS scan_id: ${TLS_SCAN_ID}"

wait_for_scan_detail "$TOKEN_A" "$DISCOVERY_V1_TLS_SCANS" "$TLS_SCAN_ID" "tls"
tls_assessment_body="$(assessment_body "$TLS_SCAN_ID" "$selection_json" "pr13g-tls-$(date +%s)")"
log "Posting TLS assessment request (expect 404, not 403)..."
post_assessment_expect "$TOKEN_A" "$tls_assessment_body" "404" "tls-assessment" >/dev/null

log "Posting foreign wallet assessment request as owner B (expect 404, not 403)..."
post_assessment_expect "$TOKEN_B" "$wallet_assessment_body" "404" "foreign-assessment" >/dev/null

run_consumer_unit_test

log "PR13g smoke completed successfully."
