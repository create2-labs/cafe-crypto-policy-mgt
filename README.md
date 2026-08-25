# CAFE Crypto Policy Management

`CAFE Crypto Policy Management` (*`CPM`*) is the Crypto Policy Management service for CAFE [cafe-crypto-policy-mgt](github.com/create2-labs/cafe-crypto-policy-mgt).


1. [CAFE Crypto Policy Management](#cafe-crypto-policy-management)
   1. [Role and boundaries](#role-and-boundaries)
   2. [Capability Providers (ADR 2026-08-03)](#capability-providers-adr-2026-08-03)
   3. [Repository layout](#repository-layout)
   4. [Discovery → CPM contract (`cafe.discovery.wallet.observed` v0.1)](#discovery--cpm-contract-cafediscoverywalletobserved-v01)
      1. [Envelope (`walletobserved.Event`)](#envelope-walletobservedevent)
      2. [Payload (`walletobserved.Payload`)](#payload-walletobservedpayload)
      3. [Exported vocabulary](#exported-vocabulary)
      4. [Canonical fixture](#canonical-fixture)
   5. [Assessment output model](#assessment-output-model)
   6. [Compatibility evaluation](#compatibility-evaluation)
   7. [Ranking and policy decision output](#ranking-and-policy-decision-output)
   8. [Inbound explicit assessment request](#inbound-explicit-assessment-request)
   9. [Health and metrics endpoints](#health-and-metrics-endpoints)
      1. [Version endpoint (CPM-OPS-3)](#version-endpoint-cpm-ops-3)
         1. [Version flow (end-to-end)](#version-flow-end-to-end)
   10. [Outbound CPM events](#outbound-cpm-events)
      1. [Producer-side documentation](#producer-side-documentation)
   11. [Read APIs](#read-apis)
   12. [Explore no-deployable-candidate observability (IMM-OPS-1)](#explore-no-deployable-candidate-observability-imm-ops-1)
   13. [Auth/Authz contract (AUTH-00)](#authauthz-contract-auth-00)
   14. [AUTH-01 runtime authentication wiring](#auth-01-runtime-authentication-wiring)
       1. [CP-PERSIST runtime](#cp-persist-runtime)
       2. [Platform drafts API (CPM-DRAFT-1 contract)](#platform-drafts-api-cpm-draft-1-contract)
   15. [AUTH-03 owner-scoped persistence](#auth-03-owner-scoped-persistence)
   16. [Durable CP storage (`CPM_STORE`)](#durable-cp-storage-cpm_store)
   17. [AUTH-04 auth error contract and observability](#auth-04-auth-error-contract-and-observability)
   18. [Option A integration (Discovery v1 wallet scans)](#option-a-integration-discovery-v1-wallet-scans)
   19. [Run locally](#run-locally)
   20. [IMM-OPS-1 smoke script (`scripts/test-imm-ops-1.sh`)](#imm-ops-1-smoke-script-scriptstest-imm-ops-1sh)

## Role and boundaries

- Discovery observes wallets, persists scan artifacts, and maps observations to the shared wire contract from `cafe-contracts`.
- CPM validates policy documents, selects compatible Capability Provider routes, assesses observations, and emits policy outcomes.
- Remediation consumes policy outcomes and plans or executes migration work — **it is not a Capability Provider** and does not live in CPM.

**Separation (ADR 2026-08-03):** a **Crypto Policy (CP)** is CAFE’s persisted intent + accepted snapshot; a **Capability Provider** is an external solution profile (e.g. Nicetry) described by `ProviderManifest`; **remediation** is a downstream consumer. CPM owns provider compatibility/ranking/persist gates; `cafe-persistence` stores CP JSON **opaquely** (no Nicetry / provider logic).

CPM does not depend on Discovery’s database or internal domain structs. CPM depends on Discovery only through the HTTP/JWT contracts required by the product workflow (session validation, scan authorization). Inbound integration is explicitly user-triggered via `policy.assessment.requested.v0.1`; `cafe.discovery.wallet.observed` remains informational.

Normative provider model: [ADR — Capability Provider abstraction](https://github.com/create2-labs/cafe-adr/blob/main/ADR_20260803_cp_provider_abstraction.md) and [PR plan](https://github.com/create2-labs/cafe-adr/blob/main/ADR_20260803_cp_provider_abstraction_PR_PLAN.md). Field-level schemas: [`openapi/cpm-v1.yaml`](./openapi/cpm-v1.yaml) (do not treat this README as a second OpenAPI).

Wallet control proof for CP persistence (EOA) is specified in [`docs/CP_PERSIST.md`](./docs/CP_PERSIST.md). **CP-PERSIST V1 is signed off independently** through that document (Part VI frozen decisions); [`workplans/WORKPLAN_API.md`](./workplans/WORKPLAN_API.md) remains the broader API workplan.

**CP-PERSIST (EOA, no drafts — ADR 2026-08-24 / RD-P1–P2):** clients **must** obtain the canonical authorization message from CPM via `POST /api/cpm/v1/wallet-challenges` (stateless; request includes hashed `payload`; CPM computes `payload_sha256`) before signing. Normative persist is **`POST /api/cpm/v1/policies`** with `signed_message` + `signature` + the same `payload`. **No public `/drafts*`** in the OpenAPI contract (handlers may lag until RD-P5 — contrat ahead of runtime). Server hash lib: [`internal/payloadhash`](./internal/payloadhash/) (JCS RFC 8785 + SHA-256; shared vectors [`internal/contract/testdata/payload_sha256/`](./internal/contract/testdata/payload_sha256/)). Spec: [`docs/CP_PERSIST.md`](./docs/CP_PERSIST.md) v1.0.0. **No Redis** / `ProofStore` in this model. Session auth (JWT) and wallet signature are orthogonal.

**Historical CP-PERSIST V1 (draft-based, superseded):** former normative persist was `POST /api/cpm/v1/drafts/{draft_id}/persist` (OpenAPI PR2 [#51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51) … PR7). Runtime still implements that path until RD-P4/P5/P7 remove it.

**Provider persist gate (CPM-P6):** normative body `cafe.crypto_policy.v0.2` + `accepted_provider_snapshot` (pinned refs, soft findings) — see [Capability Providers](#capability-providers-adr-2026-08-03). **Remaining runtime gap:** Smart Account / non-EOA CP persist (`UNSUPPORTED_WALLET_TYPE`).

## Capability Providers (ADR 2026-08-03)

CPM’s catalogue and explore/persist paths follow the **posture + solution profile** model (no business policy graph: no normative `nodes`, `node_path`, `graphEdges`, `compatibility_rules`, or `allowed` / `selected_node_ids` / `selected_edge_ids`).

| Concept | Where | Role |
| --- | --- | --- |
| **Crypto Policy** | `cpm_pq_account_validation_v1` | CAFE intention: `required_posture` + `allowed_providers` (e.g. `["nicetry"]`) — not a concrete technique |
| **ProviderManifest** | `cafe.provider_manifest.v0.1` | Declares solution profiles (`resulting_posture`, signature, constraints, chain support, references, `maturity`, `claim_status`) |
| **Selection wire** | `PolicySelectionRequest` | Internal transitional only; **removed from explore & assessment input** in CPM-P9a; couche B lives in persist `user_constraints` (CPM-P10) |
| **Persisted CP** | `cafe.crypto_policy.v0.2` | Normative hashed `payload` at `POST …/policies` (signed) — closed set incl. top-level `accepted_findings` + snapshot (RD-P1 contract; runtime wires in RD-P5) |

**Posture naming**

- Domain / catalogue / persist body: **`required_posture`** (Crypto Policy, CP payload).
- Explore / NATS assessment input **v0.2**: **`crypto_policy_id`** + scan context (`policy_context` HTTP / `observation` NATS). Required posture lives on the catalogue CP. Legacy `selection_request` / `target_posture` → HTTP **400** / NATS validation error.
- Provider solution: **`resulting_posture`**. Hard compatibility (CPM-P9b+) requires Crypto Policy posture to align with the profile’s resulting posture (ADR §7).

**`key_rotation_model`:** on solution profiles / persist `user_constraints` (P10); **not** on explore/assessment input after P9a.

**Hard vs soft (Nicetry pilot, ADR §7)**

- **Hard** failures reject the candidate (`incompatible.provider.*` — posture, chain, rotation, continuity, new wallet, wallet type, unresolved ref, etc.).
- **Soft** findings on ranked candidates: `requires_bundler`, `requires_local_signer_state` (`severity: warning`; do not block ranking). Persist must list them in `accepted_provider_snapshot.accepted_findings` when the snapshot flags require them.
- `requires_wallet_control_proof` is **persist-only** (CP-PERSIST EOA signature), not an explore soft finding.

**`claim_status` / `maturity`:** ranked and rejected explore candidates expose `maturity` and `claim_status` from the resolved profile. **`claim_status=declared` is a declaration only** — never present it as CAFE-audited or execution-observed proof (`cafe_reviewed` / `externally_audited` / `executed_observed` are distinct levels).

**Persist gate (ADR §9 / CPM-P10)** — before cafe-persistence write:

- `schema_version` = `cafe.crypto_policy.v0.2`
- `crypto_policy_id`, `required_posture`, `user_constraints`, `solution_profile_ref`, `accepted_provider_snapshot` required
- legacy `template_id` / `selection_request` / top-level couche B fields → **400** `CRYPTO_POLICY_PAYLOAD_INVALID`
- CPM **rejoue couche A puis couche B** against the snapshot (+ `policy_context` wallet/chains); couche B KO → `PROVIDER_USER_CONSTRAINTS_INCOMPATIBLE`
- snapshot `references` must be **pinned** (commit/version must not be empty or `unpinned_pending_fixture` — the shipped Nicetry fixture is pinned as of **CPM-P7**)
- soft finding codes listed when account/constraint flags require them; `chain_support_used.status` must not be `planned`
- stable error codes: `CRYPTO_POLICY_PAYLOAD_INVALID`, `PROVIDER_REFS_UNPINNED`, `PROVIDER_SOFT_FINDINGS_REQUIRED`, `PROVIDER_CHAIN_PLANNED`, `PROVIDER_SCAN_COMPAT_FAILED`, `PROVIDER_USER_CONSTRAINTS_INCOMPATIBLE` (OpenAPI `WalletAuthorizationErrorCode`)

Schemas: OpenAPI `CryptoPolicyPersistPayload`, `UserConstraints`, `AcceptedProviderSnapshot`, `SolutionProfileRef`. Package overview: `internal/domain/provider`.

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/cafe-cpm` | Service entrypoint |
| `internal/app`, `internal/config` | Bootstrap and configuration |
| `internal/domain/walletobserved` | Thin re-export of shared `cafe.discovery.wallet.observed` v0.1 wire types |
| `internal/domain/vocabulary` | Exported strings for account kind, algorithms, PQ posture, subject type |
| `internal/domain/provider` | `ProviderManifest` v0.1 loader/registry + ADR §7 hard/soft helpers; Nicetry fixture under `testdata/provider_manifest_nicetry_v0_1.json` |
| `internal/domain/policy` | Policy domain contracts (`PolicySelectionRequest`, catalogue `CryptoPolicy` with `required_posture` + `allowed_providers`, explorers, `CryptoPolicyPersistPayload` / persist gate) — **no** business policy graph |
| `internal/api` | PR17 read-only HTTP APIs for policy inspection and decision exploration |
| `internal/persistence` | Durable CP via **cafe-persistence** (`cphttp.Client`); see [Durable CP storage](#durable-cp-storage-cpm_store). `OwnerScopedStore` is compiled **only for tests** (`-tags dev`) — not used at runtime in deployed images. Opaque JSON only — no provider/Nicetry evaluation here. |
| `internal/walletauth` | CP-PERSIST V1 canonical wallet authorization message builder and EIP-191 / `personal_sign` verifier (PR3); used at persist time by PR4 handlers |
| `internal/payloadhash` | Server-authoritative `payload_sha256` (closed hashed set → RFC 8785 JCS → SHA-256 hex; findings sort+dedupe) — RD-P2; wired by RD-P4/P5 |
| `docs/` | Integration narratives — [`docs/CPM_OPTION_A_INTEGRATED.md`](./docs/CPM_OPTION_A_INTEGRATED.md) (Option A v1 flow); [`docs/CP_PERSIST.md`](./docs/CP_PERSIST.md) (EOA wallet control proof for CP persistence) |
| `scripts/` | Operational helpers — [`scripts/test-imm-ops-1.sh`](./scripts/test-imm-ops-1.sh) (IMM-OPS-1 explore observability smoke); Option A v1 smoke lives in [`cafe-deploy`](https://github.com/create2-labs/cafe-deploy/scripts/test-discovery-v1-wallet-scans-to-cpm.sh) |
| `internal/metrics` | Prometheus registry and CPM application counters (IMM-OPS-1) |
| `internal/integration/nats` | NATS integration for inbound explicit assessment requests + outbound CPM event publication |

## Discovery → CPM contract (`cafe.discovery.wallet.observed` v0.1)

`internal/domain/walletobserved` re-exports the shared contract from `github.com/create2-labs/cafe-contracts/observation/wallet/v01` so CPM code can keep stable local imports without duplicating wire structs.

Normative identifiers (see `internal/domain/walletobserved/contract.go`):

- `event_type`: `cafe.discovery.wallet.observed`
- `event_version`: `v0.1`
- `producer`: `cafe-discovery` (required by `Event.Validate()` for v0.1)

### Envelope (`walletobserved.Event`)

| JSON field | Description |
| --- | --- |
| `event_id` | Stable id (idempotence / deduplication) |
| `event_type` / `event_version` | Must match constants above |
| `occurred_at` | Event timestamp (RFC3339 in JSON) |
| `correlation_id` / `causation_id` | Tracing back to scans or jobs |
| `producer` | Must be `cafe-discovery` |
| `subject` | `type` + `id` (see vocabulary) |
| `payload` | Normalized observation (below) |

### Payload (`walletobserved.Payload`)

Observed (policy inputs from Discovery / scanners):

| JSON field | Description |
| --- | --- |
| `chain_ids` | Numeric EVM chain IDs (e.g. `1`, `8453`) |
| `account_kind` | Exported account kind (vocabulary) |
| `current_algorithm` | Exported algorithm id (vocabulary) |
| `public_key_exposed` | Exposure semantics for policy |
| `is_multichain` | Observed across more than one chain |
| `observed_at` | Observation time |


| JSON field | Description |
| --- | --- |
| `current_pq_posture` | `classical_only` \| `hybrid` \| `full_pq` \| `unknown` |

### Exported vocabulary

Subject type (v0.1): `wallet`.

Account kinds: `eoa`, `erc4337_smart_account`, `delegated_eoa_7702`, `contract_account`, `unknown`.

Algorithms: `secp256k1_ecrecover`, `mldsa44`, `mldsa65`, `falcon512`, or any non-empty string with prefix `hybrid_` (hybrid profiles).

PQ posture: `classical_only`, `hybrid`, `full_pq`, `unknown`.

`Event.Validate()` is provided by the shared `cafe-contracts` type and checks envelope constants, producer, subject type/id, and payload vocabulary.

### Canonical fixture

Golden JSON: [`internal/domain/walletobserved/testdata/discovery_wallet_observed_v01.json`](./internal/domain/walletobserved/testdata/discovery_wallet_observed_v01.json). Tests assert unmarshal → `Validate()` → JSON round-trip.

## Assessment output model

`internal/domain/policy/assessment_result.go` defines `CryptoPolicyAssessmentResult` as a standalone typed output model for policy assessment results. It is intentionally separate from `CryptoPolicyInstance` so policy content and runtime assessment outcomes remain decoupled for persistence and future API/event payloads.

## Compatibility evaluation 

`internal/domain/policy/compatibility_result.go` defines `PolicyCompatibilityEvaluator`, which classifies a single validated `CryptoPolicyInstance` against a `walletobserved.Payload` and `PolicySelectionRequest`. It returns `PolicyCompatibilityResult` with one of: `compatible_and_deployable`, `compatible_but_not_deployable` (e.g. empty instance scope `chain_ids`), or `incompatible`, with structured `AssessmentFinding` entries. Template-backed candidates use `required_posture` + `solution_profile_ref` (no graph topology).

When `CPM_PROVIDER_MANIFEST_PATHS` is set (default: Nicetry fixture), explore resolves each instance `solution_profile_ref` and applies ADR §7 **hard** provider checks (`required_posture` / wire `target_posture` vs `solution_profile.resulting_posture`, rotation model, chain support, continuity, wallet type, etc.). Hard fail → `rejected_candidate` with stable codes such as `incompatible.provider.posture`, `incompatible.provider.chain`, `incompatible.provider.rotation`, `incompatible.provider.continuity`, `incompatible.provider.new_wallet`, `incompatible.provider.wallet_type`. Ranked candidates keep ADR §7 **soft** findings `requires_bundler` and `requires_local_signer_state` (`severity: warning`; they do not block ranking). `requires_wallet_control_proof` is persist-only (CP-PERSIST stamp). **`claim_status=declared` remains a declaration, not audited/executed proof.** Ranked/rejected explore candidates expose structured fields (`candidate_id`, `required_posture`, `resulting_posture`, `solution_profile_ref`, `maturity`, `claim_status`) without graph topology arrays. See [Capability Providers](#capability-providers-adr-2026-08-03).

Normative draft persist (`POST /drafts/{draft_id}/persist`) requires a `cafe.crypto_policy.v0.2` payload with `crypto_policy_id`, `user_constraints`, and `accepted_provider_snapshot`: soft findings listed in `accepted_findings`, provider `references` pinned (not `unpinned_pending_fixture`), and CPM rejeu couche A+B against the snapshot. Persistence stores the JSON opaquely — no Nicetry logic in `cafe-persistence`.

## Ranking and policy decision output 

`internal/domain/policy/policy_decision.go` defines `PolicyDecisionEvaluator`, `PolicyDecision`, `RankedPolicy`, and `RejectedPolicy`. It applies deterministic first-version ranking over compatible candidates:
1) exclude incompatible routes,
2) better alignment of selection `target_posture` with candidate `required_posture` / profile `resulting_posture`,
3) higher maturity,
4) better chain coverage,
5) better address-continuity matching,
6) avoid new wallet creation when allowed,
7) final lexical tie-break on normalized stable `policy_id` (derived from instance id while no dedicated policy id exists).

Soft findings and `claim_status` ride along on ranked rows; they do not change the ranking key order above.
## Inbound explicit assessment request

`internal/integration/nats` now contains a consumer for `policy.assessment.requested.v0.1` (`cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01`).

Behavior:
- validates inbound payloads with shared `cafe-contracts/cafenatsv01` contracts.
- maps `selection_request` into `policy.PolicySelectionRequest` and delegates execution to a thin handler interface.
- normalizes wallet subject identifiers (`wallet:0x...`) to canonical lowercase for machine-facing consistency.
- uses inbound `event_id` as idempotence key via `IdempotencyStore`.
- treats duplicate deliveries and replayed `event_id` values as no-op.
- releases in-flight claim on handler failure so retry is allowed.
- ignores non-matching subjects (including informational `cafe.discovery.wallet.observed`), so observation publications do not auto-trigger assessment.

Tests in `internal/integration/nats/assessment_consumer_test.go` cover:
- first delivery
- duplicate delivery suppression
- replay after preloaded processed state (simulated post-restart behavior)
- retry after transient handler failure
- non-triggering behavior for `cafe.discovery.wallet.observed`
- canonical lowercase normalization of wallet subject identifiers before handler delegation

## Health and metrics endpoints

| Endpoint | Auth | Purpose |
| --- | --- | --- |
| `GET /healthz` | Public | Liveness (`<service-name> ok`) |
| `GET /metrics` | Public | Prometheus scrape (CPM application registry; IMM-OPS-1 explore counter) |
| `GET /version` | Public | Deployed service version (`{"version": "…"}`; CPM-OPS-3, Discovery-aligned) |

`GET /health` is not registered in CPM runtime.

### Version endpoint (CPM-OPS-3)

Public `GET /version` returns the running image version for Platform Status (**US18** / **CPM-UI-7A**):

```bash
curl http://localhost:8082/version
```

Response:

```json
{
  "version": "v1.2.3"
}
```

The response shape **must** remain `{"version": "..."}`; frontend and `cafe-deploy` rely on it (same contract as Discovery).

#### Version flow (end-to-end)

1. **GitHub Action** (`docker-rc.yml` / `docker-release.yml`): sets `APP_VERSION` from the Git tag or RC label (e.g. `v1.2.3`) and passes `--build-arg APP_VERSION=...` to `Dockerfile`.
2. **Dockerfile**: embeds the resolved version in the binary via `-ldflags` (`internal/version`). Optional runtime override: env `APP_VERSION`.
3. **CPM container**: serves `GET /version` on the main HTTP listener (`CPM_HTTP_ADDR`, default `:8082` locally, `:8080` in compose).
4. **Infra** (`cafe-deploy`, follow-up PR): NGINX proxies `location = /api/cpm/version` → `http://cafe-cpm:8080/version`.
5. **Frontend** (**CPM-UI-7A**, after deploy): `platformService.getCpmVersion()` calls `/api/cpm/version` and displays the CPM version on Platform Status.

Edge routing for `/api/cpm/version` is tracked in [`cafe-deploy`](https://github.com/create2-labs/cafe-deploy); this repo owns the service endpoint only.

## Outbound CPM events 

`internal/integration/nats/outbound_producer.go` publishes shared `cafenatsv01` contracts:

- `cafe.cpm.events.policy.assessment.completed.v0_1`
- `cafe.cpm.events.policy.remediation.requested.v0_1`

Producer behavior is replay-safe and deterministic:

- duplicate identity key: `{subject}:{event_id}`
- duplicate with same payload hash: allowed and deterministic (same JSON projection)
- duplicate with divergent payload hash: rejected

`PolicyRemediationRequested` mapping keeps `auto_start_remediation` intent explicit by appending `informational_only=true` to `correlation_ref` when auto-start is false.

### Producer-side documentation

- Discovery README: [Data structure (CPM export contract)](https://github.com/create2-labs/cafe-discovery/blob/main/README.md#data-structure-cpm-export-contract)
- CAFE developer guide: [Discovery to CPM](https://github.com/create2-labs/cafe-documentation/blob/main/03-cafe-developer-guide.md#discovery-to-cpm-normalized-wallet-observation)

## Read APIs

CPM exposes read-only catalogue APIs backed by local Crypto Policy and provider manifest files loaded at startup.

Environment variables:

- `CPM_CRYPTO_POLICY_PATHS` (comma-separated, default: `/app/policy/crypto_policy_pq_account_validation_v1.json`) — ships Crypto Policy `cpm_pq_account_validation_v1` (`required_posture` + `allowed_providers: ["nicetry"]`)
- `CPM_PROVIDER_MANIFEST_PATHS` (comma-separated, default: `/app/policy/provider_manifest_nicetry_v0_1.json`): Capability Provider manifests (`ProviderManifest` v0.1). Loaded for `GET /providers` and explore hard/soft checks. Fixture source: `internal/domain/provider/testdata/provider_manifest_nicetry_v0_1.json` (refs pinned — NiceTry `commit` `40a1286d18dee2a92631da82a52e484fa9a3628c`; Ephemeral-Keys-Protocol `version` `ac140c71d400449adec18884c4fd3373592292f3` = main HEAD SHA, no release tags yet — **CPM-P7**).

Catalogue responses are posture + `allowed_providers` / provider-manifest oriented; they do **not** return templates, instances, or a business policy graph.

**Catalogue startup signals (ADR §7.2.1 family 1 / CPM-P11a):** after loading Crypto Policies and provider manifests, CPM emits structured logs (no `/admin` endpoint):

- `WARN catalogue: posture orphanage …` when a CP has empty `allowed_providers` or no allowed profile with `resulting_posture == required_posture` (chain `planned` / wallet type are **not** part of this static check — no false alert for posture-OK + planned-only).
- `ERROR catalogue: malformed suggested_user_constraints …` when a profile’s `suggested_user_constraints` contradict `constraints` / signature; the registry marks that profile `Erroneous` (explore still returns `compatibility_status=erroneous` per CPM-P9b).

Optional Prometheus counters: `cpm_catalogue_posture_orphan_total`, `cpm_catalogue_malformed_manifest_total`.

**Runtime signals (ADR §7.2.1 family 2 / CPM-P11b):** contextual to a scan + CP (+ user constraints). Distinct from catalogue startup signals:

| Signal | When | Log / metric |
| --- | --- | --- |
| Aucun scan-compatible | Explore HTTP 200 with empty `scan_compatible_providers` and non-empty `rejected_candidates` | `event=cpm.explore.no_deployable_candidate` + `adr_signal=runtime.no_scan_compatible` ; counter `cpm_explore_no_deployable_candidate_total` (IMM-OPS-1) |
| Couche B KO | Persist gate fails with `PROVIDER_USER_CONSTRAINTS_INCOMPATIBLE` (scan-compatible snapshot rejected by `user_constraints`) | `event=cpm.persist.user_constraints_incompatible` + `adr_signal=runtime.no_provider_after_user_constraints` ; counter `cpm_persist_user_constraints_incompatible_total` |

These are **not** catalogue/admin alerts and **not** fired at startup/load.

Endpoints:

- `GET /api/cpm/v1/crypto-policies`
- `GET /api/cpm/v1/crypto-policies/{crypto_policy_id}`
- `GET /api/cpm/v1/providers`
- `GET /api/cpm/v1/providers/{provider_id}`
- `POST /api/cpm/v1/policies/decisions/explore` (wire v0.2: `crypto_policy_id` + `policy_context`; output `scan_compatible_providers` via couche A match)

`POST /api/cpm/v1/policies/decisions/explore` accepts:

- `crypto_policy_id` (required) — catalogue Crypto Policy id
- `policy_context` (wallet scan context: `scan_id`, `wallet_address` / `target_address`, `wallet_type`, `chain_ids`, `current_algorithm`, `current_pq_posture`, `scanned_at`, `status` — converted server-side into the evaluator’s normalized payload)
- optional top-level `scan_id` (`string`): when present and non-empty, AUTH-02 delegates to Discovery (`can-read`); if `policy_context.scan_id` is also set, it must match. Discovery’s `scan_id` is the id of **one persisted scan result row** (a new scan run creates a new row with a new id; stable for that row’s lifetime — see `WORKPLAN_API.md` §2.2 and Discovery `wallet_policy_context` docs).

Legacy `selection_request` / couche-B fields are **rejected with HTTP 400**.

and returns `PolicyDecision` with public key **`scan_compatible_providers`** (couche A: posture, wallet type, deployable chain + capabilities including `rotate_signer` when `per_userop`, soft findings, indicative `suggested_user_constraints`) plus optional `rejected_candidates` and `warnings`. Couche B does not influence explore membership.

**Couche A match:** for each provider in `allowed_providers`, resolve solution profiles and apply ADR §5.3 / §7 couche A. Contradictory `suggested_user_constraints` → `compatibility_status=erroneous` (not scan-compatible).
## Explore no-deployable-candidate observability (IMM-OPS-1)

When `POST …/decisions/explore` returns HTTP **200** with **no** scan-compatible provider and **non-empty** `rejected_candidates`, CPM emits platform observability (REQ9). This is **not** an HTTP error — it is the ADR §7.2.1 family-2 signal **aucun scan-compatible** (`adr_signal=runtime.no_scan_compatible`). Discovery context is usable but no catalog route is deployable on the requested chain set. Distinct from persist couche B KO (`adr_signal=runtime.no_provider_after_user_constraints`).

**Hook:** `internal/api/read_api.go` — after building the explore decision, before `respondJSON(200)`. The explore JSON response uses `scan_compatible_providers`.

**Structured log** (`event=cpm.explore.no_deployable_candidate`):

- May include `scan_id`, `requested_chain_ids`, `observed_chain_ids`, `missing_chain_ids`, `rejection_codes`, candidate instance/template ids, `request_id`.
- Wallet address is **never** logged in clear text — only `wallet_address_hash` (normalized address, SHA-256 truncated).

**Prometheus counter:** `cpm_explore_no_deployable_candidate_total`

| Label | Meaning |
| --- | --- |
| `rejection_code` | Dominant code for the event (priority: `incompatible.chain_scope`, else first stable blocking code, else `unknown`). **One increment per explore event.** |
| `wallet_type` | Canonical account kind from `policy_context`, or `unknown`. |
| `binding` | `discovery` when `scan_id` or Discovery context is present; else `unknown`. |
| `missing_chain_count` | Bucket: `0`, `1`, `2`, `3`, `4_plus`, or `unknown` (minimum missing chains among `incompatible.chain_scope` rejections). |

High-cardinality values (`scan_id`, wallet address/hash, `chain_ids`, policy/catalog ids, tenant/owner/request ids) are **not** used as Prometheus labels.

**Metrics endpoint:** `GET /metrics` (public, same route class as `/healthz`). Uses a dedicated CPM Prometheus registry (`internal/metrics`). Grafana scrape and dashboards are **IMM-OPS-2** (`cafe-deploy`).

Tracking: [`workplans/IMMUTABILITE_PR.md`](./workplans/IMMUTABILITE_PR.md) § IMM-OPS-1.

## Auth/Authz contract (AUTH-00)

AUTH-00 freezes the cross-repo contract required for CPM authenticated rollout:

- JWT expectations;
- principal model;
- error payload schema for 401/403;
- scan authorization request/response outcomes;
- route classification policy.

See [`AUTH_CONTRACT.md`](./AUTH_CONTRACT.md) and typed models in `internal/authz/contract.go`.

## AUTH-01 runtime authentication wiring

CPM now applies authentication middleware to classified business routes.

Environment variables:

- `CPM_AUTH_REQUIRED` (default: `true`)
- `CAFE_SESSION_JWT_VALIDATION_URL` (required when auth is enabled)
- `CAFE_SESSION_JWT_VALIDATION_TIMEOUT_SEC` (default: `3`)
- `CAFE_SESSION_JWT_VALIDATION_SERVICE_TOKEN` (optional placeholder for service-to-service auth)
- `CAFE_SCAN_AUTHORIZATION_URL` (required for scan-bound operations)
- `CAFE_SCAN_AUTHORIZATION_TIMEOUT_SEC` (default: `3`)
- `CAFE_SCAN_AUTHORIZATION_SERVICE_TOKEN` (optional placeholder for service-to-service auth)
- `CAFE_POLICY_REFERENCE_INTERNAL_SERVICE_TOKEN` (required when `CPM_AUTH_REQUIRED=true` for `POST /internal/policies/references/scan`; shared secret for Discovery → CPM scan-reference check — WORKPLAN PR5)
- `CPM_AUTH_CLOCK_SKEW_SEC` (default: `30`)

### CP-PERSIST runtime

CP-PERSIST V1 does **not** add new mandatory runtime environment variables. Wallet authorization is verified at persist time from the request body (`signed_message`, `signature`) on `POST /api/cpm/v1/drafts/{draft_id}/persist` (PR4). Optional `CPM_WALLET_AUTH_DOMAIN` (or request host fallback) is embedded in canonical messages from `POST /api/cpm/v1/wallet-challenges`. See [`docs/CP_PERSIST.md`](./docs/CP_PERSIST.md).

**Provider snapshot gate (CPM-P10 / ADR §9):** the draft `payload` must be a normative `cafe.crypto_policy.v0.2` document with `crypto_policy_id`, `user_constraints`, `required_posture`, `solution_profile_ref`, and `accepted_provider_snapshot` (pinned `references`, soft findings in `accepted_findings`, no `planned` chain). CPM rejoue couche A+B against the snapshot before the opaque write to cafe-persistence. The shipped Nicetry fixture refs are pinned (**CPM-P7**); `unpinned_pending_fixture` (or empty commit/version) still fails the gate. Details: [Capability Providers](#capability-providers-adr-2026-08-03) and OpenAPI `CryptoPolicyPersistPayload`.

**V2 optional (not V1):** `CPM_REDIS_URL` and ephemeral proof stores may be introduced later for advanced workflows — not required for CP-PERSIST V1.

Remaining client-side gaps are documented in [`docs/CP_PERSIST.md`](./docs/CP_PERSIST.md#expected-implementation-gaps-after-pr4). Smokes: backend [`test-cpm-cp-persist-t4-draft-persist.sh`](https://github.com/create2-labs/cafe-deploy/blob/main/scripts/test-cpm-cp-persist-t4-draft-persist.sh); Web UI [`test-cpm-cp-persist-t5-web-ui-flow.sh`](https://github.com/create2-labs/cafe-deploy/blob/main/scripts/test-cpm-cp-persist-t5-web-ui-flow.sh); CLI [`test-cpm-cp-persist-t6-cli-flow.sh`](https://github.com/create2-labs/cafe-deploy/blob/main/scripts/test-cpm-cp-persist-t6-cli-flow.sh). E2E product doc: [cafe-documentation `docs/security/cp-persist-v1.md`](https://github.com/create2-labs/cafe-documentation/blob/main/docs/security/cp-persist-v1.md).
Important:
- **`CPM_AUTH_REQUIRED=false`** disables the entire auth middleware: user JWT routes and **`POST /internal/policies/references/scan`** are unauthenticated at CPM — use only in controlled local dev. **Staging/production** should keep **`CPM_AUTH_REQUIRED=true`** and set **`CAFE_POLICY_REFERENCE_INTERNAL_SERVICE_TOKEN`** (see `cafe-deploy` env templates and **WORKPLAN_API_PR** PR9).
- CPM validates **Discovery-issued session JWTs**.
- CPM does not define a separate JWT model/secret.
- CPM does not issue or validate a CPM-specific JWT. It accepts the same Bearer session token issued by Discovery and delegates authoritative cryptographic validation to Discovery.
- The current Discovery session token is a PQC hybrid JWS JSON envelope encoded as base64url, carrying EdDSA and ML-DSA-65 signatures.
- Current Discovery session-token semantics do not rely on issuer/audience validation, so CPM AUTH-01 does not enforce `iss`/`aud`.
- Identity is derived from Discovery-validated claims, primarily `user_id` and optionally `email`.
- CPM may perform local structural and expiry fast-fail checks, but it only injects request principal after Discovery validation succeeds.
- If the Discovery validation endpoint returns only pass/fail (without claims), CPM falls back to claims parsed from the already validated token payload.
- `CAFE_SESSION_JWT_VALIDATION_URL` must point to an internal-only Discovery endpoint.
- Optional config: `CAFE_SESSION_JWT_VALIDATION_SERVICE_TOKEN` for service-to-service protection (temporary placeholder; to be replaced by first-class service identity in later auth hardening).
- AUTH-02 adds fail-closed scan authorization for requests carrying a `scan_id` field in the JSON body by delegating to Discovery scan visibility checks.

Public routes:

- `GET /healthz`
- `GET /metrics`
- `GET /version`

Authenticated business routes:

- `GET /api/cpm/v1/crypto-policies`
- `GET /api/cpm/v1/crypto-policies/{crypto_policy_id}`
- `GET /api/cpm/v1/providers`
- `GET /api/cpm/v1/providers/{provider_id}`
- `POST /api/cpm/v1/policies/decisions/explore`

Additional authenticated routes (owner-scoped draft / policy payloads):

- `POST /api/cpm/v1/drafts` — upsert platform draft (`DraftUpsertRequest`)
- `GET /api/cpm/v1/drafts?id=...` — single draft by id (query `id` required; no list without `id`)
- `DELETE /api/cpm/v1/drafts?id=...` — remove platform draft (W1 unblock)
- `POST /api/cpm/v1/policies` — upsert `{ "id", "scan_id?", "payload" }`
- `GET /api/cpm/v1/policies?id=...`

### Platform drafts API (CPM-DRAFT-1 contract)

Canonical spec: [`workplans/CPM_DRAFT_1_PR.md`](workplans/CPM_DRAFT_1_PR.md), OpenAPI [`openapi/cpm-v1.yaml`](openapi/cpm-v1.yaml), WORKPLAN [`workplans/WORKPLAN_API.md`](workplans/WORKPLAN_API.md) §4.4.1.

**Decisions (frozen):**

- **`draft_id`:** client-supplied `id` on every `POST` (generate UUID on first save; reuse on update).
- **Response:** `draft_id`, `saved_at`, `status: "server_draft"` — no `draft_version` until versioning exists.
- **`DELETE`:** `204` if removed; `404` if unknown / out of scope / already deleted (product effect is idempotent).

**POST body (`DraftUpsertRequest`):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "payload": { "selected_candidate_id": "cpx_pq_account_validation_v1" }
}
```

**POST response (`DraftUpsertResponse`):**

```json
{
  "draft_id": "550e8400-e29b-41d4-a716-446655440001",
  "saved_at": "2026-06-02T10:00:00.000Z",
  "status": "server_draft"
}
```

**Rejected client fields:** `owner_user_id`, `tenant_id`, `binding` (owner scope from JWT only; `binding` forbidden on drafts).

**Structured errors (`4xx`):**

| Code | HTTP | When |
|------|------|------|
| `DRAFT_ID_REQUIRED` | 400 | Missing `id` on POST or missing query `id` on GET/DELETE |
| `DRAFT_PAYLOAD_REQUIRED` | 400 | Missing `payload` or not a JSON object |
| `DRAFT_SCAN_ID_INVALID` | 400 | `scan_id` present but not a valid UUID |
| `DRAFT_BINDING_FORBIDDEN` | 400 | Client sent `binding` in draft upsert body |
| `DRAFT_OWNER_FIELDS_FORBIDDEN` | 400 | Client sent `owner_user_id` or `tenant_id` |
| `DRAFT_NOT_FOUND` | 404 | GET unknown id; DELETE unknown / already deleted |

**curl examples** (replace `$TOKEN`, `$SCAN_ID`, `$DRAFT_ID`):

```bash
curl -sS -X POST "https://localhost/api/cpm/v1/drafts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$DRAFT_ID\",\"scan_id\":\"$SCAN_ID\",\"payload\":{\"selected_candidate_id\":\"cpx_pq_account_validation_v1\"}}"

curl -sS "https://localhost/api/cpm/v1/drafts?id=$DRAFT_ID" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X DELETE "https://localhost/api/cpm/v1/drafts?id=$DRAFT_ID" \
  -H "Authorization: Bearer $TOKEN" -w "\nHTTP %{http_code}\n"
```

**Not accepted:** `{ "draft": { ... } }` request body; `{ "item": { ... } }` Go-struct leak on POST response (removed in CPM-DRAFT-1B).

## AUTH-03 owner-scoped persistence

Draft and policy records are **owner-scoped**: access is always tied to the authenticated principal (`user_id`, optional `tenant_id`), never to client-supplied owner fields.

Rules enforced by the `PolicyStore` layer (`internal/persistence`):

- owner is derived from authenticated principal context;
- cross-user read/update is rejected;
- write operations require a valid principal;
- no API accepts client payload owner identity as authoritative.

**Runtime storage** is durable via **cafe-persistence** (HTTP `internal/cp/v1`). See [Durable CP storage](#durable-cp-storage-cpm_store).

**Tests only:** `OwnerScopedStore` (`owner_scoped_store_dev.go`, `//go:build dev`) is an in-process fake used by unit/handler tests so they do not require a running cafe-persistence instance. It is **not** linked into production binaries and **must not** be used for staging, production, or normal local stack runs.

## Durable CP storage (`CPM_STORE`)

Normative detail: [`docs/PERS_D5C_REMOVE_MEMORY.md`](./docs/PERS_D5C_REMOVE_MEMORY.md) (follows [PERS-D5b](./docs/PERS_D5B_ROLLOUT.md) staging/prod rollout).

| Context | `CPM_STORE` | Backend |
| --- | --- | --- |
| **Deployed CPM** (dev stack, staging, prod) | `persistence` (default) | `cphttp.Client` → cafe-persistence |
| **Unit / handler tests** | `memory` (only with `-tags dev`) | `OwnerScopedStore` — in-process, **tests only** |
| **Production binary** | `memory` | **Rejected** at startup |

Environment variables (runtime — **persistence required**):

- `CPM_STORE` (default: `persistence`) — only `persistence` is valid in deployed images.
- `CPM_PERSISTENCE_URL` — cafe-persistence origin (e.g. `http://cafe-persistence:8082`).
- `CAFE_PERSISTENCE_SERVICE_TOKEN` — bearer for `internal/cp/v1`.
- `CPM_PERSISTENCE_TIMEOUT_SEC` (default: `15`).

**Postgres only — no Redis CP cache (P0):** cafe-persistence stores durable CP in Postgres (`crypto_policy_*` tables) via `internal/cp/v1`. Unlike wallet/TLS scans (Postgres + Redis cache in cafe-persistence), **there is no Redis layer for CP** in P0. CPM never talks to Postgres or Redis directly. Optional Redis CP accelerators are P1+ only (ADR §8.2). See [`cafe-deploy/docs/RUNBOOK_CP_PERSISTENCE.md`](https://github.com/create2-labs/cafe-deploy/blob/main/docs/RUNBOOK_CP_PERSISTENCE.md#storage-postgres-only-no-redis-cp-cache).

**Why `memory` still exists:** handler tests (`draft_persist`, wallet challenge, policy references, etc.) exercise owner-scoped routes against an in-memory `PolicyStore` without Postgres, Redis, or HTTP to cafe-persistence. That keeps CI fast and removes a hard dependency on the data plane for CPM-only test runs.

**What `memory` is not:** not a rollback path, not a dev-stack shortcut, not a supported runtime mode in Docker images. If cafe-persistence is down, CPM returns **503** on durable CP operations (ADR §5.5) — restore persistence; do not set `CPM_STORE=memory`.

Run tests (includes in-memory store code):

```bash
go test -tags dev ./...
```

Deploy / local stack wiring: [`cafe-deploy` env templates](https://github.com/create2-labs/cafe-deploy) and [`RUNBOOK_CP_PERSISTENCE.md`](https://github.com/create2-labs/cafe-deploy/blob/main/docs/RUNBOOK_CP_PERSISTENCE.md).

Legacy anonymous records strategy (AUTH-03):

- authenticated owner-scoped reads do not expose legacy anonymous draft/policy data;
- no backfill migration in the AUTH-03 rollout;
- test fixtures can be dropped or regenerated freely.

## AUTH-04 auth error contract and observability

CPM now returns stable JSON payloads for authn/authz failures:

```json
{
  "code": "AUTHZ_SCAN_FORBIDDEN",
  "message": "scan access denied",
  "details": {},
  "request_id": "req_..."
}
```

Contract guarantees:
- `code`, `message`, `details`, and `request_id` are always present in auth-related error payloads.
- `X-Request-Id` is propagated when provided and generated otherwise.
- The same request id is echoed in both response header and JSON payload.

AUTH-04 stable codes and mappings:
- `AUTH_UNAUTHENTICATED` -> `401`
- `AUTH_VALIDATION_UNAVAILABLE` -> `503`
- `AUTHZ_SCAN_ID_MALFORMED` -> `400`
- `AUTHZ_SCAN_ID_CONFLICT` -> `400`
- `AUTHZ_SCAN_FORBIDDEN` -> `403`
- `AUTHZ_SCAN_UNAVAILABLE` -> `503`
- `AUTHZ_OWNER_FORBIDDEN` -> `403`
- `AUTHZ_PRINCIPAL_REQUIRED` -> `401`

Observability:
- structured auth decision logs include request id, method, route class, category, outcome, and safe reason code.
- logs may include authenticated `user_id` and `tenant_id` only after principal resolution.
- logs must not include raw bearer tokens, authorization headers, full claims, emails, secrets, or request bodies.

Metrics:
- decision counter name: `cpm_auth_decisions_total`.
- label set: `category` (`authn`, `scan_authz`, `owner_authz`), `outcome` (`allowed`, `denied`, `unavailable`, `malformed`), `code` (stable auth code or `OK`), `route` (low-cardinality route class).
- no high-cardinality labels (no user id, scan id, raw path, token, or request id).

Audit hook:
- AUTH-04 adds a small internal audit sink interface with default no-op implementation.
- events are emitted for scan authorization denied and owner access denied decisions.
- no external audit storage is implemented in this phase.

## Option A integration (Discovery v1 wallet scans)

**Option A** is the post-V1 path where CPM uses **real wallet scans** from the authenticated **Discovery backend** (not mock `scan_id` placeholders or direct DB access). See [`workplans/CPM_post_v_1_option_a_scan_context.md`](./workplans/CPM_post_v_1_option_a_scan_context.md) for product intent, Option A vs Option B, and data-flow constraints.

Canonical end-to-end narrative: [`docs/CPM_OPTION_A_INTEGRATED.md`](./docs/CPM_OPTION_A_INTEGRATED.md) (Discovery v1 list/detail → CPM explore with **`policy_context`** → persist). Maintainer field mapping: [Discovery `CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md`](https://github.com/create2-labs/cafe-discovery/blob/main/docs/CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md). Public summary: [cafe-documentation `docs/architecture/cpm-option-a-v1-flow.md`](https://github.com/create2-labs/cafe-documentation/blob/main/docs/architecture/cpm-option-a-v1-flow.md). Product and technical specs (English): [functional-specifications.md](https://github.com/create2-labs/cafe-documentation/blob/main/functional-specifications.md), [technical-specifications.md](https://github.com/create2-labs/cafe-documentation/blob/main/technical-specifications.md).

Integrators use Discovery **`/discovery/v1/wallets/scans`** (edge: **`/api/discovery/v1/wallets/scans`**) and CPM **`/api/cpm/v1/...`** only.

## Run locally

```bash
go test ./...
# Default bind: :8082 (override with CPM_HTTP_ADDR)
go run ./cmd/cafe-cpm
```

With repo test fixtures and auth disabled:

```bash
export CPM_AUTH_REQUIRED=false
export CPM_CRYPTO_POLICY_PATHS=internal/domain/policy/testdata/crypto_policy_pq_account_validation_v1.json
export CPM_PROVIDER_MANIFEST_PATHS=internal/domain/provider/testdata/provider_manifest_nicetry_v0_1.json
go run ./cmd/cafe-cpm
```

## IMM-OPS-1 smoke script (`scripts/test-imm-ops-1.sh`)

Validates explore no-deployable-candidate observability (unit tests, HTTP smoke, optional local CPM startup). Requires `curl` and `jq` (or `python3`) for smoke JSON assertions.

```bash
./scripts/test-imm-ops-1.sh unit          # go test (IMM-OPS-1 packages)
./scripts/test-imm-ops-1.sh smoke         # against running CPM (default http://127.0.0.1:8082)
./scripts/test-imm-ops-1.sh smoke -v      # + explore JSON, /metrics lines, structured log
./scripts/test-imm-ops-1.sh local           # start CPM with fixtures, smoke, stop
./scripts/test-imm-ops-1.sh local -v      # recommended first run
./scripts/test-imm-ops-1.sh all             # unit + local
```

**What the smoke test does**

1. `GET /healthz`, `GET /metrics`, and `GET /version`
2. **No-candidate case** — `POST …/decisions/explore` (transitional until CPM-P9; production catalogue no longer ships explore instances after CPM-P8 — smoke may see empty candidates until P9)
3. Prints **catalogue snapshot** via `GET /crypto-policies` (intention: `required_posture` + `allowed_providers`)
4. Asserts log `cpm.explore.no_deployable_candidate` and increment of `cpm_explore_no_deployable_candidate_total` when rejected candidates are present
5. **Negative case** — explore with deployable targets when candidates exist → metric must **not** increment again

**Environment variables**

| Variable | Default | Role |
| --- | --- | --- |
| `CPM_BASE_URL` | `http://127.0.0.1:8082` | Smoke target when not using `local` |
| `CPM_HTTP_ADDR` | `:0` (random port) | Listen address in `local` mode |
| `CPM_LOG_FILE` | `$TMPDIR/cpm-imm-ops-1-<timestamp>-<pid>.log` | Server log path (`local`) |
| `VERBOSE` / `-v` | off | Full explore JSON, metrics snippets, catalog snapshot |
| `SKIP_UNIT` / `SKIP_SMOKE` | `0` | For `all` mode |

**Examples**

Local CPM (auth off, `binding=discovery`, structured log assertions):

```bash
./scripts/test-imm-ops-1.sh local -v
```

Against **cafe-deploy** dev stack (`CPM_AUTH_REQUIRED=true` on `cafe-cpm:8082`):

```bash
DISCOVERY_BASE=http://localhost:8080 CPM_BASE_URL=http://127.0.0.1:8082 ./scripts/test-imm-ops-1.sh smoke -v
```

The script signs up/signs in via `cafe-deploy/scripts/lib/discovery-test-user.sh` and omits `scan_id` from explore payloads unless `SCAN_ID` is set (`binding=unknown` — still exercises `incompatible.chain_scope` and increments the Prometheus counter).

Auth-off target only:

```bash
CPM_SKIP_AUTH=1 CPM_BASE_URL=http://127.0.0.1:8082 ./scripts/test-imm-ops-1.sh smoke -v
```
