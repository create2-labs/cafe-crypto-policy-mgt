# Cafe CPM — backlog

Items deferred; not blocking current IMM work unless noted.

## Livré (IMM-W1 — ne pas réimplémenter DELETE)

**`DELETE /api/cpm/v1/drafts?id=…`** — série **IMM-W1-1…3** mergée ([#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41)–[#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44)) dans **`cafe-crypto-policy-mgt`** (pas dans Discovery).

| Couche | Rôle |
|--------|------|
| **Discovery IMM-9** ([#76](https://github.com/create2-labs/cafe-discovery/pull/76)) | `POST /scan` → **409** `CPM_EXISTS_FOR_WALLET_TARGET` si policy/draft (lookup CPM **IMM-9b**) |
| **CPM IMM-W1** | `OwnerScopedStore.DeleteDraft` + route **`DELETE /api/cpm/v1/drafts?id=…`** → **204** \| **404** |

**Validation smoke (stack locale :8080 / :8082) :**

- `cafe-deploy/scripts/test-cpm-imm-w1-delete-draft.sh` — OK
- `cafe-deploy/scripts/test-discovery-imm9-wallet-scan-w1-cpm-block.sh` — step 5 (DELETE draft → rescan **200**) — OK

**Tests unitaires :** `go test ./internal/app/ ./internal/persistence/ -run DeleteDraft`

---

## IMM-9b follow-up — Unify wallet address normalization (2 PR)

**Règle canonique (une seule) :** `persistence.NormalizeWalletTargetAddress` — `0x` + 42 car. hex minuscules. Helpers partagés dans `internal/persistence/wallet_target.go` :

| Helper | Usage |
|--------|--------|
| `NormalizeWalletSubjectID` | NATS consumer — `wallet:0x…` ou hex nu ; invalide / non-wallet → pass-through |
| `WalletSubjectIDFromAddress` | `POST …/policies/assessment/request` — adresse Discovery → `wallet:0x…` (erreur si invalide) |

**Implémentation locale (branche à créer depuis `main`) :** code prêt sur le workspace ; à découper en **2 PR** :

| PR | Branche suggérée | Fichiers | Titre suggéré |
|----|------------------|----------|---------------|
| **1** | `cpm/imm-9b-wallet-normalization-nats` | `wallet_target.go` (+ tests `NormalizeWalletSubjectID`), `assessment_consumer.go`, tests NATS existants | `cpm: unify wallet subject normalization (NATS, IMM-9b)` |
| **2** | `cpm/imm-9b-wallet-normalization-assessment` (après merge PR1) | `assessment_request.go`, `assessment_request_test.go`, tests `WalletSubjectIDFromAddress` | `cpm: assessment request uses canonical wallet subject id` |

**Acceptance :** `go test ./internal/persistence/ ./internal/integration/nats/ ./internal/app/ -run 'Assessment|WalletSubject|NormalizeWallet'` inchangé en comportement pour adresses valides.

**Repos :** `cafe-crypto-policy-mgt` uniquement.

---

## IMM-9b follow-up — Factor internal policy-reference handlers

**Context:** `internal/app/policy_reference_internal.go` (`POST /internal/policies/references/scan`) and `internal/app/policy_reference_wallet_target_internal.go` (`POST …/references/wallet-target`) share the same boilerplate: read/limit body, JSON decode, `user_id` + `tenant_id` → `authz.Principal`, validate principal, map persistence errors.

**Improvement:** Extract shared helpers (e.g. `readInternalServiceJSON`, `principalFromInternalUserTenant`) without changing HTTP contracts or auth route inventory.

**Acceptance:** `policy_reference_internal_test.go` and `policy_reference_wallet_target_internal_test.go` unchanged in behaviour; less duplication for future internal lookups.

**Repos:** `cafe-crypto-policy-mgt` only.

**Priorité:** plus tard (refactor confort).

---

## W1 / IMM-9b — Lookup by `target_address` in payload only (not `scan_id`)

**Context:** Internal lookup `POST /internal/policies/references/wallet-target` and `CountActiveWalletCPMContext` match owner policies/drafts by extracting a wallet address from **stored payload** (`policy_context`, `selected_wallet_policy_context`, `wallet_address`, etc.). This is intentional: Discovery must not resolve address via `policy.scan_id → Discovery detail` before allocating a new `scan_id` (**IMM-9b**).

**Implication:** Records with payload that has **no** denormalized address (e.g. PR5-style `{ "x": 1 }` with only `PolicyRecord.ScanID` set) do **not** participate in W1 for any address — rescan is not blocked. Real flows (explore/persist, UI, deploy scripts) must persist `target_address` (or equivalent) in the payload.

**Not a bug:** Do not “fix” by joining on `scan_id` and calling Discovery from CPM for W1. If product requires blocking in legacy shapes, migrate payloads or enforce address fields on `POST …/policies` / `POST …/drafts` at write time.

**Repos:** Document in `workplans/IMMUTABILITE_PR.md` / optional persist validation if product wants stricter guarantees.

**Priorité:** doc / validation produit optionnelle — **pas** une nouvelle implémentation lookup.

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

**Priorité:** plus tard.
