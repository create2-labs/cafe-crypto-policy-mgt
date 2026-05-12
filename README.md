# CAFE Crypto Policy Management

`CAFE Crypto Policy Management` (*`CPM`*) is the Crypto Policy Management service for CAFE [cafe-crypto-policy-mgt](github.com/create2-labs/cafe-crypto-policy-mgt).

## Role and boundaries

- Discovery observes wallets, persists scan artifacts, and maps observations to the shared wire contract from `cafe-contracts`.
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
| `internal/persistence` | Owner-scoped in-memory persistence (`OwnerScopedStore`) for drafts and persisted policy records exposed under `/api/v1/cpm/*` |
| `scripts/` | Operational helpers (e.g. `wallet-scan-and-cpm-policy.sh` for Discovery + CPM over HTTPS) |
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

- `policy_context` (wallet scan context: `scan_id`, `wallet_address`, `wallet_type`, `chain_ids`, `current_algorithm`, `current_pq_posture`, `scanned_at`, `status` — converted server-side into the evaluator’s normalized payload)
- `selection_request` (`PolicySelectionRequest`)
- optional top-level `scan_id` (`string`): when present and non-empty, AUTH-02 delegates to Discovery (`can-read`); if `policy_context.scan_id` is also set, it must match. Evaluation uses the mapped observation derived from `policy_context` plus `selection_request`. Discovery’s `scan_id` is the id of **one persisted scan result row** (a new scan run creates a new row with a new id; stable for that row’s lifetime — see `WORKPLAN_API.md` §2.2 and Discovery `wallet_policy_context` docs).

and returns `PolicyDecision` output that keeps the distinction between:

- `incompatible`
- `compatible_but_not_deployable`
- `compatible_and_deployable`

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
- `CPM_AUTH_CLOCK_SKEW_SEC` (default: `30`)

Important:
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

Public route:

- `GET /healthz`

Authenticated business routes:

- `GET /api/v1/policies/catalog`
- `GET /api/v1/policies/templates`
- `GET /api/v1/policies/instances`
- `POST /api/v1/policies/decisions/explore`

Additional authenticated routes (owner-scoped draft / policy payloads):

- `POST /api/v1/cpm/drafts` — upsert `{ "id", "scan_id?", "payload" }` (`owner_user_id` / `tenant_id` rejected from client)
- `GET /api/v1/cpm/drafts?id=...`
- `POST /api/v1/cpm/policies` — upsert `{ "id", "scan_id?", "payload" }`
- `GET /api/v1/cpm/policies?id=...`

## AUTH-03 owner-scoped persistence foundation

CPM now includes an owner-scoped persistence primitive in `internal/persistence/owner_scoped_store.go` for draft/policy records.

Rules enforced by this layer:
- owner is always derived from authenticated principal context (`user_id`, optional `tenant_id`);
- cross-user read/update is rejected;
- write operations require a valid principal;
- no API should accept owner identity from client payload as authoritative.

Legacy anonymous records strategy (AUTH-03):
- current rollout treats legacy anonymous draft/policy data as inaccessible to authenticated owner-scoped reads;
- no backfill migration is performed in this PR;
- local/dev anonymous datasets can be dropped or regenerated during rollout.

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

## Scripted Discovery → CPM flow (HTTPS-friendly)

[`scripts/wallet-scan-and-cpm-policy.sh`](./scripts/wallet-scan-and-cpm-policy.sh) signs in to **cafe-discovery**, queues `POST /discovery/scan`, polls `GET /discovery/cbom/{address}`, maps CBOM fields into **`policy_context`** for **`POST /api/v1/policies/decisions/explore`**, then optionally persists a minimal record via **`POST /api/v1/cpm/policies`**. In-line help: `./scripts/wallet-scan-and-cpm-policy.sh --help`.

- Defaults for local dev: **`http://localhost:8080`** (**`DISCOVERY_BASE`**) and **`http://localhost:8082`** (**`CPM_BASE`**) when unset in the script. **`go run ./cmd/cafe-cpm`** listens on **`:8082`** by default (`CPM_HTTP_ADDR`, overridable).
- Use **`https://`** in `DISCOVERY_BASE` / `CPM_BASE` behind TLS gateways as needed.
- For self-signed or unknown CAs locally, set **`CURL_INSECURE=1`** (curl `-k`).
- Discovery’s immediate **`POST /discovery/scan` response does not include the internal NATS scan UUID**; the script correlates by wallet address → CBOM. See [cafe-discovery README](https://github.com/create2-labs/cafe-discovery/blob/main/README.md#post-discoveryscan).

Cross-repo narrative: [`cafe-documentation/03-cafe-developer-guide.md`](https://github.com/create2-labs/cafe-documentation/blob/main/03-cafe-developer-guide.md).

## Run locally

```bash
go test ./...
# Default bind: :8082 (override with CPM_HTTP_ADDR)
go run ./cmd/cafe-cpm
```
