#!/usr/bin/env bash
# wallet-scan-and-cpm-policy.sh — Discovery (JWT) → wallet scan → CBOM → CPM decisions/explore → optional persist
# Help: ./wallet-scan-and-cpm-policy.sh --help   | Requires: curl, jq
#
set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/cpm-route-paths.sh
source "${_SCRIPT_DIR}/lib/cpm-route-paths.sh"
# shellcheck source=lib/discovery-route-paths.sh
source "${_SCRIPT_DIR}/lib/discovery-route-paths.sh"

show_help() {
  local self
  self=$(basename "$0")
  cat <<EOF
Usage: ${self} [--help|-h]

No data arguments: configure everything via environment variables.

Required
  DISCOVERY_EMAIL      Discovery user email
  DISCOVERY_PASSWORD   Password
  WALLET_ADDRESS       EVM address to scan (0x…)

Optional
  DISCOVERY_BASE       cafe-discovery root URL (no trailing slash)
                       default: http://localhost:8080
  CPM_BASE             cafe-crypto-policy-mgt root URL (no trailing slash)
                       default: http://localhost:8082
  TURNSTILE_TOKEN      Turnstile token for sign-in; default: dev-pass (Discovery dev / no secret)
  POLL_INTERVAL_SEC    Delay between CBOM polls (seconds); default: 5
  POLL_MAX_ATTEMPTS    Max CBOM polling attempts; default: 60
  CPM_SKIP_AUTH=1      Omit Authorization header to CPM (local dev without CPM auth only)
  SKIP_PERSIST=1       Stop after POST .../policies/decisions/explore (skip .../cpm/policies)
  SCAN_ID_BINDING      Optional scan_id only for persist payload if you have a Discovery UUID
  EXTRA_SCAN_BODY      Extra JSON merged into POST /discovery/v1/scan; default: {}
  CURL_INSECURE=1      curl -k (self-signed TLS or unknown CA locally)
  CURL_REDIRECT=0      Disable curl redirect following (-L off)

Steps
  1) POST /auth/signin
  2) POST /discovery/v1/scan (returns scan_id + status requested)
  3) Poll GET .../discovery/cbom/{address} until 200
  4) POST .../api/cpm/v1/policies/decisions/explore with policy_context + selection_request (no top-level scan_id unless AUTH-02 is wired)
  5) POST .../api/cpm/v1/policies (unless SKIP_PERSIST=1)

Example
  export DISCOVERY_BASE=http://localhost:8080 CPM_BASE=http://localhost:8082
  export DISCOVERY_EMAIL='user@example.com' DISCOVERY_PASSWORD='secret'
  export WALLET_ADDRESS='0x742d35Cc6634C0532925a3b844Bc454e4438f44e'
  ./${self}
  # DISCOVERY_BASE / CPM_BASE default to http://localhost:8080 and http://localhost:8082 unless set.

See cafe-crypto-policy-mgt README: "Scripted Discovery → CPM flow".
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
: "${WALLET_ADDRESS:?}"

DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
CPM_BASE="${CPM_BASE:-http://localhost:8082}"

TURNSTILE_TOKEN="${TURNSTILE_TOKEN:-dev-pass}"
POLL_INTERVAL_SEC="${POLL_INTERVAL_SEC:-5}"
POLL_MAX_ATTEMPTS="${POLL_MAX_ATTEMPTS:-60}"
EXTRA_SCAN_BODY="${EXTRA_SCAN_BODY:-{}}"

command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

DISCOVERY_BASE="${DISCOVERY_BASE%/}"
CPM_BASE="${CPM_BASE%/}"

die() { echo "error: $*" >&2; exit 1; }

# No curl -f: always capture body + HTTP status (needed to debug 4xx/5xx over HTTPS).
CURL_OPTS=( -sS )
[[ "${CURL_INSECURE:-}" == "1" ]] && CURL_OPTS+=( -k )
if [[ "${CURL_REDIRECT:-1}" != "0" ]]; then
  CURL_OPTS+=( -L )
fi

# json_post URL [extra curl args...]  stdin=JSON body
# On success (2xx): response body on stdout. On failure: message + body on stderr; return 1.
json_post() {
  local url="$1"
  shift
  local tmp code
  tmp=$(mktemp)
  code=$(curl "${CURL_OPTS[@]}" "$@" \
    -o "$tmp" \
    -w "%{http_code}" \
    -X POST \
    -H 'Content-Type: application/json' \
    --data-binary @- \
    "$url") || true
  local body
  body=$(cat "$tmp" 2>/dev/null || true)
  rm -f "$tmp"

  if [[ -z "$code" || "$code" == "000" ]]; then
    echo "" >&2
    echo "error: POST $url -> no HTTP status (connection or TLS failure?)" >&2
    [[ -n "$body" ]] && { echo "response body:" >&2; printf '%s\n' "$body" >&2; }
    echo "hint: try CURL_INSECURE=1 for self-signed certs, or check URL/port." >&2
    return 1
  fi

  case "$code" in
    2??)
      printf '%s' "$body"
      return 0
      ;;
    *)
      echo "" >&2
      echo "error: POST $url -> HTTP $code" >&2
      [[ -n "$body" ]] && { echo "response body:" >&2; printf '%s\n' "$body" >&2; }
      return 1
      ;;
  esac
}

