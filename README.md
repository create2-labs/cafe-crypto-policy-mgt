# CAFE Crypto Policy Management

`CAFE Crypto Policy Management` (*`CPM`*) is the Crypto Policy Management service for CAFE [cafe-crypto-policy-mgt](github.com/create2-labs/cafe-crypto-policy-mgt).

## Role and boundaries

- Discovery observes wallets, persists scan artifacts, and  maps observations to the shared wire contract from `cafe-contracts`.
- CPM validates policy documents, selects compatible routes, assesses observations, and emits policy outcomes. 
- Remediation consumes policy outcomes and plans or executes migration work.

CPM does not depend on Discovery’s database or internal domain structs. Inbound integration is explicitly user-triggered via `policy.assessment.requested.v0.1`; `cafe.discovery.wallet.observed` remains informational.

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/cafe-cpm` | Service entrypoint |
| `internal/app`, `internal/config` | Bootstrap and configuration |
| `internal/domain/walletobserved` | Thin re-export of shared `cafe.discovery.wallet.observed` v0.1 wire types |
| `internal/domain/vocabulary` | Exported strings for account kind, algorithms, PQ posture, subject type |
| `internal/domain/policy` | Policy domain contracts (`PolicySelectionRequest`, `PolicyGraphCatalog`, `CryptoPolicyTemplate`, `CryptoPolicyInstance`, `CryptoPolicyValidationResult`, `CryptoPolicyAssessmentResult`, `PolicyCompatibilityResult` + `PolicyCompatibilityEvaluator`, and PR13 `PolicyDecision` models/evaluator) |
| `internal/api` | PR17 read-only HTTP APIs for policy inspection and decision exploration |
| `internal/persistence` | Placeholder for future persistence adapters |
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

`internal/domain/policy/compatibility_result.go` defines `PolicyCompatibilityEvaluator`, which classifies a single validated `CryptoPolicyInstance` against a `walletobserved.Payload` and `PolicySelectionRequest`. It returns `PolicyCompatibilityResult` with one of: `compatible_and_deployable`, `compatible_but_not_deployable` (e.g. empty instance scope `chain_ids`), or `incompatible`, with structured `AssessmentFinding` entries. Template-backed instances that omit `node_path` must pass the matching `CryptoPolicyTemplate` so the node path can be resolved.

## Ranking and policy decision output 

`internal/domain/policy/policy_decision.go` defines `PolicyDecisionEvaluator`, `PolicyDecision`, `RankedPolicy`, and `RejectedPolicy`. It applies deterministic first-version ranking over compatible candidates:
1) exclude incompatible routes,
2) better target-posture alignment,
3) higher maturity,
4) better chain coverage,
5) better address-continuity matching,
6) avoid new wallet creation when allowed,
7) final lexical tie-break on normalized stable `policy_id` (derived from instance id while no dedicated policy id exists).

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

## Health endpoint contract

CPM exposes `GET /healthz` as its service health endpoint. `GET /health` is not registered in CPM runtime.

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

CPM now exposes read-only APIs backed by local policy files loaded at startup. These endpoints are for inspection and exploration only.

Environment variables:

- `CPM_POLICY_CATALOG_PATH` (default: `/app/policy/policy_graph_catalog_valid.json`)
- `CPM_POLICY_TEMPLATE_PATHS` (comma-separated, default: `/app/policy/crypto_policy_template_valid.json`)
- `CPM_POLICY_INSTANCE_PATHS` (comma-separated, default: `/app/policy/crypto_policy_instance_valid.json`)

Endpoints:

- `GET /api/v1/policies/catalog`
- `GET /api/v1/policies/templates`
- `GET /api/v1/policies/instances`
- `POST /api/v1/policies/decisions/explore`

`POST /api/v1/policies/decisions/explore` accepts:

- `observation` (`walletobserved.Payload`)
- `selection_request` (`PolicySelectionRequest`)

and returns `PolicyDecision` output that keeps the distinction between:

- `incompatible`
- `compatible_but_not_deployable`
- `compatible_and_deployable`

## Run locally

```bash
go test ./...
go run ./cmd/cafe-cpm
```
