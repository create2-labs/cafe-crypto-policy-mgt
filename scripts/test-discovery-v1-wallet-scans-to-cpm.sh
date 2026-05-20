#!/usr/bin/env bash
# test-discovery-v1-wallet-scans-to-cpm.sh
# Option A smoke: Discovery AuthN ->
#   GET /discovery/v1/wallets/scans -> GET …/wallets/scans/{scan_id} ->
#   CPM POST …/policies/decisions/explore -> optional persist.
set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"
# shellcheck source=lib/cpm-route-paths.sh
source "${_SCRIPT_DIR}/lib/cpm-route-paths.sh"
# shellcheck source=lib/discovery-route-paths.sh
source "${_SCRIPT_DIR}/lib/discovery-route-paths.sh"

show_help() {
  local self
  self=$(basename "$0")
  cat <<EOF
Usage: ${self} [--help|-h]

No positional data arguments: configure behavior via environment variables.

Required
  DISCOVERY_EMAIL                 Discovery user email
  DISCOVERY_PASSWORD              Discovery password

Optional
  DISCOVERY_BASE                   Discovery root URL (no trailing slash)
                                  default: http://localhost:8080
  CPM_BASE                        CPM root URL (no trailing slash)
                                  default: http://localhost:8082
  DISCOVERY_V1_WALLET_SCANS_LIST_PATH
                                  Wallet scans list path (appended to DISCOVERY_BASE)
                                  default: /discovery/v1/wallets/scans (direct backend :8080)
                                  Edge (nginx): /api/discovery/v1/wallets/scans
  DISCOVERY_WALLET_CONTEXTS_PATH  Deprecated alias for DISCOVERY_V1_WALLET_SCANS_LIST_PATH
  SCAN_ID                         Select one authorized scan by scan_id (UUID)
  CPM_EXPLORE_PATH                CPM explore endpoint path
                                  default: /api/cpm/v1/policies/decisions/explore
  CPM_PERSIST_PATH                CPM persist endpoint path
                                  default: /api/cpm/v1/policies
  TURNSTILE_TOKEN                 Sign-in token; default: dev-pass
  TARGET_POSTURE                  selection_request.target_posture
                                  default: hybrid
  ALLOW_NEW_WALLET                true/false; default: false
  ADDRESS_CONTINUITY_REQUIRED     true/false; default: true
  KEY_ROTATION_REQUIRED           true/false; default: true
  RECOVERY_REQUIRED               true/false; default: true
  MINIMUM_MATURITY                integer; default: 1
  APPROVAL_MODE                   string; default: manual
  SKIP_PERSIST=1                  Stop after CPM explore (default behavior)
  CPM_SKIP_AUTH=1                 Local dev escape hatch: omit Authorization to CPM.
                                  Do not use in staging/prod.

Security / transport
  CURL_INSECURE=1                 curl -k for local self-signed TLS
  CURL_REDIRECT=0                 default is 0 to avoid forwarding auth headers across redirects

Examples
  export DISCOVERY_EMAIL='user@example.com' DISCOVERY_PASSWORD='secret'
  export DISCOVERY_BASE='http://localhost:8080' CPM_BASE='http://localhost:8082'
  ./${self}

  SCAN_ID='your-scan-id' ./${self}
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

: "${DISCOVERY_EMAIL:?}"
: "${DISCOVERY_PASSWORD:?}"

DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
CPM_BASE="${CPM_BASE:-http://localhost:8082}"
DISCOVERY_V1_WALLET_SCANS_LIST_PATH="${DISCOVERY_V1_WALLET_SCANS_LIST_PATH:-${DISCOVERY_WALLET_CONTEXTS_PATH:-${DISCOVERY_V1_WALLET_SCANS}}}"
CPM_EXPLORE_PATH="${CPM_EXPLORE_PATH:-${CPM_POLICIES_DECISIONS_EXPLORE}}"
CPM_PERSIST_PATH="${CPM_PERSIST_PATH:-${CPM_POLICIES}}"

TURNSTILE_TOKEN="${TURNSTILE_TOKEN:-dev-pass}"

TARGET_POSTURE="${TARGET_POSTURE:-hybrid}"
ALLOW_NEW_WALLET="${ALLOW_NEW_WALLET:-false}"
ADDRESS_CONTINUITY_REQUIRED="${ADDRESS_CONTINUITY_REQUIRED:-true}"
KEY_ROTATION_REQUIRED="${KEY_ROTATION_REQUIRED:-true}"
RECOVERY_REQUIRED="${RECOVERY_REQUIRED:-true}"
MINIMUM_MATURITY="${MINIMUM_MATURITY:-1}"
APPROVAL_MODE="${APPROVAL_MODE:-manual}"

SKIP_PERSIST="${SKIP_PERSIST:-1}"

command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

DISCOVERY_BASE="${DISCOVERY_BASE%/}"
CPM_BASE="${CPM_BASE%/}"

die() { echo "error: $*" >&2; exit 1; }

bool_to_json() {
  local v
  v=$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')
  case "$v" in
    1|true|yes|y|on) printf 'true' ;;
    0|false|no|n|off|"") printf 'false' ;;
    *) die "invalid boolean value '$1' (expected true/false style value)" ;;
  esac
}

