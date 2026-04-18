# cafe-cpm

`cafe-cpm` is the standalone Crypto Policy Management service for CAFE.

## Boundaries

- Discovery observes and normalizes wallet/account state.
- CPM validates policy documents, selects compatible policy routes, assesses observations, and emits policy outcomes.
- Remediation consumes policy outcomes and plans/executes remediation.

## Explicit non-goals in this stream

- CPM does not scan wallets.
- CPM does not execute remediation.
- CPM does not depend on Discovery database internals.
- Discovery monolith refactoring is out of scope for this PR stream.

## Repository bootstrap scope (PR0)

- minimal Go service skeleton
- minimal config loading from environment variables
- minimal app bootstrap (`/healthz`)
- placeholder packages for API, policy domain, NATS integration, and persistence
- baseline GitHub Actions workflow skeleton aligned with `cafe-discovery`

## Discovery → CPM observation contract (PR1)

- Normative event model: `internal/domain/walletobserved` (`discovery.wallet.observed` v0.1).
- Shared exported vocabulary (account kinds, algorithms, PQ posture, subject types): `internal/domain/vocabulary`.
- Canonical JSON fixture: `internal/domain/walletobserved/testdata/discovery_wallet_observed_v01.json`.

## Run locally

```bash
go test ./...
go run ./cmd/cafe-cpm
```