SIGNIN_PAYLOAD=$(jq -nc \
  --arg email "$DISCOVERY_EMAIL" \
  --arg password "$DISCOVERY_PASSWORD" \
  --arg tt "$TURNSTILE_TOKEN" \
  '{email:$email, password:$password, turnstile_token:$tt}')

echo "Signing in to Discovery..."
SIGNIN_BODY=$(jq -cn --argjson body "$SIGNIN_PAYLOAD" '$body' | json_post "${DISCOVERY_BASE}/auth/signin") \
  || die "sign-in failed"

JWT=$(echo "$SIGNIN_BODY" | jq -r '.token // empty')
[[ -n "$JWT" ]] || die "no token in sign-in response: $SIGNIN_BODY"

DISCOVERY_HEADERS=( -H "Authorization: Bearer ${JWT}" )
CPM_HEADERS=()
if [[ "${CPM_SKIP_AUTH:-}" != "1" ]]; then
  CPM_HEADERS=( -H "Authorization: Bearer ${JWT}" )
fi

echo "Queueing wallet scan for ${WALLET_ADDRESS} ..."
SCAN_PAYLOAD=$(jq -nc \
  --arg addr "$WALLET_ADDRESS" \
  --argjson extra "$EXTRA_SCAN_BODY" \
  '{address:$addr} + $extra')

SCAN_RESP=$(jq -cn --argjson body "$SCAN_PAYLOAD" '$body' | json_post "${DISCOVERY_BASE}${DISCOVERY_V1_SCAN}" "${DISCOVERY_HEADERS[@]}") \
  || die "queue scan failed"

echo "$SCAN_RESP" | jq .

V1_SCAN_ID=$(echo "$SCAN_RESP" | jq -r '.scan_id // empty')
if [[ -n "$V1_SCAN_ID" && -z "${SCAN_ID_BINDING:-}" ]]; then
  SCAN_ID_BINDING="$V1_SCAN_ID"
  echo "Using scan_id from v1 scan response for persist: ${SCAN_ID_BINDING}"
fi

echo "Waiting for CBOM (scanner + persistence async) ..."
CBOM=""
http_last=""
for attempt in $(seq 1 "$POLL_MAX_ATTEMPTS"); do
  # URL-encode path segment for address
  enc_addr=$(jq -rn --arg u "$WALLET_ADDRESS" '$u|@uri')
  raw=$(curl "${CURL_OPTS[@]}" "${DISCOVERY_HEADERS[@]}" -w '\n%{http_code}' "${DISCOVERY_BASE}/discovery/cbom/${enc_addr}") || true
  http_last=$(echo "$raw" | tail -n1)
  body=$(echo "$raw" | sed '$d')
  if [[ "$http_last" == "200" ]]; then
    CBOM="$body"
    break
  fi
  if [[ "$http_last" != "404" ]]; then
    die "unexpected CBOM status ${http_last}: ${body}"
  fi
  echo "  attempt ${attempt}/${POLL_MAX_ATTEMPTS}: not ready (${http_last}), sleeping ${POLL_INTERVAL_SEC}s"
  sleep "$POLL_INTERVAL_SEC"
done
[[ -n "$CBOM" ]] || die "timed out waiting for CBOM (${POLL_MAX_ATTEMPTS} attempts)"

echo "CBOM received."
echo "$CBOM" | jq '{address, type: .type, algorithm, key_exposed: .key_exposed, networks: .networks, scanned_at: .scanned_at}'