CURL_OPTS=( -sS )
[[ "${CURL_INSECURE:-}" == "1" ]] && CURL_OPTS+=( -k )
if [[ "${CURL_REDIRECT:-0}" != "0" ]]; then
  CURL_OPTS+=( -L )
fi

http_json() {
  local method="$1"
  local url="$2"
  shift 2
  local tmp code
  tmp=$(mktemp)
  code=$(curl "${CURL_OPTS[@]}" "$@" \
    -o "$tmp" \
    -w "%{http_code}" \
    -X "$method" \
    "$url") || true
  local body
  body=$(jq -Rs '.' <"$tmp" | jq -r '.')
  rm -f "$tmp"

  if [[ -z "$code" || "$code" == "000" ]]; then
    echo "error: ${method} ${url} -> no HTTP status (connection reset, refused, or TLS failure?)" >&2
    [[ -n "$body" ]] && { echo "response body:" >&2; printf '%s\n' "$body" >&2; }
    echo "hint: ensure Discovery is listening (e.g. docker compose exposes :8080); try: curl -svS '${DISCOVERY_BASE}/health'." >&2
    echo "hint: sign-in is ${DISCOVERY_BASE}/auth/signin (not under /api). Nginx edge in cafe-deploy proxies /api/* only, not /auth." >&2
    echo "hint: CURL_INSECURE=1 only helps HTTPS with untrusted certs; it does not fix connection reset when the port is wrong or the process is down." >&2
    return 1
  fi

  case "$code" in
    2??)
      printf '%s' "$body"
      return 0
      ;;
    *)
      echo "error: ${method} ${url} -> HTTP $code" >&2
      [[ -n "$body" ]] && { echo "response body:" >&2; printf '%s\n' "$body" >&2; }
      return 1
      ;;
  esac
}

json_post() {
  local url="$1"
  shift
  http_json POST "$url" -H 'Content-Type: application/json' "$@" --data-binary @-
}

json_get() {
  local url="$1"
  shift
  http_json GET "$url" "$@"
}

SIGNIN_PAYLOAD=$(jq -nc \
  --arg email "$DISCOVERY_EMAIL" \
  --arg password "$DISCOVERY_PASSWORD" \
  --arg tt "$TURNSTILE_TOKEN" \
  '{email:$email, password:$password, turnstile_token:$tt}')

echo "Signing in to Discovery..."
SIGNIN_BODY=$(jq -cn --argjson body "$SIGNIN_PAYLOAD" '$body' | json_post "${DISCOVERY_BASE}/auth/signin") \
  || die "sign-in failed"

JWT=$(printf '%s' "$SIGNIN_BODY" | jq -r '.token // empty')
[[ -n "$JWT" ]] || die "no token in sign-in response"

DISCOVERY_HEADERS=( -H "Authorization: Bearer ${JWT}" )
CPM_HEADERS=()
if [[ "${CPM_SKIP_AUTH:-}" != "1" ]]; then
  CPM_HEADERS=( -H "Authorization: Bearer ${JWT}" )
fi

CONTEXTS_URL="${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS_LIST_PATH}"
echo "Loading authenticated wallet scans from Discovery v1 (GET ${DISCOVERY_V1_WALLET_SCANS_LIST_PATH})..."
CONTEXTS_RAW=$(json_get "$CONTEXTS_URL" "${DISCOVERY_HEADERS[@]}") \
  || die "failed to load wallet scans"

