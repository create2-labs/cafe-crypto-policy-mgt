# Crypto Policy persistence workflow for wallets

1. [Crypto Policy persistence workflow for wallets](#crypto-policy-persistence-workflow-for-wallets)
   1. [Document versions](#document-versions)
   2. [Purpose](#purpose)
   3. [Scope](#scope)
   4. [Out of scope](#out-of-scope)
   5. [Expected implementation gaps (after PR4)](#expected-implementation-gaps-after-pr4)
2. [Part I — Functional specifications](#part-i--functional-specifications)
   1. [Definitions](#definitions)
      1. [Discovery scan](#discovery-scan)
      2. [CP explore decision](#cp-explore-decision)
      3. [Crypto Policy / CP](#crypto-policy--cp)
      4. [CP draft](#cp-draft)
      5. [Persisted CP](#persisted-cp)
      6. [Wallet control proof](#wallet-control-proof)
   2. [Core product rule](#core-product-rule)
   3. [Interface-independent wallet authorization requirement](#interface-independent-wallet-authorization-requirement)
   4. [Functional workflow — EOA wallet](#functional-workflow--eoa-wallet)
      1. [Step 1 - Discovery scan](#step-1---discovery-scan)
      2. [Step 2 — CP exploration](#step-2--cp-exploration)
      3. [Step 3 — CP draft creation](#step-3--cp-draft-creation)
      4. [Step 4 — Canonical message preparation](#step-4--canonical-message-preparation)
      5. [Step 5 — Wallet signature](#step-5--wallet-signature)
      6. [Step 6 — CP persistence with signed authorization](#step-6--cp-persistence-with-signed-authorization)
   5. [Functional workflow summary](#functional-workflow-summary)
   6. [Required UX behavior](#required-ux-behavior)
      1. [Web UI](#web-ui)
      2. [CLI](#cli)
      3. [Direct API](#direct-api)
   7. [Error semantics](#error-semantics)
      1. [Stateless message helper errors (`POST /api/cpm/v1/wallet-challenges`)](#stateless-message-helper-errors-post-apicpmv1wallet-challenges)
      2. [Persistence errors (`POST /api/cpm/v1/drafts/{draft_id}/persist`)](#persistence-errors-post-apicpmv1draftsdraft_idpersist)
3. [Part II — Technical specifications](#part-ii--technical-specifications)
   1. [10. Backend enforcement](#10-backend-enforcement)
   2. [11. Proposed CPM API contract](#11-proposed-cpm-api-contract)
      1. [11.1 Canonical message helper (mandatory before sign)](#111-canonical-message-helper-mandatory-before-sign)
      2. [11.2 Not in V1: `POST /api/cpm/v1/wallet-challenges/verify`](#112-not-in-v1-post-apicpmv1wallet-challengesverify)
      3. [11.3 Persist draft (normative)](#113-persist-draft-normative)
   3. [12. Challenge message format](#12-challenge-message-format)
   4. [13. Stateless authorization model](#13-stateless-authorization-model)
      1. [13.0 V1 authorization flow](#130-v1-authorization-flow)
      2. [13.1 Replay control (V1)](#131-replay-control-v1)
      3. [13.2 Durable persisted policy metadata](#132-durable-persisted-policy-metadata)
      4. [13.3 V2 optional hardening (not V1)](#133-v2-optional-hardening-not-v1)
   5. [14. Address normalization](#14-address-normalization)
   6. [15. Expiration and replay protection](#15-expiration-and-replay-protection)
   7. [16. Security requirements](#16-security-requirements)
   8. [17. Audit requirements](#17-audit-requirements)
   9. [18. OpenAPI requirements](#18-openapi-requirements)
   10. [19. Testing requirements](#19-testing-requirements)
       1. [19.1 Unit tests](#191-unit-tests)
       2. [19.2 API tests](#192-api-tests)
       3. [19.3 Non-regression tests](#193-non-regression-tests)
4. [Part III — Stories and tasks](#part-iii--stories-and-tasks)
   1. [Epic](#epic)
   2. [Baseline / non-regression stories](#baseline--non-regression-stories)
   3. [User stories](#user-stories)
   4. [Implementation tasks](#implementation-tasks)
      1. [CP-PERSIST-T1 — Specification (PR1)](#cp-persist-t1--specification-pr1)
      2. [CP-PERSIST-T2 — OpenAPI contract (PR2)](#cp-persist-t2--openapi-contract-pr2)
      3. [CP-PERSIST-T3 — Canonical message and EIP-191 verifier (PR3)](#cp-persist-t3--canonical-message-and-eip-191-verifier-pr3)
      4. [CP-PERSIST-T4 — Persist enforcement (PR4)](#cp-persist-t4--persist-enforcement-pr4)
      5. [CP-PERSIST-T5 — Web UI integration (PR5)](#cp-persist-t5--web-ui-integration-pr5)
      6. [CP-PERSIST-T6 — CLI integration (PR6)](#cp-persist-t6--cli-integration-pr6)
      7. [CP-PERSIST-T7 — Documentation and E2E validation (PR7)](#cp-persist-t7--documentation-and-e2e-validation-pr7)
   5. [Tracking table](#tracking-table)
5. [Part IV — PR breakdown](#part-iv--pr-breakdown)
   1. [20. PR1 — Contract-first CP persistence specification](#20-pr1--contract-first-cp-persistence-specification)
   2. [21. PR2 — OpenAPI contract for stateless CP-PERSIST](#21-pr2--openapi-contract-for-stateless-cp-persist)
   3. [22. PR3 — Canonical message builder and EIP-191 verifier](#22-pr3--canonical-message-builder-and-eip-191-verifier)
   4. [23. PR4 — Enforce wallet signed authorization on CP persistence](#23-pr4--enforce-wallet-signed-authorization-on-cp-persistence)
   5. [24. PR5 — Web UI integration](#24-pr5--web-ui-integration)
   6. [25. PR6 — CLI integration](#25-pr6--cli-integration)
   7. [26. PR7 — Documentation and end-to-end validation](#26-pr7--documentation-and-end-to-end-validation)
6. [Part V — TODO list for future wallet types](#part-v--todo-list-for-future-wallet-types)
   1. [28. Smart contract wallets](#28-smart-contract-wallets)
   2. [29. Safe / multisig wallets](#29-safe--multisig-wallets)
   3. [30. Institutional delegated wallets](#30-institutional-delegated-wallets)
   4. [31. Contract admin / proxy admin ownership](#31-contract-admin--proxy-admin-ownership)
   5. [32. Hardware and custody providers](#32-hardware-and-custody-providers)
7. [Part VI — Frozen decisions](#part-vi--frozen-decisions)
   1. [33. Persistence endpoint](#33-persistence-endpoint)
   2. [34. Legacy `POST /api/cpm/v1/policies`](#34-legacy-post-apicpmv1policies)
   3. [35. Message helper API path](#35-message-helper-api-path)
   4. [36. Challenge message and signature format](#36-challenge-message-and-signature-format)
   5. [37. Replay policy (stateless V1)](#37-replay-policy-stateless-v1)
   6. [38. No V1 server-side challenge/proof store](#38-no-v1-server-side-challengeproof-store)
   7. [39. Signed message validity (TTL)](#39-signed-message-validity-ttl)
   8. [40. Persist ordering (transactional persist-once)](#40-persist-ordering-transactional-persist-once)
   9. [41. V2 optional hardening (not V1)](#41-v2-optional-hardening-not-v1)
   10. [42. Long-term architecture](#42-long-term-architecture)
8. [Part VII — Non-goals](#part-vii--non-goals)
9. [Part VIII — Summary](#part-viii--summary)

---

## Document versions


| Date           | Author        | Version | Comments                                                                                                                          |
| -------------- | ------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Jun 10th, 2026 | O. Lodygensky | 0.1     | First version                                                                                                                     |
| Jun 10th, 2026 | ChatGPT       | 0.2     | Clarify off-chain signature, ephemeral proof storage and PR split                                                                 |
| Jun 10th, 2026 | ChatGPT       | 0.3     | Harden stories, tasks and backend enforcement guardrails                                                                          |
| Jun 10th, 2026 | O. Lodygensky | 0.4     | Freeze API contract, TTL, Redis fail-closed and persist consume ordering                                                          |
| Jun 10th, 2026 | O. Lodygensky | 0.5     | Align PR3–PR8 breakdown and tasks with frozen decisions                                                                           |
| Jun 10th, 2026 | O. Lodygensky | 0.6     | Document CPM-owned ephemeral store, CPM_REDIS_URL and store abstractions                                                          |
| Jun 10th, 2026 | O. Lodygensky | 0.7     | Editorial cleanup; align WORKPLAN/README; clarify PR8 vs PR1 cross-links                                                          |
| Jun 10th, 2026 | O. Lodygensky | 0.8     | Document expected implementation gaps after PR1 and independent V1 sign-off                                                       |
| Jun 10th, 2026 | O. Lodygensky | 0.9     | Adopt stateless signature-at-persist model for CP-PERSIST V1; move Redis/proof store to V2                                        |
| Jun 10th, 2026 | O. Lodygensky | 0.9.1   | TOC Part VI range; wallet signed authorization wording; issued_at clock skew rule                                                 |
| Jun 10th, 2026 | O. Lodygensky | 0.9.2   | Mandatory CPM-issued canonical message via POST /wallet-challenges before sign                                                    |
| Jun 10th, 2026 | O. Lodygensky | 0.9.3   | Clarify signed-message vs server-side binding model (PR3/PR4)                                                                     |
| Jun 10th, 2026 | O. Lodygensky | 0.9.4   | Mark PR1 merged; unblock PR2 tracking                                                                                             |
| Jun 12th, 2026 | O. Lodygensky | 0.9.5   | Mark PR2 OpenAPI complete; link `[cafe-crypto-policy-mgt` PR #51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51) |
| Jun 12th, 2026 | O. Lodygensky | 0.9.6   | Mark PR3 canonical message + EIP-191 verifier implemented (CP-PERSIST-T3)                                                         |
| Jun 12th, 2026 | O. Lodygensky | 0.9.7   | Mark PR4 EOA persist enforcement implemented (CP-PERSIST-T4); update gaps and smoke script references                             |


---

## Purpose

This document defines the functional and technical workflow required to persist a Crypto Policy, abbreviated **CP**, for a wallet in CAFE.

The core rule is:

> A wallet can be discovered and analyzed without proving wallet ownership.
> A Crypto Policy can only be persisted for a wallet after proving control of that wallet.

This document focuses only on **EOA wallets** for the first implementation.

Other wallet types, such as smart contract wallets, Safe multisig wallets and institutional delegated wallets, are explicitly out of scope for the first implementation and are listed in the TODO section.

---

## Scope

This document covers:

- Wallet-only Crypto Policy persistence.
- EOA wallets.
- Wallet control proof through a stateless signed authorization message (EIP-191 verified at persist time).
- Wallet signed authorization enforcement for all interfaces:
  - Web UI.
  - CLI.
  - Direct API usage.
- Transition from non-actionable draft to persisted CP.
- Backend-side enforcement in CPM.
- Stateless signature-at-persist model for V1 (mandatory CPM-issued canonical message via `POST /wallet-challenges`; verified at persist time).
- Functional and technical API expectations.
- Suggested implementation split by PR.

## Out of scope

This document does not cover:

- TLS endpoints as CP persistence targets.
- Smart contract wallets.
- Safe or multisig wallets.
- Institutional delegation workflows.
- On-chain deployment of policy logic.
- Automatic remediation.
- Policy execution.
- Cross-chain ownership aggregation beyond the selected `chain_id`.

Important CAFE rule:

> CPM assessment and persistence flows are wallet-only.
> TLS endpoints may remain visible in Discovery, history, CBOM or risk inventory, but TLS endpoints are not CPM persistence targets.

---

## Expected implementation gaps (after PR4)

PR1 is **docs-only** and freezes the CP-PERSIST V1 contract (stateless signature-at-persist). **PR2** ([`cafe-crypto-policy-mgt` PR #51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51)) documents the normative OpenAPI contract in [`openapi/cpm-v1.yaml`](../openapi/cpm-v1.yaml). **PR3** ([PR #52](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/52)) implements `internal/walletauth`, `POST /api/cpm/v1/wallet-challenges`, and EIP-191 verification primitives. **PR4** (CP-PERSIST-T4) implements `POST /api/cpm/v1/drafts/{draft_id}/persist`, transactional persist-once semantics, EOA blocking on legacy `POST /api/cpm/v1/policies`, and ownership metadata without raw signatures. **PR5** (CP-PERSIST-T5, [`cafe-frontend` PR #84](https://github.com/create2-labs/cafe-frontend/pull/84)) wires the Web UI to the same frozen V1 flow in **`VITE_CPM_DATA_SOURCE=api`** mode.

The following gaps are **expected** after PR5 Web UI delivery. They are **not** documentation defects; each maps to a later PR in Part IV.

1. **CLI still uses legacy persist scripts.** **PR6** will add `cafe cpm wallet-challenge create` and `cafe cpm draft persist`, and upgrade `cafe-deploy/scripts/test-discovery-v1-wallet-scans-to-cpm.sh` to the compliant sign + persist path. The **Web UI** (PR5) already uses `wallet-challenges` → `personal_sign` → `drafts/{draft_id}/persist` for EOA in API mode. Session auth and wallet signature remain **orthogonal**: session auth identifies the user; wallet signature proves control of the EOA for the persist action.
2. **End-to-end product documentation consolidation remains open.** **PR7** will finalize cross-repo README alignment, troubleshooting, and manual E2E scenarios. Backend smoke: `cafe-deploy/scripts/test-cpm-cp-persist-t4-draft-persist.sh`. Web UI + contract smoke: `cafe-deploy/scripts/test-cpm-cp-persist-t5-web-ui-flow.sh` (optional vitest slice via `CAFE_FRONTEND_DIR`).
3. **CP-PERSIST V1 is signed off independently through this document** (Part VI frozen decisions). [`workplans/WORKPLAN_API.md`](../workplans/WORKPLAN_API.md) remains a broader API workplan and may keep its global proposal status.
4. **Smart Account CP persist is out of scope for CP-PERSIST V1 (backend and frontend).** V1 normative persist is **EOA-only** (`wallet-challenges` → EIP-191 sign → `drafts/{draft_id}/persist`). **Backend enforcement:** when the platform draft payload carries `wallet_type` other than `eoa`, **`POST /api/cpm/v1/wallet-challenges`** and **`POST /api/cpm/v1/drafts/{draft_id}/persist`** return **422** `UNSUPPORTED_WALLET_TYPE` (*only EOA wallets are supported for CP persistence in V1* — see handlers in `internal/app/wallet_challenge_routes.go`, `internal/app/draft_persist_routes.go`). Legacy **`POST /api/cpm/v1/policies`** with Discovery-bound payloads (`selected_wallet_policy_context`) is blocked without signed authorization (**403** `WALLET_CONTROL_PROOF_REQUIRED`) for product flows; there is **no V1 Smart Account wallet-control proof** (EIP-1271, Safe, multisig, etc. — **Part V** TODO). **Explore** (`POST …/policies/decisions/explore`) and **platform draft save** (`POST …/drafts`) still accept `smart_account` scan context; only **actionable CP persist** is unavailable. **Frontend:** the SPA in **`VITE_CPM_DATA_SOURCE=api`** has no wired non-EOA persist client — see [`cafe-frontend` / `docs/cpm-developer.md`](../../cafe-frontend/docs/cpm-developer.md) (*Known limitation: Smart Account*).

**Not a V1 gap:** Redis, `CPM_REDIS_URL`, `ChallengeStore` / `ProofStore` or `wallet_control_proof_id` — these are **V2 optional** hardening, not required for CP-PERSIST V1.

---

# Part I — Functional specifications

## Definitions

### Discovery scan

A **Discovery scan** observes public information about a wallet and produces a scan result.

A scan may include:

- `scan_id`
- `wallet_address`
- `chain_ids`
- `wallet_type`
- `current_pq_posture`
- risk indicators
- historical information
- policy context input for CPM

Discovery does not require proof of wallet ownership.

### CP explore decision

A CP explore decision queries CPM policy data and may propose candidate policies or explain rejected policies.
It must not create an official policy for the wallet.

### Crypto Policy / CP

A **Crypto Policy** describes the selected cryptographic posture, migration path or remediation plan associated with a scan result.

Examples:

- Classical-only wallet posture.
- PQ-ready migration recommendation.
- Hybrid migration path.
- Required algorithm transition.
- Wallet migration target posture.

### CP draft

A **CP draft** is a saved, non-actionable preparation of a Crypto Policy.

A draft can be created before proving wallet control.

A draft is not actionable.

### Persisted CP

A **persisted CP** is an official Crypto Policy associated with a wallet in CAFE. A persisted CP is created from a CP draft.

A CP can only become persisted after successful wallet control proof.

### Wallet control proof

A **wallet control proof** is a backend-verified proof that the authenticated user controls the wallet or is authorized to act for it.

For the first implementation, wallet control proof is limited to **EOA signature verification**.

---

## Core product rule

The workflow must enforce the following separation:


| What                   | How                                                                             |
| ---------------------- | ------------------------------------------------------------------------------- |
| Discovery scan         | allowed without wallet control proof                                            |
| CP explore             | allowed without wallet control proof; non-persistent; non-actionable            |
| CP draft               | allowed without wallet control proof; saved as unverified draft; non-actionable |
| CP persist             | requires wallet control proof; creates an official wallet CP                    |
| CP update              | requires wallet control proof or a still-valid authorization                    |
| CP delete / deactivate | require wallet control proof since no prior valid delegation model exists       |


Update and delete semantics may be refined in later PRs, but the model must not introduce a loophole where a user can persist or modify a CP for an EOA wallet without proving wallet control.

---

## Interface-independent wallet authorization requirement

Wallet signed authorization is mandatory for EOA CP persistence regardless of the interface used.

The following interfaces must all go through the same backend-enforced persist contract:

- Web UI.
- CLI.
- Direct API usage.
- Future automation tools.
- Future admin or integration interfaces, unless explicitly exempted by a documented internal service contract.

The Web UI may provide a MetaMask-based signing experience.

The CLI may use a local wallet, hardware wallet, private key signer, external signer or wallet provider.

UI and CLI **must** call `POST /api/cpm/v1/wallet-challenges` to obtain the canonical message before signing, then **must** submit `signed_message` + `signature` to `POST /api/cpm/v1/drafts/{draft_id}/persist`.

The client **must** obtain the canonical authorization message from CPM before signing by calling:

```http
POST /api/cpm/v1/wallet-challenges
```

This helper is **stateless**: it validates draft / scan / wallet bindings and returns the canonical message to sign, but it stores nothing server-side.

At persist time, CPM does not rely on stored challenge state. CPM verifies that the submitted `signed_message` exactly matches the canonical message expected for the wallet, chain, scan, draft, action and validity window (see §12 binding model), then verifies the EIP-191 / `personal_sign` signature. User and tenant scope are enforced separately via session/JWT and draft/scan ownership.

Advanced clients must not invent an alternative message format. They may only sign the canonical message returned by CPM.

The frontend must never be considered the source of trust.

The backend must reject EOA CP persistence when no valid signed wallet authorization is verified at persist time.

---

## Functional workflow — EOA wallet

### Step 1 - Discovery scan

The user scans a wallet. No ownership proof is required.

This is already implemented.

Rationale:

- Wallet data is public on-chain.
- Auditors, analysts and external observers must be able to inspect a wallet.
- Discovery is read-only.

---

### Step 2 — CP exploration

The user asks CPM to explore policy decisions from the Discovery context. No ownership proof is required.

This is already implemented.

---

### Step 3 — CP draft creation

The user saves a CP draft. No ownership proof is required.

This is already implemented.

Rationale:

- A consultant or analyst may prepare recommendations before involving the wallet owner.
- A product manager or security team may prepare scenarios.
- The proof is only required when the user attempts to persist the CP.

---

### Step 4 — Canonical message preparation

Before persisting a CP for an EOA wallet, the client must obtain the **canonical authorization message** from CPM for the user to sign.

This is implemented in **PR3** (`POST /api/cpm/v1/wallet-challenges`).

The client **must** obtain the canonical authorization message from CPM before signing by calling:

```http
POST /api/cpm/v1/wallet-challenges
```

This helper is **stateless**: it validates draft / scan / wallet bindings and returns the canonical message to sign, but it stores nothing server-side.

At persist time, CPM does not rely on stored challenge state. CPM verifies that the submitted `signed_message` exactly matches the canonical message expected for the wallet, chain, scan, draft, action and validity window (see §12 binding model), then verifies the EIP-191 / `personal_sign` signature. User and tenant scope are enforced separately via session/JWT and draft/scan ownership.

Advanced clients must not invent an alternative message format. They may only sign the canonical message returned by CPM.

---

### Step 5 — Wallet signature

The user signs the canonical message locally with the EOA wallet using **EIP-191 / `personal_sign`**.

This is a **client-side** step (Web UI, CLI, MetaMask, hardware wallet, or external signer). CPM verifies the signature at persist time in **PR4**; it is not implemented inside the CPM service itself.

The signature is off-chain:

- no blockchain transaction is sent
- no gas is paid
- no on-chain state is modified
- the private key never leaves the wallet

For the Web UI, this is typically done through MetaMask or another injected wallet provider.

For the CLI, this may be done through a local EOA signer, hardware wallet, external wallet provider or another signing adapter.

The private key must never be sent to the backend.

---

### Step 6 — CP persistence with signed authorization

The client calls the normative persist endpoint with the signed authorization.

This is implemented in **PR4** (`POST /api/cpm/v1/drafts/{draft_id}/persist`).

```http
POST /api/cpm/v1/drafts/{draft_id}/persist
```

```json
{
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "signed_message": "string",
  "signature": "0x..."
}
```

A CLI-like persistence flow still exists in `cafe-deploy/scripts/test-discovery-v1-wallet-scans-to-cpm.sh` (legacy `POST /api/cpm/v1/policies`); it is **blocked for EOA wallet payloads** after **PR4** (`WALLET_CONTROL_PROOF_REQUIRED`). The compliant backend smoke is `cafe-deploy/scripts/test-cpm-cp-persist-t4-draft-persist.sh` (wallet-challenges → sign → normative persist). Full CLI/product migration remains **PR6** / **PR7**.

The backend verifies at persist time (clients are **not** trusted):

- authenticated **user** and **tenant** via existing session/JWT (server-side — not fields in the signed message)
- draft exists and belongs to the authenticated user/tenant scope (server-side ownership)
- draft is not already persisted
- draft is linked to `scan_id`
- scan is linked to `wallet_address`
- wallet type is `eoa`
- `signed_message` **exactly matches** the canonical message CPM expects for **wallet**, **chain**, **scan**, **draft**, `**action = persist_crypto_policy`** and **validity window** (`issued_at`, `expires_at`) — see §12 binding model
- `expires_at` is not in the past
- `issued_at` is not in the future beyond allowed clock skew (**30 seconds** recommended, aligned with CPM session clock skew)
- validity window (`expires_at` − `issued_at`) is not greater than **10 minutes**
- signature is EIP-191 / `personal_sign` compatible
- recovered signer address equals normalized `wallet_address`
- request fields match draft / scan / wallet bindings

If all checks pass, CPM creates the persisted policy from the draft in a **transactional persist-once** operation.

If CP creation fails before the draft is marked persisted, the user may **retry with the same signature** while it is still valid (V1 acceptable behavior).

The raw signature must not be stored in the durable database. The persisted CP may store minimal audit metadata only (`ownership_status`, `wallet_control_method`, `wallet_control_verified_at`).

`POST /api/cpm/v1/policies` must **not** persist an EOA CP without this signed authorization flow.

---

## Functional workflow summary

```text
1. User scans wallet
   -> no proof required

2. User explores CP decisions
   -> no proof required
   -> non-persistent

3. User saves CP draft
   -> no proof required
   -> draft is unverified and non-actionable

4. Client obtains canonical message from CPM
   -> POST /api/cpm/v1/wallet-challenges (mandatory; stateless helper, stores nothing)

5. User signs message locally (EIP-191 / personal_sign)
   -> UI, CLI, MetaMask, hardware wallet or other signer

6. Client persists draft
   -> POST /api/cpm/v1/drafts/{draft_id}/persist with signed_message + signature

7. Backend verifies session, bindings, message freshness and EIP-191 signature
   -> transactional persist-once; replay after success -> DRAFT_ALREADY_PERSISTED
```

---

## Required UX behavior

### Web UI

The UI must clearly distinguish:

```text
Draft CP
Unverified draft
Ready to sign
Wallet verified
Persisted CP
```

The UI must not display an unverified draft as an active wallet policy.

When the user clicks “Persist policy” or equivalent, the UI must:

1. obtain the canonical message from CPM (`POST /api/cpm/v1/wallet-challenges` — mandatory before sign)
2. ask the wallet provider to sign the message (EIP-191 / `personal_sign`)
3. call `POST /api/cpm/v1/drafts/{draft_id}/persist` with `signed_message` + `signature`

The UI must **not** rely on `POST /api/cpm/v1/wallet-challenges/verify` as a V1 security step (not part of the V1 contract).

If the wallet signature fails, the CP remains a draft.

If CP creation fails before the draft is marked persisted, the UI may retry persist with the **same signature** while it is still valid.

### CLI

The current CLI is `cafe-frontend/scripts/cafe.sh`; it is based on the API.

The current CLI must be enhanced to manage CPM persistence workflows.

The CLI must follow the same backend workflow as the UI and direct API.

The CLI must not provide a `--force` or `--skip-wallet-proof` option for CP persistence.

A possible CLI sequence:

```bash
cafe discovery scan wallet --address 0xabc... --chain-id 1

cafe cpm draft create --scan-id <scan_id> --policy-template <template_id>

cafe cpm wallet-challenge create \
  --wallet 0xabc... \
  --chain-id 1 \
  --scan-id <scan_id> \
  --draft-id <draft_id>

cafe wallet sign --message-file <canonical_message.txt>

cafe cpm draft persist \
  --draft-id <draft_id> \
  --wallet 0xabc... \
  --chain-id 1 \
  --scan-id <scan_id> \
  --signed-message-file <canonical_message.txt> \
  --signature 0x...
```

The signing mechanism may vary, but the backend verification at persist time must remain identical.

### Direct API

Direct API users must follow the same sequence.

Calling the persistence endpoint without a valid `signed_message` and `signature` must return an error.

Expected error:

```json
{
  "error": "WALLET_CONTROL_PROOF_REQUIRED",
  "message": "Persisting a Crypto Policy for a wallet requires a valid signed wallet authorization."
}
```

---

## Error semantics

### Stateless message helper errors (`POST /api/cpm/v1/wallet-challenges`)


| Case                      | HTTP status | Error code                |
| ------------------------- | ----------- | ------------------------- |
| Missing wallet address    | 400         | `WALLET_ADDRESS_REQUIRED` |
| Invalid wallet address    | 400         | `INVALID_WALLET_ADDRESS`  |
| Missing chain id          | 400         | `CHAIN_ID_REQUIRED`       |
| Unknown draft             | 404         | `DRAFT_NOT_FOUND`         |
| Unknown scan              | 404         | `SCAN_NOT_FOUND`          |
| Draft and scan mismatch   | 409         | `DRAFT_SCAN_MISMATCH`     |
| Draft and wallet mismatch | 409         | `DRAFT_WALLET_MISMATCH`   |
| Unsupported wallet type   | 422         | `UNSUPPORTED_WALLET_TYPE` |


For the first implementation, only `wallet_type = eoa` is supported for persistence.

### Persistence errors (`POST /api/cpm/v1/drafts/{draft_id}/persist`)


| Case                              | HTTP status | Error code                                                                  |
| --------------------------------- | ----------- | --------------------------------------------------------------------------- |
| Missing signed_message/signature  | 400         | `WALLET_CONTROL_PROOF_REQUIRED` *(missing `signed_message` or `signature`)* |
| Invalid signature                 | 401         | `INVALID_WALLET_SIGNATURE`                                                  |
| Recovered address mismatch        | 401         | `WALLET_SIGNATURE_ADDRESS_MISMATCH`                                         |
| Expired signed message            | 410         | `WALLET_AUTHORIZATION_EXPIRED`                                              |
| `issued_at` too far in the future | 400         | `WALLET_AUTHORIZATION_NOT_YET_VALID`                                        |
| Message validity window too long  | 400         | `WALLET_AUTHORIZATION_VALIDITY_TOO_LONG`                                    |
| Message field mismatch (draft)    | 409         | `WALLET_AUTHORIZATION_DRAFT_MISMATCH`                                       |
| Message field mismatch (scan)     | 409         | `WALLET_AUTHORIZATION_SCAN_MISMATCH`                                        |
| Message field mismatch (wallet)   | 409         | `WALLET_AUTHORIZATION_WALLET_MISMATCH`                                      |
| Message field mismatch (chain)    | 409         | `WALLET_AUTHORIZATION_CHAIN_MISMATCH`                                       |
| Message field mismatch (action)   | 409         | `WALLET_AUTHORIZATION_ACTION_MISMATCH`                                      |
| Unsupported wallet type           | 422         | `UNSUPPORTED_WALLET_TYPE`                                                   |
| Draft already persisted           | 409         | `DRAFT_ALREADY_PERSISTED`                                                   |
| Draft not found                   | 404         | `DRAFT_NOT_FOUND`                                                           |


**V2 note:** `POST /api/cpm/v1/wallet-challenges/verify` and proof-handle errors (`WALLET_CONTROL_PROOF_`*) are **not** part of CP-PERSIST V1.

---

# Part II — Technical specifications

## 10. Backend enforcement

The wallet signature requirement must be enforced by the CPM backend at persist time.

The frontend is not trusted.

The CLI is not trusted.

Any direct API caller is not trusted.

The persistence handler must reject EOA persist requests without a valid EIP-191 / `personal_sign` signature over the canonical authorization message.

`POST /api/cpm/v1/policies` must **not** allow EOA Crypto Policy persistence without valid signed authorization. Legacy or pre-CP_PERSIST callers of this route must receive `WALLET_CONTROL_PROOF_REQUIRED` (or an equivalent blocking error) for EOA wallet flows.

The only normative EOA persist route is:

```text
POST /api/cpm/v1/drafts/{draft_id}/persist
```

Persist ordering (frozen for stateless V1):

```text
1. Validate session auth, draft ownership, draft-not-already-persisted, scan/wallet bindings and EOA scope.
2. Validate signed_message content, freshness (max 10 minutes) and EIP-191 signature; recover signer address.
3. Create the persisted CP from the draft in a transactional persist-once operation.
4. If step 3 fails before the draft is marked persisted, the client may retry with the same signature while still valid.
```

Required invariant:

```text
No persisted CP for an EOA wallet can be created without a valid signed wallet authorization verified at persist time.
```

Recommended test name:

```text
TestPersistPolicyRequiresWalletSignatureForEOA
```

---

## 11. Proposed CPM API contract

### 11.1 Canonical message helper (mandatory before sign)

```http
POST /api/cpm/v1/wallet-challenges
```

**V1 role:** mandatory stateless helper — clients must obtain the canonical message from CPM before signing. Stores nothing server-side.

The client **must** obtain the canonical authorization message from CPM before signing by calling:

```http
POST /api/cpm/v1/wallet-challenges
```

This helper is **stateless**: it validates draft / scan / wallet bindings and returns the canonical message to sign, but it stores nothing server-side.

At persist time, CPM does not rely on stored challenge state. CPM verifies that the submitted `signed_message` exactly matches the canonical message expected for the wallet, chain, scan, draft, action and validity window (see §12 binding model), then verifies the EIP-191 / `personal_sign` signature. User and tenant scope are enforced separately via session/JWT and draft/scan ownership.

Advanced clients must not invent an alternative message format. They may only sign the canonical message returned by CPM.

Request:

```json
{
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "draft_id": "uuid",
  "action": "persist_crypto_policy"
}
```

Response:

```json
{
  "message": "string",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "draft_id": "uuid",
  "action": "persist_crypto_policy",
  "issued_at": "date-time",
  "expires_at": "date-time"
}
```

### 11.2 Not in V1: `POST /api/cpm/v1/wallet-challenges/verify`

`POST /api/cpm/v1/wallet-challenges/verify` is **not** part of the CP-PERSIST V1 security path. Signature verification happens at persist time. A future **V2** optional endpoint may support pre-validation UX only.

### 11.3 Persist draft (normative)

```http
POST /api/cpm/v1/drafts/{draft_id}/persist
```

This endpoint expresses the domain transition from draft to persisted policy.

Request:

```json
{
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "signed_message": "string",
  "signature": "0x..."
}
```

Response:

```json
{
  "policy_id": "uuid",
  "draft_id": "uuid",
  "scan_id": "uuid",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "status": "persisted",
  "ownership_status": "verified",
  "wallet_control_method": "eoa_signature",
  "persisted_at": "date-time"
}
```

`POST /api/cpm/v1/policies` is **not** the CP persistence endpoint for this workflow. For EOA wallets, it must **not** create a persisted CP without signed authorization and must return `WALLET_CONTROL_PROOF_REQUIRED` (or an equivalent blocking error).

---

## 12. Challenge message format

**Decision (frozen for CP-PERSIST V1):** EIP-191 / `personal_sign`-compatible signed message.

The message must be deterministic, human-readable and canonical.

Normative message structure:

```text
CAFE Crypto Policy Persistence

Domain: <frontend_or_api_domain>
Action: persist_crypto_policy
Wallet: <wallet_address>
Chain ID: <chain_id>
Scan ID: <scan_id>
Draft ID: <draft_id>
Issued At: <issued_at>
Expiration Time: <expires_at>

By signing this message, I prove control of the wallet and authorize CAFE to persist the selected Crypto Policy draft for this wallet.
```

**Binding model (signed message vs server-side):**


| Enforced via signed message        | Enforced server-side only                          |
| ---------------------------------- | -------------------------------------------------- |
| `wallet_address`                   | `user_id` (session/JWT)                            |
| `chain_id`                         | `tenant_id` (session/JWT)                          |
| `scan_id`                          | draft ownership (user/tenant scope)                |
| `draft_id`                         | scan ownership / authorization (user/tenant scope) |
| `action` (`persist_crypto_policy`) | EOA wallet type                                    |
| `issued_at`, `expires_at`          | draft not already persisted                        |


The signed message **does not** include `user_id` or `tenant_id`. CPM binds the authorization to a user/tenant by requiring a valid session and verifying that the target draft and scan belong to that principal before accepting persist.

**V1 stateless rules:**

- The backend does **not** store the message server-side in V1.
- Clients **must** obtain the canonical message from `POST /api/cpm/v1/wallet-challenges` before signing; advanced clients must not invent an alternative message format.
- At persist time, CPM does not rely on stored challenge state. CPM verifies that `signed_message` **exactly matches** the canonical message expected for wallet, chain, scan, draft, action and validity window (§12 binding model), then verifies the EIP-191 signature. User/tenant scope is enforced separately via session/JWT and draft/scan ownership.
- Maximum validity window: **10 minutes** (`expires_at` − `issued_at`).
- `issued_at` must not be in the future beyond allowed clock skew (**30 seconds** recommended).
- `expires_at` must not be in the past at persist time.
- `Challenge ID` / `Nonce` fields are **optional in V1**; freshness is enforced via `issued_at` / `expires_at`.

The backend must verify signatures using EIP-191 `personal_sign` semantics in V1.

Future versions may support SIWE / EIP-4361 more formally.

---

## 13. Stateless authorization model

CP-PERSIST V1 uses a **stateless signed authorization message** submitted with the persist request.

There is **no V1 server-side challenge store** and **no V1 server-side proof store**.

### 13.0 V1 authorization flow

```text
1. Client obtains canonical message from CPM (`POST /api/cpm/v1/wallet-challenges` — mandatory).
2. User signs locally (EIP-191 / personal_sign).
3. Client submits signed_message + signature with POST /drafts/{draft_id}/persist.
4. Backend verifies bindings, freshness, signature and transactional persist-once.
```

**No V1 requirements for:** Redis, `CPM_REDIS_URL`, `ChallengeStore`, `ProofStore`, `wallet_control_proof_id`, backend-stored challenge or proof.

### 13.1 Replay control (V1)

Replay is **controlled**, not claimed impossible:

- short message validity window (max 10 minutes);
- strict binding to `draft_id`, `scan_id`, `wallet_address`, `chain_id`, `action`;
- transactional **draft can be persisted once** — replay after success returns `DRAFT_ALREADY_PERSISTED`;
- binding mismatches return explicit errors;
- parallel duplicate submits: only one persist succeeds.

If CP creation fails before the draft is marked persisted, retry with the **same signature** while valid is acceptable in V1.

### 13.2 Durable persisted policy metadata

Persisted CP records may include:

```text
policy_id
draft_id
scan_id
wallet_address
chain_id
ownership_status
wallet_control_method
wallet_control_verified_at
persisted_at
created_by_user_id
tenant_id
```

Recommended values:

```text
ownership_status = verified
wallet_control_method = eoa_signature
```

The persisted CP must not store the raw signature or reusable proof artifacts.

### 13.3 V2 optional hardening (not V1)

Future optional enhancements may include:

```text
CPM-owned ephemeral store (Redis via CPM_REDIS_URL)
ChallengeStore / ProofStore abstractions
wallet_control_proof_id single-use handles
POST /api/cpm/v1/wallet-challenges/verify as optional pre-validation UX
delegation, admin workflows, strict one-time authorization tokens
```

V2 must not weaken V1 backend enforcement at persist time.

---

## 14. Address normalization

EOA addresses must be normalized before comparison.

For Ethereum-compatible chains:

- accept `0x` hexadecimal format
- validate length
- normalize for storage and comparison
- use case-insensitive comparison unless checksum validation is explicitly enforced
- optionally store checksum representation for display

All comparisons between signed message wallet address, draft wallet address, scan wallet address and recovered signature address must use normalized address comparison.

---

## 15. Expiration and replay protection

Signed authorization message requirements (V1):

```text
max validity window: 10 minutes (expires_at - issued_at)
expires_at must not be in the past at persist time
issued_at must not be in the future beyond 30 seconds clock skew (recommended)
action: persist_crypto_policy
signed-message binding: wallet, chain, scan, draft, action, issued_at, expires_at
server-side binding: user, tenant (session/JWT + draft/scan ownership)
persist-once: transactional draft state
```

Replay protection must prevent (within threat model):

- persisting the same draft twice (DRAFT_ALREADY_PERSISTED)
- using a signature for another draft, wallet, chain or scan (binding mismatch errors)
- using an expired signed message (WALLET_AUTHORIZATION_EXPIRED)
- using a signature from another user’s draft (owner-scoped draft checks)
- bypassing proof via POST /api/cpm/v1/policies for EOA flows

Replay within the validity window before the first successful persist is absorbed by transactional persist-once semantics.

---

## 16. Security requirements

The backend must never receive a private key.

The backend must never ask the user to provide a private key.

The signed message must be explicit and human-readable.

The signed message must not be ambiguous.

The action must be included in the signed message.

The draft id must be included in the signed message.

The scan id must be included in the signed message.

The chain id must be included in the signed message.

The message must include `issued_at` and `expires_at`; `expires_at` must not be in the past at persist time; `issued_at` must not be in the future beyond **30 seconds** clock skew (recommended).

The persistence handler is the **final enforcement point** for wallet control proof in V1.

Clients are not trusted; the backend verifies the signed message and all server-side bindings.

---

## 17. Audit requirements

The system should be able to answer:

```text
Who persisted this CP?
For which wallet?
For which chain?
From which draft?
From which scan?
Using which proof method?
When was wallet control verified?
When was the CP persisted?
```

Suggested durable audit fields:

```text
created_by_user_id
tenant_id
wallet_address
chain_id
scan_id
draft_id
wallet_control_method
wallet_control_verified_at
persisted_at
```

The durable audit trail must not store raw signature bytes or reusable proof handles.

---

## 18. OpenAPI requirements

The OpenAPI contract must document:

- `POST /api/cpm/v1/drafts/{draft_id}/persist` (normative; `signed_message` + `signature`)
- `POST /api/cpm/v1/wallet-challenges` (mandatory stateless canonical message helper)
- request / response schemas
- error schemas
- EOA-only limitation for the first release
- `WALLET_CONTROL_PROOF_REQUIRED` and wallet authorization errors (§ Error semantics)
- `UNSUPPORTED_WALLET_TYPE`

The OpenAPI contract must **not** document `POST /api/cpm/v1/wallet-challenges/verify` as a V1 security requirement.

The OpenAPI description must explicitly state:

```text
Wallet signed authorization is required for EOA CP persistence regardless of whether the caller is the Web UI, CLI or a direct API client.
Signature verification happens at POST /drafts/{draft_id}/persist.
```

---

## 19. Testing requirements

### 19.1 Unit tests

Required tests:

```text
Stateless helper returns canonical message for valid EOA draft bindings
Stateless helper fails for unknown draft / scan / wallet mismatch
Persist draft fails without signed_message and signature
Persist draft fails with invalid signature
Persist draft fails with expired signed message
Persist draft fails when issued_at is too far in the future (> 30s clock skew)
Persist draft fails with validity window > 10 minutes
Persist draft fails with message/draft/scan/wallet/chain/action mismatch
Persist draft fails when recovered address mismatches wallet
Persist draft succeeds with valid signature
Persist draft returns DRAFT_ALREADY_PERSISTED on replay after success
Persist draft allows retry with same signature if CP creation fails before draft marked persisted
POST /api/cpm/v1/policies blocks EOA persist without signed authorization
Persisted CP includes ownership metadata
Persisted CP does not store raw signature
```

### 19.2 API tests

Required API test scenarios:

```text
UI-like flow:
  create draft
  POST /api/cpm/v1/wallet-challenges (mandatory)
  sign message locally
  persist draft with signed_message + signature

CLI-like flow:
  create draft
  POST /api/cpm/v1/wallet-challenges (mandatory)
  sign externally
  persist draft

Direct API negative flow:
  create draft
  call persist without signature
  expect WALLET_CONTROL_PROOF_REQUIRED

Legacy/pre-CP_PERSIST negative flow:
  use an existing persistence path or script without signed authorization
  expect WALLET_CONTROL_PROOF_REQUIRED
```

### 19.3 Non-regression tests

Required non-regression tests:

```text
Discovery scan still works without wallet proof
CP explore still works without wallet proof
Draft creation still works without wallet proof
Only persistence requires wallet signed authorization
TLS targets cannot be persisted as CP targets
Existing CP persistence paths cannot bypass wallet proof for EOA
```

---

# Part III — Stories and tasks

This chapter maps product stories and implementation tasks to the PR sequence in [Part IV — PR breakdown](#part-iv--pr-breakdown).

Status convention: `✅ done` | `🟡 in progress` | `⚪ planned`.

---

## Epic

**CP-PERSIST** — Wallet control proof for EOA Crypto Policy persistence.

Goal:

```text
A wallet can be scanned, explored and drafted without proving ownership.
An EOA Crypto Policy cannot be persisted without a backend-verified wallet control proof.
Signed wallet authorization is mandatory for EOA CP persistence for Web UI, CLI and direct API usage.
```

Out of scope for this epic:

```text
Smart contract wallets, Safe / multisig, institutional delegation, TLS persistence targets.
```

---

## Baseline / non-regression stories

These stories are already implemented and must remain true during CP-PERSIST implementation.


| ID                | Story                                                                                                                           | Priority | PR(s) | Status |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------- | -------- | ----- | ------ |
| **CP-PERSIST-S1** | As an analyst, I want to scan a wallet without proving ownership so I can assess public on-chain posture.                       | Must     | —     | ✅ done |
| **CP-PERSIST-S2** | As a user, I want to explore CP decisions without proof so I can compare candidate policies without creating an official CP.    | Must     | —     | ✅ done |
| **CP-PERSIST-S3** | As a consultant, I want to save a CP draft without proof so I can prepare a recommendation before the wallet owner is involved. | Must     | —     | ✅ done |


---

## User stories


| ID                 | Story                                                                                                                                                                                                                                                                                     | Priority | PR(s)        | Status                                         |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------------ | ---------------------------------------------- |
| **CP-PERSIST-S4**  | As a wallet owner using the Web UI, I want to sign an off-chain authorization with my EOA wallet so I can persist my CP without sending an on-chain transaction.                                                                                                                          | Must     | PR3–PR5      | ✅ done — [PR5 `cafe-frontend` #84](https://github.com/create2-labs/cafe-frontend/pull/84) |
| **CP-PERSIST-S5**  | As a wallet owner using the CLI, I want to sign externally and submit the signed authorization through the same CPM persist API so the CLI cannot bypass backend enforcement.                                                                                                             | Must     | PR3–PR4, PR6 | 🟡 in progress [PR3](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/52); [PR4](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/53) |
| **CP-PERSIST-S6**  | As an API integrator, I want a clear `WALLET_CONTROL_PROOF_REQUIRED` error when I call persist without a valid signed authorization.                                                                                                                                                      | Must     | PR2, PR4     | ✅ done : [PR2](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51); [PR4](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/53) |
| **CP-PERSIST-S7**  | As a security officer, I want wallet signed authorizations to be time-bound and cryptographically bound to wallet, chain, scan, draft and action (via signed message), with user/tenant enforced server-side via session and draft/scan ownership, and replay controlled at persist time. | Must     | PR3–PR4      | ✅ done: [PR3 #52](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/52); [PR4](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/53)|
| **CP-PERSIST-S8**  | As an auditor, I want persisted CPs to record minimal ownership verification metadata without storing reusable credentials or raw signatures.                                                                                                                                             | Should   | PR4          | ✅ done :[PR4](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/53) |
| **CP-PERSIST-S9**  | As a developer, I want contract-first documentation and OpenAPI before implementation so UI, CLI and API clients share one backend contract.                                                                                                                                              | Must     | PR1, PR2     | ✅ done   [PR1](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/50); [PR2](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/51)                                      |
| **CP-PERSIST-S10** | As a product owner, I want end-to-end documentation and validation so scan / explore / draft remain open while persist requires proof.                                                                                                                                                    | Should   | PR7          | ⚪ planned                                      |
| **CP-PERSIST-S11** | As a platform owner, I want CPM backend to be the only enforcement point for CP persistence so that UI, CLI, scripts or direct API calls cannot bypass wallet signed authorization.                                                                                                       | Must     | PR4          | ✅ done :[PR4](https://github/create2-labs/cafe-crypto-policy-mgt/pulls/53) |


---

## Implementation tasks

Tasks are grouped by delivery PR. Each task references the user stories it satisfies.

### CP-PERSIST-T1 — Specification (PR1)

Stories: **CP-PERSIST-S9**

```text
[x] Add CP_PERSIST.md with functional and technical rules
[x] Document EOA-only first scope
[x] Document interface-independent wallet authorization requirement
[x] Document stateless signature-at-persist API contract (§11, §13)
[x] Document mandatory CPM-issued canonical message helper (POST /wallet-challenges)
[x] Document PR sequence in Part IV
[x] Update WORKPLAN_API.md cross-links and CP-PERSIST route notes
[x] Update README cross-links for stateless CP-PERSIST V1
```

**PR1 merged** — specification deliverables complete.

### CP-PERSIST-T2 — OpenAPI contract (PR2)

Stories: **CP-PERSIST-S6**, **CP-PERSIST-S9**

```text
[x] Add OpenAPI for POST /api/cpm/v1/drafts/{draft_id}/persist (signed_message + signature)
[x] Add OpenAPI for POST /api/cpm/v1/wallet-challenges (mandatory stateless canonical message helper)
[x] Document that POST /api/cpm/v1/policies is not the CP persistence endpoint
[x] Ensure legacy/pre-CP_PERSIST persistence paths cannot bypass signed authorization
[x] Add error codes (WALLET_CONTROL_PROOF_REQUIRED, WALLET_AUTHORIZATION_*, binding errors)
[x] Mark persistence as EOA-only for the first release
[x] Do not document /wallet-challenges/verify as V1 security requirement
[x] Document that UI, CLI and direct API share the same contract
```

**PR2** — OpenAPI contract in `[openapi/cpm-v1.yaml](../openapi/cpm-v1.yaml)`; merged via [cafe-crypto-policy-mgt PR #51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51). No runtime handlers in this PR.

### CP-PERSIST-T3 — Canonical message and EIP-191 verifier (PR3)

Stories: **CP-PERSIST-S4**, **CP-PERSIST-S5**, **CP-PERSIST-S7**

```text
[x] Implement canonical message builder per §12
[x] Implement mandatory stateless POST /api/cpm/v1/wallet-challenges (no server storage)
[x] Implement EOA signature verifier and address normalization
[x] Support EIP-191 / personal_sign verification at persist time
[x] Enforce max 10-minute validity window on signed messages
[x] Reject issued_at too far in the future (30s clock skew)
[x] Add deterministic signature test vectors
[x] Add tests for wrong wallet, chain_id, draft_id, scan_id, expired message, future issued_at
[x] Canonical message binds wallet, chain, scan, draft, action, issued_at, expires_at only (not user_id/tenant_id)
[x] Do not store raw signature in durable DB
[x] Add unit tests and API tests
```

**PR3** — merged via [cafe-crypto-policy-mgt PR #52](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/52)

### CP-PERSIST-T4 — Persist enforcement (PR4)

Stories: **CP-PERSIST-S6**, **CP-PERSIST-S7**, **CP-PERSIST-S8**, **CP-PERSIST-S11**

```text
[x] Implement POST /api/cpm/v1/drafts/{draft_id}/persist with signed_message + signature
[x] Block EOA persistence via POST /api/cpm/v1/policies without signed authorization
[x] Reject missing, invalid, expired, not-yet-valid (future issued_at) or mismatched signed authorizations
[x] Enforce server-side user/tenant via session/JWT + draft/scan ownership (not via signed message fields)
[x] Transactional persist-once semantics (DRAFT_ALREADY_PERSISTED on replay)
[x] Allow retry with same signature if CP creation fails before draft marked persisted
[x] Store ownership metadata on persisted CP (no raw signature)
[x] All existing EOA CP persistence paths must require signed authorization
[x] Add regression test for the current CLI-like/pre-CP_PERSIST flow
[x] Add unit tests and API tests
```

**PR4** — handlers in `internal/app/draft_persist_routes.go`, `internal/persistence/owner_scoped_store.go` (`PersistDraftOnce`); smoke `cafe-deploy/scripts/test-cpm-cp-persist-t4-draft-persist.sh`.


Merged via [cafe-crypto-policy-mgt PR #53](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/53)


### CP-PERSIST-T5 — Web UI integration (PR5)

Stories: **CP-PERSIST-S4**

**Status:** ✅ done — [`cafe-frontend` PR #84](https://github.com/create2-labs/cafe-frontend/pull/84); smoke `cafe-deploy/scripts/test-cpm-cp-persist-t5-web-ui-flow.sh`.

```text
[x] Call POST /api/cpm/v1/wallet-challenges for canonical message (mandatory before sign)
[x] Integrate MetaMask or injected wallet provider for EIP-191 / personal_sign
[x] Call POST /api/cpm/v1/drafts/{draft_id}/persist with signed_message + signature
[x] Do not use POST /wallet-challenges/verify as V1 security step
[x] Display draft / unverified / ready to sign / persisted states
[x] Handle signature rejection; retry persist with same signature if still valid
[x] Add frontend tests
```

### CP-PERSIST-T6 — CLI integration (PR6)

Stories: **CP-PERSIST-S5**

```text
[ ] Add cafe cpm wallet-challenge create -> POST /api/cpm/v1/wallet-challenges (mandatory)
[ ] Add cafe cpm draft persist -> POST /api/cpm/v1/drafts/{draft_id}/persist with signature
[ ] Support external EIP-191 / personal_sign signing
[ ] Reject --force or --skip-wallet-proof options
[ ] Upgrade test-discovery-v1-wallet-scans-to-cpm.sh to sign + persist flow
[ ] Document CLI workflow (cpm-developer.md or equivalent)
[ ] Add CLI tests where applicable
```

### CP-PERSIST-T7 — Documentation and E2E validation (PR7)

Stories: **CP-PERSIST-S10**, **CP-PERSIST-S1**, **CP-PERSIST-S2**, **CP-PERSIST-S3**

```text
[ ] Confirm README and WORKPLAN_API remain aligned with the implemented V1 contract
[x] Update cafe-frontend/docs/cpm-developer.md with frozen V1 contract (CP-PERSIST API flow — PR5 [#84](https://github.com/create2-labs/cafe-frontend/pull/84); PR7 completes cross-repo E2E/troubleshooting)
[ ] Document scan vs explore vs draft vs persist (frozen routes §33–§42)
[ ] Document stateless V1, mandatory CPM-issued canonical message, 10-minute signed message validity
[ ] Document replay policy and transactional persist-once semantics
[ ] Add end-to-end test notes and manual test scenario
[ ] Add troubleshooting section
[ ] Confirm Discovery / explore / draft still work without proof (S1–S3)
[ ] Confirm all persistence paths require signed wallet authorization
[ ] Confirm test-discovery-v1-wallet-scans-to-cpm.sh is compliant
```

---

## Tracking table


| Task / Story                                        | PR  | Git PR                                                                | Repository                                         | Depends on | Status    |
| --------------------------------------------------- | --- | --------------------------------------------------------------------- | -------------------------------------------------- | ---------- | --------- |
| **CP-PERSIST-T1** / **S9**                          | PR1 | —                                                                     | `cafe-crypto-policy-mgt`                           | —          | ✅ done    |
| **CP-PERSIST-T2** / **S6**, **S9**                  | PR2 | [#51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51) | `cafe-crypto-policy-mgt`                           | PR1        | ✅ done    |
| **CP-PERSIST-T3** / **S4**, **S5**, **S7**          | PR3 | [#52](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/52)                                                                   | `cafe-crypto-policy-mgt`                           | PR2        | ✅ done    |
| **CP-PERSIST-T4** / **S6**, **S7**, **S8**, **S11** | PR4 | [#53](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/53)                                                                    | `cafe-crypto-policy-mgt`, `cafe-deploy` (smoke)    | PR3        | ✅ done    |
| **CP-PERSIST-T5** / **S4**                          | PR5 | [#84](https://github.com/create2-labs/cafe-frontend/pull/84)                                                                     | `cafe-frontend`, `cafe-deploy` (smoke)             | PR4        | ✅ done    |
| **CP-PERSIST-T6** / **S5**                          | PR6 | —                                                                     | `cafe-frontend` (`cafe.sh`), `cafe-deploy` (smoke) | PR4        | ⚪ planned |
| **CP-PERSIST-T7** / **S10**, **S1–S3**              | PR7 | —                                                                     | multi-repo                                         | PR5, PR6   | ⚪ planned |


Recommended delivery order:

```text
PR1 → PR2 → PR3 → PR4 → (PR5 and PR6 in parallel) → PR7
```

---

# Part IV — PR breakdown

## 20. PR1 — Contract-first CP persistence specification

**Status:** ✅ merged (Jun 10, 2026).

Repository:

```text
cafe-crypto-policy-mgt
```

Goal:

- Add this document as `CP_PERSIST.md`.
- Document functional rules and stateless signature-at-persist V1 model.
- Document EOA-only first scope.
- Document wallet authorization requirement for UI, CLI and direct API.
- Document recommended API contract and PR sequence.

Deliverables:

```text
CP_PERSIST.md (stateless V1)
WORKPLAN_API.md cross-links and CP-PERSIST route / EOA persist clarifications
README cross-links for stateless CP-PERSIST V1
```

No implementation in this PR.

**Expected implementation gaps after this PR:** see [Expected implementation gaps (after PR4)](#expected-implementation-gaps-after-pr4). OpenAPI gap closed in **PR2** ([#51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51)); backend enforcement closed in **PR4** (CP-PERSIST-T4). Remaining gaps are client migration (**PR5**–**PR6**) and E2E docs (**PR7**).

Expected commit title:

```text
docs(cpm): adopt stateless CP-PERSIST V1 signature-at-persist model
```

---

## 21. PR2 — OpenAPI contract for stateless CP-PERSIST

**Status:** ✅ done — merge via `[cafe-crypto-policy-mgt` PR #51](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/51).

OpenAPI contract in [`openapi/cpm-v1.yaml`](../openapi/cpm-v1.yaml). **PR3** handler for `POST /wallet-challenges` and **PR4** handler for `POST /drafts/{draft_id}/persist` are implemented.

Repository:

```text
cafe-crypto-policy-mgt
```

Goal:

- Add OpenAPI for `POST /api/cpm/v1/drafts/{draft_id}/persist` (`signed_message` + `signature`).
- Add OpenAPI for mandatory stateless `POST /api/cpm/v1/wallet-challenges`.
- Document that `POST /api/cpm/v1/policies` is not the CP persistence endpoint.
- Add wallet authorization error codes.
- Mark persistence as EOA-only for the first release.

Deliverables:

```text
OpenAPI schemas
Canonical message helper request / response schema
Persist draft request / response schema (signed_message + signature)
Error schema updates
```

Acceptance criteria:

```text
OpenAPI documents POST /api/cpm/v1/drafts/{draft_id}/persist with signed_message and signature.
OpenAPI documents POST /api/cpm/v1/wallet-challenges as mandatory stateless canonical message helper.
OpenAPI does not require POST /api/cpm/v1/wallet-challenges/verify for V1.
OpenAPI documents that EOA CP persistence requires wallet signed authorization.
OpenAPI documents that POST /api/cpm/v1/policies must not persist EOA CP without proof.
OpenAPI documents 10-minute max signed message validity and EIP-191 / personal_sign.
OpenAPI documents that UI, CLI and direct API share the same contract.
OpenAPI documents that only EOA is supported in this release.
```

Expected commit title:

```text
docs(openapi): add stateless EOA wallet authorization contract for CP persistence
```

---

## 22. PR3 — Canonical message builder and EIP-191 verifier

**Status:** ✅ implemented (CP-PERSIST-T3). Merge target: PR #52 stack.

Repository:

```text
cafe-crypto-policy-mgt
```

Depends on: **PR2**.

Goal:

- Implement canonical message builder per §12.
- Implement mandatory stateless `POST /api/cpm/v1/wallet-challenges` (validates bindings, returns canonical message; stores nothing).
- Implement EIP-191 / `personal_sign` verifier and address normalization.
- Enforce max 10-minute validity window on signed messages.
- Reject `issued_at` too far in the future (30-second clock skew).
- Canonical signed message binds **wallet, chain, scan, draft, action, timestamps** only — not `user_id` / `tenant_id`.

Deliverables:

```text
Canonical message builder
Mandatory stateless POST /api/cpm/v1/wallet-challenges handler
EIP-191 / personal_sign verifier
Address normalization helper
Deterministic signature test vectors
Unit tests and API tests
```

Acceptance criteria:

```text
Stateless helper returns canonical message for valid bindings; stores nothing server-side.
Invalid signature is rejected at persist verification time.
Expired signed message is rejected.
issued_at too far in the future (> 30s clock skew) is rejected.
Validity window > 10 minutes is rejected.
Recovered address mismatch is rejected.
Wrong wallet, chain_id, draft_id and scan_id are rejected.
Canonical message includes wallet, chain, scan, draft, action, issued_at, expires_at — not user_id or tenant_id.
No raw signature is persisted in durable DB.
```

Expected commit title:

```text
feat(cpm): add canonical wallet message builder and EIP-191 verifier
```

---

## 23. PR4 — Enforce wallet signed authorization on CP persistence

**Status:** ✅ done — CP-PERSIST-T4 (handlers + tests + `cafe-deploy/scripts/test-cpm-cp-persist-t4-draft-persist.sh`).

Repository:

```text
cafe-crypto-policy-mgt
```

Depends on: **PR3**.

Goal:

- Implement `POST /api/cpm/v1/drafts/{draft_id}/persist` with `signed_message` + `signature`.
- Block EOA CP persistence via `POST /api/cpm/v1/policies` without signed authorization.
- Transactional persist-once semantics; `DRAFT_ALREADY_PERSISTED` on replay after success.
- Store ownership metadata without raw signature.
- Enforce **user/tenant** via session/JWT and draft/scan ownership; enforce **wallet/chain/scan/draft/action/timestamps** via signed message exact match + EIP-191.

Deliverables:

```text
POST /api/cpm/v1/drafts/{draft_id}/persist implementation
Signature and binding validation at persist time
EOA blocking on legacy POST /api/cpm/v1/policies
Ownership metadata on persisted policy
Unit tests and API tests
```

Acceptance criteria:

```text
EOA CP persistence without signed authorization returns WALLET_CONTROL_PROOF_REQUIRED.
POST /api/cpm/v1/policies cannot persist an EOA CP without signed authorization.
EOA CP persistence with valid signature succeeds via POST /api/cpm/v1/drafts/{draft_id}/persist.
Replay after successful persist returns DRAFT_ALREADY_PERSISTED.
Client may retry with same signature if CP creation fails before draft marked persisted.
Persisted policy includes ownership_status = verified and wallet_control_method = eoa_signature.
Persisted policy does not store raw signature.
Persist rejects when session user/tenant does not own the draft or scan, even if signature is valid.
Signed message mismatch on wallet, chain, scan, draft, action or timestamps is rejected.
```

Expected commit title:

```text
feat(cpm): require wallet signed authorization to persist EOA policies
```

---

## 24. PR5 — Web UI integration

**Status:** ✅ done — CP-PERSIST-T5 ([`cafe-frontend` PR #84](https://github.com/create2-labs/cafe-frontend/pull/84); smoke `cafe-deploy/scripts/test-cpm-cp-persist-t5-web-ui-flow.sh`).

Repository:

```text
cafe-frontend
```

Depends on: **PR4**.

Goal:

- Align frontend with stateless V1 contract.
- Mandatory message helper; sign canonical message locally; persist with `signed_message` + `signature`.
- Use MetaMask or injected wallet provider for EIP-191 / `personal_sign`.

Deliverables:

```text
Canonical message helper client (POST /wallet-challenges)
Wallet signing integration
Persist draft client (POST /api/cpm/v1/drafts/{draft_id}/persist)
UI state labels
Frontend tests
```

Acceptance criteria:

```text
User cannot persist an EOA CP without signing the canonical message.
UI uses frozen V1 paths; not legacy /wallet-challenge/start|verify or POST /policies persist.
UI does not depend on /wallet-challenges/verify for V1 security.
```

Expected commit title:

```text
feat(frontend): add stateless EOA wallet authorization flow for CP persistence
```

---

## 25. PR6 — CLI integration

Repository:

```text
cafe-frontend (cafe.sh), cafe-deploy (smoke)
```

Depends on: **PR4**. May run in parallel with **PR5**.

Goal:

- Extend `cafe.sh` with mandatory message helper and persist with signature.
- Upgrade smoke script to compliant sign + persist flow.

Deliverables:

```text
cafe cpm wallet-challenge create (mandatory)
cafe cpm draft persist with signed_message + signature
no --force or --skip-wallet-proof flags
upgrade test-discovery-v1-wallet-scans-to-cpm.sh
CLI documentation
```

Expected commit title:

```text
feat(cli): support stateless EOA wallet authorization for CP persistence
```

---

## 26. PR7 — Documentation and end-to-end validation

Repositories:

```text
cafe-crypto-policy-mgt, cafe-deploy, cafe-frontend
```

Depends on: **PR5** and **PR6**.

Goal:

- Verify README / WORKPLAN alignment with implemented stateless V1.
- E2E validation; non-regression S1–S3.

Deliverables:

```text
cpm-developer.md update
smoke documentation update
manual E2E scenario: scan -> explore -> draft -> sign -> persist
troubleshooting: WALLET_CONTROL_PROOF_REQUIRED, expired authorization, binding mismatch, DRAFT_ALREADY_PERSISTED
```

Expected commit title:

```text
docs: validate stateless EOA CP persistence workflow end-to-end
```

---

# Part V — TODO list for future wallet types

The first implementation only supports EOA wallets.

The following wallet types must be handled later.

## 28. Smart contract wallets

TODO:

```text
Support EIP-1271 isValidSignature.
Detect wallet_type = smart_account or contract.
Verify signature using contract call.
Bind proof to chain_id because contract signature validity is chain-specific.
Define error semantics when RPC is unavailable.
Define replay and expiration semantics.
```

## 29. Safe / multisig wallets

TODO:

```text
Support Safe ownership model.
Detect Safe wallet.
Verify threshold signatures.
Verify owners and threshold at challenge time.
Define whether one owner is enough for draft work.
Define whether threshold is required for persistence.
Define Safe transaction / message signing flow.
```

## 30. Institutional delegated wallets

TODO:

```text
Support organization-level delegation.
Allow a wallet owner to delegate CP management to an organization.
Define delegation creation and revocation.
Define delegation expiration.
Define audit model.
Define whether delegation requires periodic re-verification.
```

## 31. Contract admin / proxy admin ownership

TODO:

```text
Support wallets or contracts controlled by admin roles.
Detect admin role where applicable.
Define whether admin role is sufficient to persist CP.
Support proxy admin patterns.
Support Ownable / AccessControl patterns where relevant.
```

## 32. Hardware and custody providers

TODO:

```text
Support custody-provider signing.
Support hardware wallet signing flows.
Define provider metadata.
Define audit requirements for institutional custody.
```

---

# Part VI — Frozen decisions

The following decisions are frozen for **CP-PERSIST V1** (stateless signature-at-persist). **CP-PERSIST V1 is signed off independently through this document** — it does not require global acceptance of `[workplans/WORKPLAN_API.md](../workplans/WORKPLAN_API.md)`. Implementation PRs must not reopen frozen decisions without an explicit spec revision.

## 33. Persistence endpoint

```text
POST /api/cpm/v1/drafts/{draft_id}/persist
```

Request body (V1):

```json
{
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "signed_message": "string",
  "signature": "0x..."
}
```

Rationale: expresses draft → persisted policy transition; wallet authorization verified at persist time.

## 34. Legacy `POST /api/cpm/v1/policies`

`POST /api/cpm/v1/policies` is **not** the CP persistence endpoint for this workflow.

For EOA wallet flows, it must **not** create a persisted CP without valid signed wallet authorization and must return `WALLET_CONTROL_PROOF_REQUIRED` (or equivalent).

## 35. Message helper API path

Clients **must** obtain the canonical authorization message from CPM before signing:

```text
POST /api/cpm/v1/wallet-challenges   # mandatory stateless helper — CPM-issued canonical message; stores nothing
```

The client **must** obtain the canonical authorization message from CPM before signing by calling:

```http
POST /api/cpm/v1/wallet-challenges
```

This helper is **stateless**: it validates draft / scan / wallet bindings and returns the canonical message to sign, but it stores nothing server-side.

At persist time, CPM does not rely on stored challenge state. CPM verifies that the submitted `signed_message` exactly matches the canonical message expected for the wallet, chain, scan, draft, action and validity window (see §12 binding model), then verifies the EIP-191 / `personal_sign` signature. User and tenant scope are enforced separately via session/JWT and draft/scan ownership.

Advanced clients must not invent an alternative message format. They may only sign the canonical message returned by CPM.

`POST /api/cpm/v1/wallet-challenges/verify` is **not** part of CP-PERSIST V1 security path (V2 optional UX only).

Legacy frontend paths `/wallet-challenge/start` and `/wallet-challenge/verify` are not part of the V1 contract.

## 36. Challenge message and signature format

V1 uses **EIP-191 / `personal_sign`** on a deterministic human-readable canonical message (see §12).

**Binding split (frozen):** the signed message binds **wallet, chain, scan, draft, action, issued_at, expires_at**. **User** and **tenant** are **not** in the signed message; CPM enforces them server-side via session/JWT and draft/scan ownership at persist time.

SIWE / EIP-4361 remains out of scope for V1.

## 37. Replay policy (stateless V1)

No Redis single-use proof in V1.

Replay is **controlled** by:

- max **10-minute** signed message validity window;
- strict binding to `draft_id`, `scan_id`, `wallet_address`, `chain_id`, `action`;
- transactional **draft persist once** (`DRAFT_ALREADY_PERSISTED` after success);
- binding mismatch errors for cross-draft / cross-wallet / cross-scan attempts.

Retry with the **same signature** after CP creation failure (before draft marked persisted) is acceptable in V1.

## 38. No V1 server-side challenge/proof store

```text
CP-PERSIST V1 does not require Redis, CPM_REDIS_URL, ChallengeStore, ProofStore,
wallet_control_proof_id, or backend-stored challenge/proof artifacts.
```

Signature verification happens at `POST /drafts/{draft_id}/persist`.

No durable database tables for challenges or proofs. No durable raw signature storage.

## 39. Signed message validity (TTL)

```text
Maximum validity window: 10 minutes (expires_at - issued_at)
expires_at must not be in the past at persist time
issued_at must not be in the future beyond 30 seconds clock skew (recommended)
```

## 40. Persist ordering (transactional persist-once)

```text
1. Validate session auth, draft ownership, bindings and EOA scope.
2. Validate signed_message content, freshness and EIP-191 signature.
3. Create persisted CP from draft in a transactional persist-once operation.
4. If step 3 fails before draft is marked persisted, client may retry with same signature while valid.
```

## 41. V2 optional hardening (not V1)

Future optional enhancements (see §13.3):

```text
CPM_REDIS_URL, ChallengeStore, ProofStore, wallet_control_proof_id
POST /api/cpm/v1/wallet-challenges/verify as optional pre-validation UX
```

V2 must not weaken persist-time verification.

## 42. Long-term architecture

```text
CAFE may later extract persistence and auth management into dedicated services.
Discovery is expected to become less stateful.
CP-PERSIST V1 remains stateless at persist time; V2 store is optional hardening only.
```

---

# Part VII — Non-goals

This workflow does not mean that CAFE takes custody of the wallet.

This workflow does not require the user to share a private key.

This workflow does not deploy anything on-chain by itself.

This workflow does not prove legal ownership of the wallet.

This workflow only proves technical control of the wallet for the purpose of persisting a Crypto Policy in CAFE.

---

# Part VIII — Summary

The required invariant for the first release is:

```text
An EOA wallet can be scanned, analyzed and drafted without wallet proof.
An EOA Crypto Policy cannot be persisted without wallet signed authorization verified at persist time.
The signed authorization is mandatory for UI, CLI and direct API usage.
The backend CPM enforces the rule at POST /api/cpm/v1/drafts/{draft_id}/persist.
```

First implementation target (stateless V1):

```text
EOA only.
POST /api/cpm/v1/wallet-challenges (mandatory stateless canonical message helper).
POST /api/cpm/v1/drafts/{draft_id}/persist with signed_message + signature.
EIP-191 / personal_sign on canonical human-readable message.
Max 10-minute signed message validity; replay controlled via bindings + persist-once.
No V1 Redis, ProofStore, ChallengeStore or wallet_control_proof_id.
POST /api/cpm/v1/policies must not persist EOA CP without signed authorization.
Session auth (JWT) and wallet signature are orthogonal.
Other wallet types remain TODO; V2 optional store-based hardening in §13.3 / §41.
```

