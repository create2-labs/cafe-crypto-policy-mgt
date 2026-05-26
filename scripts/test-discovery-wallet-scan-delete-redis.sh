#!/usr/bin/env bash
# IMM-5 smoke: two wallet scans on the same address, DELETE one scan, the other
# stays listable; DELETE the last scan removes the wallet:user Redis key.
#
# Requires a running Discovery stack (backend :8080, persistence, scanner-wallet,
# Postgres, Redis) and CPM policy-reference internal API for DELETE (no 503).
set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/discovery-route-paths.sh
source "${_SCRIPT_DIR}/lib/discovery-route-paths.sh"

ADDR="${ADDR:-0x70Af6FeA3DF8a81fA71E5E5abc2989F6880CFa21}"
CHAIN_ID="${CHAIN_ID:-1}"
DISCOVERY_BASE="${DISCOVERY_BASE:-http://localhost:8080}"
# Compte dédié par défaut (évite d'autres lignes Postgres sur la même adresse, qui
# empêcheraient la suppression Redis au dernier DELETE).
EMAIL="${EMAIL:-imm5-delete-redis-$(date +%s)@example.com}"
PASSWORD="${PASSWORD:-securepassword}"
TURNSTILE="${TURNSTILE:-dev-pass}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-cafe-postgres-dev}"
REDIS_CONTAINER="${REDIS_CONTAINER:-cafe-redis-dev}"
POSTGRES_USER="${POSTGRES_USER:-cafe}"
POSTGRES_DB="${POSTGRES_DB:-cafe}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-cafe}"
SKIP_REDIS="${SKIP_REDIS:-0}"

DISCOVERY_BASE="${DISCOVERY_BASE%/}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require_cmd curl
require_cmd jq

wait_wallet_scan() {
  local scan_id="$1"
  local max="${2:-120}"
  local i=0
  while [ "$i" -lt "$max" ]; do
    local body status
    body=$(curl -fsS "${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS}/${scan_id}" \
      -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
    status=$(jq -r '.status // empty' <<<"$body")
    echo "  ${scan_id} → ${status}"
    case "$status" in
      completed|failed) return 0 ;;
    esac
    sleep 3
    i=$((i + 1))
  done
  echo "timeout waiting for ${scan_id}" >&2
  return 1
}

post_wallet_scan() {
  curl -fsS -X POST "${DISCOVERY_BASE}${DISCOVERY_V1_SCAN}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${DISCOVERY_TOKEN}" \
    -d "$(jq -nc --arg a "$ADDR" --argjson c "$CHAIN_ID" '{address:$a, chain_ids:[$c]}')"
}

list_wallet_scans() {
  curl -fsS "${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS}?address=${ADDR}&limit=50" \
    -H "Authorization: Bearer ${DISCOVERY_TOKEN}"
}

delete_wallet_scan() {
  local scan_id="$1"
  curl -sS -o /tmp/discovery-delete-wallet-scan.json -w '%{http_code}' \
    -X DELETE "${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS}/${scan_id}" \
    -H "Authorization: Bearer ${DISCOVERY_TOKEN}"
}

assert_http_delete() {
  local scan_id="$1"
  local status
  status=$(delete_wallet_scan "$scan_id")
  if [ "$status" = "204" ]; then
    return 0
  fi
  echo "ERREUR: DELETE ${scan_id} → HTTP ${status} (attendu 204)" >&2
  jq . /tmp/discovery-delete-wallet-scan.json 2>/dev/null >&2 || cat /tmp/discovery-delete-wallet-scan.json >&2
  if [ "$status" = "503" ]; then
    echo "Indice: configurer Discovery CAFE_CPM_INTERNAL_BASE_URL + CAFE_POLICY_REFERENCE_INTERNAL_SERVICE_TOKEN (CPM policy ref pour DELETE)." >&2
  fi
  return 1
}

assert_list_contains_scan_id() {
  local list_json="$1"
  local scan_id="$2"
  if ! jq -e --arg id "$scan_id" '.items[]? | select(.scan_id == $id)' <<<"$list_json" >/dev/null; then
    echo "ERREUR: scan_id ${scan_id} absent de l'historique" >&2
    return 1
  fi
}

assert_list_missing_scan_id() {
  local list_json="$1"
  local scan_id="$2"
  if jq -e --arg id "$scan_id" '.items[]? | select(.scan_id == $id)' <<<"$list_json" >/dev/null; then
    echo "ERREUR: scan_id ${scan_id} encore présent dans l'historique" >&2
    return 1
  fi
}

