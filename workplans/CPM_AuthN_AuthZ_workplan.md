# CPM Auth-Only Rollout Plan (Multi-Repo PRs)

This document is kept as archive.

It had initially been written because we had to make CPM usable in **authenticated-only mode** end-to-end:

- all CPM API operations require authenticated user context;
- scan access is authorization-scoped;
- draft/policy persistence is owner-scoped;
- frontend behavior and deploy/e2e checks are aligned.

This document defines the necessary work to do among different repositories. The work is splitted into small PR to keep review quite simple and safe.

## Scope and principles

- No anonymous access to business CPM endpoints (`/api/cpm/*` and any retained policy APIs).
- Enforce `401` (unauthenticated) vs `403` (authenticated but unauthorized).
- Keep authn/authz decisions server-side; frontend only sends token and handles outcomes.
- Fail closed for scan-bound authorization checks.
- Feature flags can help rollout in dev/test, but must never allow anonymous business access in staging/prod.
- Keep health endpoints public unless security policy says otherwise.

## PR tracking table

| Track ID | Repo | PR Title (proposed) | Purpose | Depends on | Owner | Status | PR Link |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AUTH-00 | `cafe-crypto-policy-mgt` (+ docs/contracts if needed) | Define CPM auth/authz contract | Freeze JWT, Principal, error payload, route classification, and scan authz contract | None | TBD | Merged | [PR #18](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/18) |
| AUTH-01 | `cafe-crypto-policy-mgt` | Add JWT auth middleware for CPM APIs | Reject anonymous requests on CPM business routes; inject principal into context | AUTH-00 | TBD | Merged | [PR #19](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/19) |
| AUTH-02 | `cafe-crypto-policy-mgt` | Enforce scan authorization in CPM (fail closed) | Enforce scan-level access control for every flow carrying a `scan_id` (JSON); non-scan drafts remain owner-scoped | AUTH-01, AUTH-05 | TBD | Merged | [PR #20](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/20) |
| AUTH-03 | `cafe-crypto-policy-mgt` | Persist drafts/policies with owner scope | Store and query draft/policy records by authenticated principal | AUTH-01 | TBD | Merged | [PR #21](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/21) |
| AUTH-04 | `cafe-crypto-policy-mgt` | Add authz error contract and observability | Standardize 401/403 payloads, structured logs, metrics, audit hooks | AUTH-01, AUTH-02 | TBD | Merged | [PR #22](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/22) |
| AUTH-05 | `cafe-discovery` | Expose scan authorization lookup for CPM | Provide authoritative scan visibility checks consumed by CPM | AUTH-00 | TBD | Merged | [PR #46](https://github.com/create2-labs/cafe-discovery/pull/46) |
| AUTH-06 | `cafe-deploy` | Enforce and test auth on CPM routes in dev stack | Update stack/e2e: no-token -> 401, invalid-token -> 401, forbidden -> 403, allowed -> success | AUTH-01, AUTH-02, AUTH-05 | TBD | Merged | [PR #10](https://github.com/create2-labs/cafe-deploy/pull/10) |
| AUTH-07 | `cafe-frontend` | Harden CPM auth UX for 401/403 states | Improve CPM page UX on expired session/forbidden scan while keeping data-source boundary | AUTH-01, AUTH-04 | TBD | Merged | [PR #46](https://github.com/create2-labs/cafe-frontend/pull/46) |
| AUTH-08 | `cafe-documentation` | Publish CPM auth-only contract and runbook | Align architecture docs, API contract, and ops/security runbook | AUTH-00..AUTH-07 | TBD | Planned | TBD |

## Detailed PR breakdown

## AUTH-00 - `cafe-crypto-policy-mgt` (+ docs/contracts if needed)

**Title:** Define CPM auth/authz contract

**Why**
- AUTH-01/02/05/06/07 all rely on shared decisions (JWT claims, principal, error model, authz semantics).
- Without this, each repo may implement incompatible conventions.

**Changes**
- Define JWT expectations (issuer, audience, required claims, clock skew policy).
- Define principal shape propagated in CPM handlers (e.g. `user_id`, optional `tenant_id`, `subject`, `claims`).
- Define 401/403 error payload schema (`code`, `message`, `details`, `request_id`).
- Define scan authorization contract between CPM and Discovery:
  - allow / deny / unavailable outcomes,
  - request/response payload,
  - timeout and retry semantics.
- Define route classification rule:
  - `public health/readiness`,
  - `authenticated business endpoint`,
  - `deprecated/disabled`.
  No unclassified route allowed.

**Acceptance**
- Contract is written and referenced by all implementation PRs.
- Auth/authz behavior can be tested from spec only.

---

## AUTH-01 - `cafe-crypto-policy-mgt`

**Title:** Add JWT auth middleware for CPM APIs

**Why**
- Current CPM API surface is callable without explicit authentication middleware.
- Auth-only mode starts by making user identity mandatory for business operations.

**Changes**
- Add HTTP middleware that validates bearer JWT (issuer/audience/signature/expiry according to platform rules).
- Inventory and classify all CPM routes, then apply middleware to every authenticated business route.
- Keep `/healthz` outside auth by default.
- Add request-context principal model (`user_id`, optional `tenant_id`, claims snapshot).
- Add route-inventory test to prevent future unclassified/anonymous business routes.

**Acceptance**
- Missing/invalid/expired token returns `401`.
- Valid token reaches handler with principal in context.
- Unit/integration tests cover all authn paths.

---

## AUTH-02 - `cafe-crypto-policy-mgt`

**Title:** Enforce scan authorization in CPM (fail closed)

**Why**
- Authentication alone is insufficient; CPM must ensure caller can access referenced scans/resources.

**Changes**
- Introduce an authorization boundary in CPM (service interface + adapter).
- Implement the CPM authorization boundary and client adapter now; wire it to Discovery AUTH-05 endpoint contract once fully available.
- On requests using `scan_id` in the JSON body, perform authorization check (`can_read_scan` or equivalent).
- Fail-closed policy:
  - explicit deny -> `403` (anti-enumeration `404` is deferred to a later hardening PR);
  - authz unavailable/timeout -> `503` (or `502` by gateway policy);
  - malformed `scan_id` -> `400`.
- Ensure wallet challenge start/verify paths enforce same scan authorization.
- Apply scan authorization checks consistently to every flow carrying a `scan_id`, including selection, validation, wallet challenge, draft save, and persist flows.
- Drafts without `scan_id` remain owner-scoped but do not require scan authorization.

**Acceptance**
- Cross-user scan access is denied (`403`).
- Authorized user succeeds for same scan.
- In this rollout, forbidden scan access returns `403`; full anti-enumeration concealment is deferred to a later hardening PR.
- Tests verify that forbidden scan access is consistently rejected and never reaches CPM business logic.
- Structured tests cover allow/deny/unavailable/malformed matrix.

---

## AUTH-03 - `cafe-crypto-policy-mgt`

**Title:** Persist drafts/policies with owner scope

**Why**
- Saved drafts and persisted policies must belong to authenticated identities.

**Changes**
- Add ownership fields (`owner_user_id`, optional `tenant_id`) in persistence model.
- On create/update/read, derive owner from principal context (never from client payload).
- Enforce owner filters on reads and writes.
- Prepare migration/backfill strategy if existing records are anonymous.

**Acceptance**
- User A cannot read/update User B drafts/policies.
- All new records include owner scope.
- No endpoint allows setting owner from body.

---

## AUTH-04 - `cafe-crypto-policy-mgt`

**Title:** Add authz error contract and observability

**Status:** Merged in [PR #22](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/22)

**Why**
- Frontend and operations need deterministic auth failure behavior.

**Changes**
- Standardize error payload for 401/403 (`code`, `message`, `details`, `request_id`).
- Add structured logs with auth decision reason (without leaking sensitive claims).
- Add metrics counters (unauthenticated, forbidden, allowed).
- Optional audit event hooks for sensitive actions.

**Acceptance**
- 401/403 payloads are stable and documented.
- Dashboards/logs can distinguish authn vs authz failures.

---

## AUTH-05 - `cafe-discovery`

**Status:** merged in **`cafe-discovery`** as **[PR #46](https://github.com/create2-labs/cafe-discovery/pull/46)** (`POST /internal/authz/scans/:scanId/can-read` plus related Discovery configuration).

**Title:** Expose/confirm scan authorization lookup for CPM

**Why**
- CPM authorization depends on authoritative scan ownership/visibility information.

**Changes**
- Add/confirm API/service contract for scan visibility checks from CPM.
- Ensure contract is principal-aware and tenant-aware.
- Include latency/error budget guidance for CPM caller.
- Define explicit integration mode for CPM -> Discovery authz checks (do not leave implicit), e.g.:
  - HTTP internal endpoint:
    - `POST /internal/authz/scans/{scanId}/can-read`
    - headers: service auth + propagated principal context (`X-User-Id`, optional `X-Tenant-Id`, `X-Request-Id`);
  - or NATS request/reply contract with equivalent semantics.
- Discovery remains authority for scan visibility; CPM must not read Discovery DB directly.

**Acceptance**
- CPM can deterministically resolve whether principal may use a given scan (identified by `scan_id` in JSON; Discovery path params may still spell `scanId` in URLs).
- Contract includes explicit deny semantics and traceable reason codes.

---

## AUTH-06 - `cafe-deploy`

**Title:** Enforce and test auth on CPM routes in dev stack

**Status:** Merged in **`cafe-deploy`** as **[PR #10](https://github.com/create2-labs/cafe-deploy/pull/10)** (dev stack / e2e: no-token, invalid-token, forbidden scan, allowed scan, owner-scoped access, request-id propagation).

**Why**
- Deployment and e2e must validate auth-only behavior, not only happy paths.

**Changes**
- Update `scripts/e2e-dev-stack.sh` with CPM auth checks:
  - call without token -> expect `401`;
  - call with invalid token -> expect `401`;
  - call with valid token but forbidden scan -> expect `403`;
  - call with valid token and owned scan -> expect success.
- Optional env var for explicit CPM token if different from Discovery token.
- Keep checks deterministic and CI-friendly.

**Acceptance**
- E2E fails if CPM endpoint is anonymously accessible.
- E2E passes with valid authenticated flow.

---

## AUTH-07 - `cafe-frontend`

**Title:** Harden CPM auth UX for 401/403 states

**Status:** Merged in **`cafe-frontend`** as **[PR #46](https://github.com/create2-labs/cafe-frontend/pull/46)** (distinct from AUTH-05's **[PR #46](https://github.com/create2-labs/cafe-discovery/pull/46)** in **`cafe-discovery`** — different repository). Ships AUTH-04-style mapping (`code`, `message`, `details`, `request_id`), typed `CpmDataSourceError` kinds, UX banner + composable summaries, `cpmAuthUx` on CPM Axios calls so `401` clears session via `clearAuth()` but skips forced `/signin` redirect on Crypto Policy Management, plus interceptor regression coverage and robust `503 → unavailable` handling.

**Why**
- Frontend already sends bearer tokens but should explicitly handle auth failures from CPM.

**Changes**
- In CPM data-source (`apiCpmDataSource`), classify HTTP/backend codes (`unauthenticated`, `forbidden`, `bad_request`, `unavailable`, `unknown`), capture `request_id`, optional `serverMessage` for diagnostics.
- Composables (`useCpmPolicySelection`, persistence/validation/wallet gate) consume `summarizeCpmDataSourceThrown` / `formatCpmUxSummaryParagraph`; components do not parse raw Axios errors.
- `401`/`unauthenticated`: session-expired copy and sign-in affordance; `403`/`forbidden`: access-denied copy + reference ID (no logout); `503` authz unavailable: retry; shared Discovery Bearer token unchanged.
- README AUTH-07 subsection + Vitest coverage (data-source mapping, boundary summaries, smoke page).

**Acceptance**
- Expired session path is understandable and recoverable.
- Forbidden scan path is explicit and non-ambiguous.
- `npm run typecheck`, `npm test`, `npm run lint` (expectations per repo), `npm run build` pass on the integrating branch.

---

## AUTH-08 - `cafe-documentation`

**Title:** Publish CPM auth-only contract and runbook

**Why**
- Teams need a single source of truth after rollout.

**Changes**
- Update developer guide + platform docs with:
  - auth requirements for `/api/cpm/*`;
  - 401/403 semantics;
  - principal/ownership model;
  - troubleshooting runbook.
- Cross-link frontend, CPM, Discovery, and deploy changes.

**Acceptance**
- New engineer can implement/debug CPM auth flow from docs only.

## Risks and mitigations

- **Risk:** Break existing anonymous dev flows.
  - **Mitigation:** allow bypass only in `dev/test` with explicit guard (`CPM_AUTH_REQUIRED=true` by default; bypass impossible in staging/prod).
- **Risk:** Inconsistent auth checks between Discovery and CPM.
  - **Mitigation:** explicit shared contract and integration tests around scan authorization (`scan_id` in CPM JSON bodies).
- **Risk:** Ambiguous frontend behavior on 401/403.
  - **Mitigation:** standardized error payload and dedicated UX handling in AUTH-07.

## Done criteria (program-level)

- Anonymous calls to CPM business APIs are impossible.
- No client-provided ownership is accepted for drafts/policies.
- Scan-bound CPM operations enforce principal authorization.
- No scan-bound operation executes without Discovery-backed authorization.
- No unclassified CPM route remains exposed.
- Draft/policy persistence is owner-scoped.
- E2E stack tests fail on anonymous access and pass on authenticated flow.
- Docs reflect final contract and operations guidance.
