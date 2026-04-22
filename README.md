# cafe-cpm

`cafe-cpm` is the standalone Crypto Policy Management service for CAFE (Go module `github.com/create2-labs/cafe-cpm`).

## Role and boundaries

- Discovery observes wallets, persists scan artifacts, and (on the export path) maps observations to the shared wire contract from `cafe-contracts`.
- CPM validates policy documents, selects compatible routes, assesses observations, and emits policy outcomes. 
- Remediation consumes policy outcomes and plans or executes migration work.

CPM does not depend on Discovery’s database or internal domain structs. Inbound integration uses the explicit `discovery.wallet.observed` contract only.

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/cafe-cpm` | Service entrypoint |
| `internal/app`, `internal/config` | Bootstrap and configuration |
| `internal/domain/walletobserved` | Thin re-export of shared `discovery.wallet.observed` v0.1 wire types |
| `internal/domain/vocabulary` | Exported strings for account kind, algorithms, PQ posture, subject type |
| `internal/domain/policy`, `internal/api`, `internal/integration/nats`, `internal/persistence` | Placeholders for later PRs |

## Execution pack

Implementation order, NATS rules, and acceptance criteria: [`cafe_cpm_v1_prompts_0.5.md`](./cafe_cpm_v1_prompts_0.5.md) (pack v0.5, service model v0.1).

## Discovery → CPM contract (`discovery.wallet.observed` v0.1)

`internal/domain/walletobserved` re-exports the shared contract from `github.com/create2-labs/cafe-contracts/discoverywalletobserved/v01` so CPM code can keep stable local imports without duplicating wire structs.

Normative identifiers (see `internal/domain/walletobserved/contract.go`):

- `event_type`: `discovery.wallet.observed`
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

### Producer-side documentation

- Discovery README: [Data structure (CPM export contract)](https://github.com/create2-labs/cafe-discovery/blob/main/README.md#data-structure-cpm-export-contract)
- CAFE developer guide: [Discovery to CPM](https://github.com/create2-labs/cafe-documentation/blob/main/03-cafe-developer-guide.md#discovery-to-cpm-normalized-wallet-observation)

## Bootstrap (PR0)

- Go service skeleton, env-based config, `/healthz`
- Baseline GitHub Actions aligned with `cafe-discovery`

## Run locally

```bash
go test ./...
go run ./cmd/cafe-cpm
```