redis_wallet_key() {
  printf 'wallet:user:%s:%s' "$USER_ID" "$NORMALIZED_ADDR"
}

redis_available() {
  [ "$SKIP_REDIS" = "1" ] && return 1
  command -v docker >/dev/null 2>&1 || return 1
  docker ps --format '{{.Names}}' | grep -qx "$REDIS_CONTAINER"
}

postgres_available() {
  command -v docker >/dev/null 2>&1 || return 1
  docker ps --format '{{.Names}}' | grep -qx "$POSTGRES_CONTAINER"
}

# Active (non–soft-deleted) scan_results rows for owner+address; empty if Postgres unavailable.
count_postgres_wallet_rows() {
  if ! postgres_available; then
    echo ""
    return 0
  fi
  docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -A \
    -c "SELECT COUNT(*) FROM scan_results WHERE user_id = '${USER_ID}' AND lower(address) = lower('${NORMALIZED_ADDR}') AND deleted_at IS NULL;"
}

assert_redis_wallet_key() {
  local expect_exists="$1" # 1 = must exist, 0 = must not exist
  local label="$2"

  if ! redis_available; then
    echo "Redis: skip (${label}) — container ${REDIS_CONTAINER} absent ou SKIP_REDIS=1"
    return 0
  fi

  local key exists
  key=$(redis_wallet_key)
  exists=$(docker exec "$REDIS_CONTAINER" redis-cli EXISTS "$key" | tr -d '\r\n')
  echo "Redis EXISTS ${key} → ${exists} (${label})"

  if [ "$expect_exists" = "1" ] && [ "$exists" != "1" ]; then
    echo "ERREUR: clé Redis attendue présente: ${key}" >&2
    docker exec "$REDIS_CONTAINER" redis-cli KEYS "wallet:user:${USER_ID}:*" >&2 || true
    return 1
  fi
  if [ "$expect_exists" = "0" ] && [ "$exists" != "0" ]; then
    echo "ERREUR: clé Redis attendue absente: ${key}" >&2
    docker exec "$REDIS_CONTAINER" redis-cli GET "$key" >&2 || true
    return 1
  fi
  return 0
}

resolve_normalized_addr() {
  local list_json
  list_json=$(list_wallet_scans)
  NORMALIZED_ADDR=$(jq -r '.items[0].address // empty' <<<"$list_json")
  if [ -z "$NORMALIZED_ADDR" ]; then
    NORMALIZED_ADDR=$(echo "$ADDR" | tr '[:upper:]' '[:lower:]')
    echo "WARN: address normalisée depuis ADDR (${NORMALIZED_ADDR})" >&2
  else
    echo "Address normalisée (API): ${NORMALIZED_ADDR}"
  fi
  export NORMALIZED_ADDR
}

echo "Discovery base: ${DISCOVERY_BASE}"
echo "Wallet address: ${ADDR}"
echo "Chain id: ${CHAIN_ID}"

curl -sS -X POST "${DISCOVERY_BASE}/auth/signup" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" --arg t "$TURNSTILE" \
    '{email:$e, password:$p, confirm_password:$p, turnstile_token:$t}')" >/dev/null || true

SIGNIN_JSON=$(curl -fsS -X POST "${DISCOVERY_BASE}/auth/signin" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$EMAIL" --arg p "$PASSWORD" --arg t "$TURNSTILE" \
    '{email:$e, password:$p, turnstile_token:$t}')")

DISCOVERY_TOKEN=$(jq -r '.token // empty' <<<"$SIGNIN_JSON")
USER_ID=$(jq -r '.user.id // empty' <<<"$SIGNIN_JSON")

if [ -z "$DISCOVERY_TOKEN" ] || [ "$DISCOVERY_TOKEN" = "null" ]; then
  echo "ERREUR: signin sans token" >&2
  exit 1
fi
if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
  echo "ERREUR: signin sans user.id" >&2
  exit 1
fi
echo "USER_ID=${USER_ID}"

SCAN1_BODY=$(post_wallet_scan)
SCAN1=$(jq -r '.scan_id // empty' <<<"$SCAN1_BODY")
if [ -z "$SCAN1" ]; then
  echo "ERREUR: premier POST scan sans scan_id" >&2
  exit 1
fi
echo "SCAN1=${SCAN1}"
wait_wallet_scan "$SCAN1"

SCAN2_BODY=$(post_wallet_scan)
SCAN2=$(jq -r '.scan_id // empty' <<<"$SCAN2_BODY")
if [ -z "$SCAN2" ] || [ "$SCAN1" = "$SCAN2" ]; then
  echo "ERREUR: second POST scan invalide (scan_id=${SCAN2})" >&2
  exit 1
