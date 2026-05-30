# Cafe CPM — backlog

Items deferred; not blocking current IMM work unless noted.

## IMM-9b follow-up — Unify wallet address normalization

**Context:** IMM-9b introduced `persistence.NormalizeWalletTargetAddress` (`internal/persistence/wallet_target.go`). NATS assessment code still uses a parallel `normalizeHexAddress` in `internal/integration/nats/assessment_consumer.go` (`normalizeWalletSubjectID`).

**Improvement:** Reuse `persistence.NormalizeWalletTargetAddress` from the NATS consumer (map `error` → `ok bool` at the call site if needed). One canonical normalization rule for CPM (0x prefix, lowercase, 20-byte hex).

**Acceptance:** Existing NATS / assessment tests pass; no behaviour change for valid `wallet:0x…` subject IDs.

**Repos:** `cafe-crypto-policy-mgt` only.

---

## IMM-9b follow-up — Factor internal policy-reference handlers

**Context:** `internal/app/policy_reference_internal.go` (`POST /internal/policies/references/scan`) and `internal/app/policy_reference_wallet_target_internal.go` (`POST …/references/wallet-target`) share the same boilerplate: read/limit body, JSON decode, `user_id` + `tenant_id` → `authz.Principal`, validate principal, map persistence errors.

**Improvement:** Extract shared helpers (e.g. `readInternalServiceJSON`, `principalFromInternalUserTenant`) without changing HTTP contracts or auth route inventory.

**Acceptance:** `policy_reference_internal_test.go` and `policy_reference_wallet_target_internal_test.go` unchanged in behaviour; less duplication for future internal lookups.

**Repos:** `cafe-crypto-policy-mgt` only.

---

## W1 / workplan — Implement `DELETE /api/cpm/v1/drafts?id=…`

**Context:** **WORKPLAN_API.md** §2.4 / §4.4 and **IMMUTABILITE** W1 require platform draft removal to unblock `POST /api/discovery/v1/scan` after a draft blocks rescan. IMM-9b already **counts** drafts in `CountActiveWalletCPMContext` (409 path works). **`owner_routes.go`** exposes `POST` and `GET` on `/api/cpm/v1/drafts` only — no `DELETE` handler yet.

**Plan (PR sequence):** **IMM-W1-1…3** in [`workplans/IMMUTABILITE_PR.md`](./workplans/IMMUTABILITE_PR.md) — store `DeleteDraft` → HTTP route → tests + deploy smoke step 5.

**Gap:** Users can be blocked by a draft (`CPM_EXISTS_FOR_WALLET_TARGET`) but cannot unblock via the public API until **IMM-W1-2** ships.

**Acceptance:** After draft save, DELETE clears W1 for that `target_address`; Discovery IMM-9 smoke script completes step 5 with **200** on rescan.

**Repos:** `cafe-crypto-policy-mgt` (API + OpenAPI); Discovery / frontend consume existing contract.

---

## W1 / IMM-9b — Lookup by `target_address` in payload only (not `scan_id`)

**Context:** Internal lookup `POST /internal/policies/references/wallet-target` and `CountActiveWalletCPMContext` match owner policies/drafts by extracting a wallet address from **stored payload** (`policy_context`, `selected_wallet_policy_context`, `wallet_address`, etc.). This is intentional: Discovery must not resolve address via `policy.scan_id → Discovery detail` before allocating a new `scan_id` (**IMM-9b**).

**Implication:** Records with payload that has **no** denormalized address (e.g. PR5-style `{ "x": 1 }` with only `PolicyRecord.ScanID` set) do **not** participate in W1 for any address — rescan is not blocked. Real flows (explore/persist, UI, deploy scripts) must persist `target_address` (or equivalent) in the payload.

**Not a bug:** Do not “fix” by joining on `scan_id` and calling Discovery from CPM for W1. If product requires blocking in legacy shapes, migrate payloads or enforce address fields on `POST …/policies` / `POST …/drafts` at write time.

**Repos:** Document in `workplans/IMMUTABILITE_PR.md` / persist validation if product wants stricter guarantees; lookup logic in `cafe-crypto-policy-mgt`.

---

## IMM-10 follow-up — Centraliser les chemins Discovery (upstream vs public)

**Context:** IMM-10 (W7/W2) et PR13g appellent Discovery avec des littéraux dupliqués, p.ex. :

- `internal/app/auth.go` — `GET …/discovery/v1/wallets/scans` (liste `limit=1`, `latest=true`)
- `internal/app/assessment_request.go` — `GET …/discovery/v1/wallets/scans/{scan_id}` (détail)
- `internal/app/authz_scan_test.go` — mêmes paths dans les mocks HTTP

Le merge `main` ↔ `imm10` a divergé sur **`/discovery/v1/...`** (appel serveur → backend `:8080`) vs **`/api/discovery/v1/...`** (contrat public / edge WORKPLAN). Pour `CAFE_DISCOVERY_HTTP_BASE` pointant sur le backend Discovery, le couple correct est **`/discovery/v1`** (aligné Fiber / compose), pas `/api/discovery/v1` sur `:8080` direct.

**Amélioration:** Introduire des constantes ou helpers CPM (p.ex. package `discoverypaths` / extension de `cpmroutes`, ou petit module partagé) :

- `DiscoveryUpstreamWalletScans` = `/discovery/v1/wallets/scans`
- `DiscoveryUpstreamWalletScanByID(scanID)` = `/discovery/v1/wallets/scans/{scan_id}`
- Optionnel, nommées à part : chemins **public** `/api/discovery/v1/...` si un jour CPM cible l’edge — ne pas les confondre avec l’upstream.

Réutiliser ces symboles dans `auth.go`, `assessment_request.go` et les tests (mocks) pour éviter une prochaine régression au merge.

**Acceptance:** Un seul endroit définit les paths upstream ; `go test ./internal/app` inchangé ; smoke `cafe-deploy/scripts/test-cpm-imm10-wallet-scan-w7-w2-guards.sh` OK avec `CAFE_DISCOVERY_HTTP_BASE` → backend.

**Repos:** `cafe-crypto-policy-mgt` ; aligner doc `internal/config` et éventuellement `scripts/lib/discovery-route-paths.sh` (bash) sur le même vocabulaire.
