# CPM Post-V1 Option A — Multi-Repo PR Plan

Cross-repository work plan for connecting the **Crypto Policy Management (CPM)** experience to **real authenticated Discovery wallet scans** via a CPM-ready wallet-policy-context API, **without** a giant single PR or a frontend-first spike.

**Repositories:** `cafe-discovery`, `cafe-crypto-policy-mgt` (CPM), `cafe-frontend`.

**Related design notes:** `cafe-crypto-policy-mgt/cpm_post_v_1_option_a_scan_context.md`, `cafe-crypto-policy-mgt/CPM_AuthN_AuthZ_workplan.md`.

---

## Objective

Deliver **Option A**: Discovery wallet scan (backed today by Discovery persistence / DB flows) feeds **normalized, owner-scoped wallet policy contexts** to the frontend; the user selects a context; CPM **explore / validate / persist** runs against **`scan_id` + `policy_context` + `selection_request`** so policies are bound to real scans—not CBOM placeholders or forged observations.

Hard rules from product/architecture:

- **Backend first**, then script/API contract validation, then frontend.
- **Do not remove** legacy scan or CBOM endpoints.
- **Do not** claim Discovery is DB-free today; Persistence Service remains the **target** authoritative scan-data owner—Discovery backend still has DB access in the interim.
- **Respect AUTH-██** already merged (principal, scan binding, fail-closed behavior where configured).
- **Frontend:** no direct DB; no unauthenticated Persistence Service calls; **no `mock-discovery-scan-placeholder` in API mode** once Option A wiring lands.

---

## Architecture decision

**Option A (short term):**

1. Discovery exposes **authenticated, owner-scoped** `GET /discovery/wallet-policy-contexts` (edge: `GET /api/discovery/wallet-policy-contexts` with `/api` prefix stripped by nginx).
2. Discovery continues to derive listed rows from existing persistence (today: scan-result style storage); responses are **DTOs**, not raw ORM/schema dumps.
3. CPM **`POST /api/v1/policies/decisions/explore`** accepts the **Option A envelope**: top-level `scan_id` (for AUTH-02 when present), `policy_context`, `selection_request`.
4. CPM **`POST /api/v1/cpm/policies`** persists owner-scoped records with **`scan_id` populated** when operating in Option A mode (explicit rejection if missing where required).
5. Frontend loads contexts via authenticated HTTP against Discovery (gateway or direct base URL configurable), selects one, and feeds that object into existing CPM data flows.

AUTH-05 / internal Discovery authz endpoints remain the path for **binary** scan checks where CPM delegates **can-read**; listing remains Discovery-owned.

---

## Current state (inspected — 2026-05-11)

Evidence from this workspace snapshot:

### cafe-discovery