fi
echo "SCAN2=${SCAN2}"
wait_wallet_scan "$SCAN2"

LIST_JSON=$(list_wallet_scans)
echo "$LIST_JSON" | jq .
assert_list_contains_scan_id "$LIST_JSON" "$SCAN1"
assert_list_contains_scan_id "$LIST_JSON" "$SCAN2"
LIST_TOTAL=$(jq -r '.total // 0' <<<"$LIST_JSON")
if [ "$LIST_TOTAL" != "2" ]; then
  echo "ERREUR: historique déjà pollué pour cette adresse (total=${LIST_TOTAL}, attendu 2 pour ce smoke)." >&2
  echo "Ce compte a d'autres scans Postgres actifs : la clé Redis wallet:user ne sera pas supprimée au DELETE du 2e scan de ce run." >&2
  echo "Relance avec un compte neuf, par ex. :" >&2
  echo "  EMAIL=\"imm5-delete-redis-\$(date +%s)@example.com\" $0" >&2
  exit 1
fi
echo "OK: deux scans listables pour la même adresse (total=2)"

resolve_normalized_addr
assert_redis_wallet_key 1 "après 2 scans terminés"

echo
echo "DELETE SCAN1 (SCAN2 doit rester listable, clé Redis conservée)..."
assert_http_delete "$SCAN1"

LIST_AFTER_DEL1=$(list_wallet_scans)
echo "$LIST_AFTER_DEL1" | jq .
assert_list_missing_scan_id "$LIST_AFTER_DEL1" "$SCAN1"
assert_list_contains_scan_id "$LIST_AFTER_DEL1" "$SCAN2"
echo "OK: SCAN2 toujours listable après DELETE SCAN1"

SCAN1_GET_STATUS=$(curl -sS -o /tmp/discovery-get-scan1.json -w '%{http_code}' \
  "${DISCOVERY_BASE}${DISCOVERY_V1_WALLET_SCANS}/${SCAN1}" \
  -H "Authorization: Bearer ${DISCOVERY_TOKEN}")
if [ "$SCAN1_GET_STATUS" != "404" ]; then
  echo "ERREUR: GET SCAN1 après DELETE → HTTP ${SCAN1_GET_STATUS} (attendu 404)" >&2
  jq . /tmp/discovery-get-scan1.json 2>/dev/null >&2 || true
  exit 1
fi
echo "OK: GET SCAN1 → 404"

assert_redis_wallet_key 1 "après DELETE SCAN1 (SCAN2 reste en Postgres)"

echo
echo "DELETE SCAN2 (dernière ligne pour l'adresse → clé Redis supprimée)..."
assert_http_delete "$SCAN2"

LIST_AFTER_DEL2=$(list_wallet_scans)
echo "$LIST_AFTER_DEL2" | jq .
assert_list_missing_scan_id "$LIST_AFTER_DEL2" "$SCAN1"
assert_list_missing_scan_id "$LIST_AFTER_DEL2" "$SCAN2"
echo "OK: plus aucun des deux scan_id dans l'historique filtré"

PG_REMAINING=""
if postgres_available; then
  PG_REMAINING=$(count_postgres_wallet_rows | tr -d '\r\n')
  echo "Postgres lignes actives (deleted_at IS NULL) pour ${NORMALIZED_ADDR}: ${PG_REMAINING}"
fi
if [ -n "$PG_REMAINING" ] && [ "$PG_REMAINING" != "0" ]; then
  echo "ERREUR: il reste ${PG_REMAINING} ligne(s) Postgres pour cette adresse — IMM-5 conserve la clé Redis wallet:user (comportement attendu)." >&2
  echo "Ce run n'a supprimé que SCAN1/SCAN2 ; d'autres scan_id historiques existent encore pour 0x…" >&2
  echo "Relance avec EMAIL neuf (voir ci-dessus) ou supprime les autres scans avant ce smoke." >&2
  docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "SELECT id, status, deleted_at IS NULL AS active, created_at FROM scan_results WHERE user_id = '${USER_ID}' AND lower(address) = lower('${NORMALIZED_ADDR}') ORDER BY created_at DESC LIMIT 20;" >&2 || true
  exit 1
fi

assert_redis_wallet_key 0 "après DELETE SCAN2 (dernière ligne Postgres active pour l'adresse)"

echo
echo "OK: IMM-5 delete + Redis — 2 scans même adresse, DELETE partiel conserve liste + Redis, DELETE final supprime la clé."
