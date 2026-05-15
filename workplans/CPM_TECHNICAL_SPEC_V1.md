# CAFE CPM Frontend - Technical Specification v1

CPM means **Crypto Policy Manager**.  
This document defines the frontend/backend contracts and technical lifecycle.

## 1. Architecture principles

- Frontend is a rendering and editing client.
- CPM backend is authoritative for path compatibility and policy validity.
- Frontend local validation is UX-only and non-normative.
- Graph and node states are driven by CPM responses, not hardcoded rules.

## 2. Frontend logical components

- `ScanSelector`
- `WalletChallengeGate`
- `CryptoPolicyTemplateSelector`
- `PolicyGraph`
- `PolicyNodeParameterEditor`
- `PolicySummaryPanel`
- `DraftActions`

Recommended state domains:
- `scanState`
- `selectionState`
- `graphViewState`
- `draftState`
- `validationState`
- `walletChallengeState`

## 3. Contracted data models

```ts
type CandidateStatus = "compatible" | "partial" | "incompatible";
type NodeUiState = "active" | "inactive" | "locked" | "invalid";
type EdgeUiState = "active" | "inactive" | "locked" | "invalid";
type WalletType = "EOA" | "SMART_ACCOUNT" | "UNKNOWN";
type DraftStatus = "local_only" | "server_draft" | "validation_failed" | "validated" | "persisted";
type CurrentPqPosture = "unknown" | "classical" | "hybrid" | "pq_ready" | "full_pq";

interface PolicyReason {
  code: string;
  severity: "info" | "warning" | "blocking";
  message: string;
  source?: "discovery" | "cpm" | "template" | "chain" | "user";
  evidenceRef?: string;
  scope?: {
    candidateId?: string;
    nodeId?: string;
    edgeId?: string;
    parameterKey?: string;
  };
}

interface PolicySelectionRequest {
  scanId: string;
  walletObservationId?: string;
  scanResultVersion?: string;
  chainIds?: number[];
  currentPqPosture?: CurrentPqPosture;
  walletAddress?: string;
  walletType?: WalletType;
  policyTemplateId?: string;
  context?: Record<string, unknown>;
}

interface PolicySelectionResponse {
  requestId: string;
  generatedAt: string;
  policyTemplateId?: string;
  policyTemplateVersion?: string;
  candidates: PolicyPathCandidate[];
  nodeDefinitions: PolicyNodeDefinition[];
  graphEdges: PolicyGraphEdge[];
}

interface PolicyPathCandidate {
  candidateId: string;
  status: CandidateStatus;
  score?: number;
  reasons: PolicyReason[];
  nodeInstances: PolicyNodeInstance[];
  edgeIds: string[];
}

interface PolicyNodeDefinition {
  nodeType: string;
  label: string;
  description?: string;
  parameterDefinitions: PolicyParameterDefinition[];
}

interface PolicyNodeInstance {
  nodeId: string;
  nodeType: string;
  uiState: NodeUiState;
  reasons: PolicyReason[];
  parameterValues: PolicyParameterValue[];
}

interface PolicyGraphEdge {
  edgeId: string;
  sourceNodeId: string;
  targetNodeId: string;
  uiState: EdgeUiState;
  reasons: PolicyReason[];
}

interface PolicyParameterDefinition {
  key: string;
  label: string;
  valueType: "enum" | "number" | "text" | "bool" | "json";
  required: boolean;
  constraints?: Record<string, unknown>;
}

/** Leaf JSON literals; may appear inside nested object/array structure only. */
type JsonPrimitive = string | number | boolean | null;

/** Index-signature form expresses recursive JSON maps; implementations may use an equivalent interface. */
interface JsonObject {
  [key: string]: JsonValue;
}

type JsonArray = JsonValue[];

/** Full JSON tree (RFC 8259-style literals) for structured configuration. */
type JsonValue = JsonPrimitive | JsonArray | JsonObject;

/**
 * Stored payload for `valueType: "json"` only: root must be a JSON object or JSON array.
 * Root-level JSON primitives (a lone string, number, boolean, or JSON `null`) are not valid configured values;
 * clients use `valueType: "text" | "number" | "bool"` for those. Nested structure may still contain primitives.
 */
type JsonParameterValue = JsonObject | JsonArray;

interface PolicyParameterValue {
  key: string;
  /** Per `PolicyParameterDefinition.valueType`: string (enum/text), number, boolean, or `JsonParameterValue` for json; `null` means unset, not a configured JSON null. */
  value: string | number | boolean | JsonParameterValue | null;
}

interface PolicyDraft {
  draftId?: string;
  draftVersion: string;
  draftStatus: DraftStatus;
  scanId: string;
  selectionResponseId?: string;
  policyTemplateId?: string;
  policyTemplateVersion?: string;
  selectedCandidateId?: string;
  selectedNodeIds: string[];
  selectedEdgeIds: string[];
  parameterValues: Record<string, PolicyParameterValue[]>;
  nodeInstances?: PolicyNodeInstance[]; // optional UX snapshot
  graphEdges?: PolicyGraphEdge[]; // optional UX snapshot
  localValidation?: {
    status: "unknown" | "pass" | "fail";
    issues: PolicyReason[];
  };
  challengeState?: "unverified" | "verified";
  normalizedPolicy?: CryptoPolicyPayload;
  updatedAt: string;
}

interface PolicyValidationRequest {
  draft: PolicyDraft;
}

interface PolicyValidationResponse {
  valid: boolean;
  issues: PolicyReason[];
  normalizedPolicy?: CryptoPolicyPayload;
  normalizedDraft?: PolicyDraft;
}

interface CryptoPolicyPayload {
  schemaVersion: string;
  policyKind: string;
  targetWalletPosture: string;
  selectedPath: string[]; // ordered list of node IDs
  selectedEdges: string[];
  nodeParameters: Record<string, PolicyParameterValue[]>;
  chainIds: number[];
  remediationHints?: string[]; // provisional and non-normative for v1
}

interface PersistedCryptoPolicy {
  policyId: string;
  policyVersion: string;
  basedOnDraftId: string;
  scanId: string;
  policyTemplateId?: string;
  policyTemplateVersion?: string;
  normalizedPolicy?: CryptoPolicyPayload;
  normalizedPolicyRef?: string;
  createdAt: string;
  status: "persisted" | "superseded";
}
```

