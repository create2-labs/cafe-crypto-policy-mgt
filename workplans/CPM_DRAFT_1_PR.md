# CPM-DRAFT-1 — Platform draft API contract (contract-first)

> **Context:** `POST /api/cpm/v1/drafts` returns **400** when the frontend sends `{ draft: PolicyDraft }`. This is a **contract drift**, not a frontend-only bug.  
> **Policy:** No temporary frontend adapter to the current backend shape. Stabilize CPM contract first, then **CPM-DRAFT-2** (frontend adoption).

**Canonical prefix:** `/api/cpm/v1`  
**Implementation reference (today):** `internal/app/owner_routes.go` (`ownerScopedUpsertRequest`)

**Related plans:** [`WORKPLAN_API.md`](./WORKPLAN_API.md) §2.4 / §4.4, [`openapi/cpm-v1.yaml`](../openapi/cpm-v1.yaml), [`README.md`](../README.md), frontend [`CPM-DRAFT-2`](../cafe-frontend/IMMUTABILITE_PR.md) (after CPM-DRAFT-1 lands).

---

## Contract decisions (frozen for CPM-DRAFT-1)

### D1 — Who generates `draft_id`?

| Option | Description |
|--------|-------------|
| **A (recommended)** | **Client-supplied `id` required** on every `POST /drafts` (upsert key). |
| B | Backend-generated `id` on first save; client omits `id` on create. |

**Decision: Option A.**

**Rationale:**

- Matches **current backend** (`decodeOwnerScopedUpsertRequest` requires `id`).
- Matches **WORKPLAN_API** §2.4 / §4.4 (`POST` body `{ id, scan_id?, payload }`).
- Matches **README** and existing owner-route tests (`{"id":"draft-1",...}`).
- Supports **idempotent upsert** and stable W1 unblock (`DELETE …/drafts?id=…`) without an extra list API.
- Frontend can generate `crypto.randomUUID()` on first save and reuse the same id on update (no owner-scoped draft list on backend).

**Client rule:** first platform save → generate `id` once; subsequent saves → same `id`. Do **not** call bare `GET /drafts` without `id` for discovery.

---

### D2 — `draft_version` in response?

| Option | Description |
|--------|-------------|
| **Minimal (recommended)** | Response exposes only fields backed by persistence: `draft_id`, `saved_at`, `status`. |
| Full | Expose `draft_version` even when store has no semantic version. |

**Decision: Minimal response** — no `draft_version` until store/API define a real version counter or etag.

**Mapping:**

- `draft_id` ← record `id`
- `saved_at` ← record `updated_at` (RFC3339 UTC)
- `status` ← constant `"server_draft"` (platform work-in-progress; not persisted policy)

---

### D3 — `DELETE` semantics

| HTTP | When |
|------|------|
| **204** | Draft existed for this owner and was removed. |
| **404** | Unknown id, out of scope, or **already deleted** (second delete). |

**Product idempotence:** after a successful delete, the draft is gone — a repeat `DELETE` still yields **404**, but the **desired end state** (no platform draft) is already satisfied. Document this for W1 unblock UX; do not treat 404 on repeat as a hard failure.

**Never:** delete the Discovery scan row referenced by `scan_id`.

---

## Target contract (normative)

### `POST /api/cpm/v1/drafts` — upsert platform draft

**Request (`DraftUpsertRequest`):**

```json
{
  "id": "string (required, non-empty)",
  "scan_id": "uuid (optional)",
  "payload": { }
}
```

| Field | Rule |
|-------|------|
| `id` | Required. Client-generated stable key for upsert. |
| `payload` | Required. JSON object (may be empty `{}`). |
| `scan_id` | If present: valid UUID string. Recommended for Discovery→CPM flows. |
| `binding` | **Forbidden** on drafts in CPM-DRAFT-1 → **400** `DRAFT_BINDING_FORBIDDEN`. |
| `owner_user_id`, `tenant_id` | **Forbidden** on client body → **400** `DRAFT_OWNER_FIELDS_FORBIDDEN`. |

**Response (`DraftUpsertResponse`) — 200:**

```json
{
  "draft_id": "same-as-request-id",
  "saved_at": "2026-06-02T10:00:00.000Z",
  "status": "server_draft"
}
```

**Not accepted (drift to remove):**

- Request: `{ "draft": { ...PolicyDraft } }` (frontend legacy shape).
- Response: `{ "item": { "ID": "...", ... } }` (Go struct leak).
- Response: `{ "draftId", "draftVersion", ... }` without OpenAPI alignment (frontend expects camelCase metadata — **CPM-DRAFT-2** maps from snake_case contract).

---

### `GET /api/cpm/v1/drafts?id={draftId}` — single draft by id

| Query | Rule |
|-------|------|
| `id` | **Required**. Missing → **400** `DRAFT_ID_REQUIRED`. |

**Response — 200 (`DraftRecord`):**

```json
{
  "id": "draft-1",
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "payload": { },
  "created_at": "2026-06-01T12:00:00.000Z",
  "updated_at": "2026-06-02T10:00:00.000Z"
}
```

**Out of scope:** owner-scoped **list** without `id` (no `GET /drafts` array). Proactive W1 uses Discovery + internal lookup, not draft listing.

---

### `DELETE /api/cpm/v1/drafts?id={draftId}`

See **D3** above. Cross-owner: **404** (do not leak existence) — align with current policy DELETE tests.

---

## Structured errors (owner routes + drafts)

All draft-related **4xx/5xx** (and owner auth failures) MUST return:

```json
{
  "code": "MACHINE_CODE",
  "message": "Human-readable summary",
  "details": {},
  "request_id": "req_..."
}
```

