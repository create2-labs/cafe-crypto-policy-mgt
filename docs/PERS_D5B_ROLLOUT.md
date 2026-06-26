# PERS-D5b — `CPM_STORE=persistence` rollout

**Status:** Completed — staging/prod on durable CP via env (`cafe-deploy`).  
**Follow-up:** [PERS-D5c](./PERS_D5C_REMOVE_MEMORY.md) removes the production memory path.

**Prerequisite:** [PERS-D5a](https://github.com/create2-labs/cafe-crypto-policy-mgt) — HTTP client to `internal/cp/v1`.

## Behaviour (historical D5b window)

| `CPM_STORE` | Backend | Survives CPM restart |
|-------------|---------|----------------------|
| `memory` | `OwnerScopedStore` | No |
| `persistence` | `cphttp.Client` → cafe-persistence | Yes |

When `persistence` is selected, CPM **does not** fall back to memory on read/write errors (503 per ADR §5.5).

## Required env (with `CPM_STORE=persistence`)

| Variable | Purpose |
|----------|---------|
| `CPM_PERSISTENCE_URL` | Origin, e.g. `http://cafe-discovery-persistence:8082` |
| `CAFE_PERSISTENCE_SERVICE_TOKEN` | Bearer for `internal/cp/v1` |
| `CPM_PERSISTENCE_TIMEOUT_SEC` | Client timeout (default 15s) |

## Rollback (D5b only — superseded by D5c)

During the D5b stability window, rollback was: set `CPM_STORE=memory` and redeploy **cafe-cpm** only. **No longer available** after [PERS-D5c](./PERS_D5C_REMOVE_MEMORY.md).

Deploy runbook: [`cafe-deploy/docs/RUNBOOK_CP_PERSISTENCE.md`](../../cafe-deploy/docs/RUNBOOK_CP_PERSISTENCE.md).

## Smoke

```bash
# From cafe-deploy (stack running, CPM_STORE=persistence)
./scripts/test-cpm-cp-persist-d5-restart-survival.sh
```