Notes:
- `scanId` can be sufficient in `PolicySelectionRequest` when CPM can resolve Discovery observation server-side.
- Discovery context fields (`walletObservationId`, `scanResultVersion`, `chainIds`, `currentPqPosture`) are optional enrichments.
- In `PolicyDraft`, normative selection is represented by `selectionResponseId`, `selectedCandidateId`, `selectedNodeIds`, `selectedEdgeIds`, and `parameterValues`.
- `nodeInstances` and `graphEdges` in draft are optional UX snapshots for rendering convenience.
- **`valueType: "json"`:** `PolicyParameterValue.value` must be a `JsonParameterValue` when set (object or array root). Root JSON primitives and JSON `null` as the sole value are invalid for this type; local UX validation may reject them; CPM remains authoritative on validation. **`null` on `valueType: "json"`** still means *unset* / missing value, not “the user chose JSON null.” Primitives belong in `enum` / `text` / `number` / `bool` as appropriate.

## 4. End-to-end technical lifecycle

1. User selects wallet scan.
2. Frontend sends `PolicySelectionRequest` to CPM.
3. CPM returns `PolicySelectionResponse` with compatible/partial/incompatible path candidates.
4. Frontend renders graph from response only, including backend-provided `graphEdges`.
5. User selects a path candidate and edits parameter values.
6. Frontend optionally performs local UX validation.
7. Frontend sends `PolicyValidationRequest` to CPM.
8. CPM returns `PolicyValidationResponse` (authoritative).
9. Frontend may save backend draft.
10. Frontend may persist validated policy as `PersistedCryptoPolicy`.
11. Remediation is out of scope for v1.