CONTEXTS=$(printf '%s' "$CONTEXTS_RAW" | jq -c '
  def as_contexts:
    if type == "array" then .
    elif type == "object" then (.contexts // .items // .results // .data // [.] )
    else [] end;
  as_contexts
  | map({
      scan_id: (.scanId // .scan_id // empty),
      wallet_address: (.walletAddress // .wallet_address // .target_address // .address // empty),
      wallet_type: (.walletType // .wallet_type // .account_kind // "unknown"),
      chain_ids: ((.chainIds // .chain_ids) | if type == "array" then . else [] end),
      current_pq_posture: (.currentPQPosture // .current_pq_posture // "unknown"),
      scanned_at: (.scannedAt // .scanned_at // .created_at // .observed_at // empty),
      status: (.status // .scanStatus // .scan_status // "unknown"),
      raw: .
    })
  | map(select(.scan_id != ""))
')

ELIGIBLE_CONTEXTS=$(printf '%s' "$CONTEXTS" | jq -c '
  map(select(((.status // "") | ascii_downcase) == "completed"))
')

eligible_count=$(printf '%s' "$ELIGIBLE_CONTEXTS" | jq 'length')
if [[ "$eligible_count" -eq 0 ]]; then
  die "no eligible completed wallet scans found. Run a wallet scan first (POST ${DISCOVERY_V1_SCAN})."
fi

if [[ -n "${SCAN_ID:-}" ]]; then
  SELECTED_CONTEXT=$(printf '%s' "$ELIGIBLE_CONTEXTS" | jq -c --arg sid "$SCAN_ID" '
    map(select(.scan_id == $sid)) | .[0] // empty
  ')
  if [[ -z "$SELECTED_CONTEXT" ]]; then
    echo "Available eligible scan_id values:" >&2
    printf '%s' "$ELIGIBLE_CONTEXTS" | jq -r '.[].scan_id' >&2
    die "SCAN_ID '${SCAN_ID}' was not found in your authenticated eligible wallet scans"
  fi
else
  if [[ "$eligible_count" -eq 1 ]]; then
    SELECTED_CONTEXT=$(printf '%s' "$ELIGIBLE_CONTEXTS" | jq -c '.[0]')
  else
    echo "Multiple eligible wallet scans found:"
    printf '%s' "$ELIGIBLE_CONTEXTS" | jq -r '
      (["scan_id","wallet_address","wallet_type","chain_ids","current_pq_posture","scanned_at","status"] | @tsv),
      (.[] | [
        .scan_id,
        .wallet_address,
        .wallet_type,
        ((.chain_ids // []) | tostring),
        (.current_pq_posture // ""),
        (.scanned_at // ""),
        (.status // "")
      ] | @tsv)
    '
    die "multiple eligible wallet scans; rerun with SCAN_ID=<id>"
  fi
fi

SELECTED_SCAN_ID=$(printf '%s' "$SELECTED_CONTEXT" | jq -r '.scan_id // empty')
[[ -n "$SELECTED_SCAN_ID" ]] || die "selected wallet scan is missing scan_id"
SELECTED_CHAINS=$(printf '%s' "$SELECTED_CONTEXT" | jq -c '.chain_ids // []')

echo "Selected wallet scan:"
printf '%s\n' "$SELECTED_CONTEXT" | jq '{scan_id,wallet_address,wallet_type,chain_ids,current_pq_posture,scanned_at,status}'

enc_scan_id=$(jq -rn --arg u "$SELECTED_SCAN_ID" '$u|@uri')
DETAIL_URL="${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS}/${enc_scan_id}"
echo "Loading wallet scan detail (GET ${DISCOVERY_V1_WALLET_SCANS}/{scan_id})..."
SCAN_DETAIL=$(json_get "$DETAIL_URL" "${DISCOVERY_HEADERS[@]}") \
  || die "failed to load wallet scan detail for scan_id=${SELECTED_SCAN_ID}"
POLICY_CONTEXT_JSON=$(printf '%s' "$SCAN_DETAIL" | jq -c '
  {
    scan_id: (.scan_id // ""),
    status: (.status // ""),
    result: (.result // empty)
  }
  | select(.scan_id != "" and (.result | type) == "object")
')
[[ -n "$POLICY_CONTEXT_JSON" ]] || die "wallet scan detail missing result (scan may not be terminal yet)"

SELECTION_JSON=$(jq -nc \
  --arg target_posture "$TARGET_POSTURE" \
  --argjson target_chain_ids "$SELECTED_CHAINS" \
  --argjson allow_new_wallet "$(bool_to_json "$ALLOW_NEW_WALLET")" \
  --argjson address_continuity_required "$(bool_to_json "$ADDRESS_CONTINUITY_REQUIRED")" \
  --argjson key_rotation_required "$(bool_to_json "$KEY_ROTATION_REQUIRED")" \
  --argjson recovery_required "$(bool_to_json "$RECOVERY_REQUIRED")" \
  --arg minimum_maturity "$MINIMUM_MATURITY" \
  --arg approval_mode "$APPROVAL_MODE" '
  {
    target_posture: $target_posture,
    target_chain_ids: $target_chain_ids,
    require_multichain: (($target_chain_ids | length) > 1),
    allow_new_wallet: $allow_new_wallet,
    address_continuity_required: $address_continuity_required,
    key_rotation_required: $key_rotation_required,
    recovery_required: $recovery_required,
    minimum_maturity: ($minimum_maturity | tonumber),
    approval_mode: $approval_mode
  }
')

REQUEST_BODY=$(jq -nc \
  --arg scan_id "$SELECTED_SCAN_ID" \
  --argjson policy_context "$POLICY_CONTEXT_JSON" \
  --argjson sel "$SELECTION_JSON" '
  {
    scan_id: $scan_id,
    policy_context: $policy_context,
    selection_request: $sel
  }
')

echo "Calling CPM decisions/explore ..."
EXPLORE_RESP=$(jq -cn --argjson body "$REQUEST_BODY" '$body' | json_post "${CPM_BASE}${CPM_EXPLORE_PATH}" "${CPM_HEADERS[@]}") \
  || die "CPM decisions/explore failed"
printf '%s\n' "$EXPLORE_RESP" | jq .

TOP_POLICY=$(printf '%s' "$EXPLORE_RESP" | jq -r '.decision.selected_policy_id // empty')
[[ -n "$TOP_POLICY" ]] || die "no ranked policy returned (selected_policy_id empty)"

if [[ "$SKIP_PERSIST" == "1" ]]; then
  echo "SKIP_PERSIST=1 — done after explore. scan_id=${SELECTED_SCAN_ID} selected_policy=${TOP_POLICY}"
  exit 0
fi

POLICY_ID=$(uuidgen 2>/dev/null | tr '[:upper:]' '[:lower:]' || openssl rand -hex 16)
PERSIST_PAYLOAD=$(jq -nc \
  --arg workflow "${_SCRIPT_NAME}" \
  --arg selected_policy_id "$TOP_POLICY" \
  --arg selected_scan_id "$SELECTED_SCAN_ID" \
  --argjson selected_wallet_policy_context "$SELECTED_CONTEXT" \
  --argjson cpm_decision "$(printf '%s' "$EXPLORE_RESP" | jq -c '.decision')" '
  {
    workflow: $workflow,
    selected_scan_id: $selected_scan_id,
    selected_policy_id: $selected_policy_id,
    selected_wallet_policy_context: $selected_wallet_policy_context,
    cpm_decision: $cpm_decision
  }
')

PERSIST_BODY=$(jq -nc \
  --arg id "$POLICY_ID" \
  --arg scan "$SELECTED_SCAN_ID" \
  --argjson payload "$PERSIST_PAYLOAD" \
  '{id:$id, scan_id:$scan, payload:$payload}')

echo "Persisting policy ${POLICY_ID} for scan_id=${SELECTED_SCAN_ID} ..."
PERSIST_RESP=$(jq -cn --argjson body "$PERSIST_BODY" '$body' | json_post "${CPM_BASE}${CPM_PERSIST_PATH}" "${CPM_HEADERS[@]}") \
  || die "CPM persist failed"
printf '%s\n' "$PERSIST_RESP" | jq .
echo "Done. policy_id=${POLICY_ID} selected_crypto_route=${TOP_POLICY}"
