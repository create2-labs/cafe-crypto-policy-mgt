# PERS-D5c — Remove production memory store path

**Status:** D5c removes `OwnerScopedStore` from production binaries.  
**Prerequisite:** [PERS-D5b](./PERS_D5B_ROLLOUT.md) — staging/prod on `CPM_STORE=persistence` with a documented stability window.

## Behaviour

| Context | `CPM_STORE` | Backend |
|---------|-------------|---------|
| **Deployed CPM** (dev stack, staging, prod) | `persistence` (default) | `cphttp.Client` → cafe-persistence |
| **Unit / handler tests** | `memory` (only with `-tags dev`) | `OwnerScopedStore` — **tests only** |
| **Production binary** | `memory` | **Rejected** at startup |

When `persistence` is selected, CPM **does not** fall back to memory on read/write errors (503 per ADR §5.5). There is **no** env rollback to memory in staging/prod after D5c.

## Why `OwnerScopedStore` / `CPM_STORE=memory` still exist

`OwnerScopedStore` is kept **only to support `go test`** — not as a runtime option for deployed or docker-compose stacks.

Handler and persistence tests (`draft_persist`, wallet challenge, internal policy references, owner routes, etc.) need a `PolicyStore` implementation that:

- enforces the same owner-scoped semantics as cafe-persistence;
- runs in-process with no HTTP, Postgres, or service token;
- keeps the CPM CI job independent of a live cafe-persistence container.

That implementation lives behind `//go:build dev`. Production `go build` (Docker image, staging, prod) does **not** include it.

**`CPM_STORE=memory` is not:**

- a rollback path when cafe-persistence is unavailable;
- a shortcut for local `docker compose` or `go run` without persistence;
- a supported configuration in shipped `cafe-cpm` images.

**`CPM_STORE=memory` is:**

- selected automatically when tests build with `-tags dev` and need an in-memory `PolicyStore`;
- rejected if someone sets it on a production binary at startup.

## Required env (runtime)

| Variable | Purpose |
|----------|---------|
| `CPM_PERSISTENCE_URL` | Origin, e.g. `http://cafe-persistence:8082` |
| `CAFE_PERSISTENCE_SERVICE_TOKEN` | Bearer for `internal/cp/v1` |
| `CPM_PERSISTENCE_TIMEOUT_SEC` | Client timeout (default 15s) |

`CPM_STORE` may be omitted (defaults to `persistence`) or set explicitly to `persistence`.

## Tests

```bash
# Full suite (includes OwnerScopedStore-backed tests)
go test -tags dev ./...
```

CI (`Dockerfile`) runs two passes:

1. **Without** `-tags dev` — `TestNewPolicyStoreRejectsMemoryInProductionBuild` (prod binary must reject `CPM_STORE=memory`).
2. **With** `-tags dev` — full `go test -tags dev ./...`.

There is no documented `go run -tags dev` + `CPM_STORE=memory` workflow; use the cafe-deploy stack with `CPM_STORE=persistence` for manual end-to-end checks.

## Incident response

If cafe-persistence is unavailable, CPM returns **503** on durable CP operations. Restore cafe-persistence or scale it — **do not** attempt `CPM_STORE=memory` rollback (removed from production binaries in D5c).

Deploy runbook: [`cafe-deploy/docs/RUNBOOK_CP_PERSISTENCE.md`](../../cafe-deploy/docs/RUNBOOK_CP_PERSISTENCE.md).

## Smoke

```bash
# From cafe-deploy (stack running, CPM_STORE=persistence)
./scripts/test-cpm-cp-persist-d5-restart-survival.sh
```