- **Endpoint implemented (PR A1 — livré sur `main`, ex. cafe-discovery #48):** `ListWalletPolicyContexts` on `GET /discovery/wallet-policy-contexts`; registered in `internal/app/container.go` under the JWT-protected `/discovery` API group.
- **Service layer:** `internal/service/wallet_policy_context.go` builds `WalletPolicyContextDTO` with `scan_id`, `wallet_address`, `wallet_type`, `chain_ids`, `current_pq_posture`, `scanned_at`, `status`; maps DB `SUCCESS` → **`completed`**; **does not** invent chain IDs for unknown networks (empty slice).
- **Tests:** `internal/handler/wallet_policy_context_test.go` (401 missing/invalid JWT, basic JSON shape, `SUCCESS` → `completed`); `internal/service/wallet_policy_context_test.go` (owner isolation user A vs B, unknown network → empty `chain_ids`).

### cafe-crypto-policy-mgt

- **Explore:** `internal/api/read_api.go` binds `POST /api/v1/policies/decisions/explore` with `decisionExploreRequest` (`scan_id`, `policy_context`, `selection_request`) and converts `policy_context` via `internal/api/explore_policy_context.go`.
- **Tests:** `internal/api/read_api_test.go` includes `TestDecisionExplore_optionA_policy_context` with Option A-shaped body returning 200.
- **Persist:** `internal/app/owner_routes.go` — `POST /api/v1/cpm/policies` uses `ownerScopedUpsertRequest` with **`scan_id` as a string field** (currently **optional** at decode — empty allowed); `OwnerScopedStore` stores `ScanID` on `PolicyRecord`.
- **Scripts:** `scripts/test-discovery-v1-wallet-scans-to-cpm.sh` smoke (Discovery **v1** wallet scans → CPM explore); `CURL_REDIRECT` default **0**. CI: `scripts/ci/assert-scripts-use-discovery-wallet-scans-v1-only.sh`.

### cafe-frontend

- **Placeholder still active:** `src/cpm/useCpmPolicySelection.ts` defines `CPM_SELECTION_CONTEXT_SCAN_ID = 'mock-discovery-scan-placeholder'` and mock wallet address; `onMounted` loads templates + selection immediately.
- **No Playwright/Cypress** in `package.json` yet — **F5** implies adding or using another agreed E2E approach.

### Edge routing (cafe-deploy template)

- `location /api/` proxies to Discovery with **prefix stripped** → backend sees `/discovery/...` paths as configured in Discovery (`nginx.conf.template`).

---

## Target flow

```text
Discovery wallet scan (existing flows; DB-backed today)
       ↓
Persistence / scan_results (today in Discovery boundary; PS is long-term owner)
       ↓
GET /discovery/wallet-policy-contexts (JWT, owner-scoped DTO list)
       ↓
Frontend: DiscoveryWalletPolicyContextDataSource + useCpmScanContext
       ↓
POST /api/v1/policies/decisions/explore
  { scan_id, policy_context, selection_request }
       ↓
CPM evaluate → graph / ranked candidates → validation / EOA / persist
       ↓
POST /api/v1/cpm/policies with non-empty scan_id binding (Option A)
```

---

## Known contract mismatches (resolve in early PRs, not implicitly in UI)

| Area | Requested narrative / UX enum | Current code behavior |
|------|------------------------------|----------------------|
| `current_pq_posture` | `classical \| hybrid \| pq_ready \| unknown` | Discovery emits **`classical_only`**, **`hybrid`**, **`full_pq`** from NIST level mapping; CPM contract tests use **`classical_only`**. Align via mapping, vocabulary change, or documented canonical set. |
| `wallet_type` | `EOA \| SMART_ACCOUNT \| UNKNOWN` | Discovery domain uses **`EOA`**, **`AA`**, **`Contract`** (`internal/domain/models.go`). CPM normalizes some wire values in `normalizeWireAccountKind`. Document and align consumer types. |

These do **not** block listing the PR sequence but **must** appear in **A2** and be closed or explicitly deferred before **F1**.

---

## PR sequence, branches, roll-out order, et suivi

**Ordre de merge suggéré** : phases **A → B → C → F → D** (l’ordre des lignes du tableau ci-dessous). La phase **C** (scripts + tests de contrat) avant **F** (frontend).

**Note — PR A1** : aucun travail PR supplémentaire n’est nécessaire pour A1 ; la fonctionnalité est **déjà implémentée et mergée** sur **`cafe-discovery`** (`main`, ex. `feat(discovery): add authenticated wallet policy contexts for CPM`, PR **#48**). La ligne A1 dans le tableau sert de **référence / jalons** ; une PR ne devrait être rouverte **que** pour des **durcissements optionnels** (tests annexes, format `scanned_at`, etc.). Le travail suivant côté Discovery pour Option A est surtout **A2** (documentation / contrat).

Remplir **Statut**, **N° PR**, **Lien**, **Assigné**, **Notes** au fil des travaux. Valeurs suggérées pour **Statut** : `à faire` · `en cours` · `PR ouverte` · `en revue` · `mergé` · `bloqué`.

| # | PR | Dépôt (repo) | Branche | Objectif (résumé) | Statut | N° PR | Lien | Assigné | Notes |
|---:|----|--------------|---------|-------------------|--------|-------|------|---------|-------|
| 1 | A1 | cafe-discovery | `option-a/a1-discovery-wallet-policy-contexts` | GET wallet-policy-contexts (JWT, DTO, pagination) | mergé | 48 | | | Déjà sur `main` — pas de PR à créer pour livrer la feature (voir note ci-dessus). Optionnel : durcissements/tests. |
| 2 | A2 | cafe-discovery | `option-a/a2-discovery-api-docs` | Doc API / contrat vs `/discovery/scans` | | | | | |
| 3 | B1 | cafe-crypto-policy-mgt | `option-a/b1-cpm-explore-option-a-payload` | Explore Option A + compat legacy | | | | | |
| 4 | B2 | cafe-crypto-policy-mgt | `option-a/b2-cpm-persist-scan-binding` | Persist lié à `scan_id` (Option A) | | | | | |
| 5 | C1 | cafe-crypto-policy-mgt | `option-a/c1-option-a-script` | Script smoke Option A | | | | | |
| 6 | C2 | cafe-crypto-policy-mgt (+ cafe-discovery si besoin) | `option-a/c2-contract-api-tests` | Tests contrat API répétables | | | | | |
| 7 | F1 | cafe-frontend | `option-a/f1-frontend-wallet-policy-context-data-source` | Data source Discovery contexts | | | | | |
| 8 | F2 | cafe-frontend | `option-a/f2-frontend-cpm-scan-context` | Composable `useCpmScanContext` | | | | | |
| 9 | F3 | cafe-frontend | `option-a/f3-frontend-cpm-scan-selector` | UI sélecteur de scan CPM | | | | | |
| 10 | F4 | cafe-frontend | `option-a/f4-frontend-feed-scan-context-to-cpm` | Brancher contexte réel sur CPM API | | | | | |
| 11 | F5 | cafe-frontend | `option-a/f5-frontend-e2e` | E2E parcours Option A | | | | | |
| 12 | D1 | cafe-crypto-policy-mgt (+ liens discovery / frontend) | `option-a/d1-option-a-docs` | Doc finale architecture / limites | | | | | |

---

## PR A1 — Discovery wallet-policy-contexts endpoint

**Statut livré — ne pas refaire cette PR pour implémenter l’endpoint** : déjà présent dans **`cafe-discovery`** (`main`, PR [#48](https://github.com/create2-labs/cafe-discovery/pull/48)). Le détail ci-dessous documente **l’intent et les critères d’acceptance** pour alignement équipe ; toute évolution nouvelle passe par une **PR dédiée** (durcissement) ou bascule sur **A2+**.

**Branch (historique / optionnel)** : `option-a/a1-discovery-wallet-policy-contexts`

### 1. Goal

Expose authenticated, owner-scoped **CPM-ready wallet policy contexts** for the current user as `GET /discovery/wallet-policy-contexts` (`GET /api/discovery/wallet-policy-contexts` through nginx), with JWT required, pagination, normalized status, strict `chain_ids` behavior, and DTO-only responses—**without** leaking raw DB internals.

### 2. Scope (expected until verified)

- `cafe-discovery/internal/handler/discovery.go` — `ListWalletPolicyContexts`
- `cafe-discovery/internal/app/container.go` — route registration
- `cafe-discovery/internal/service/wallet_policy_context.go`
- `cafe-discovery/internal/handler/wallet_policy_context_test.go`
- `cafe-discovery/internal/service/wallet_policy_context_test.go`
- Middleware: JWT on `/discovery` group (existing pattern)

### 3. Acceptance criteria

- Missing `Authorization` → **401**.
- Invalid JWT → **401**.
- Authenticated user → **200** with `{ contexts, total, limit, offset, count }`.
- User A **never** sees user B scans (repository + handler guarantees).
- `SUCCESS` in persistence surfaces as **`completed`** in JSON.
- Unknown / unmapped networks → **`chain_ids: []`** (no `[1]` fallback).
- Response contains **only** advertised façade fields (`scan_id`, `wallet_address`, `wallet_type`, `chain_ids`, `current_pq_posture`, `scanned_at`, `status`).
- **`scanned_at`:** RFC3339 UTC (existing code uses RFC3339Nano in places — confirm stability for clients).

### 4. Tests

- Unit/integration: extend handler tests if any acceptance row above lacks HTTP-level coverage (e.g. cross-user isolation at Fiber layer mirrors service tests).
- Manual: curls in handler test comments; via nginx `/api/discovery/wallet-policy-contexts` on a stack.

### 5. Explicit non-goals

- Removing `/discovery/scans` or CBOM routes.
- Moving Discovery to Persistence Service–only reads.
- Broad AuthZ refactor outside existing JWT ownership model.
- Changing `scan_id` semantics beyond documenting short-term UUID = `scan_results` row id.

### 6. Suggested commit message

`discovery: harden wallet-policy-contexts listing for Option A`

### 7. Suggested PR title

`Option A: Discovery GET /discovery/wallet-policy-contexts`

### 8. PR description template

```markdown
## Summary
(Historique) Implémentation livrée sur `cafe-discovery` main — voir PR #48. Ne pas rouvrir sauf durcissement ciblé.

## Acceptance / testing
See `CPM_OPTION_A_PR_PLAN.md` § PR A1 — `cd cafe-discovery && GOWORK=off go test ./... -count=1`.

## Out of scope
Legacy scan listing, CBOM, Persistence Service extraction, nginx changes (already strips `/api`).

## Risks / follow-ups
PQ posture / wallet_type vocabulary alignment vs CPM façade — tracked for A2.
```

---

## PR A2 — Discovery API documentation / DTO contract

**Branch:** `option-a/a2-discovery-api-docs`

### 1. Goal

Publish a **maintainer-facing API contract** that distinguishes **`/discovery/scans`** (generic scan history) from **`/discovery/wallet-policy-contexts`** (CPM-oriented façade), documents **direct vs nginx URLs**, JWT requirements, pagination, **`scan_id` semantics**, status normalization, and **strict `chain_ids`** rules—including **explicit enum alignment** (`current_pq_posture`, `wallet_type`) versus CPM and frontend unions.

### 2. Scope (expected until verified)

- `cafe-discovery/docs/**/*.md` (new file, e.g. `OPTION_A_WALLET_POLICY_CONTEXTS_API.md`), or README section if docs-only change is undesirable.
- Optional: generated OpenAPI snippets if Discovery already publishes them (**no** behavior change unless generation requires stubs).

### 3. Acceptance criteria

- Document states **JWT required** and **owner-scoped** semantics.
- **Both** URLs documented: backend `…/discovery/wallet-policy-contexts` vs edge `…/api/discovery/wallet-policy-contexts`.
- **`scan_id`:** short-term = UUID of **one persisted scan result row** in `scan_results` (new execution → new row → new id; stable for that row’s lifetime); future `walletObservationId` / `policySubjectRef` called out—**explicitly**.
- Pagination fields match actual JSON envelope.
- Contract mismatch table (**PQ posture**, **`SMART_ACCOUNT` vs `AA`**) resolved or flagged with **blocking issue** linked for B1/F1.

### 4. Tests

- Documentation review checklist; markdown link sanity (optional CI).

### 5. Explicit non-goals

- CPM explore/persist wording beyond cross-links (defer to D1 for full architecture).
- Nginx/terraform edits unless a doc typo fix is unavoidable.

### 6. Suggested commit message

`docs: Discovery wallet-policy-contexts vs scans API`

### 7. Suggested PR title

`docs: Wallet policy contexts API (Option A contract)`

### 8. PR description template

```markdown
## Summary
Documents the Discovery wallet-policy-contexts endpoint for Option A consumers and contrasts it with /discovery/scans.

## Acceptance
 reviewers confirm URL matrix, enums, pagination, scan_id semantics, chain_ids strict rules.

## Non-goals
No production code changes unless OpenAPI regeneration requires trivial annotation.
```

---

## PR B1 — CPM explore accepts Option A payload

**Branch:** `option-a/b1-cpm-explore-option-a-payload`

### 1. Goal

Ensure **`POST /api/v1/policies/decisions/explore`** treats **`scan_id` + `policy_context` + `selection_request`** as the **default recommended contract**, preserves evaluator output shape (**`decision`** / **`ranked_candidates`**), retains **backward compatibility** if any legacy envelope still exists on the wire, and documents **AUTH-02** limitations (**CPM cannot verify scan ownership alone** unless scan auth delegation succeeds—stay honest in README / godoc).

### 2. Scope (expected until verified)

- `cafe-crypto-policy-mgt/internal/api/read_api.go` — decode path, validation errors
- `cafe-crypto-policy-mgt/internal/api/explore_policy_context.go` — wire → evaluator mapping
- `cafe-crypto-policy-mgt/internal/app/auth.go`, `extractScanIDsForAuthorization`, related tests (**AUTH-02**)
- Tests: `internal/api/read_api_test.go`, `internal/app/authz_scan_test.go`, `internal/app/app_test.go`

### 3. Acceptance criteria

- Option A-shaped body (**including** mirrored `scan_id` in top-level + `policy_context` when AUTH-02 active) → **200** where authz succeeds.
- **Clear 4xx** on malformed **`policy_context`**, conflicting `scan_id`s, invalid `wallet_type`, invalid `current_pq_posture`, bad `scanned_at`.
- **Legacy body** remains **200** if still supported (**document** deprecation path).
- **`target_chain_ids` empty**: behavior documented and covered by tests (evaluator/rules product expectation—not silent defaults).
- `DisallowUnknownFields` behavior unchanged or consciously relaxed with tests.

### 4. Tests

- `cd cafe-crypto-policy-mgt && GOWORK=off go test ./... -count=1` with new table-driven cases where gaps exist.

### 5. Explicit non-goals

- Implementing broad cross-service AuthZ beyond existing AUTH-02 wiring.
- CBOM ingestion in explore handler.

### 6. Suggested commit message

`cpm: document and harden Option A decisions/explore envelope`

### 7. Suggested PR title

`Option A: CPM decisions/explore payload contract`

### 8. PR description template

```markdown
## Summary
Aligns POST /api/v1/policies/decisions/explore with Option A (`scan_id` + `policy_context` + `selection_request`) and tests edge cases.

## AUTH-02
Notes when scan_id triggers Discovery delegation and known limitations short-term.

## Tests
go test ./... -count=1
```

---

## PR B2 — CPM persist binds `scan_id`

**Branch:** `option-a/b2-cpm-persist-scan-binding`

### 1. Goal

When operating in **Option A mode**, **`POST /api/v1/cpm/policies`** must **persist a non-empty **`scan_id`** (same UUID as Discovery context)**, reject or fail clearly if omitted**, keep **legacy** payloads working only when explicitly flagged or when `scan_id` omission remains valid for older flows—and return the stored **`ScanID`** in **`item`** so clients verify binding.

### 2. Scope (expected until verified)

- `cafe-crypto-policy-mgt/internal/app/owner_routes.go` — `decodeOwnerScopedUpsertRequest`, handlers
- `cafe-crypto-policy-mgt/internal/persistence/owner_scoped_store.go`
- Payload shape from scripts: optional nested `selected_scan_id`, `selected_wallet_policy_context`, etc.—**decide single canonical persistence JSON** vs storing rich object inside `payload` only (document outcome).
- Tests: `internal/app/owner_routes_test.go` + integration-style tests mirroring AUTH behavior

### 3. Acceptance criteria

- Persist with **`scan_id` set** → `PolicyRecord.ScanID` persisted; **GET** returns same **`scan_id`**.
- Option A persist path (**define trigger:** e.g. presence of `selected_wallet_policy_context` or `workflow=option-a` flag) → **reject 4xx** if `scan_id` empty **or** if top-level **`scan_id`** missing while Option A discriminator present.
- Owner-scoping unchanged (**403**/errors per existing semantics).
- **No** mandate to store CBOM snapshot as primary input for Option A.

### 4. Tests

- New tests covering Option A discriminator + missing `scan_id` → expected **4xx**.
- Regression: persist without Option A discriminator still behaves as today if supported.

### 5. Explicit non-goals

- Database schema migrations beyond existing in-memory / current store (**no** unrelated persistence refactor).
- Removing legacy drafts/policies without `scan_id`.

### 6. Suggested commit message

`cpm: require scan binding for Option A persist`

### 7. Suggested PR title

`Option A: bind persisted CPM policies to scan_id`

### 8. PR description template

```markdown
## Summary
Ensures persisted policies record Discovery scan_id for Option A flows and rejects ambiguous saves.

## Backward compatibility
Preserves legacy persists where scan_id-less upserts remain valid — call out explicitly in changelog.

## Tests
go test ./... -count=1
```

---

## PR C1 — Script Option A finalization (Discovery **v1** wallet scans)

**Branch:** `option-a/c1-option-a-script`

### 1. Goal

Make **`scripts/test-discovery-v1-wallet-scans-to-cpm.sh`** the **authoritative bash smoke contract** for Option A: Discovery sign-in → **`GET /discovery/v1/wallets/scans`** → **`GET /discovery/v1/wallets/scans/{scan_id}`** → deterministic scan selection (**`SCAN_ID`** or uniqueness rules) → CPM explore (Option A body) → optional persist. **Do not** call the removed **`/discovery/wallet-policy-contexts`** façade (PR11a on `cafe-discovery`). No CBOM / legacy scan polling scripts.

### 2. Scope (expected until verified)

- `cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh`
- **`scripts/ci/assert-scripts-use-discovery-wallet-scans-v1-only.sh`** (shared guard); CI job invokes it plus **`bash -n`** on the smoke script.

### 3. Acceptance criteria

- Default path lists wallet scans via **v1** (direct **`/discovery/v1/wallets/scans`**); **help** cites edge **`/api/discovery/v1/wallets/scans`** and CPM **`/api/cpm/v1/...`** paths.
- **`CURL_REDIRECT`** default **`0`**; never prints JWT or passwords.
- Multi-scan selection behaves per documented rules (**fail if ambiguous without `SCAN_ID`**).
- **CI / local guard:** **`scripts/**/*.sh`** must contain **no** substring spelling the removed legacy wallet-policy-contexts route (**`grep`** or the versioned **`assert`** script).
- Works with **`SKIP_PERSIST=1`**; full persist exercised once **B2** semantics are merged.

### 4. Tests

```bash
bash scripts/ci/assert-scripts-use-discovery-wallet-scans-v1-only.sh
bash -n cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh
# shellcheck (if installed): shellcheck cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh
```

Manual: `SKIP_PERSIST=1` against local stack.

### 5. Explicit non-goals

- Auto-provisioning wallets or scans unless already scripted elsewhere.

### 6. Suggested commit message

`chore(scripts): migrate Option A smoke test to Discovery v1 wallet scans`

### 7. Suggested PR title

`ci(scripts): Discovery v1 wallet scans smoke + guard legacy route references`

### 8. PR description template

```markdown
## Summary
Option A bash smoke uses Discovery **v1** `GET /discovery/v1/wallets/scans` + detail `…/wallets/scans/{scan_id}` and CPM explore/persist; CI asserts no legacy Discovery wallet-*-contexts route string in active `scripts/**/*.sh`.

## How to validate
`bash scripts/ci/assert-scripts-use-discovery-wallet-scans-v1-only.sh`, `bash -n scripts/test-discovery-v1-wallet-scans-to-cpm.sh`, shellcheck (optional), manual `SKIP_PERSIST=1` run.

## Non-goals
No legacy CBOM or scan-polling smoke scripts.
```

---

## PR C2 — Contract/API tests

**Branch:** `option-a/c2-contract-api-tests`

### 1. Goal

Add **repeatable, CI-friendly** tests that freeze **Discovery** + **CPM** public contracts **before** substantial frontend merges.

### 2. Scope (expected until verified)

- `cafe-crypto-policy-mgt/internal/...` new `contract` or integrate into `internal/app/app_test.go` if docker-free httptest stubs suffice.
- `cafe-discovery/internal/handler/...` augment only if Discovery lacks coverage for any row in acceptance matrix (**prefer** reuse of existing suites).
- Optionally **GitHub Actions** workflow snippet (only if repos already CI).

### 3. Acceptance criteria matrix (minimum)

**Discovery**

- No JWT → **401**
- Valid JWT → **200**
- `contexts[]` shape stable (keys asserted)
- Owner isolation
- Unknown network mapping → **`chain_ids` empty**, not **`[1]`**
- **`status === "completed"`** for successful persisted scans (normalized)

**CPM**

- Option A explore body → **200** (fixture store)
- Invalid body → **clear 4xx**
- Persist with **`scan_id`** → **`item.scan_id`** non-empty (**post-B2** rules)
- Option A persist without **`scan_id`** → **explicit failure**

### 4. Tests

Primarily **`go test`**. Scripts from **C1** as supplementary local gate.

### 5. Explicit non-goals

- Full docker-compose E2E of three services (**F5** handles UI-heavy paths).
- Load or security fuzzing.

### 6. Suggested commit message

`test: add Option A Discovery + CPM contract coverage`

### 7. Suggested PR title

`tests: Option A API contracts (Discovery + CPM)`

### 8. PR description template

```markdown
## Summary
Adds automated contract tests aligning with Option A flows.

## Covers
Listed matrix in `CPM_OPTION_A_PR_PLAN.md` § PR C2.

## Non-goals
Browser E2E (separate PR F5).
```

---

## PR F1 — Frontend `DiscoveryWalletPolicyContextDataSource`

**Branch:** `option-a/f1-frontend-wallet-policy-context-data-source`

### 1. Goal

Introduce a dedicated **authenticated** data-access type that lists Discovery wallet-policy-context pages, **separate** from **`CpmDataSource`**, with configurable base path (**direct Discovery** vs **gateway `/api/discovery`**), resilient error surfaces (**401 / 403 / 404 / network**), and **no CPM coupling**.

### 2. Scope (expected until verified)

- New module under `cafe-frontend/src/` (e.g. `discovery/walletPolicyContextDataSource.ts` or adjacent to HTTP client factories).
- Existing auth HTTP helpers / Axios instances (**match conventions**).
- Unit specs with mocked transport.

### 3. Acceptance criteria

- Implements **`listWalletPolicyContexts({ limit?, offset? })`** returning **`WalletPolicyContextPage`**.
- TypeScript **`WalletPolicyContext`** matches facade after **enum alignment** (**A2**).
- Structured errors surfaced (no swallowed failures).
- No visible CPM page behavior change (**no** mandatory UI wiring).

### 4. Tests

- Vitest unit tests (**mock adapter**).

### 5. Explicit non-goals

- `useCpmScanContext`, scan selector UI, touching persistence.
- Removing mock CPM data sources.

### 6. Suggested commit message

`frontend: Discovery wallet-policy-context listing data source`

### 7. Suggested PR title

`Frontend: DiscoveryWalletPolicyContextDataSource`

### 8. PR description template

```markdown
## Summary
Adds Discovery wallet-policy-contexts client abstraction (authenticated).

## Dependency
Depends on finalized API enums from cafe-discovery docs PR (A2).

## Tests
npm test (vitest).

## Non-goals
CPM wiring / UI selectors.
```

---

## PR F2 — Frontend `useCpmScanContext`

**Branch:** `option-a/f2-frontend-cpm-scan-context`

### 1. Goal

Composable that **owns list + selection lifecycle**: loading / empty / error / pagination hints, **`?scanId=`** route sync, rejecting unknown or unauthorized scans, emitting a **signal** when selection changes so downstream validation resets can subscribe—without calling CPM until a valid context exists.

### 2. Scope (expected until verified)

- `cafe-frontend/src/cpm/**/*.ts` (new composable)
- Minimal shell hook-up only if tests require (**hidden** demo route discouraged).

### 3. Acceptance criteria

- **API mode:** **no dummy scan id** placeholders.
- Mock/demo mode continues to isolate fake scan contexts (**existing mock strategies** — document).
- On invalid **`scanId`** query → explicit error UX state surfaced to future UI.
- Public API exposes **`selectedScanId`** + **`selectedContext`** + async load.

### 4. Tests

- Vitest: state machine / query-param parsing / guard rails.

### 5. Explicit non-goals

- Full CPM page redesign.
- Persist flow.

### 6. Suggested commit message

`frontend: composable for CPM scan context selection`

### 7. Suggested PR title

`Frontend: useCpmScanContext composable`

### 8. PR description template

```markdown
## Summary
Adds scan context selection composable (Discovery contexts only).

## Contract
 Honors ?scanId=; blocks CPM fetch until validated context chosen.

## Non-goals
Visible CPM page integration (later PRs).
```

---

## PR F3 — CPM scan selector UI

**Branch:** `option-a/f3-frontend-cpm-scan-selector`

### 1. Goal

Expose scan selection UX on the CPM experience: loader, empty (**“Run a wallet scan first”**), auth/network errors with retry, **multi-select list + summary card** (**wallet**, **wallet type**, **chain ids**, **PQ posture**, **scannedAt**, **`status`**), fed **only** from wallet-policy-contexts API.

### 2. Scope (expected until verified)

- `cafe-frontend/src/components/cpm/**/*.vue`
- Possibly `src/views/**` mounting new block.

### 3. Acceptance criteria

- Never calls **`/discovery/scans`** directly for Option A UX.
- No raw persistence fields surfaced.
- **Does not trigger** Policy explore/load graph until downstream PR wires it (acceptable to emit events only—or guard via prop **if** wired early).

### 4. Tests

- Component tests (**Vitest + Vue test utils**) for states matrix.

### 5. Explicit non-goals

- Persist / validation / wallet challenge wiring.

### 6. Suggested commit message

`frontend: wallet scan context picker on CPM page`

### 7. Suggested PR title

`Frontend: CPM wallet policy scan selector`

### 8. PR description template

```markdown
## Summary
Adds visible scan selection UI sourcing wallet-policy-contexts endpoint.

## UX states
Matches loading / empty / auth / retry / summary requirements.

## Non-goals
CPM evaluate calls (defer F4 until selection stable).
```

---

## PR F4 — Feed selected context into CPM selection

**Branch:** `option-a/f4-frontend-feed-scan-context-to-cpm`

### 1. Goal

Replace **mock / placeholder IDs** when **API mode** is enabled: bind **`selectionScanId`** (and companion wallet/type/address posture fields) from **`WalletPolicyContext`**, propagate into **`getPolicySelection` / explore** payload (`scan_id`, `policy_context`, `selection_request` per **`apiCpmDataSource`**), reset validation + EOA + incompatible drafts **on scan change**.

### 2. Scope (expected until verified)

- `cafe-frontend/src/cpm/useCpmPolicySelection.ts` — initialization + watchers
- `cafe-frontend/src/cpm/apiCpmDataSource.ts` — mapping if needed
- Validation / wallet challenge modules per AUTH-██ behavior

### 3. Acceptance criteria

- **API mode**: **never** emits **`mock-discovery-scan-placeholder`** in outbound requests (**assert via tests**/spies).
- No `getPolicySelection` until valid context (**if enforced** product-side).
- Change scan ⇒ validation reset + wallet challenge cleared + drafts not silently revived when incompatible (**unit/integration tests** documenting behavior).

### 4. Tests

- Extend `useCpmPolicySelection.spec.ts`, `apiCpmDataSource.spec.ts`, adjacent specs.

### 5. Explicit non-goals

- Designing new graph visualizations unrelated to feeding context.

### 6. Suggested commit message

`frontend: drive CPM selection from Discovery scan context`

### 7. Suggested PR title

`Frontend: bind CPM policy selection to real scan context`

### 8. PR description template

```markdown
## Summary
Feeds selected Discovery wallet-policy-context into CPM API mode (drops mock scan id placeholder).

## Testing
Adds unit/integration coverage asserting request bodies.

## Depends on
B1/B2 finalized semantics for scan_id coupling + AUTH posture.
```

---

## PR F5 — Frontend E2E

**Branch:** `option-a/f5-frontend-e2e`

### 1. Goal

Prove **critical UI journeys** aligned with acceptance (happy **+ negative** flows). Today **Vitest-only** toolchain — introducing **Playwright or equivalent** acceptable if approved at repo standards level.

### 2. Scope (expected until verified)

- New E2E dir + **`package.json` scripts**.
- Fixtures or mocks for deterministic auth (align with **`CPM_SKIP_AUTH`** / staging creds governance — document safe defaults).

### 3. Acceptance criteria

**Happy path:** login → wallet context visible → selection → templates/candidates loaded → edits → validation → persist with visible **`scan_id`** binding acknowledgement.

**Negatives:** no scans → empty; multiple scans → explicit picker; bogus **`scanId`** param → error; auth failure (401 surfaces); transient network (**retry UI** exercised); malformed explore response → surfaced error (**mock server** acceptable).

### 4. Tests

New E2E suite + CI step (optional if runners absent).

### 5. Explicit non-goals

- Perf testing or Lighthouse.

### 6. Suggested commit message

`test(e2e): Option A CPM + Discovery scans flow`

### 7. Suggested PR title

`E2E: CPM scan context + persistence flow`

### 8. PR description template

```markdown
## Summary
Adds automated browser/regression flows for Option A CPM UX.

## Setup
 Documents required env/secrets mocks for CI/local.

## Non-goals
Replaces bash contract tests (those remain authoritative for pure API).
```

---

## PR D1 — Final documentation

**Branch:** `option-a/d1-option-a-docs`

### 1. Goal

Produce **integrated Option A narrative** spanning **Discovery + CPM + frontend**, including diagrams, endpoint matrix, payloads, AuthN/Z model (**JWT everywhere public**), scripts, **`scan_id` roadmap**, Persistence Service positioning, **`/discovery/scans` distinction**, limits (short-term inability to forge CBOM-backed observation for explore when using façade fields only—clarified), and **`test-discovery-v1-wallet-scans-to-cpm.sh`**.

### 2. Scope (expected until verified)

- `cafe-crypto-policy-mgt/docs/` + cross-links (`README.md` sections pointing to authoritative doc).
- Optional `cafe-documentation` PR if architectural docs live there centrally.

### 3. Acceptance criteria

- **Mermaid (or ASCII) architecture diagram**: scan → Persistence (target) vs current Discovery DB → contexts API → frontend → explore → persist.
- Full URL table (**direct nginx vs localhost dev** ports).
- **Known limitations**: scan ownership parity, enum vocabulary, Persistence Service rollout.
- **Manual validation commands** appendix (reuse section below).

### 4. Tests

Doc review checklist.

### 5. Explicit non-goals

Replacing granular AUTH workplan tables—**reference** instead of duplicating verbatim.

### 6. Suggested commit message

`docs: Option A end-to-end flow (Discovery ↔ CPM ↔ frontend)`

### 7. Suggested PR title

`Docs: Complete Option A CAFE crypto policy integration`

### 8. PR description template

```markdown
## Summary
Final Option A narrative + diagrams consolidating multi-repo rollout.

## Reviewers
@platform + FE + Discovery owners.

## Note
Depends on factual accuracy of landed code from A–F series.
```

---

## Risks and open questions

- **PQ posture vocabulary** drift (**Discovery façade** vs **`cafe-contracts`/v01 enums** vs **frontend UX copy**) — unblock before frontend types freeze.
- **Wallet type taxonomy** (**`SMART_ACCOUNT` vs Discovery `AA`**) — normalization layer ownership (Discovery emit vs frontend map vs CPM only).
- **AUTH-02** when Discovery delegation slow/unavailable (**fail-closed** UX on CPM) — coordinated messaging for F3/F4.
- **Persist payload canonical shape** (**B2**) — nested `selected_wallet_policy_context` vs flat `payload` blob only.
- **`scanned_at` source**: entity `UpdatedAt` vs **`ScannedAt`** — document field semantics per D1.
- **F5 infra**: absence of Playwright baseline → **budget** infra PR or accept extended Vitest + MSW (**decision**).

---

## Manual validation commands

```bash
# Discovery unit + integration suites
cd cafe-discovery && GOWORK=off go test ./... -count=1

# CPM unit + integration suites
cd cafe-crypto-policy-mgt && GOWORK=off go test ./... -count=1

# Option A bash smoke test (SKIP_PERSIST until persist rules verified)
SKIP_PERSIST=1 \
  DISCOVERY_EMAIL='user@example.com' DISCOVERY_PASSWORD='secret' \
  ./cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh

# Script syntax / optional shellcheck (C1 expectation)
bash -n cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh

# Frontend unit tests after F-series land
cd cafe-frontend && npm test

# Frontend typecheck/lint gates
cd cafe-frontend && npm run typecheck && npm run lint
```

---

## Global non-goals (entire initiative)

- Big-bang refactor of Persistence Service ingestion or ripping DB out of Discovery prematurely.
- Retiring **`GET /discovery/cbom/*`** or scan queue endpoints.
- Frontend calling Persistence Service or SQL directly.

---

_End of PR plan._
