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

**Gap:** Users can be blocked by a draft (`CPM_EXISTS_FOR_WALLET_TARGET`) but cannot unblock via the public API until this route exists (UI / `cafe-deploy/scripts/test-discovery-imm9-wallet-scan-w1-cpm-block.sh` step 5 skips rescan-after-delete when DELETE is not routed).

**Improvement:** Add `DELETE …/drafts?id=…` → **204** | **404** (idempotent), owner-scoped, same auth as other owner routes; document in `openapi/cpm-v1.yaml`.

**Acceptance:** After draft save, DELETE clears W1 for that `target_address`; Discovery IMM-9 smoke script completes step 5 with **200** on rescan.

**Repos:** `cafe-crypto-policy-mgt` (API + OpenAPI); Discovery / frontend consume existing contract.

---

## W1 / IMM-9b — Lookup by `target_address` in payload only (not `scan_id`)

**Context:** Internal lookup `POST /internal/policies/references/wallet-target` and `CountActiveWalletCPMContext` match owner policies/drafts by extracting a wallet address from **stored payload** (`policy_context`, `selected_wallet_policy_context`, `wallet_address`, etc.). This is intentional: Discovery must not resolve address via `policy.scan_id → Discovery detail` before allocating a new `scan_id` (**IMM-9b**).

**Implication:** Records with payload that has **no** denormalized address (e.g. PR5-style `{ "x": 1 }` with only `PolicyRecord.ScanID` set) do **not** participate in W1 for any address — rescan is not blocked. Real flows (explore/persist, UI, deploy scripts) must persist `target_address` (or equivalent) in the payload.

**Not a bug:** Do not “fix” by joining on `scan_id` and calling Discovery from CPM for W1. If product requires blocking in legacy shapes, migrate payloads or enforce address fields on `POST …/policies` / `POST …/drafts` at write time.

**Repos:** Document in `workplans/IMMUTABILITE_PR.md` / persist validation if product wants stricter guarantees; lookup logic in `cafe-crypto-policy-mgt`.