| Code | HTTP | When |
|------|------|------|
| `DRAFT_ID_REQUIRED` | 400 | `POST` missing/empty `id`; `GET`/`DELETE` missing query `id` |
| `DRAFT_PAYLOAD_REQUIRED` | 400 | `POST` missing `payload` or not a JSON object |
| `DRAFT_SCAN_ID_INVALID` | 400 | `scan_id` present but not a valid UUID |
| `DRAFT_BINDING_FORBIDDEN` | 400 | Client sent `binding` in draft upsert body |
| `DRAFT_OWNER_FIELDS_FORBIDDEN` | 400 | Client sent `owner_user_id` or `tenant_id` |
| `DRAFT_NOT_FOUND` | 404 | `GET` unknown id; `DELETE` unknown/already deleted (owner-safe) |
| `AUTH_UNAUTHENTICATED` | 401 | Missing/invalid session (AUTH-04) |
| `AUTHZ_PRINCIPAL_REQUIRED` | 401 | Principal missing |
| `AUTHZ_OWNER_FORBIDDEN` | 403 | Cross-owner access |
| `INTERNAL_ERROR` | 500 | Unexpected persistence failure |

Legacy `{"error":"id is required"}` on owner routes is **deprecated** — replaced in **CPM-DRAFT-1B**.

---

## PR breakdown

### CPM-DRAFT-1A — Contract freeze + docs

**Status:** ✅ done (docs gelés — pas de PR GitHub requise pour marquer le plan)

**Branch:** `cpm/cpm-draft-1a-contract-docs`

| Deliverable | File(s) |
|-------------|---------|
| This plan + decisions | `workplans/CPM_DRAFT_1_PR.md` |
| OpenAPI drafts aligned | `openapi/cpm-v1.yaml` |
| WORKPLAN_API draft contract | `workplans/WORKPLAN_API.md` §4.4 + appendix |
| README + curl examples | `README.md` |
| Tracking | `workplans/IMMUTABILITE_PR.md` (CPM table) |

**Out of scope:** Go handler changes (1B), tests (1C).

---

### CPM-DRAFT-1B — Backend DTO / validation / errors

**Branch:** `cpm/cpm-draft-1b-backend-dto` (depends on 1A merged or stacked)

| Deliverable | File(s) |
|-------------|---------|
| `DraftUpsertRequest` / `DraftUpsertResponse` types + decode | `internal/app/owner_routes.go` (or `draft_contract.go`) |
| Structured error writer for owner POST/GET/DELETE drafts | `internal/app/owner_routes.go`, `api_constants.go` |
| Map store `DraftRecord` → API `DraftRecord` / `DraftUpsertResponse` | `internal/app/owner_routes.go` |
| Remove `{item: Go struct}` leak on POST drafts | `internal/app/owner_routes.go` |

**Acceptance:**

- `POST` with valid `{id, scan_id, payload}` → 200 + `DraftUpsertResponse`.
- `POST` with `{draft:{...}}` → 400 `DRAFT_PAYLOAD_REQUIRED` or `DRAFT_ID_REQUIRED` (not opaque JSON error).
- Owner fields rejected; JWT-only owner scope unchanged.

---

### CPM-DRAFT-1C — Contract tests

**Branch:** `cpm/cpm-draft-1c-contract-tests`

| Test | File |
|------|------|
| POST create/update happy path | `internal/app/owner_routes_test.go` |
| 400 matrix (id, payload, scan_id, owner fields) | `owner_routes_drafts_contract_test.go` (new) |
| GET with/without id | same |
| DELETE 204 / second 404 / cross-owner 404 | same |
| Contract test runner | `scripts/test-cpm-draft-1-contract.sh` (`go test`; `--smoke` → cafe-deploy) |
| Integration smoke (stack up) | [`cafe-deploy/scripts/test-cpm-draft-1-contract.sh`](../../cafe-deploy/scripts/test-cpm-draft-1-contract.sh) |

---

## CPM-DRAFT-2 — Frontend adoption (after 1A–1C)

**Repository:** `cafe-frontend`  
**Depends on:** CPM-DRAFT-1 merged.

| Task | Notes |
|------|-------|
| Map `savePolicyDraft` → `DraftUpsertRequest` | Build `id` (client UUID), `scan_id`, `payload` from `PolicyDraft` |
| Map response → `BackendDraftSaveResponse` | `draft_id`, `saved_at`, `status` |
| Remove “server mock” copy in API mode | `CpmBackendDraftPersistence.vue` |
| Structured error UX | Map `code` from CPM to `CpmDataSourceError` / boundary copy |
| Tests | `apiCpmDataSource.spec.ts` exact query/body |

**Do not** add `listDrafts()` — still no owner list API.

---

## curl examples (CPM-DRAFT-1A)

Replace `$TOKEN`, `$SCAN_ID`, `$DRAFT_ID` (client-generated UUID).

```bash
# Upsert platform draft (create or update)
curl -sS -X POST "https://localhost/api/cpm/v1/drafts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$DRAFT_ID\",\"scan_id\":\"$SCAN_ID\",\"payload\":{\"selected_candidate_id\":\"cpx_pq_account_validation_v1\"}}"

# Read draft by id
curl -sS "https://localhost/api/cpm/v1/drafts?id=$DRAFT_ID" \
  -H "Authorization: Bearer $TOKEN"

# Delete platform draft (W1 unblock)
curl -sS -X DELETE "https://localhost/api/cpm/v1/drafts?id=$DRAFT_ID" \
  -H "Authorization: Bearer $TOKEN" -w "\nHTTP %{http_code}\n"
```

---

## Git policy

Branches may be created locally per sub-PR. **No** automated commit, push, or merge unless explicitly requested.
