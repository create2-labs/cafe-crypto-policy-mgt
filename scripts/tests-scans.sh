#!/usr/bin/env bash
set -euo pipefail

ADDR="${ADDR:-0x70Af6FeA3DF8a81fA71E5E5abc2989F6880CFa21}"
CHAIN_ID="${CHAIN_ID:-1}"
DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
EMAIL="${EMAIL:-user2@example.com}"
PASSWORD="${PASSWORD:-securepassword}"
TURNSTILE="${TURNSTILE:-dev-pass}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-cafe-postgres-dev}"
POSTGRES_USER="${POSTGRES_USER:-cafe}"
POSTGRES_DB="${POSTGRES_DB:-cafe}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-cafe}"

DISCOVERY_BASE="${DISCOVERY_BASE%/}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require_cmd curl
require_cmd jq

##############################################################################
# Helper
# wait_wallet_scan <scan_id> [<max_seconds>]
##############################################################################
wait_wallet_scan() {
  local scan_id="$1"
  local max="${2:-120}"
  local i=0
  while [ "$i" -lt "$max" ]; do
    local body status
    body=$(curl -fsS "${DISCOVERY_BASE}/discovery/v1/wallets/scans/${scan_id}" \
      -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
    status=$(jq -r '.status // empty' <<<"$body")
    echo "  ${scan_id} → ${status}"
    case "$status" in
      completed|failed) echo "$body" | jq .; return 0 ;;
    esac
    sleep 3
    i=$((i + 1))
  done
  echo "timeout waiting for ${scan_id}" >&2
  return 1
}

post_wallet_scan() {
  curl -fsS -X POST "${DISCOVERY_BASE}/discovery/v1/scan" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${DISCOVERY_TOKEN}" \
    -d "$(jq -nc --arg a "$ADDR" --argjson c "$CHAIN_ID" '{address:$a, chain_ids:[$c]}')"
}

assert_created_at_not_zero() {
  local list_json="$1"
  local zero_count
  zero_count=$(jq '[.items[]? | select(.created_at == "0001-01-01T00:00:00Z" or .created_at == "" or .created_at == null)] | length' <<<"$list_json")
  if [ "$zero_count" -ne 0 ]; then
    echo "ERREUR: ${zero_count} scan(s) avec created_at à zéro" >&2
    return 1
  fi
}

assert_list_contains_scan_id() {
  local list_json="$1"
  local scan_id="$2"
  if ! jq -e --arg id "$scan_id" '.items[]? | select(.scan_id == $id)' <<<"$list_json" >/dev/null; then
    echo "ERREUR: scan_id ${scan_id} absent de l'historique" >&2
    return 1
  fi
}

assert_chain_filter_before_pagination() {
  local filtered_json="$1"
  local expected_scan_id="$2"
  local total item_count actual_scan_id

  total=$(jq -r '.total' <<<"$filtered_json")
  item_count=$(jq -r '.items | length' <<<"$filtered_json")
  actual_scan_id=$(jq -r '.items[0].scan_id // empty' <<<"$filtered_json")

  if [ "$total" -lt 2 ]; then
    echo "ERREUR: filtre chain_id incomplet, total=${total}" >&2
    return 1
  fi
  if [ "$item_count" -ne 1 ]; then
    echo "ERREUR: pagination chain_id attend 1 item, got ${item_count}" >&2
    return 1
  fi
  if [ "$actual_scan_id" != "$expected_scan_id" ]; then
    echo "ERREUR: address+chain_id+limit=1&offset=1 devrait retourner ${expected_scan_id}, got ${actual_scan_id}" >&2
    return 1
  fi
}

assert_chain_id_requires_address() {
  local status
  status=$(curl -sS -o /tmp/discovery-chain-only-response.json -w '%{http_code}' \
    "${DISCOVERY_BASE}/discovery/v1/wallets/scans?chain_id=${CHAIN_ID}" \
    -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
  if [ "$status" != "400" ]; then
    echo "ERREUR: chain_id sans address devrait retourner 400, got ${status}" >&2
    cat /tmp/discovery-chain-only-response.json >&2 || true
    return 1
  fi
}

print_postgres_rows_if_available() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker absent: skip vérification Postgres"
    return 0
  fi
  if ! docker ps --format '{{.Names}}' | grep -qx "$POSTGRES_CONTAINER"; then
    echo "container ${POSTGRES_CONTAINER} absent: skip vérification Postgres"
    return 0
  fi

  echo
  echo "Postgres rows for ${ADDR}:"
  docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "SELECT id, address, status, risk_score, created_at FROM scan_results WHERE lower(address) = lower('${ADDR}') ORDER BY created_at DESC, id DESC;"
}


