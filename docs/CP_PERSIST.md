# Crypto Policy persistence workflow for wallets

1. [Crypto Policy persistence workflow for wallets](#crypto-policy-persistence-workflow-for-wallets)
   1. [Document versions](#document-versions)
   2. [Purpose](#purpose)
   3. [Scope](#scope)
   4. [Out of scope](#out-of-scope)
   5. [Expected implementation gaps (after PR1)](#expected-implementation-gaps-after-pr1)
2. [Part I — Functional specifications](#part-i--functional-specifications)
   1. [Definitions](#definitions)
   2. [Core product rule](#core-product-rule)
   3. [Interface-independent challenge requirement](#interface-independent-challenge-requirement)
   4. [Functional workflow — EOA wallet](#functional-workflow--eoa-wallet)
   5. [Functional workflow summary](#functional-workflow-summary)
   6. [Required UX behavior](#required-ux-behavior)
   7. [Error semantics](#error-semantics)
3. [Part II — Technical specifications](#part-ii--technical-specifications)
   1. [Backend enforcement](#backend-enforcement)
   2. [Proposed CPM API contract](#proposed-cpm-api-contract)
   3. [Challenge message format](#challenge-message-format)
   4. [Ephemeral authorization model](#ephemeral-authorization-model)
   5. [Address normalization](#address-normalization)
   6. [Expiration and replay protection](#expiration-and-replay-protection)
   7. [Security requirements](#security-requirements)
   8. [Audit requirements](#audit-requirements)
   9. [OpenAPI requirements](#openapi-requirements)
   10. [Testing requirements](#testing-requirements)
4. [Part III — Stories and tasks](#part-iii--stories-and-tasks)
   1. [Epic](#epic)
   2. [Baseline / non-regression stories](#baseline--non-regression-stories)
   3. [User stories](#user-stories)
   4. [Implementation tasks](#implementation-tasks)
   5. [Tracking table](#tracking-table)
5. [Part IV — PR breakdown](#part-iv--pr-breakdown)
6. [Part V — TODO list for future wallet types](#part-v--todo-list-for-future-wallet-types)
7. [Part VI — Frozen decisions](#part-vi--frozen-decisions) (§33–§43)
8. [Part VII — Non-goals](#part-vii--non-goals)
9. [Part VIII — Summary](#part-viii--summary)

---

## Document versions


| Date           | Author        | Version | Comments      |
| -------------- | ------------- | ------- | ------------- |
| Jun 10th, 2026 | O. Lodygensky | 0.1     | First version |
| Jun 10th, 2026 | ChatGPT       | 0.2     | Clarify off-chain signature, ephemeral proof storage and PR split |
| Jun 10th, 2026 | ChatGPT       | 0.3     | Harden stories, tasks and backend enforcement guardrails |
| Jun 10th, 2026 | O. Lodygensky | 0.4     | Freeze API contract, TTL, Redis fail-closed and persist consume ordering |
| Jun 10th, 2026 | O. Lodygensky | 0.5     | Align PR3–PR8 breakdown and tasks with frozen decisions |
| Jun 10th, 2026 | O. Lodygensky | 0.6     | Document CPM-owned ephemeral store, CPM_REDIS_URL and store abstractions |
| Jun 10th, 2026 | O. Lodygensky | 0.7     | Editorial cleanup; align WORKPLAN/README; clarify PR8 vs PR1 cross-links |
| Jun 10th, 2026 | O. Lodygensky | 0.8     | Document expected implementation gaps after PR1 and independent V1 sign-off |


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
- Wallet control proof through a signed challenge.
- Challenge enforcement for all interfaces:
  - Web UI.
  - CLI.
  - Direct API usage.
- Transition from non-actionable draft to persisted CP.
- Backend-side enforcement in CPM.
- CPM-owned ephemeral store for wallet challenges and proofs (`ChallengeStore` / `ProofStore`, `CPM_REDIS_URL` in V1).
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

## Expected implementation gaps (after PR1)

PR1 is **docs-only** and freezes the CP-PERSIST V1 contract. It intentionally does not implement runtime behavior.

The following gaps are **expected** after PR1 merge. They are **not** PR1 defects; each maps to a later PR in Part IV.

1. **EOA persistence enforcement is not implemented yet.** This is **PR5**. No EOA CP persistence path may remain callable without `wallet_control_proof_id`. Existing `POST /api/cpm/v1/policies` must be blocked, migrated or made compliant for EOA flows.

2. **OpenAPI is not implemented yet.** This is **PR2** and is the **mandatory gate** before backend implementation.

3. **`CPM_REDIS_URL` is decided and documented.** **PR3** will implement runtime config loading, `ChallengeStore` / `ProofStore`, Redis adapter, TTL, namespace and fail-closed behavior. Deploy wiring may land with PR3 or an adjacent deploy change.

4. **Frontend and CLI still use legacy or mock persistence flows.** **PR6** and **PR7** will migrate them to the frozen CP-PERSIST V1 flow. Session auth and wallet challenge are **orthogonal**: session auth identifies the user; wallet challenge proves control of the EOA for the persist action.

5. **CP-PERSIST V1 is signed off independently through this document** (Part VI frozen decisions). [`workplans/WORKPLAN_API.md`](../workplans/WORKPLAN_API.md) remains a broader API workplan and may keep its global proposal status.

Do not expose PR3–PR4 endpoints to product users without **PR5** enforcement unless they are feature-flagged or otherwise unreachable from product flows.

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

## Interface-independent challenge requirement

Wallet control challenge is mandatory regardless of the interface used.

The following interfaces must all go through the same backend-enforced challenge flow:

- Web UI.
- CLI.
- Direct API usage.
- Future automation tools.
- Future admin or integration interfaces, unless explicitly exempted by a documented internal service contract.

The Web UI may provide a MetaMask-based signing experience.

The CLI may use a local wallet, hardware wallet, private key signer, external signer or wallet provider.

However, both UI and CLI must call the same CPM challenge APIs and must receive the same backend authorization before CP persistence.

The frontend must never be considered the source of trust.

The backend must reject CP persistence when no valid wallet control proof exists.

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

### Step 4 — Wallet challenge preparation

Before persisting a CP for an EOA wallet, the user must produce an off-chain wallet signature proving control of the wallet.

This is not implemented.

The signature is off-chain:

- no blockchain transaction is sent
- no gas is paid
- no on-chain state is modified
- the private key never leaves the wallet

However, because the signature authorizes a backend-side action — persisting an official CP for a wallet — CPM must verify the proof before accepting the persistence request.


The CPM backend creates a challenge containing:

- wallet address
- chain id
- scan id
- draft id
- action
- nonce
- expiration time

The client asks the wallet to sign this exact message.

The signed message is then submitted back to CPM for verification.

This:
- is simple to test
- avoids message-format drift between UI and CLI
- gives CPM a clear anti-replay model
- makes persistence authorization fully backend-enforced

---

### Step 5 — Wallet signature

The user signs the challenge message.

This is not implemented.


For the Web UI, this will typically be done through MetaMask or another injected wallet provider.

For the CLI, this may be done through:

- a local EOA signer
- a hardware wallet
- an external wallet provider
- a private key stored outside CAFE
- another signing adapter

The private key must never be sent to the backend.  

The backend only receives:

- challenge identifier
- wallet address
- chain id
- signature
- optional signature metadata

---

### Step 6 — Challenge verification

The client submits the signed challenge.

This is not implemented yet.

The signature is off-chain:

* no blockchain transaction is sent
* no gas is paid
* no on-chain state is modified
* the private key never leaves the wallet

However, because the signature authorizes a backend-side action — persisting an official Crypto Policy for a wallet — CPM must verify the proof before accepting the persistence request.

The backend verifies:

* challenge exists
* challenge is not expired
* challenge was created by the same authenticated user
* challenge belongs to the same tenant or organization
* challenge was not already consumed
* `wallet_address` matches the challenge
* `chain_id` matches the challenge
* recovered EOA address matches `wallet_address`
* `draft_id` still exists
* `draft_id` is linked to the same `scan_id`
* `scan_id` is linked to the same `wallet_address`
* action is `persist_crypto_policy`

If verification succeeds, CPM creates a short-lived persistence authorization.

Example response:

```json
{
  "wallet_control_proof_id": "uuid",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "draft_id": "uuid",
  "scan_id": "uuid",
  "action": "persist_crypto_policy",
  "verified_at": "2026-06-10T12:25:00Z",
  "expires_at": "2026-06-10T12:35:00Z"
}
```

The `wallet_control_proof_id` is not a permanent delegation.

It is a short-lived authorization to persist the specific draft for the specific wallet and scan.

The `wallet_control_proof_id` must not be stored in the durable database as a reusable authorization.

Recommended first implementation:

```text
Store `wallet_control_proof_id` in the CPM ephemeral store with a 10-minute TTL.
```

The ephemeral proof must be:

* short-lived
* single-use
* bound to the authenticated user
* bound to the tenant or organization
* bound to the wallet address
* bound to the chain id
* bound to the scan id
* bound to the draft id
* bound to the action `persist_crypto_policy`

At persist time, the ephemeral proof must be atomically consumed or reserved in the CPM ephemeral store **before** CP creation. If CP creation fails, the proof remains consumed and the user must sign again.

The raw signature must not be stored in the ephemeral store or in the durable database for V1.

The persisted CP may store minimal audit metadata, for example:

```text
ownership_status = verified
wallet_control_method = eoa_signature
wallet_control_verified_at = <timestamp>
```

The persisted CP must not store the ephemeral proof, raw signature or reusable proof artifacts.

---

### Step 7 — CP persistence

The user persists the draft into an official CP.

A CLI-like persistence flow already exists in `cafe-deploy/scripts/test-discovery-v1-wallet-scans-to-cpm.sh`, but it does not implement the wallet control challenge yet.

The existing flow must therefore be treated as pre-`CP_PERSIST` behavior and must be upgraded before being considered compliant.

The backend verifies:

- authenticated user
- draft exists
- draft belongs to the user or tenant scope
- draft is not already persisted
- draft is linked to an EOA wallet
- `wallet_control_proof_id` exists
- proof is valid
- proof is not expired
- proof is not already consumed
- proof is bound to the same `draft_id`
- proof is bound to the same `scan_id`
- proof is bound to the same `wallet_address`
- proof is bound to the same `chain_id`
- proof action is `persist_crypto_policy`

If all checks pass, CPM **atomically consumes or reserves the proof in the CPM ephemeral store**, then creates the persisted policy from the draft.

If CP creation fails after the proof has been consumed, the proof **remains consumed**. The user must create a new challenge and sign again.

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

4. User requests wallet challenge
   -> challenge bound to wallet, chain, scan, draft, user, tenant, action

5. User signs challenge
   -> UI or CLI, same backend contract

6. Backend verifies signature
   -> creates short-lived wallet_control_proof_id

7. User persists draft
   -> POST /api/cpm/v1/drafts/{draft_id}/persist with wallet_control_proof_id

8. CPM atomically consumes proof, then creates persisted CP
   -> if CP creation fails, proof stays consumed; user must re-sign
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

1. request a challenge from CPM (`POST /api/cpm/v1/wallet-challenges`)
2. ask the wallet provider to sign the message (EIP-191 / `personal_sign`)
3. submit the signature to CPM (`POST /api/cpm/v1/wallet-challenges/verify`)
4. receive a `wallet_control_proof_id`
5. call `POST /api/cpm/v1/drafts/{draft_id}/persist` with the proof

If the wallet signature fails, the CP remains a draft.

If CP creation fails after the proof was consumed, the UI must prompt the user to sign a new challenge.

### CLI

The current CLI is `cafe-frontend/scripts/cafe.sh`; it is based on the API.

The current CLI must be enhanced to manage CPM persistence workflows.

The CLI must follow the same backend workflow.

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

cafe wallet sign \
  --challenge-id <challenge_id>

cafe cpm wallet-challenge verify \
  --challenge-id <challenge_id> \
  --wallet 0xabc... \
  --chain-id 1 \
  --signature 0x...

cafe cpm draft persist \
  --draft-id <draft_id> \
  --wallet-control-proof-id <wallet_control_proof_id>
```

The signing mechanism may vary, but the backend verification must remain identical.

### Direct API

Direct API users must follow the same sequence.

Calling the persistence endpoint without a valid `wallet_control_proof_id` must return an error.

Expected error:

```json
{
  "error": "WALLET_CONTROL_PROOF_REQUIRED",
  "message": "Persisting a Crypto Policy for a wallet requires a valid wallet control proof."
}
```

---

## Error semantics

### Challenge creation errors


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
| Ephemeral store unavailable | 503       | `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` |


For the first implementation, only `wallet_type = eoa` is supported for persistence.

### Challenge verification errors


| Case                       | HTTP status | Error code                          |
| -------------------------- | ----------- | ----------------------------------- |
| Unknown challenge          | 404         | `CHALLENGE_NOT_FOUND`               |
| Expired challenge          | 410         | `CHALLENGE_EXPIRED`                 |
| Consumed challenge         | 409         | `CHALLENGE_ALREADY_CONSUMED`        |
| Invalid signature          | 401         | `INVALID_WALLET_SIGNATURE`          |
| Recovered address mismatch | 401         | `WALLET_SIGNATURE_ADDRESS_MISMATCH` |
| Wrong user or tenant       | 403         | `CHALLENGE_OUT_OF_SCOPE`            |
| Ephemeral store unavailable | 503        | `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` |


### Persistence errors


| Case                        | HTTP status | Error code                              |
| --------------------------- | ----------- | --------------------------------------- |
| Missing proof               | 400         | `WALLET_CONTROL_PROOF_REQUIRED`         |
| Unknown proof               | 404         | `WALLET_CONTROL_PROOF_NOT_FOUND`        |
| Expired proof               | 410         | `WALLET_CONTROL_PROOF_EXPIRED`          |
| Consumed proof              | 409         | `WALLET_CONTROL_PROOF_ALREADY_CONSUMED` |
| Proof does not match draft  | 409         | `WALLET_CONTROL_PROOF_DRAFT_MISMATCH`   |
| Proof does not match wallet | 409         | `WALLET_CONTROL_PROOF_WALLET_MISMATCH`  |
| Proof does not match scan   | 409         | `WALLET_CONTROL_PROOF_SCAN_MISMATCH`    |
| Unsupported wallet type     | 422         | `UNSUPPORTED_WALLET_TYPE`               |
| Draft already persisted     | 409         | `DRAFT_ALREADY_PERSISTED`               |
| Ephemeral store unavailable | 503         | `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` |


---

# Part II — Technical specifications

## 10. Backend enforcement

The wallet challenge requirement must be enforced by CPM backend.

The frontend is not trusted.

The CLI is not trusted.

Any direct API caller is not trusted.

The persistence handler must reject requests without a valid backend-side wallet control proof.

`POST /api/cpm/v1/policies` must **not** allow EOA Crypto Policy persistence without a valid `wallet_control_proof_id`. Legacy or pre-CP_PERSIST callers of this route must receive `WALLET_CONTROL_PROOF_REQUIRED` (or an equivalent blocking error) for EOA wallet flows.

The only normative EOA persist route is:

```text
POST /api/cpm/v1/drafts/{draft_id}/persist
```

Persist ordering (frozen):

```text
1. Validate draft, proof bindings and EOA scope.
2. Atomically consume or reserve the proof in the CPM ephemeral store.
3. Create the persisted CP from the draft.
4. If step 3 fails, the proof remains consumed; the user must sign a new challenge.
```

Required invariant:

```text
No persisted CP for an EOA wallet can be created without a valid wallet_control_proof_id.
```

Recommended test name:

```text
TestPersistPolicyRequiresWalletControlProofForEOA
```

---

## 11. Proposed CPM API contract

### 11.1 Create challenge

```http
POST /api/cpm/v1/wallet-challenges
```

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
  "challenge_id": "uuid",
  "message": "string",
  "nonce": "string",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "draft_id": "uuid",
  "action": "persist_crypto_policy",
  "expires_at": "date-time"
}
```

### 11.2 Verify challenge

```http
POST /api/cpm/v1/wallet-challenges/verify
```

Request:

```json
{
  "challenge_id": "uuid",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "signature": "0x..."
}
```

Response:

```json
{
  "wallet_control_proof_id": "uuid",
  "wallet_address": "0xabc...",
  "chain_id": 1,
  "scan_id": "uuid",
  "draft_id": "uuid",
  "action": "persist_crypto_policy",
  "method": "eoa_signature",
  "verified_at": "date-time",
  "expires_at": "date-time"
}
```

### 11.3 Persist draft

```http
POST /api/cpm/v1/drafts/{draft_id}/persist
```

This endpoint expresses the domain transition from draft to persisted policy.

Request:

```json
{
  "wallet_control_proof_id": "uuid"
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

`POST /api/cpm/v1/policies` is **not** the CP persistence endpoint for this workflow. For EOA wallets, it must **not** create a persisted CP without `wallet_control_proof_id` and must return `WALLET_CONTROL_PROOF_REQUIRED` (or an equivalent blocking error).

---

## 12. Challenge message format

**Decision (frozen for CP-PERSIST V1):** EIP-191 / `personal_sign`-compatible signed message.

The message should be deterministic and human-readable.

Recommended message structure:

```text
CAFE Crypto Policy Persistence

Domain: <frontend_or_api_domain>
Action: persist_crypto_policy
Wallet: <wallet_address>
Chain ID: <chain_id>
Scan ID: <scan_id>
Draft ID: <draft_id>
Challenge ID: <challenge_id>
Nonce: <nonce>
Issued At: <issued_at>
Expiration Time: <expires_at>

By signing this message, I prove control of the wallet and authorize CAFE to persist the selected Crypto Policy draft for this wallet.
```

The exact message must be stored server-side in an ephemeral store with TTL, not in the durable database.

The backend must verify the signature against the exact stored message, not against a reconstructed message that might differ in formatting.

The backend must verify signatures using EIP-191 `personal_sign` semantics in V1.

Future versions may support SIWE / EIP-4361 more formally.

---

## 13. Ephemeral authorization model

Wallet challenges and wallet control proofs are transient authorization artifacts.

They must not be stored as durable database records.

### 13.0 CPM ephemeral store ownership and deployment

**Decision (frozen for CP-PERSIST V1):**

```text
CPM owns the wallet challenge/proof ephemeral store.
```

The store is internal to the **CPM bounded context**. It must not depend on:

- Discovery scan cache semantics or keyspaces;
- the Discovery database;
- Discovery internal domain structs;
- any other service’s Redis usage patterns.

CPM may depend on Discovery only through the HTTP/JWT contracts required by the product workflow (session validation, scan authorization). The challenge/proof store is **not** delegated to Discovery auth.

**Redis is the recommended V1 adapter** for the CPM ephemeral store. Redis is an **implementation detail**, not a business dependency on Discovery or any other product domain.

Future implementation must use store abstractions first:

```text
ChallengeStore
ProofStore
```

Handlers must depend on these interfaces, not on a Redis client used directly in route code. A handler must not know which Redis instance backs the store — only the configured CPM ephemeral store.

**Configuration (frozen target):**

```text
CPM_REDIS_URL
```

Examples:

```text
# Local dev — shared internal Redis on the Docker network (logical DB separation)
CPM_REDIS_URL=redis://redis:6379/1

# Production target — dedicated CPM Redis instance
CPM_REDIS_URL=redis://redis-cpm:6379/0
```

Rules:

- use the `redis://` URL scheme, not `http://`;
- Redis must remain **internal to the Docker network** — do not publish Redis ports on the host for this use case;
- changing from a shared internal instance to a dedicated CPM instance must be a **deployment-only** change (`CPM_REDIS_URL`), with **no** API or handler code change.

**Mandatory key namespace** (even when using a separate logical DB index):

```text
cpm:wallet_challenge:<challenge_id>
cpm:wallet_control_proof:<wallet_control_proof_id>
```

The `cpm:*` prefix is required to avoid collision with any other Redis usage on the same instance.

**Dev vs production strategy:**

```text
Local dev / simple compose:
  CPM may use the existing internal Redis instance via CPM_REDIS_URL.
  Recommended local URL: redis://redis:6379/1
  Namespace cpm:* remains mandatory.

Production target:
  CPM should use a dedicated Redis instance (for example redis-cpm).
  Recommended prod URL: redis://redis-cpm:6379/0
```

A separate logical DB index (for example `/1`) is acceptable for local dev and simple staging. Production should use a dedicated CPM Redis instance or strictly operated instance/logical separation. Migrating from shared to dedicated Redis is a deployment change only.

**Long-term architecture note:**

```text
CAFE may later extract persistence and auth management into dedicated services.
Discovery is expected to become less stateful and should eventually avoid direct Redis/Postgres ownership.
This does not change CP-PERSIST V1: the wallet challenge/proof store belongs to CPM and is accessed through a configurable CPM ephemeral store (CPM_REDIS_URL in V1).
```

**Other frozen V1 store rules:**

- no durable database tables for challenges, proofs or raw signatures;
- single-use proof semantics;
- proof is atomically consumed or reserved in the CPM ephemeral store **before** CP creation (see §10);
- if the ephemeral store is unavailable, challenge create, challenge verify and CP persist must **fail closed** with **503** `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` — no fallback may allow EOA CP persistence without wallet control proof.

The durable database must only store the persisted CP and minimal audit metadata.

### 13.1 Ephemeral wallet challenge

Key format (Redis adapter):

```text
cpm:wallet_challenge:<challenge_id>
```

Fields:

```text
user_id
tenant_id
wallet_address
chain_id
scan_id
draft_id
action
message
nonce
expires_at
created_at
```

TTL:

```text
10 minutes
```

The challenge message is stored only to make signature verification deterministic and replay-safe.

It must not be persisted in the durable database.

### 13.2 Ephemeral wallet control proof

Key format (Redis adapter):

```text
cpm:wallet_control_proof:<wallet_control_proof_id>
```

Fields:

```text
challenge_id
user_id
tenant_id
wallet_address
chain_id
scan_id
draft_id
action
method = eoa_signature
verified_at
expires_at
consumed = false
```

TTL:

```text
10 minutes
```

The `wallet_control_proof_id` is a transient authorization handle.

It must not be stored in the durable database as a reusable credential.

At persist time, the proof must be **atomically consumed or reserved in the CPM ephemeral store before** the persisted CP is created. If CP creation fails after consume, the proof **remains consumed**.

The raw signature must not be stored in the ephemeral store or in the durable database for V1.

### 13.3 Durable persisted policy metadata

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

The persisted CP must not store the ephemeral proof, raw signature or reusable proof artifacts.

---

## 14. Address normalization

EOA addresses must be normalized before comparison.

For Ethereum-compatible chains:

- accept `0x` hexadecimal format
- validate length
- normalize for storage and comparison
- use case-insensitive comparison unless checksum validation is explicitly enforced
- optionally store checksum representation for display

All comparisons between:

- challenge wallet address
- draft wallet address
- scan wallet address
- recovered signature address

must use normalized address comparison.

---

## 15. Expiration and replay protection

Challenge requirements:

```text
challenge TTL: 10 minutes
challenge nonce: random, unique, single-use
challenge action: persist_crypto_policy
challenge binding: user, tenant, wallet, chain, scan, draft
```

Proof requirements:

```text
proof TTL: 10 minutes
proof usage: single-use
proof binding: user, tenant, wallet, chain, scan, draft, action
proof consume: atomically before CP creation; remains consumed if CP creation fails
```

Replay protection must prevent:

- using a challenge twice
- using a signature from another challenge
- using a proof for another draft
- using a proof for another wallet
- using a proof for another chain
- using a proof for another scan
- using a proof from another user or tenant
- using a proof after expiration

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

The challenge must expire.

The nonce must be random and unique.

The proof must be single-use.

The persistence handler must be the final enforcement point.

If the CPM ephemeral store is unavailable, CPM must fail closed with **503** `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` for challenge create, challenge verify and persist operations. No fallback may allow EOA CP persistence without wallet control proof.

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
When was the wallet control proof verified?
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

The durable audit trail must not store a reusable `wallet_control_proof_id`, raw signature or durable proof artifacts.

---

## 18. OpenAPI requirements

The OpenAPI contract must document:

- `POST /api/cpm/v1/wallet-challenges`
- `POST /api/cpm/v1/wallet-challenges/verify`
- `POST /api/cpm/v1/drafts/{draft_id}/persist`
- request schemas
- response schemas
- error schemas
- EOA-only limitation for the first release
- `WALLET_CONTROL_PROOF_REQUIRED`
- `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE`
- `UNSUPPORTED_WALLET_TYPE`

The OpenAPI description must explicitly state:

```text
Wallet control challenge is required for CP persistence regardless of whether the caller is the Web UI, CLI or a direct API client.
```

---

## 19. Testing requirements

### 19.1 Unit tests

Required tests:

```text
Create challenge for EOA draft succeeds
Create challenge fails for unknown draft
Create challenge fails for draft / scan mismatch
Create challenge fails for wallet mismatch
Verify challenge succeeds with valid EOA signature
Verify challenge fails with invalid signature
Verify challenge fails with expired challenge
Verify challenge fails with consumed challenge
Verify challenge fails when recovered address mismatches wallet
Persist draft fails without wallet_control_proof_id
Persist draft fails with expired proof
Persist draft fails with consumed proof
Persist draft fails with proof for another draft
Persist draft fails with proof for another wallet
Persist draft succeeds with valid proof
Persist draft atomically consumes proof before CP creation
Persist draft leaves proof consumed if CP creation fails after consume
POST /api/cpm/v1/policies blocks EOA persist without wallet_control_proof_id
Persist draft blocks all other persistence paths that do not provide wallet_control_proof_id
Persist draft rejects legacy or pre-CP_PERSIST CLI-like flows without proof
Persisted CP includes ownership metadata
Challenge create/verify/persist return 503 when ephemeral store is unavailable
```

### 19.2 API tests

Required API test scenarios:

```text
UI-like flow:
  create draft
  create challenge
  verify signature
  persist draft

CLI-like flow:
  create draft
  create challenge
  verify externally-produced signature
  persist draft

Direct API negative flow:
  create draft
  call persist without proof
  expect WALLET_CONTROL_PROOF_REQUIRED

Legacy/pre-CP_PERSIST negative flow:
  use an existing persistence path or script without challenge proof
  expect WALLET_CONTROL_PROOF_REQUIRED
```

### 19.3 Non-regression tests

Required non-regression tests:

```text
Discovery scan still works without wallet proof
CP explore still works without wallet proof
Draft creation still works without wallet proof
Only persistence requires wallet proof
TLS targets cannot be persisted as CP targets
Existing CP persistence paths cannot bypass wallet proof
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
The challenge flow is mandatory for Web UI, CLI and direct API usage.
```

Out of scope for this epic:

```text
Smart contract wallets, Safe / multisig, institutional delegation, TLS persistence targets.
```

---

## Baseline / non-regression stories

These stories are already implemented and must remain true during CP-PERSIST implementation.

| ID | Story | Priority | PR(s) | Status |
| --- | --- | --- | --- | --- |
| **CP-PERSIST-S1** | As an analyst, I want to scan a wallet without proving ownership so I can assess public on-chain posture. | Must | — | ✅ done |
| **CP-PERSIST-S2** | As a user, I want to explore CP decisions without proof so I can compare candidate policies without creating an official CP. | Must | — | ✅ done |
| **CP-PERSIST-S3** | As a consultant, I want to save a CP draft without proof so I can prepare a recommendation before the wallet owner is involved. | Must | — | ✅ done |

---

## User stories

| ID | Story | Priority | PR(s) | Status |
| --- | --- | --- | --- | --- |
| **CP-PERSIST-S4** | As a wallet owner using the Web UI, I want to sign an off-chain challenge with my EOA wallet so I can persist my CP without sending an on-chain transaction. | Must | PR2–PR6 | ⚪ planned |
| **CP-PERSIST-S5** | As a wallet owner using the CLI, I want to sign externally and submit the proof through the same CPM APIs so the CLI cannot bypass backend enforcement. | Must | PR2–PR5, PR7 | ⚪ planned |
| **CP-PERSIST-S6** | As an API integrator, I want a clear `WALLET_CONTROL_PROOF_REQUIRED` error when I call persist without a valid proof. | Must | PR2, PR5 | ⚪ planned |
| **CP-PERSIST-S7** | As a security officer, I want wallet control proofs to be ephemeral, single-use, TTL-bound and bound to user, tenant, wallet, chain, scan and draft. | Must | PR3–PR5 | ⚪ planned |
| **CP-PERSIST-S8** | As an auditor, I want persisted CPs to record minimal ownership verification metadata without storing reusable credentials. | Should | PR5 | ⚪ planned |
| **CP-PERSIST-S9** | As a developer, I want contract-first documentation and OpenAPI before implementation so UI, CLI and API clients share one backend contract. | Must | PR1, PR2 | 🟡 in progress |
| **CP-PERSIST-S10** | As a product owner, I want end-to-end documentation and validation so scan / explore / draft remain open while persist requires proof. | Should | PR8 | ⚪ planned |
| **CP-PERSIST-S11** | As a platform owner, I want CPM backend to be the only enforcement point for CP persistence so that UI, CLI, scripts or direct API calls cannot bypass wallet control proof. | Must | PR5 | ⚪ planned |

---

## Implementation tasks

Tasks are grouped by delivery PR. Each task references the user stories it satisfies.

### CP-PERSIST-T1 — Specification (PR1)

Stories: **CP-PERSIST-S9**

```text
[x] Add CP_PERSIST.md with functional and technical rules
[x] Document EOA-only first scope
[x] Document interface-independent challenge requirement
[x] Document recommended API contract and ephemeral proof model
[x] Document CPM-owned ephemeral store, CPM_REDIS_URL and store abstractions (§13.0)
[x] Document PR sequence in Part IV
[x] Update WORKPLAN_API.md cross-links and CP-PERSIST route notes
[x] Update README cross-links and planned CPM_REDIS_URL note
```

PR1 doc deliverables above are complete in this branch; story **S9** / **T1** remain **`in progress`** until the PR is reviewed and merged.

### CP-PERSIST-T2 — OpenAPI contract (PR2)

Stories: **CP-PERSIST-S6**, **CP-PERSIST-S9**

```text
[ ] Add OpenAPI for POST /api/cpm/v1/drafts/{draft_id}/persist
[ ] Document that POST /api/cpm/v1/policies is not the CP persistence endpoint
[ ] Ensure legacy/pre-CP_PERSIST persistence paths cannot bypass proof
[ ] Add wallet challenge create / verify schemas
[ ] Add persist-draft request / response schema with wallet_control_proof_id
[ ] Add error codes (WALLET_CONTROL_PROOF_REQUIRED, WALLET_CONTROL_PROOF_STORE_UNAVAILABLE, challenge errors)
[ ] Mark persistence as EOA-only for the first release
[ ] Document that UI, CLI and direct API share the same contract
```

### CP-PERSIST-T3 — Ephemeral challenge store (PR3)

Stories: **CP-PERSIST-S7**

```text
[ ] Document CPM-owned ephemeral store and CPM_REDIS_URL (§13.0)
[ ] Define ChallengeStore and ProofStore abstractions (handlers must not use Redis directly)
[ ] Implement Redis adapter behind ChallengeStore and ProofStore (CPM_REDIS_URL)
[ ] Use mandatory key namespace cpm:wallet_challenge:* and cpm:wallet_control_proof:*
[ ] TTL = 10 minutes for challenges and proofs
[ ] Implement single-use and atomic consume-or-reserve semantics for proofs
[ ] Fail closed with 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE if store is unavailable
[ ] No durable DB tables for wallet_challenges or wallet_control_proofs
[ ] No raw signature storage in ephemeral store or durable DB
[ ] Add tests for store unavailable / expired / already consumed
[ ] Add unit tests
```

### CP-PERSIST-T4 — EOA signature verification (PR4)

Stories: **CP-PERSIST-S4**, **CP-PERSIST-S5**, **CP-PERSIST-S7**

```text
[ ] Implement POST /api/cpm/v1/wallet-challenges
[ ] Implement POST /api/cpm/v1/wallet-challenges/verify
[ ] Implement EOA signature verifier and address normalization
[ ] Verify exact stored challenge message, not reconstructed message
[ ] Support EIP-191 / personal_sign verification
[ ] Normalize signature recovery edge cases, including v = 27/28 vs 0/1 if needed
[ ] Add deterministic signature test vectors
[ ] Add tests for wrong wallet, wrong chain_id, wrong draft_id, wrong scan_id
[ ] Bind proof to user, tenant, wallet, chain, scan, draft and action
[ ] Mark challenge consumed after successful verification
[ ] Do not store raw signature in ephemeral store or durable DB
[ ] Return 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE when ephemeral store is unavailable
[ ] Add unit tests and API tests
```

### CP-PERSIST-T5 — Persistence enforcement (PR5)

Stories: **CP-PERSIST-S6**, **CP-PERSIST-S7**, **CP-PERSIST-S8**, **CP-PERSIST-S11**

```text
[ ] Implement POST /api/cpm/v1/drafts/{draft_id}/persist with wallet_control_proof_id
[ ] Block EOA persistence via POST /api/cpm/v1/policies without wallet_control_proof_id (block, migrate or make compliant)
[ ] Reject missing, mismatched, expired or consumed proofs
[ ] Atomically consume or reserve proof in CPM ephemeral store before CP creation
[ ] No fallback if ephemeral store is unavailable — fail closed with 503
[ ] If CP creation fails after consume, leave proof consumed; user must re-sign
[ ] Store ownership metadata on persisted CP (no raw signature or reusable proof artifacts)
[ ] All existing EOA CP persistence paths must require wallet control proof
[ ] Add regression test for the current CLI-like/pre-CP_PERSIST flow
[ ] Add unit tests and API tests
```

### CP-PERSIST-T6 — Web UI integration (PR6)

Stories: **CP-PERSIST-S4**

```text
[ ] Migrate from /wallet-challenge/start|verify to /wallet-challenges and /wallet-challenges/verify
[ ] Request challenge from CPM on persist action
[ ] Integrate MetaMask or injected wallet provider for EIP-191 / personal_sign
[ ] Submit signature and receive wallet_control_proof_id
[ ] Call POST /api/cpm/v1/drafts/{draft_id}/persist with proof (not POST /policies)
[ ] Display draft / unverified / ready to sign / verified / persisted states
[ ] Handle signature rejection and CP creation failure after consumed proof (re-sign flow)
[ ] Add frontend tests
```

### CP-PERSIST-T7 — CLI integration (PR7)

Stories: **CP-PERSIST-S5**

```text
[ ] Add cafe cpm wallet-challenge create -> POST /api/cpm/v1/wallet-challenges
[ ] Add cafe cpm wallet-challenge verify -> POST /api/cpm/v1/wallet-challenges/verify
[ ] Add cafe cpm draft persist -> POST /api/cpm/v1/drafts/{draft_id}/persist
[ ] Support external EIP-191 / personal_sign signing
[ ] Reject --force or --skip-wallet-proof options
[ ] Upgrade test-discovery-v1-wallet-scans-to-cpm.sh to challenge + persist flow
[ ] Document CLI workflow (cpm-developer.md or equivalent)
[ ] Add CLI tests where applicable
```

### CP-PERSIST-T8 — Documentation and E2E validation (PR8)

Stories: **CP-PERSIST-S10**, **CP-PERSIST-S1**, **CP-PERSIST-S2**, **CP-PERSIST-S3**

```text
[ ] Confirm README and WORKPLAN_API remain aligned with the implemented V1 contract
[ ] Update cafe-frontend/docs/cpm-developer.md with frozen V1 contract
[ ] Document scan vs explore vs draft vs persist (frozen routes §33–§41)
[ ] Document EOA-only scope, CPM-owned ephemeral store, CPM_REDIS_URL, cpm:* namespace
[ ] Document 10-minute TTL, single-use proof, consume-before-create
[ ] Document 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE, no fallback persist, and re-sign flow
[ ] Add end-to-end test notes and manual test scenario
[ ] Add troubleshooting section
[ ] Confirm Discovery / explore / draft still work without proof (S1–S3)
[ ] Confirm all persistence paths require wallet control proof (UI, CLI, smoke, API)
[ ] Confirm test-discovery-v1-wallet-scans-to-cpm.sh is compliant
```

---

## Tracking table

| Task / Story | PR | Repository | Depends on | Status |
| --- | --- | --- | --- | --- |
| **CP-PERSIST-T1** / **S9** | PR1 | `cafe-crypto-policy-mgt` | — | 🟡 in progress (doc complete; awaiting merge) |
| **CP-PERSIST-T2** / **S6**, **S9** | PR2 | `cafe-crypto-policy-mgt` | PR1 | ⚪ planned |
| **CP-PERSIST-T3** / **S7** | PR3 | `cafe-crypto-policy-mgt` | PR2 | ⚪ planned |
| **CP-PERSIST-T4** / **S4**, **S5**, **S7** | PR4 | `cafe-crypto-policy-mgt` | PR3 | ⚪ planned |
| **CP-PERSIST-T5** / **S6**, **S7**, **S8**, **S11** | PR5 | `cafe-crypto-policy-mgt` | PR4 | ⚪ planned |
| **CP-PERSIST-T6** / **S4** | PR6 | `cafe-frontend` | PR5 | ⚪ planned |
| **CP-PERSIST-T7** / **S5** | PR7 | `cafe-frontend` (`cafe.sh`), `cafe-deploy` (smoke) | PR5 | ⚪ planned |
| **CP-PERSIST-T8** / **S10**, **S1–S3** | PR8 | multi-repo | PR6, PR7 | ⚪ planned |

Recommended delivery order:

```text
PR1 → PR2 → PR3 → PR4 → PR5 → (PR6 and PR7 in parallel) → PR8
```

---

# Part IV — PR breakdown

## 20. PR1 — Contract-first CP persistence specification

Repository:

```text
cafe-crypto-policy-mgt
```

Goal:

- Add this document as `CP_PERSIST.md`.
- Document functional rules.
- Document EOA-only first scope.
- Document challenge requirement for UI, CLI and direct API.
- Document recommended API contract.
- Document PR sequence.

Deliverables:

```text
CP_PERSIST.md
WORKPLAN_API.md cross-links and CP-PERSIST route / EOA persist clarifications
README cross-links and planned CPM_REDIS_URL note
```

No implementation in this PR.

**Expected implementation gaps after this PR:** see [Expected implementation gaps (after PR1)](#expected-implementation-gaps-after-pr1). Reviewers must not treat current runtime behavior (proof-free `POST /api/cpm/v1/policies`, missing OpenAPI, no Redis store, legacy frontend paths) as documentation defects.

Expected commit title:

```text
docs(cpm): define wallet control proof for CP persistence
```

---

## 21. PR2 — OpenAPI contract for EOA wallet challenge

Repository:

```text
cafe-crypto-policy-mgt
```

Goal:

- Add OpenAPI definitions for wallet challenge creation and verification.
- Add OpenAPI definitions for `POST /api/cpm/v1/drafts/{draft_id}/persist`.
- Document that `POST /api/cpm/v1/policies` is not the CP persistence endpoint for this workflow.
- Add error codes.
- Mark persistence as EOA-only for the first release.

Deliverables:

```text
OpenAPI schemas
Challenge request / response schema
Challenge verification request / response schema
Persist draft request / response schema
Existing persistence route behavior
Error schema updates
```

Acceptance criteria:

```text
OpenAPI documents POST /api/cpm/v1/wallet-challenges and /wallet-challenges/verify.
OpenAPI documents POST /api/cpm/v1/drafts/{draft_id}/persist.
OpenAPI documents that CP persistence requires wallet control proof.
OpenAPI documents that POST /api/cpm/v1/policies must not persist EOA CP without proof.
OpenAPI documents 10-minute TTL, single-use proof and EIP-191 / personal_sign for V1.
OpenAPI documents WALLET_CONTROL_PROOF_STORE_UNAVAILABLE (503) when the CPM ephemeral store is unavailable.
OpenAPI documents that challenge is required for UI, CLI and API.
OpenAPI documents that only EOA is supported in this release.
```

Expected commit title:

```text
docs(openapi): add EOA wallet challenge contract for CP persistence
```

---

## 22. PR3 — CPM backend ephemeral challenge store

Repository:

```text
cafe-crypto-policy-mgt
```

Depends on: **PR2** (frozen key format, TTL and store semantics documented in OpenAPI or internal config).

Goal:

- Implement the **CPM-owned** ephemeral store for wallet challenges and proofs (§13.0, §38, §39).
- Introduce `ChallengeStore` and `ProofStore` abstractions; handlers must not use a Redis client directly.
- Provide a **Redis adapter** configured via `CPM_REDIS_URL` (recommended V1 implementation detail).
- Use mandatory `cpm:*` key namespace and a **10-minute** TTL.
- Implement single-use and atomic consume-or-reserve semantics for proofs (§37, §41).
- Fail closed with **503** `WALLET_CONTROL_PROOF_STORE_UNAVAILABLE` when the store is unavailable (§40).
- Do not add durable database tables for challenges, proofs or raw signatures.

Deliverables:

```text
ChallengeStore and ProofStore interfaces
Redis adapter (CPM_REDIS_URL, redis:// scheme)
cpm:wallet_challenge:<challenge_id> and cpm:wallet_control_proof:<wallet_control_proof_id> key formats
10-minute TTL constant
atomic single-use / consume-or-reserve methods
503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE mapping
unit tests (available, expired, consumed, store unavailable)
```

Deployment note: wiring `CPM_REDIS_URL` in runtime compose/env is a **separate deploy change**, not part of this PR’s code contract. V1 local example: `redis://redis:6379/1`. Production target: `redis://redis-cpm:6379/0`.

Acceptance criteria:

```text
CPM owns the ephemeral store; no dependency on Discovery cache or keyspaces.
Handlers depend on ChallengeStore / ProofStore, not on Redis directly.
Challenge can be created and retrieved through the CPM ephemeral store.
Challenge expires automatically after 10 minutes.
Challenge can be atomically marked as consumed or deleted.
Proof can be created and retrieved through the CPM ephemeral store.
Proof expires automatically after 10 minutes.
Proof can be atomically consumed or reserved (used by PR5 persist handler).
All store operations return 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE when the store is unavailable.
No durable DB migration is introduced for wallet_challenges or wallet_control_proofs.
No raw signatures are stored in the ephemeral store or durable DB.
```

Expected commit title:

```text
feat(cpm): add ephemeral wallet challenge store
```

---

## 23. PR4 — CPM backend EOA signature verification

Repository:

```text
cafe-crypto-policy-mgt
```

Depends on: **PR3** (challenge and proof stores).

Goal:

- Implement frozen V1 challenge routes (§35): `POST /api/cpm/v1/wallet-challenges` and `POST /api/cpm/v1/wallet-challenges/verify`.
- Verify EIP-191 / `personal_sign` signatures against the **exact stored challenge message** (§36).
- Recover signer address and compare to normalized wallet address.
- On success, create a **single-use** `wallet_control_proof_id` in the CPM ephemeral store with **10-minute** TTL (§37, §39).
- Mark the challenge as consumed after successful verification.
- Do not store the raw signature in the ephemeral store or durable DB (§38).

Deliverables:

```text
POST /api/cpm/v1/wallet-challenges handler
POST /api/cpm/v1/wallet-challenges/verify handler
deterministic EIP-191 human-readable challenge message builder
EOA signature verifier (personal_sign)
address normalization helper
signature recovery edge-case handling (v = 27/28 vs 0/1)
deterministic signature test vectors
unit tests
API tests
```

Acceptance criteria:

```text
POST /api/cpm/v1/wallet-challenges creates a challenge bound to user, tenant, wallet, chain, scan, draft and action.
Challenge message follows §12 and is stored exactly in the CPM ephemeral store.
Valid EIP-191 / personal_sign signature creates wallet_control_proof_id in the CPM ephemeral store (10-minute TTL, single-use).
Invalid signature is rejected.
Expired challenge is rejected.
Consumed challenge is rejected.
Recovered address mismatch is rejected.
Verification uses the exact stored challenge message, not a reconstructed variant.
Wrong wallet, wrong chain_id, wrong draft_id and wrong scan_id are rejected.
Challenge is consumed after successful verification.
Verify returns 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE when the ephemeral store is unavailable.
No raw signature is persisted in the ephemeral store or durable DB.
```

Expected commit title:

```text
feat(cpm): verify EOA wallet challenge signatures
```

---

## 24. PR5 — Enforce wallet proof on CP persistence

Repository:

```text
cafe-crypto-policy-mgt
```

Depends on: **PR4** (challenge verify and proof creation).

Goal:

- Implement `POST /api/cpm/v1/drafts/{draft_id}/persist` with `wallet_control_proof_id`.
- Block EOA CP persistence via `POST /api/cpm/v1/policies` without `wallet_control_proof_id`.
- Reject persistence without proof.
- Reject mismatched, expired or consumed proofs.
- Atomically consume or reserve the proof in the CPM ephemeral store **before** CP creation.
- If CP creation fails after consume, leave the proof consumed; user must re-sign.
- Store ownership metadata on persisted CP without raw signature or reusable proof artifacts.
- Fail closed with **503** if the ephemeral store is unavailable — no fallback persist without proof.

Deliverables:

```text
POST /api/cpm/v1/drafts/{draft_id}/persist implementation
Proof validation and atomic consume-or-reserve via ProofStore
EOA blocking on legacy POST /api/cpm/v1/policies (block, migrate or make compliant)
Ownership metadata on persisted policy
Unit tests
API tests
```

Acceptance criteria:

```text
EOA CP persistence without proof returns WALLET_CONTROL_PROOF_REQUIRED.
All existing EOA CP persistence paths require wallet_control_proof_id.
POST /api/cpm/v1/policies cannot persist an EOA CP without wallet_control_proof_id.
EOA CP persistence with valid proof succeeds via POST /api/cpm/v1/drafts/{draft_id}/persist.
Proof is consumed atomically in the CPM ephemeral store before CP creation.
If CP creation fails after consume, proof remains consumed and cannot be reused.
No fallback allows EOA CP persistence when the ephemeral store is unavailable.
Valid proof cannot be reused for another draft, wallet or persist attempt.
Persisted policy includes ownership_status = verified.
Persisted policy includes wallet_control_method = eoa_signature.
Persisted policy does not store raw signature or reusable proof artifacts.
```

Expected commit title:

```text
feat(cpm): require wallet control proof to persist EOA policies
```

---

## 25. PR6 — Web UI integration

Repository:

```text
cafe-frontend
```

Goal:

- Align frontend with frozen V1 contract (`/wallet-challenges`, `/drafts/{draft_id}/persist`).
- Use MetaMask or injected wallet provider for EIP-191 / `personal_sign`.
- Persist draft only after backend proof verification.
- Display clear draft / verified / persisted states.

Deliverables:

```text
Challenge creation client (POST /api/cpm/v1/wallet-challenges)
Wallet signing integration (EIP-191 / personal_sign)
Challenge verification client (POST /api/cpm/v1/wallet-challenges/verify)
Persist draft client (POST /api/cpm/v1/drafts/{draft_id}/persist)
UI state labels and re-sign flow after consumed-proof CP failure
Frontend tests
```

Acceptance criteria:

```text
User cannot persist an EOA CP without signing the challenge.
UI uses frozen V1 API paths, not legacy /wallet-challenge/start|verify or POST /policies persist.
UI displays unverified drafts as non-actionable.
UI displays persisted CP only after successful backend persistence.
UI handles signature rejection and re-sign after consumed proof gracefully.
```

Expected commit title:

```text
feat(frontend): add EOA wallet challenge flow for CP persistence
```

---

## 26. PR7 — CLI integration

Repository:

```text
cafe-frontend (cafe-frontend/scripts/cafe.sh)
```

Depends on: **PR5**. May run in parallel with **PR6**.

Goal:

- Extend `cafe.sh` with frozen V1 CPM wallet-challenge and persist commands (§35, §33).
- Call `POST /api/cpm/v1/wallet-challenges` and `/wallet-challenges/verify` — not legacy `/wallet-challenge/start|verify`.
- Persist via `POST /api/cpm/v1/drafts/{draft_id}/persist` with `wallet_control_proof_id` — not `POST /policies`.
- Support external EIP-191 / `personal_sign` signing (local signer, hardware wallet, etc.).
- Ensure the CLI cannot bypass backend proof requirements.
- Upgrade `cafe-deploy/scripts/test-discovery-v1-wallet-scans-to-cpm.sh` to the compliant flow.

Deliverables:

```text
cafe cpm wallet-challenge create  -> POST /api/cpm/v1/wallet-challenges
cafe cpm wallet-challenge verify  -> POST /api/cpm/v1/wallet-challenges/verify
cafe cpm draft persist            -> POST /api/cpm/v1/drafts/{draft_id}/persist
no --force or --skip-wallet-proof flags
CLI documentation (cpm-developer.md or equivalent)
upgrade test-discovery-v1-wallet-scans-to-cpm.sh smoke
CLI tests where applicable
```

Acceptance criteria:

```text
CLI follows the same challenge flow as the UI and direct API.
CLI uses frozen V1 paths only.
CLI persist fails without wallet_control_proof_id.
CLI persist fails against POST /api/cpm/v1/policies for EOA without proof.
CLI docs explain EIP-191 signing, 10-minute TTL and mandatory wallet proof.
Smoke script test-discovery-v1-wallet-scans-to-cpm.sh passes with challenge + persist flow.
User must re-run challenge flow if persist fails after proof consume.
```

Expected commit title:

```text
feat(cli): support EOA wallet challenge flow for CP persistence
```

---

## 27. PR8 — Documentation and end-to-end validation

Repositories:

```text
cafe-crypto-policy-mgt
cafe-deploy
cafe-frontend
cafe-discovery (cross-links only, if needed)
```

Depends on: **PR6** and **PR7** (clients aligned with frozen V1 contract).

Goal:

- Verify and finalize README / WORKPLAN_API cross-links after implementation (PR1 doc baseline is already in place).
- Document the full frozen V1 workflow end-to-end (§33–§43) in user-facing and developer docs.
- Validate non-regression: scan, explore and draft remain open; persist requires proof.
- Validate that no client or smoke script can bypass wallet control proof.

Deliverables:

```text
Confirm README and WORKPLAN_API remain aligned with implemented V1 contract and runtime behavior
cafe-frontend/docs/cpm-developer.md update (V1 paths, EIP-191, re-sign after consumed proof)
cafe-deploy smoke documentation update
manual E2E test scenario (scan -> explore -> draft -> challenge -> verify -> persist)
troubleshooting: WALLET_CONTROL_PROOF_REQUIRED, 503 store unavailable, expired/consumed proof, re-sign
confirm test-discovery-v1-wallet-scans-to-cpm.sh is the canonical compliant smoke
non-regression checklist for S1–S3 baseline stories
```

Acceptance criteria:

```text
Docs explain scan vs explore vs draft vs persist with frozen V1 routes.
Docs document POST /api/cpm/v1/drafts/{draft_id}/persist as the only EOA persist route.
Docs document that POST /api/cpm/v1/policies must not persist EOA CP without proof.
Docs explain EIP-191 / personal_sign, 10-minute TTL, single-use proof and consume-before-create ordering.
Docs explain UI, CLI and direct API all require the same challenge flow.
Docs explain EOA-only first release and list unsupported wallet types.
Docs explain 503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE and re-sign after consumed-proof CP failure.
E2E smoke passes: scan, explore, draft without proof; persist only with valid wallet_control_proof_id.
All persistence paths (UI, CLI, smoke scripts, direct API) require wallet control proof.
```

Expected commit title:

```text
docs: document EOA CP persistence workflow
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

The following decisions are frozen for **CP-PERSIST V1**. **CP-PERSIST V1 is signed off independently through this document** — it does not require global acceptance of [`workplans/WORKPLAN_API.md`](../workplans/WORKPLAN_API.md). Implementation PRs must not reopen frozen decisions without an explicit spec revision.

## 33. Persistence endpoint

```text
POST /api/cpm/v1/drafts/{draft_id}/persist
```

Request body: `{ "wallet_control_proof_id": "uuid" }` only.

Rationale: expresses the domain transition draft → persisted policy; authorization is easier to document and test than overloading `POST /policies`.

## 34. Legacy `POST /api/cpm/v1/policies`

`POST /api/cpm/v1/policies` is **not** the CP persistence endpoint for this workflow.

For EOA wallet flows, it must **not** create a persisted CP without `wallet_control_proof_id` and must return `WALLET_CONTROL_PROOF_REQUIRED` (or an equivalent blocking error).

## 35. Challenge API paths

```text
POST /api/cpm/v1/wallet-challenges
POST /api/cpm/v1/wallet-challenges/verify
```

Legacy frontend paths such as `/wallet-challenge/start` and `/wallet-challenge/verify` are not part of the V1 contract.

## 36. Challenge message and signature format

V1 uses **EIP-191 / `personal_sign`** on a deterministic human-readable message (see §12).

SIWE / EIP-4361 remains out of scope for V1.

## 37. Proof reuse policy

`wallet_control_proof_id` is **single-use**.

A consumed proof cannot be reused for another draft, wallet, scan, chain, user or tenant.

## 38. Ephemeral storage model

```text
CPM owns the wallet challenge/proof ephemeral store.
Redis is the recommended V1 adapter (implementation detail, not a Discovery dependency).
ChallengeStore and ProofStore abstractions; handlers must not use Redis directly.
No durable database tables for challenges or proofs
No raw signature storage in ephemeral store or durable DB for V1
Mandatory key namespace: cpm:wallet_challenge:* and cpm:wallet_control_proof:*
```

The durable database stores only the persisted CP and minimal audit metadata.

CPM must not depend on Discovery cache, Discovery database, Discovery internal structs or Discovery Redis key semantics.

## 39. TTL

```text
10 minutes
```

Applies to both wallet challenges and wallet control proofs.

## 40. Ephemeral store unavailable

CPM must **fail closed** when the CPM ephemeral store is unavailable for challenge or proof operations.

```text
HTTP 503
error: WALLET_CONTROL_PROOF_STORE_UNAVAILABLE
```

Applies to challenge create, challenge verify and persist. No fallback may allow EOA CP persistence without wallet control proof.

## 41. Persist ordering: consume proof before CP creation

```text
1. Validate draft, proof bindings and EOA scope.
2. Atomically consume or reserve the proof in the CPM ephemeral store.
3. Create the persisted CP from the draft.
4. If step 3 fails, the proof remains consumed; the user must create a new challenge and sign again.
```

Do not create the CP first and consume the proof afterward. Re-signing off-chain is acceptable; replay or double persistence is not.

## 42. CPM_REDIS_URL

```text
CPM_REDIS_URL=redis://redis:6379/1        # local dev — shared internal instance, logical DB /1
CPM_REDIS_URL=redis://redis-cpm:6379/0   # production target — dedicated CPM instance
```

Use the `redis://` scheme. Redis must remain internal to the Docker network (no host port publish for this use case). Changing the URL must be a deployment-only migration path.

## 43. Long-term architecture

```text
CAFE may later extract persistence and auth management into dedicated services.
Discovery is expected to become less stateful and should eventually avoid direct Redis/Postgres ownership.
This does not change CP-PERSIST V1: the wallet challenge/proof store belongs to CPM and is accessed through CPM_REDIS_URL and store abstractions.
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
An EOA Crypto Policy cannot be persisted without wallet control proof.
The proof challenge is mandatory for UI, CLI and direct API usage.
The backend CPM enforces the rule.
```

First implementation target:

```text
EOA only.
POST /api/cpm/v1/wallet-challenges and /wallet-challenges/verify.
POST /api/cpm/v1/drafts/{draft_id}/persist.
EIP-191 / personal_sign challenge for V1.
CPM-owned ephemeral store (Redis adapter via CPM_REDIS_URL; mandatory cpm:* namespace).
10-minute TTL, single-use wallet_control_proof_id.
ChallengeStore and ProofStore abstractions; handlers must not use Redis directly.
Atomic proof consume before CP creation; re-sign if CP creation fails.
503 WALLET_CONTROL_PROOF_STORE_UNAVAILABLE when ephemeral store is down; no fallback persist.
POST /api/cpm/v1/policies must not persist EOA CP without proof.
Other wallet types remain TODO.
```