## 5. Wallet challenge technical behavior

- For `EOA` scans, ownership verification is required before:
  - persisting actionable policy;
  - triggering remediation-related operations (future scope).
- Frontend can still allow exploration + local draft editing in `unverified` mode.
- UI must expose `unverified` vs `verified` clearly.

Challenge endpoints (provisional):
- `POST /api/cpm/wallet-challenge/start`
  - input: `scanId`, `walletAddress`, `provider`
  - output: `challengeId`, `messageToSign`, `expiresAt`
- `POST /api/cpm/wallet-challenge/verify`
  - input: `challengeId`, `walletAddress`, `signature`, `provider`
  - output: `status`, `verifiedAt`, `expiresAt`

## 6. API surface (provisional)

- `POST /api/cpm/policy-selection` -> `PolicySelectionResponse`
- `POST /api/cpm/policy-validation` -> `PolicyValidationResponse`
- `POST /api/cpm/drafts` -> persisted draft metadata
- `POST /api/cpm/policies` -> `PersistedCryptoPolicy`
- `GET /api/cpm/policy-templates` -> list of versioned crypto policy templates
- `GET /api/cpm/drafts/:draftId` -> `PolicyDraft`
- `GET /api/cpm/policies/:policyId` -> `PersistedCryptoPolicy`

### 6.1 Addendum - authentication and authorization requirements

CPM API endpoints are **not anonymous**.  
All `\/api\/cpm\/*` operations must require an authenticated user context.

Requirements:
- Frontend sends `Authorization: Bearer <jwt>` on every CPM API call.
- Backend must reject missing/invalid tokens with `401 Unauthorized`.
- Backend must enforce per-user/per-tenant authorization with `403 Forbidden` when authenticated users are not allowed to access a resource.
- `scanId` usage in selection/validation/draft/persist flows is authorization-scoped: a user can only reference scans they are allowed to access.
- Draft and persisted policy records are owned/scope-bound and must be readable/writable only by authorized principals.

Rationale:
- Persisted crypto policies and server drafts are user-scoped artifacts and cannot be safely managed through anonymous APIs.
- Scan access already requires authentication in Discovery; CPM must keep the same security boundary to avoid cross-user access and inconsistent identity context.

## 7. Local draft persistence

- Local drafts are non-normative UX artifacts.
- Local storage key should be versioned (example: `cpm:policy-drafts:v1`).
- Local draft structure includes:
  - `schemaVersion`
  - `draftVersion`
  - `policyTemplateVersion`
- Local draft includes explicit `draftStatus`.
- On schema/version mismatch, frontend must warn and prevent silent restore.

## 8. Validation strategy

- Local validation:
  - immediate UX feedback;
  - never treated as final truth.
- Backend validation:
  - required before persistence;
  - can reject drafts that look locally valid.

## 9. Error model requirements

- Locked/invalid nodes must include backend-provided reasons.
- Graph edges must include backend-provided `uiState` and reasons.
- Validation rejection must include node/parameter scoped issues when possible.
- Challenge failures must distinguish:
  - user rejected signature;
  - challenge expired;
  - provider unavailable;
  - wallet address mismatch.

## 10. Versioning and policy persistence

- Policy templates are versioned.
- Drafts are versioned.
- Persisted policies are versioned and either immutable or revisioned.
- Normalized policy payload must include schema version.
- Frontend must always display active versions used for selection/validation/persistence.

## 11. Acceptance criteria (technical)

- Graph rendering is fully driven by `PolicySelectionResponse`.
- Invalid/locked nodes and edges always expose reasons.
- Policy templates and drafts are versioned.
- Backend validation can reject a locally valid-looking draft.
- Local drafts are non-normative.
- Persisted policies are versioned and immutable or revisioned.
- Remediation remains out of scope for v1.

