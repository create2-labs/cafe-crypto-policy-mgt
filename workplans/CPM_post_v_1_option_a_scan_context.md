# CPM Post-V1 — Option A: Authenticated scan context through Discovery backend

**Canonical definition of Option A** (product intent, constraints, Option A vs Option B). For the reconciled **v1 API** rollout and merged PR index, see [`CPM_OPTION_A_PR_PLAN.md`](./CPM_OPTION_A_PR_PLAN.md), [`docs/CPM_OPTION_A_INTEGRATED.md`](../docs/CPM_OPTION_A_INTEGRATED.md), and Discovery [`CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md`](https://github.com/create2-labs/cafe-discovery/blob/main/docs/CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md).

## 1. Purpose

This document describes the work required to connect the CPM frontend page to real wallet scan data after the CPM frontend V1 work is complete.

The objective is to replace the current placeholder scan context used by CPM with authenticated user-owned wallet scan contexts produced by Discovery and stored through the Persistence Service.

This document focuses on Option A:

```text
CPM frontend
  -> authenticated Discovery backend API
  -> scan data backed by Persistence Service / DB
```



Option A is the short-term integration path. It respects the current AuthN/AuthZ architecture, where authentication and authorization are still embedded in the Discovery backend.

This document does not describe option B, the future extraction of the Persistence Service into a fully independent repository or service boundary. That remains a later architecture step.

Option B
>CPM frontend appelle une API scan-context portée par le Persistence Service extrait, avec AuthN/AuthZ équivalent.

---

## 2. Current situation

The CPM frontend V1 already provides the policy workflow:

```text
policy template selection
  -> policy candidate graph
  -> candidate selection
  -> PolicyDraft construction
  -> parameter editing
  -> local draft save / reload
  -> backend draft save
  -> authoritative validation
  -> EOA challenge gate when required
  -> persisted validated policy
```

However, the CPM page still lacks a real user scan-context source.

Today, the CPM selection context still relies on a placeholder scan id in the frontend when real Discovery scan wiring is absent.

This means that the CPM page can prove the policy journey, but it does not yet allow the user to select one of their real wallet scans as the input for policy selection.

---

## 3. Current Discovery architecture reality

Today, `cafe-discovery` contains at least two service responsibilities:

1. Discovery backend;
2. Persistence Service.

The Persistence Service already exists and is responsible for scan lifecycle persistence, including scan events and persisted scan state.

However, the Discovery backend still has direct database access in the current codebase. It still initializes Postgres, runs migrations, and wires repositories, including scan-related repositories.

Therefore, this document must not assume that the Discovery backend is already DB-free.

The short-term implementation should use the Discovery backend as the authenticated façade for scan-context access.

The target architecture remains that scan data ownership should move behind the Persistence Service boundary, but that refactor is not a prerequisite for this Option A implementation.

---

## 4. Option A decision

Option A means:

```text
The CPM frontend loads user-owned wallet policy contexts from the Discovery backend.
```

The Discovery backend enforces AuthN/AuthZ and exposes a normalized API over scan data that is currently persisted through the Persistence Service / DB layer.

The CPM frontend must not:

- access the database directly;
- access an unauthenticated Persistence Service endpoint directly;
- rely on frontend mock scan placeholders in API mode;
- consume raw database rows as CPM policy input;
- infer policy eligibility purely on the client.

The Discovery backend must return normalized CPM-ready wallet policy contexts, not internal persistence entities.

---

## 5. Intended data flow

```text
User
  -> launches a wallet scan from Discovery UI
  -> Discovery backend authenticates and authorizes the request
  -> Discovery orchestration emits scan events
  -> scanner performs wallet scan
  -> Persistence Service stores scan lifecycle and result data
  -> Discovery backend exposes authenticated wallet policy contexts
  -> CPM frontend loads eligible wallet policy contexts
  -> user selects one scan context
  -> CPM frontend builds PolicySelectionRequest
  -> CPM API returns PolicySelectionResponse
  -> user configures, validates, and persists a crypto policy
```

Short-term Option A:

```text
CPM frontend
  -> Discovery backend authenticated scan-context API
  -> Persistence Service / DB-backed scan data
```

Future target option:

```text
CPM frontend
  -> extracted Persistence Service or scan-context service
  -> equivalent AuthN/AuthZ boundary
  -> scan data
```

---

## 6. Product model

The CPM page should not list raw scans as generic technical records.

It should list wallet policy contexts.

A wallet policy context is a normalized view of a scan result that is suitable for CPM policy selection.

**`scan_id` (short-term):** it identifies **one persisted scan result row** (not the wallet as a permanent product id). A **new** scan execution that persists a **new** result therefore yields a **new** row and **its own** `scan_id`; the id is **stable for the lifetime of that row** (AUTH-02 / CPM binding refer to that row).

Example conceptual model:

```ts
type WalletPolicyContext = {
  scanId: string
  subjectKind: 'wallet'
  walletAddress?: string
  walletType: 'EOA' | 'SMART_ACCOUNT' | 'UNKNOWN'
  chainIds: number[]
  currentPQPosture: 'classical' | 'hybrid' | 'pq_ready' | 'unknown'
  scanStatus: 'completed'
  scannedAt: string
  displayName?: string
  source: 'discovery'
}
```

The exact type name and fields can evolve, but the intent is stable:

```text
CPM consumes policy contexts, not raw scan rows.
```

---

## 7. API boundary

### 7.1 Candidate endpoint

A possible short-term Discovery backend endpoint:

```http
GET /api/discovery/wallet-policy-contexts
```

Alternative names:

```http
GET /api/scans/wallet-policy-contexts
GET /api/policy-contexts/wallets
GET /api/cpm/wallet-policy-contexts
```

The endpoint should return only contexts owned by the authenticated user.

Example response:

```json
{
  "items": [
    {
      "scanId": "scan_01HX...",
      "subjectKind": "wallet",
      "walletAddress": "0xabc...",
      "walletType": "EOA",
      "chainIds": [1, 8453],
      "currentPQPosture": "classical",
      "scanStatus": "completed",
      "scannedAt": "2026-05-10T12:00:00Z",
      "displayName": "0xabc... on Ethereum/Base",
      "source": "discovery"
    }
  ]
}
```

### 7.2 Optional single-context endpoint

For deep links and validation:

```http
GET /api/discovery/wallet-policy-contexts/{scanId}
```

This endpoint should return:

- `200` when the scan exists, belongs to the user, is eligible, and can be used by CPM;
- `401` when the user is not authenticated;
- `403` when the user is authenticated but not allowed to access the scan;
- `404` when the scan does not exist or should not be revealed;
- `409` or `422` when the scan exists but is not eligible for CPM policy selection.

---

## 8. AuthN/AuthZ requirements

AuthN/AuthZ is currently embedded in the Discovery backend.

Therefore, Option A must enforce the existing Discovery AuthN/AuthZ model.

Requirements:

- the scan-context endpoint must require authentication;
- the endpoint must only return scans owned by the authenticated user;
- the endpoint must not leak the existence of scans owned by other users;
- the endpoint must return stable error semantics for unauthorized, forbidden, not found, empty, and network-failure cases;
- the frontend must not bypass this API by calling the Persistence Service or database directly;
- the frontend must not treat the presence of a `scanId` in the URL as proof of authorization.

The CPM API should also be allowed to reject a `PolicySelectionRequest` referencing a scan that is not authorized, not found, stale, or ineligible.

In other words, the Discovery scan-context API is the first authorization boundary, but CPM server-side checks may still be required for defense in depth.

---

## 9. Frontend architecture

The existing `CpmDataSource` should remain the abstraction for CPM operations:

```text
policy templates
policy selection
policy validation
backend draft save
EOA challenge
persisted policy
```

The scan-context source is different. It belongs to the Discovery boundary.

Recommended frontend structure:

```text
src/discovery/walletPolicyContextsDataSource.ts
src/discovery/useWalletPolicyContexts.ts
src/cpm/useCpmScanContext.ts
```

Alternative naming:

```text
src/cpm/scanContext/useCpmScanContext.ts
src/cpm/scanContext/discoveryWalletPolicyContextDataSource.ts
```

Recommended separation:

```text
Discovery scan context data source
  -> loads authenticated user wallet policy contexts

CpmDataSource
  -> performs CPM policy operations

useCpmScanContext
  -> bridges selected wallet policy context into CPM selection state
```

The UI should not load raw scan results directly inside `CryptoPolicyManagement.vue`.

A composable should own:

- loading wallet policy contexts;
- selected context state;
- URL sync with `?scanId=`;
- empty/loading/error states;
- selected scan validation;
- reset behavior when the selected context changes.

---

## 10. CPM page UX

The CPM page should behave as follows.

### 10.1 Initial load

```text
1. Load authenticated user wallet policy contexts.
2. If no context exists, show an empty state explaining that the user must run a wallet scan first.
3. If `?scanId=` exists, try to select the matching context.
4. If `?scanId=` is invalid, not owned, not found, or ineligible, show an explicit error state.
5. If no `?scanId=` exists, require explicit user selection or select the most recent completed context depending on product decision.
6. Once a valid context is selected, call CPM policy selection with that context.
```

### 10.2 Empty state

When the user has no completed eligible wallet scan:

```text
No eligible wallet scan found.
Run a wallet scan in Discovery before defining a crypto policy.
```

The UI may include a link to the wallet scan page.

### 10.3 Selected context

The selected wallet policy context should be visible near the policy workflow:

```text
Selected wallet scan
- wallet address
- wallet type
- chain ids
- current PQ posture
- scan date
```

### 10.4 Changing context

When the selected scan context changes:

- reset current policy selection if it depends on the previous scan;
- clear validation state;
- clear EOA challenge state if tied to wallet address / scan id;
- preserve local draft only if compatibility is explicitly confirmed;
- update URL if `?scanId=` deep linking is enabled.

---

## 11. PolicySelectionRequest integration

Current V1 uses `scanId` as part of the CPM selection context.

For Option A, the first implementation can continue to feed `scanId` into `PolicySelectionRequest`.

Example:

```ts
type PolicySelectionRequest = {
  scanId: string
  policyTemplateId: string
  // existing fields...
}
```

However, the working document should keep the future model open.

Possible future options:

```ts
type PolicySubjectRef = {
  kind: 'wallet_observation'
  id: string
}
```

or:

```ts
type PolicySubjectRef = {
  kind: 'scan'
  id: string
}
```

Recommended short-term decision:

```text
Use scanId for Option A if this matches the current CPM contract.
Do not block implementation on a larger subject-ref redesign.
Document subjectRef as a future evolution.
```

---

## 12. Backend work — Discovery side

### 12.1 Add authenticated endpoint

Add or reuse an authenticated Discovery backend endpoint returning wallet policy contexts for the current user.

The endpoint must:

- require authentication;
- enforce ownership;
- query the current persisted scan source;
- return only completed and eligible wallet scans;
- normalize fields for CPM;
- avoid exposing internal DB models;
- avoid exposing scans from other users;
- provide stable error semantics.

### 12.2 Normalize scan results

Create a mapping layer from persisted scan data to wallet policy context DTO.

The mapping should be explicit and tested.

Example responsibilities:

```text
scan result entity / persisted observation
  -> wallet address
  -> wallet type
  -> chain ids
  -> current PQ posture
  -> completed status
  -> scan timestamp
  -> display metadata
```

### 12.3 Ownership and authorization

The endpoint must use the same AuthN/AuthZ model as the rest of the Discovery backend.

The user id must come from the authenticated request context, not from query parameters.

Bad pattern:

```http
GET /wallet-policy-contexts?userId=...
```

Preferred pattern:

```http
GET /wallet-policy-contexts
Authorization: session/JWT/cookie according to existing app conventions
```

### 12.4 Eligibility filtering

Only return contexts that CPM can use.

Possible eligibility rules:

- scan status is `completed`;
- scan type is wallet-compatible;
- required wallet fields are present;
- scan result is not deleted;
- scan belongs to the authenticated user;
- scan is not too old if product later introduces freshness constraints.

---

## 13. Frontend work — CPM side

### 13.1 Add Discovery scan-context data source

Create a frontend data source for wallet policy contexts.

Example interface:

```ts
export interface WalletPolicyContextDataSource {
  listWalletPolicyContexts(): Promise<WalletPolicyContext[]>
  getWalletPolicyContext(scanId: string): Promise<WalletPolicyContext>
}
```

This should use the existing frontend HTTP client and auth/session conventions.

### 13.2 Add composable

Create a composable that manages scan context state:

```ts
useCpmScanContext()
```

Responsibilities:

- load contexts;
- expose `contexts`, `selectedContext`, `selectedScanId`;
- sync with route query `?scanId=`;
- select a context;
- expose loading and error states;
- validate selected scan availability;
- reset dependent CPM states when selection changes.

### 13.3 Wire into CPM page

Update the CPM page so that:

- no CPM policy selection is requested in API mode until a valid scan context exists;
- the selected scan id replaces the placeholder scan id;
- changing selected scan refreshes the policy selection response;
- EOA gate receives the wallet address / wallet type from the selected context when available;
- errors are displayed as first-class UI states.

### 13.4 Preserve mock mode

Mock mode should remain available.

In mock mode:

- fixture-based wallet policy contexts may be returned;
- the placeholder may still exist internally if needed for demos;
- tests can run offline.

In API mode:

- the frontend must not silently fall back to `mock-discovery-scan-placeholder`;
- if no scan context is available, the UI must show an empty or error state.

---

## 14. Suggested PR sequence

### PR A1 — Discovery API contract and DTO

Goal:

Define the wallet policy context DTO and authenticated endpoint contract.

Scope:

- DTO types;
- endpoint route definition;
- no broad persistence refactor;
- no CPM frontend wiring yet.

Acceptance criteria:

- endpoint requires authentication;
- endpoint returns only current-user wallet policy contexts;
- no raw DB rows exposed;
- tests cover empty, unauthorized, forbidden/not found, and happy path.

---

### PR A2 — Discovery mapping from persisted scans to wallet policy contexts

Goal:

Map persisted wallet scan results to normalized CPM-ready contexts.

Scope:

- mapping layer;
- eligibility filtering;
- unit tests;
- no frontend change.

Acceptance criteria:

- completed wallet scans become wallet policy contexts;
- incomplete/failed/deleted/non-wallet scans are excluded or marked ineligible according to product rules;
- mapping is deterministic.

---

### PR A3 — Frontend Discovery scan-context data source

Goal:

Add the frontend API client/composable for wallet policy contexts.

Scope:

- data source interface;
- API-backed implementation;
- mock implementation for tests/demo;
- typed error model if needed.

Acceptance criteria:

- can list wallet policy contexts;
- supports single-context lookup if endpoint exists;
- handles 401/403/404/empty/network errors;
- no CPM page behavior change yet unless behind a feature flag.

---

### PR A4 — CPM scan selector and route sync

Goal:

Render the scan selector on the CPM page and support `?scanId=`.

Scope:

- scan selector UI;
- selected scan state;
- URL query sync;
- empty/loading/error states;
- no policy persistence behavior changes.

Acceptance criteria:

- user can choose one of their wallet policy contexts;
- invalid `?scanId=` is handled explicitly;
- no policy selection is requested with an invalid scan id.

---

### PR A5 — Feed selected scan context into CPM policy selection

Goal:

Replace placeholder scan id in API mode with the selected real scan context.

Scope:

- connect `useCpmScanContext` with `useCpmPolicySelection`;
- refresh policy selection when scan changes;
- reset validation and EOA challenge state when scan changes;
- preserve mock mode.

Acceptance criteria:

- API mode never uses the placeholder scan id;
- `PolicySelectionRequest.scanId` is the selected authorized scan id;
- changing scan refreshes the CPM graph/candidates;
- EOA gate uses selected wallet context when available.

---

### PR A6 — Documentation and end-to-end tests

Goal:

Document the Discovery -> CPM scan-context flow and add end-to-end coverage.

Scope:

- docs update;
- e2e happy path;
- e2e empty state;
- e2e unauthorized/forbidden state if test infrastructure allows;
- regression tests against placeholder usage in API mode.

Acceptance criteria:

- contributor documentation explains Option A;
- tests prove the user can select a real wallet scan and start CPM policy selection;
- tests prove placeholder scan id is not used in API mode.

---

## 15. Testing strategy

### Backend tests

- authenticated user sees only their own completed wallet scan contexts;
- unauthenticated request returns 401;
- request for another user's scan returns 403 or 404 according to security convention;
- failed/running/deleted scans are excluded or marked ineligible;
- mapping from persisted scan data to wallet policy context is deterministic;
- no raw persistence model leaks into the API response.

### Frontend unit tests

- data source maps API responses to typed contexts;
- composable handles loading, empty, success and error states;
- route `?scanId=` selects the expected context;
- invalid `?scanId=` produces an explicit error;
- changing selected context resets dependent CPM state.

### Integration / e2e tests

- user logs in;
- user has at least one completed wallet scan;
- user opens CPM page;
- user selects wallet scan;
- CPM calls policy selection with the selected `scanId`;
- graph and candidates render;
- validation and persist flow still work;
- API mode does not use `mock-discovery-scan-placeholder`.

---

## 16. Security considerations

The selected scan id is user-controlled input when it comes from the URL.

Therefore:

- the frontend must not trust `?scanId=`;
- Discovery backend must enforce ownership;
- CPM backend should reject unauthorized or invalid scan references;
- logs should not leak sensitive wallet metadata beyond existing logging policy;
- error messages should avoid confirming the existence of another user's scans;
- scan contexts should expose only fields needed by CPM.

---

## 17. Non-goals

This Option A work does not include:

- extracting the Persistence Service from the Discovery repository;
- removing all DB access from the Discovery backend;
- redesigning the full Discovery persistence model;
- implementing Remediation;
- turning the CPM frontend into a policy engine;
- replacing the existing CPM `CpmDataSource` abstraction;
- implementing a generic policy subject model for all future asset types unless needed for compatibility.

---

## 18. Open questions

1. Should Option A expose `scanId` only, or introduce `walletObservationId` immediately?
2. Should the scan-context endpoint live under `/api/discovery`, `/api/scans`, or `/api/cpm`?
3. Should the CPM page auto-select the latest completed wallet scan or require explicit selection?
4. Should `?scanId=` be only a deeplink hint, or should the selected scan be persisted in local user preferences?
5. Should CPM API re-check scan ownership by calling Discovery/Persistence, or trust a signed/validated context passed from the frontend?
6. Should policy selection be allowed for stale scans?
7. Should failed scans appear in the selector as disabled items, or be hidden entirely?
8. How will this evolve once the Persistence Service is extracted from the Discovery repository?

---

## 19. Recommended short-term decision

Implement Option A before attempting to extract the Persistence Service.

The pragmatic short-term target is:

```text
CPM frontend
  -> authenticated Discovery backend scan-context endpoint
  -> normalized wallet policy contexts
  -> selected scanId
  -> CPM PolicySelectionRequest
```

This keeps AuthN/AuthZ correct, avoids direct DB access, avoids unauthenticated Persistence Service access, and delivers the missing product link between Discovery scans and CPM policy selection.

The later architecture can move the scan-context endpoint behind an extracted Persistence Service once equivalent AuthN/AuthZ guarantees exist.