##############################################################################
# main
##############################################################################

echo "Discovery base: ${DISCOVERY_BASE}"
echo "Wallet address: ${ADDR}"
echo "Chain id: ${CHAIN_ID}"

signup_body=$(curl -sS -X POST "${DISCOVERY_BASE}/auth/signup" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" --arg t "$TURNSTILE" \
    '{email:$e, password:$p, confirm_password:$p, turnstile_token:$t}')")
echo "signup response:"
jq . <<<"$signup_body" || echo "$signup_body"

DISCOVERY_TOKEN=$(curl -fsS -X POST "${DISCOVERY_BASE}/auth/signin" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" --arg t "$TURNSTILE" \
    '{email:$e, password:$p, turnstile_token:$t}')" | jq -r '.token')
export DISCOVERY_TOKEN

if [ -z "$DISCOVERY_TOKEN" ] || [ "$DISCOVERY_TOKEN" = "null" ]; then
  echo "ERREUR: signin sans token" >&2
  exit 1
fi

SCAN1_BODY=$(post_wallet_scan)
echo "$SCAN1_BODY" | jq .
SCAN1=$(jq -r '.scan_id // empty' <<<"$SCAN1_BODY")
if [ -z "$SCAN1" ]; then
  echo "ERREUR: premier POST scan sans scan_id" >&2
  exit 1
fi

echo "SCAN1=${SCAN1}"
wait_wallet_scan "$SCAN1"

SCAN1_DETAIL=$(curl -fsS "${DISCOVERY_BASE}/discovery/v1/wallets/scans/${SCAN1}" \
  -H "Authorization: Bearer ${DISCOVERY_TOKEN}" \
  | jq '{scan_id, status, result}')
echo "$SCAN1_DETAIL"

SCAN2_BODY=$(post_wallet_scan)
echo "$SCAN2_BODY" | jq .
SCAN2=$(jq -r '.scan_id // empty' <<<"$SCAN2_BODY")
if [ -z "$SCAN2" ]; then
  echo "ERREUR: second POST scan sans scan_id" >&2
  exit 1
fi

if [ "$SCAN1" = "$SCAN2" ]; then
  echo "ERREUR: même scan_id" >&2
  exit 1
fi
echo "OK: scan_id distincts"

echo "SCAN2=${SCAN2}"
wait_wallet_scan "$SCAN2"

echo
echo "Detail SCAN1 after SCAN2 completion:"
curl -fsS "${DISCOVERY_BASE}/discovery/v1/wallets/scans/${SCAN1}" \
  -H "Authorization: Bearer ${DISCOVERY_TOKEN}" | jq .

echo
echo "Wallet scan history:"
LIST_JSON=$(curl -fsS "${DISCOVERY_BASE}/discovery/v1/wallets/scans?address=${ADDR}&limit=50" \
  -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
echo "$LIST_JSON" | jq .

total=$(jq -r '.total' <<<"$LIST_JSON")
if [ "$total" -lt 2 ]; then
  echo "ERREUR: historique incomplet, total=${total}" >&2
  exit 1
fi
assert_created_at_not_zero "$LIST_JSON"
assert_list_contains_scan_id "$LIST_JSON" "$SCAN1"
assert_list_contains_scan_id "$LIST_JSON" "$SCAN2"
echo "OK: historique multi-lignes, scan_id présents et created_at non zéro"

echo
echo "Wallet scan history filtered by address + chain_id, with pagination:"
FILTERED_JSON=$(curl -fsS "${DISCOVERY_BASE}/discovery/v1/wallets/scans?address=${ADDR}&chain_id=${CHAIN_ID}&limit=1&offset=1" \
  -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
echo "$FILTERED_JSON" | jq .
assert_chain_filter_before_pagination "$FILTERED_JSON" "$SCAN1"
echo "OK: address+chain_id filtre l'historique avant pagination"

echo
echo "Wallet scan history with chain_id only must fail:"
assert_chain_id_requires_address
echo "OK: chain_id sans address retourne 400"

print_postgres_rows_if_available
