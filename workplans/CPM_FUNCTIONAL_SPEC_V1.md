# CAFE CPM Frontend - Functional Specification v1

CPM means **Crypto Policy Manager**.  
This document defines product behavior for the CPM page in frontend.

## 1. Purpose

The CPM page lets a user build a crypto policy draft from a wallet scan, preview it, validate it through CPM backend, and persist it when appropriate.

Core capabilities:
- select a wallet scan;
- request and display policy path candidates from CPM;
- render graph topology (nodes + edges) returned by CPM;
- select a path candidate and configure node parameters;
- apply a crypto policy template from the catalog;
- save local draft and reload local draft;
- send draft to CPM for authoritative validation;
- persist validated policy on backend;
- support EOA wallet ownership challenge (multi-provider).

## 2. Scope

### In scope
- end-to-end functional lifecycle from scan selection to policy persistence;
- UI behavior for candidate graph, node states, parameter editing, and reasons;
- draft handling (local + backend);
- EOA challenge rules for actionable actions.

### Out of scope (v1)
- remediation execution;
- frontend framework migration decisions;
- backend implementation internals.

## 3. Functional lifecycle

1. User selects a wallet scan.
2. Frontend sends `PolicySelectionRequest` to CPM.
3. CPM returns compatible, partial, and incompatible path candidates in `PolicySelectionResponse`.
4. Frontend renders graph from CPM response, including explicit edges (no inferred topology).
5. User selects a path candidate and configures node parameters.
6. Frontend may perform local UX validation (non-authoritative).
7. Frontend sends draft to CPM for authoritative validation.
8. Backend draft can be saved.
9. Validated policy can be persisted.
10. Remediation execution remains out of scope for v1.

## 4. Authority model

- Frontend is **not** the authoritative engine for graph compatibility or policy validity.
- CPM backend is the source of truth for:
  - path compatibility;
  - node/parameter constraints;
  - policy validity.
- Frontend can do local checks only for UX guidance.
- Discovery context may be passed in `PolicySelectionRequest` (`walletObservationId`, `scanResultVersion`, `chainIds`, `currentPqPosture`) when available.
- `scanId` may be sufficient if CPM can resolve Discovery observation server-side.

## 5. Wallet challenge behavior

- If wallet type is `EOA`, ownership verification is required **before**:
  - persisting an actionable policy;
  - requesting remediation-related actions (future scope).
- Exploration and local draft editing may still be allowed without challenge, but UI must clearly mark state as `unverified`.
- Challenge must support multiple EVM-compatible providers (example: MetaMask, Rabby, WalletConnect, Coinbase Wallet).

## 6. User interactions

### Graph and nodes
- Graph is rendered from `PolicySelectionResponse`.
- Graph edges are rendered from backend-provided topology, not inferred from node order.
- Left-click selects node/path when allowed by CPM response.
- Context action (right-click or keyboard equivalent) opens parameter editing.
- Invalid and locked nodes must always expose structured reasons from backend response.

### Policy template from catalog
- User can select a crypto policy template from catalog.
- Template application updates selected path + parameter defaults.
- Template and resulting draft must display version metadata.

### Draft actions
- Save local draft.
- Reload local draft.
- Save backend draft.
- Persist validated policy.
- Draft status must be explicit: `local_only`, `server_draft`, `validation_failed`, `validated`, `persisted`.
- Normative draft selection must track: `selectionResponseId`, `selectedCandidateId`, `selectedNodeIds`, `selectedEdgeIds`, and `parameterValues`.
- Graph/node snapshots may be stored for UX convenience but are not normative.

## 7. Error and state handling

Functional errors to surface:
- no scans available;
- backend selection request failure;
- no compatible paths;
- parameter value invalid;
- backend validation rejected;
- local draft unreadable/version mismatch;
- EOA challenge rejected/expired/provider unavailable.
- backend validation rejected despite local UX-valid draft.

UX principles:
- no silent failure;
- action-oriented error messages;
- preserve current edits whenever safe.

## 8. Acceptance criteria

- Graph is rendered from CPM response, not hardcoded frontend rules.
- Invalid and locked nodes always expose structured reasons.
- Policy templates and drafts are versioned.
- Backend validation can reject a locally valid-looking draft.
- Local drafts are non-normative.
- Persisted policies are versioned and immutable or revisioned.
- Remediation hints in normalized payload are provisional and non-normative for v1.
- Remediation remains out of scope for v1.

## 9. Open product decisions

1. Which path selection UX is default when multiple compatible candidates exist?
2. How strict should template auto-application be (strict vs adaptive UX mode)?
3. Draft persistence policy: immutable snapshots or mutable latest draft?
4. EOA challenge session TTL and re-challenge rules?
5. Minimum provider support list for v1?