# Map CBOM → policy_context for decisions/explore (network names → chain_ids; no synthetic default chain).
POLICY_CTX=$(echo "$CBOM" | jq -c --arg wal "$WALLET_ADDRESS" '
  def chain_map:
    {
      "ethereum-mainnet": 1,
      "mainnet": 1,
      "ethereum": 1,
      "polygon": 137,
      "base": 8453,
      "arbitrum": 42161,
      "arbitrum-one": 42161,
      "optimism": 10,
      "bsc": 56,
      "avalanche": 43114,
      "sepolia": 11155111
    };
  (.networks // []) as $nets
  | ($nets | map(ascii_downcase | chain_map[.]) | map(select(. != null))) as $mapped
  | ((.chainIds // .chain_ids) | if type == "array" then . else [] end) as $direct
  | (if ($direct | length) > 0 then $direct else $mapped end) as $chain_ids
  | ((.algorithm // "") | ascii_downcase) as $alg
  | (if (($alg | test("ecdsa")) or ($alg | test("secp256k1"))) then "secp256k1_ecrecover"
     elif (($alg == "") or (.type == null)) then "secp256k1_ecrecover"
     elif ($alg != "") then $alg
     else "secp256k1_ecrecover"
     end) as $curr_alg
  | {
      wallet_address: $wal,
      wallet_type: (
        if ((.type // "") | ascii_downcase) == "eoa" then "EOA"
        elif (((.type // "") | ascii_downcase) == "smart_account")
             or (((.type // "") | ascii_downcase) == "erc4337_smart_account") then "AA"
        else (.type // "unknown")
        end
      ),
      chain_ids: $chain_ids,
      current_algorithm: $curr_alg,
      current_pq_posture: (
        if ((.nist_level // 99) <= 1) then "classical_only"
        elif ((.nist_level // 0) >= 5) then "full_pq"
        else "hybrid"
        end
      ),
      scanned_at: (.scanned_at // (now | strftime("%Y-%m-%dT%H:%M:%SZ")))
    }
')

SELECTION_JSON=$(jq -nc \
  --argjson chains "$(echo "$POLICY_CTX" | jq '.chain_ids')" \
  '
  {
    target_posture: "hybrid",
    target_chain_ids: $chains,
    require_multichain: (($chains | length) > 1),
    allow_new_wallet: false,
    address_continuity_required: true,
    key_rotation_required: true,
    recovery_required: true,
    minimum_maturity: 1,
    approval_mode: "manual"
  }
')

# Omit top-level scan_id unless CPM AUTH-02 + Discovery can-read are wired for your scan UUID.
REQUEST_BODY=$(jq -nc \
  --argjson ctx "$POLICY_CTX" \
  --argjson sel "$SELECTION_JSON" \
  '{policy_context: $ctx, selection_request: $sel}')

echo "Calling CPM decisions/explore ..."
EXPLORE_RESP=$(jq -cn --argjson body "$REQUEST_BODY" '$body' | json_post "${CPM_BASE}${CPM_POLICIES_DECISIONS_EXPLORE}" "${CPM_HEADERS[@]}") \
  || die "CPM decisions/explore failed"

echo "$EXPLORE_RESP" | jq .

TOP_POLICY=$(echo "$EXPLORE_RESP" | jq -r '.decision.selected_policy_id // empty')
[[ -n "$TOP_POLICY" ]] || die "no ranked policy returned (selected_policy_id empty). Response above."

if [[ "${SKIP_PERSIST:-}" == "1" ]]; then
  echo "SKIP_PERSIST=1 — done after explore. Selected policy: ${TOP_POLICY}"
  exit 0
fi

POLICY_ID=$(uuidgen 2>/dev/null | tr '[:upper:]' '[:lower:]' || openssl rand -hex 16)
SCAN_ID_BINDING="${SCAN_ID_BINDING:-}"

PERSIST_PAYLOAD=$(echo "$EXPLORE_RESP" | jq -nc \
  --arg wal "$WALLET_ADDRESS" \
  --arg pid "$TOP_POLICY" \
  --argjson cbom "$(echo "$CBOM" | jq -c '{address,algorithm,type,key_exposed,networks,nist_level,scanned_at}')" \
  --argjson decision "$(echo "$EXPLORE_RESP" | jq -c '.decision')" '
  {
    workflow: "wallet-scan-and-cpm-policy.sh",
    wallet_address: $wal,
    preferred_policy_id: $pid,
    cbom_snapshot: $cbom,
    cpm_decision: $decision
  }
')

PERSIST_BODY=$(jq -nc \
  --arg id "$POLICY_ID" \
  --arg scan "$SCAN_ID_BINDING" \
  --argjson payload "$PERSIST_PAYLOAD" \
  '{id:$id, scan_id:$scan, payload:$payload}')

echo "Persisting policy ${POLICY_ID} (owner-scoped store) ..."
PERSIST_RESP=$(jq -cn --argjson body "$PERSIST_BODY" '$body' | json_post "${CPM_BASE}${CPM_POLICIES}" "${CPM_HEADERS[@]}") \
  || die "CPM POST /cpm/policies failed"

echo "$PERSIST_RESP" | jq .
echo "Done. Policy id=${POLICY_ID}, selected_crypto_route=${TOP_POLICY}"
