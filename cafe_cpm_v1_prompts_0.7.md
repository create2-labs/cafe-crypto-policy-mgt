# CAFE Crypto Policy Management (CPM) v0.7 – Unified Cursor Pack for implementing the CPM service model v0.1 with an explicit neutral `cafe-contracts` repository and explicit user-triggered assessment flow

This file is the single source of truth for the Cursor workstream.

**Pack selection:** use **only this v0.7 document** for current implementation. The file `cafe_cpm_v1_prompts_0.6.md` is **frozen** and not maintained in parallel.

Version note:
- this document is execution pack version `v0.7`
- it guides the implementation of the CPM service model `v0.1`
- event subjects and payload examples therefore still use `v0.1` unless the service contract itself is versioned forward

What changes in v0.7:
- unlike v0.6, CPM is no longer modeled as an automatic consumer of raw Discovery observation events
- Discovery still observes and may still export `discovery.wallet.observed.v0.1`, but that event is informational and must not auto-trigger CPM assessment
- CPM assessment now starts only from an explicit user-triggered command, modeled in this pack as `policy.assessment.requested.v0.1`
- the authenticated backend or gateway may issue that explicit request after a user action; in the current architecture this will typically be the Discovery-side backend until a broader platform split exists
- `cafe-contracts` remains the neutral repository for wire contracts
- `cafe-cpm` remains the owner of policy semantics, validation, compatibility, ranking, and assessment logic
- PR0, PR1, PR2, and PR3 are already marked done in this document, so this version adds an explicit correction/alignment prompt for the work already completed

Current architectural decision:
- `cafe-contracts` is a real repository in this workstream
- `cafe-cpm` is a real repository in this workstream
- `cafe-discovery` remains the current monolithic backend for now
- `cafe-discovery` still owns its current persistence for scan results, users, plans, and related existing backend concerns
- scanners remain in their own repositories and are not part of this CPM workstream
- CPM must not depend on Discovery's database
- Discovery, CPM, and later Remediation must communicate through versioned NATS events and stable APIs
- extracting Discovery persistence is a later workstream and remains out of scope here
- Discovery-side changes are allowed only when they improve the normalized wallet observation exported to CPM
- `discovery.wallet.observed.v0.1` may be published automatically by Discovery, but it is not the trigger for CPM
- CPM must start only from an explicit assessment request, represented in this pack by `policy.assessment.requested.v0.1`
- the explicit request may be issued by the authenticated backend or gateway after a user action; for now that initiating backend will most likely still live inside `cafe-discovery`

Frontend will be defined and implemented later.

---

## 1. Master prompt for Cursor

Use this prompt at the beginning of the whole workstream.

```text
You are working on CAFE.

Goal:
Implement Crypto Policy Management (CPM) v0.1 as its own service and its own repository, `cafe-cpm`, with a neutral shared contracts repository, `cafe-contracts`, using **`cafe_cpm_v1_prompts_0.7.md`** (this file) as the authoritative implementation guide for integration semantics; PR-level prompts below remain aligned with the numbered PR list.

Current architecture assumptions:
- `cafe-contracts` is the neutral repository for shared wire contracts.
- `cafe-cpm` is a separate repository and a separate service.
- `cafe-discovery` remains the current monolithic backend for now.
- `cafe-discovery` still owns its existing persistence layer.
- scanners already live in separate repositories and are out of scope here.
- CPM must not read from or write to Discovery's database.
- Discovery and CPM must not import each other directly.
- Discovery and CPM may both depend on `cafe-contracts`.
- CPM must integrate with Discovery and later Remediation through versioned NATS events and stable APIs only.
- Later, Discovery persistence may be extracted; nothing in this workstream should block that future change.

Targeted Discovery improvement rule:
- This workstream may improve `cafe-discovery` only where doing so clearly improves the normalized wallet observation exported to CPM.
- Allowed Discovery changes are limited to:
  - cleaning up `ScanResultEntity` or its owned equivalent
  - clarifying observation vocabulary
  - deriving observation-facing posture fields
  - adding/exporting the normalized `discovery.wallet.observed.v0.1` contract
  - adding minimal NATS publication or compatibility shims required for the CPM flow
- This workstream must not become a broad Discovery architecture refactor.
- This workstream must not extract Discovery persistence.
- CPM must still consume only the exported contract, not Discovery internal structs or Discovery persistence.

Authoritative service model:
- Do not introduce a DSL.
- Do not introduce a generic rule engine.
- Keep all logic explicit, typed, deterministic, and easy to test.
- Discovery owns the internal observation model.
- `cafe-contracts` owns the shared wire-level contract packaging.
- `cafe-cpm` owns the semantics, interpretation, ranking, validation, assessment, and policy logic.
- A CPx is represented as an ordered path of nodes.
- `CryptoPolicyInstance` must not contain configurable edge parameters.
- Transition semantics must still exist, but they are enforced by compatibility rules defined in the graph catalog and/or template.
- Keep canonical policy content separate from runtime results:
  - `CryptoPolicyInstance`
  - `CryptoPolicyValidationResult`
  - `CryptoPolicyAssessmentResult`

Service boundaries:
- Discovery observes and normalizes wallet/account state.
- Discovery may export normalized observations using shared contracts from `cafe-contracts`.
- Raw Discovery observation export does not auto-trigger CPM.
- CPM validates policy documents, activates policy instances, evaluates observed subjects only when explicitly requested, and emits policy outcomes.
- Remediation consumes policy outcomes and turns them into remediation plans and execution steps.
- CPM must not scan wallets itself.
- CPM must not execute remediation itself.
- CPM owns only CPM data: catalog, templates, instances, validation results, assessments, audit metadata, and CPM-specific integration state.
- The authenticated backend or gateway may request a CPM assessment explicitly after a user action by publishing `policy.assessment.requested.v0.1` or by calling a CPM API that maps to the same semantics.

Implementation discipline:
- Implement this as a sequence of small PRs.
- Keep each PR narrowly scoped.
- Add focused tests for each PR.
- Avoid unrelated refactors.
- Prefer explicit typed fields over flexible abstractions.
- Backward compatibility is nice to keep when cheap, but it is not a blocking requirement in this workstream.
- Discovery can be improved when the PR explicitly calls for it, but only at the observation/export boundary.

Important integration rules:
- If a shared contract is needed between Discovery and CPM, package it in `cafe-contracts`.
- Do not couple CPM to Discovery internals.
- Do not couple Discovery to CPM code.
- Prefer versioned NATS subjects and typed payloads.
- Use the canonical multi-chain compatibility rules and deterministic ranking contract defined later in this document.
- Build event handling with explicit idempotence in mind from the start.
- Do not subscribe CPM directly to raw Discovery observation events as an implicit trigger.
- The canonical asynchronous trigger for CPM in this pack is `policy.assessment.requested.v0.1`.

For each PR:
1. Inspect the relevant code paths first.
2. Identify the smallest integration points.
3. Implement only the requested scope.
4. Add focused tests.
5. Summarize:
   - what changed
   - what tests were added
   - what remains for the next PR
```

---

## 2. Global instructions for every PR

Use these instructions at the beginning of every PR.

```text
Important constraints:
- Keep the scope limited to this PR only.
- Do not introduce a DSL.
- Do not introduce a generic rule engine.
- Prefer typed structs, enums/constants, and explicit helper functions.
- Avoid unrelated refactors.
- Add focused tests.
- Keep JSON shapes stable unless this PR explicitly changes them.
- Do not add configurable edge parameters to CryptoPolicyInstance.
- Transition semantics must be enforced through catalog/template compatibility rules, not through instance-level edge objects.
- Respect the service boundaries:
  - Discovery observes
  - `cafe-contracts` packages shared wire contracts
  - CPM validates/selects/assesses
  - Remediation plans/executes
- Do not add direct database coupling between CPM and Discovery.
- Discovery may be touched only when the PR explicitly says so, and only for narrow observation/export-boundary work.
- Avoid mixing remediation execution fields into canonical policy models unless the document explicitly calls for a policy-facing reference or requirement.
- Keep NATS contracts typed and versioned from the start.
- Follow the vocabulary ownership, idempotence, versioning, ranking, remediation-trigger, and multi-chain rules defined in this document.

Before coding:
1. Inspect the existing code paths affected by this PR.
2. Identify the minimal integration points.
3. Implement only the scope requested here.
4. Add tests.
5. At the end, summarize:
   - what changed
   - what tests were added
   - what remains for the next PR
```

---

## 3. Additional contract-hardening rules

These rules are normative for this workstream and must be reflected in implementation choices.

### Shared vocabulary and contract ownership

- `cafe-cpm` owns the semantics of the exported CPM-facing vocabulary:
  - policy selection
  - policy validation
  - policy assessment
  - ranking and compatibility semantics
  - the meaning of the normalized Discovery -> CPM observation contract
- `cafe-contracts` owns the technical packaging of shared wire contracts:
  - envelope and payload structs
  - exported enum/string constants used on the wire
  - minimal contract validation only
  - canonical JSON fixtures and schemas where applicable
- Discovery may keep internal legacy values, but anything exported to CPM must be mapped to the shared wire vocabulary at the integration boundary.
- Discovery and CPM must not invent new exported enum values independently. Shared wire-level additions must land in `cafe-contracts` and remain aligned with CPM semantics.

### Discovery-owned observation model versus exported contract

- Discovery owns `ScanResultEntity` and any Discovery-internal observation model evolution.
- CPM must not import Discovery internal Go structs as its canonical input model.
- The Discovery -> CPM boundary must be an explicit exported contract, initially `discovery.wallet.observed.v0.1`, packaged in `cafe-contracts`.
- This workstream may improve Discovery's internal observation model if that makes the exported contract clearer, more deterministic, or easier to validate.
- `cafe-cpm` may re-export or alias `cafe-contracts` wire types for local ergonomics, but must not fork them silently.


### Explicit assessment trigger rule

- `discovery.wallet.observed.v0.1` is an observation event, not a command.
- Discovery may publish `discovery.wallet.observed.v0.1` automatically after scans or persistence updates, but CPM must not treat that publication as a reason to start assessment automatically.
- CPM must start assessment only from an explicit user-triggered request.
- In the canonical asynchronous path of this pack, that explicit command is `policy.assessment.requested.v0.1`.
- The command issuer is the authenticated backend or gateway acting on behalf of the user. In the current architecture this will usually still be the Discovery-side backend until a later platform split exists.
- Because CPM must not read Discovery's database, `policy.assessment.requested.v0.1` must carry either:
  - an embedded normalized observation snapshot using the shared wire contract, or
  - another fully self-sufficient payload that lets CPM assess without fetching Discovery persistence.
- CPM may later expose a synchronous API that follows the same semantics, but this pack treats the explicit command event as the canonical asynchronous trigger.

### Event idempotence and replay handling

- Every inbound and outbound event must carry a stable `event_id`.
- CPM consumers must be idempotent.
- CPM must persist or otherwise track processed inbound `event_id` values for duplicate suppression.
- Assessment identity should be deterministic. The preferred first-version strategy is to derive it from a documented tuple such as:
  - `subject.id`
  - `policy_instance_id`
  - `causation_id` or originating Discovery event ID
- Replays must not create duplicate side effects such as duplicated assessments or duplicated remediation requests.
- Global event ordering is not guaranteed. Handlers must rely on IDs, references, and timestamps rather than assuming delivery order.

### Contract versioning policy

- NATS subjects and explicit event contracts must be versioned from the start.
- Additive backward-compatible payload changes may stay within the same contract version.
- Removing, renaming, or changing the meaning of an existing field requires a new event version.
- Consumers must ignore unknown fields.
- Canonical policy documents should evolve with the same discipline: semantic breaking changes require a new versioned contract or migration step.

Versioning matrix:

| Change type | Same version allowed | New version required | Notes |
| --- | --- | --- | --- |
| Add optional field | yes | no | Consumers must ignore unknown fields |
| Add new event subject | yes | no | Keep existing subjects unchanged |
| Remove field | no | yes | Breaking change |
| Rename field | no | yes | Breaking change |
| Change field meaning | no | yes | Semantic breaking change |
| Tighten enum values incompatibly | no | yes | Breaking change |
| Deprecate field but keep behavior | yes | no | Document deprecation and sunset plan |
| Sunset old version | no | yes | Requires explicit rollout plan and coexistence period |

Rollout expectation for future versions:
- when introducing `v0.2` of a contract, support `v0.1` and `v0.2` in parallel for a documented coexistence window whenever feasible
- do not switch producers and consumers in one hidden step
- prefer producer-first additive rollout, then consumer adoption, then deprecation, then sunset

### First-version deterministic ranking contract

PR13 must implement the following first-version ranking order unless this document is explicitly updated later:

1. Exclude incompatible policies first.
2. Prefer better alignment with the requested target posture.
3. Then prefer higher maturity.
4. Then prefer better chain coverage.
5. Then prefer better satisfaction of address continuity requirements.
6. Then prefer avoiding new wallet creation when the request allows a choice.
7. Final tie-break must be lexical `policy_id` (or another explicitly documented stable ID if `policy_id` is absent).

### Minimum payload required for `policy.remediation.requested`

The first version of the remediation-trigger event must include at least:

- `assessment_id`
- `policy_id`
- `policy_instance_id`
- `template_id` when applicable
- `subject`
- `selected_path.node_ids`
- `target_posture`
- `address_continuity_required`
- `key_rotation_required`
- `recovery_required`
- `provider_constraints` when applicable
- `correlation_id`
- `causation_id`

### Canonical multi-chain rules

The first version must apply these rules consistently:

- a policy is deployable only if all required `target_chain_ids` are included in the policy `chain_ids`
- `chain_ids: []` means a known route that is not currently deployable
- if `require_multichain = true`, at least two deployable target chains must match
- if the target posture is satisfied only on a strict subset of requested target chains, compatibility is false unless a later contract explicitly introduces partial coverage
- chain support is checked during compatibility filtering, not deferred to ranking

### Internal model to event payload mapping

- NATS payloads are explicit projections of internal models, not independent parallel business objects.
- Event producers must not recompute policy semantics differently from the internal services that already produced the source model.
- PR14 must define a documented mapping matrix between internal models and exported event payloads.
- The first-version mapping must cover at least:
  - Discovery normalized wallet observation export -> `discovery.wallet.observed.v0.1`
  - explicit user-triggered assessment request -> `policy.assessment.requested.v0.1`
  - `CryptoPolicyValidationResult` -> `policy.validation.completed.v0.1`
  - `CryptoPolicyAssessmentResult` -> `policy.assessment.completed.v0.1`
  - `PolicyDecision` plus the selected template/instance references -> `policy.remediation.requested.v0.1`
- The event layer must only project and transport data. It must not add extra ranking, compatibility, or remediation-selection logic.
- PR14 and the later NATS wiring PRs must include tests that verify the documented model-to-payload mapping.

### Compatibility versus deployability status

- `chain_ids: []` must not be collapsed into a plain incompatible result without explanation.
- The first-version compatibility output must distinguish at least these states:
  - `compatible_and_deployable`
  - `compatible_but_not_deployable`
  - `incompatible`
- `compatible_but_not_deployable` means the route is structurally known and may match conceptually, but it cannot currently be executed on the requested chains.
- This distinction must be preserved in internal compatibility results and in read APIs, so downstream consumers and future UIs do not confuse a known route with an invalid route.

### Policy identifier normalization for deterministic tie-breaks

- `policy_id` used for deterministic ranking tie-breaks must be stable, ASCII, uppercase, and case-normalized before comparison.
- The recommended first-version format is `CP` followed by zero-padded digits, for example `CP001`, `CP002`, `CP101`.
- Lexical comparison for the final tie-break must operate on the normalized stored value, not on locale-specific or case-insensitive transformations at runtime.
- If a ranked candidate does not natively expose a `policy_id`, CPM must derive a stable comparable identifier and document that derivation explicitly in code and tests.

---

## 4. Normative Rules Index

1. `cafe-contracts` is a real repository in this workstream.
2. `cafe-cpm` is a real repository in this workstream and must not depend on Discovery's database.
3. Discovery and CPM must not import each other directly; both may depend on `cafe-contracts`.
4. Discovery may be improved in this workstream only for narrow observation/export-boundary work.
5. Discovery remains monolithic in this workstream; persistence extraction and broad backend refactor stay out of scope.
6. CPM logic must remain explicit, typed, deterministic, and free of DSLs or generic rule engines.
7. `CryptoPolicyInstance` must not contain configurable edge parameters.
8. Transition semantics must be enforced through catalog/template compatibility rules.
9. Canonical policy content must stay separate from runtime results:
   - `CryptoPolicyInstance`
   - `CryptoPolicyValidationResult`
   - `CryptoPolicyAssessmentResult`
10. `cafe-contracts` packages shared wire contracts; `cafe-cpm` owns their semantics.
11. `discovery.wallet.observed.v0.1` is informational and must not auto-trigger CPM assessment.
12. CPM assessment starts only from an explicit request, canonically `policy.assessment.requested.v0.1` in this pack.
13. Explicit assessment requests must be self-sufficient enough for CPM to assess without reading Discovery persistence.
14. NATS contracts must be typed, versioned, replay-safe, and idempotent.
15. Event producers must project internal models; they must not recompute business semantics in the event layer.
16. Compatibility output must distinguish:
    - `compatible_and_deployable`
    - `compatible_but_not_deployable`
    - `incompatible`
17. Ranking must follow the deterministic order defined in this document, with final lexical tie-break on normalized `policy_id`.
18. `policy.remediation.requested` must include the minimum payload defined in this document.
19. Multi-chain support is evaluated during compatibility filtering, not deferred to ranking.
20. PR15 and PR16 must include replay, duplicate-suppression, and retry-behavior tests.
21. Read APIs must preserve the distinction between invalid routes and known-but-not-deployable routes.
22. Discovery-side PRs in this pack must remain narrow and must not refactor Discovery persistence or broader backend ownership.

When in doubt, update this index first, then update the detailed PR sections that depend on it.

---

## 5. Canonical Event Payload Examples

These examples are normative first-version examples for reducing interpretation variance. They are the canonical shapes that shared contracts and tests must encode.

Identifier note:
- `policy_id` is the stable normalized policy identifier used for deterministic ranking tie-breaks
- `policy_instance_id` is the identifier of a concrete `CryptoPolicyInstance` record and is distinct from `policy_id`
- `template_id` is the identifier of the source template when applicable
- `selected_path.node_ids` must contain stable catalog node identifiers (not display labels)

### `discovery.wallet.observed.v0.1`

This event represents a normalized observation. It may be produced automatically by Discovery, but it does not start CPM assessment by itself.

```json
{
  "event_id": "evt_disc_0001",
  "event_type": "discovery.wallet.observed",
  "event_version": "v0.1",
  "occurred_at": "2026-04-17T10:00:00Z",
  "correlation_id": "corr_scan_0001",
  "causation_id": "scan_job_0001",
  "producer": "cafe-discovery",
  "subject": {
    "type": "wallet",
    "id": "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
  },
  "payload": {
    "chain_ids": [1, 8453],
    "account_kind": "eoa",
    "current_algorithm": "secp256k1_ecrecover",
    "current_pq_posture": "classical_only",
    "public_key_exposed": true,
    "is_multichain": true,
    "observed_at": "2026-04-17T09:59:58Z"
  }
}
```

### `policy.assessment.requested.v0.1`

This event is the canonical asynchronous trigger for CPM in this pack. It is issued only after an explicit user action.

```json
{
  "event_id": "evt_pol_req_0001",
  "event_type": "policy.assessment.requested",
  "event_version": "v0.1",
  "occurred_at": "2026-04-17T10:00:02Z",
  "correlation_id": "corr_scan_0001",
  "causation_id": "ui_action_0001",
  "producer": "cafe-discovery",
  "subject": {
    "type": "wallet",
    "id": "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
  },
  "payload": {
    "request_id": "req_0001",
    "observation_snapshot": {
      "chain_ids": [1, 8453],
      "account_kind": "eoa",
      "current_algorithm": "secp256k1_ecrecover",
      "current_pq_posture": "classical_only",
      "public_key_exposed": true,
      "is_multichain": true,
      "observed_at": "2026-04-17T09:59:58Z"
    },
    "selection_request": {
      "target_posture": "hybrid",
      "target_chain_ids": [1, 8453],
      "require_multichain": true,
      "allow_new_wallet": false,
      "address_continuity_required": false,
      "key_rotation_required": true,
      "recovery_required": true
    },
    "requested_by": {
      "type": "user",
      "id": "user:1234"
    }
  }
}
```

### `policy.assessment.completed.v0.1`

```json
{
  "event_id": "evt_pol_assess_0001",
  "event_type": "policy.assessment.completed",
  "event_version": "v0.1",
  "occurred_at": "2026-04-17T10:00:03Z",
  "correlation_id": "corr_scan_0001",
  "causation_id": "evt_pol_req_0001",
  "producer": "cafe-cpm",
  "subject": {
    "type": "wallet",
    "id": "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
  },
  "payload": {
    "assessment_id": "assess_0001",
    "policy_id": "CP001",
    "policy_instance_id": "INST001",
    "template_id": "TPL001",
    "status": "non_compliant",
    "compatibility_status": "compatible_and_deployable",
    "reasons": [
      "Observed target posture is classical_only while requested target posture is hybrid"
    ],
    "warnings": [],
    "evaluated_at": "2026-04-17T10:00:03Z"
  }
}
```

### `policy.remediation.requested.v0.1`

```json
{
  "event_id": "evt_pol_rem_0001",
  "event_type": "policy.remediation.requested",
  "event_version": "v0.1",
  "occurred_at": "2026-04-17T10:00:04Z",
  "correlation_id": "corr_scan_0001",
  "causation_id": "evt_pol_assess_0001",
  "producer": "cafe-cpm",
  "subject": {
    "type": "wallet",
    "id": "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
  },
  "payload": {
    "assessment_id": "assess_0001",
    "policy_id": "CP001",
    "policy_instance_id": "INST001",
    "template_id": "TPL001",
    "selected_path": {
      "node_ids": ["NODE_EOA_ENTRY", "NODE_AA_ERC4337", "NODE_SIG_EIP7932", "NODE_VERIFIER_STANDARD", "NODE_TARGET_HYBRID"]
    },
    "target_posture": "hybrid",
    "address_continuity_required": false,
    "key_rotation_required": true,
    "recovery_required": true,
    "provider_constraints": {
      "allowed_provider_modes": ["third_party", "user_managed"]
    }
  }
}
```

---
## 6. Operational Acceptance Rules for PR15 and PR16

### PR15 minimum operational acceptance

- duplicate inbound `policy.assessment.requested.v0.1` events with the same `event_id` must not create duplicate assessments
- replay of a previously processed assessment request must either:
  - produce no new side effect, or
  - produce the same persisted logical assessment without duplication
- inbound handler tests must explicitly cover:
  - first delivery
  - duplicate delivery
  - replay after process restart or persisted state reload if feasible in the local design
- retry behavior must be documented:
  - when processing fails before the side effect is committed, retry is allowed
  - once the side effect is committed and `event_id` is recorded, retry must be suppressed or become a no-op
- if Discovery continues to publish `discovery.wallet.observed.v0.1`, tests must verify that CPM does not treat those observation events as implicit triggers

### PR16 minimum operational acceptance

- duplicate publication attempts for the same logical assessment/remediation trigger must not create divergent payloads
- event producer tests must explicitly cover:
  - assessment completed publication
  - remediation requested publication
  - duplicate suppression or deterministic republishing behavior
- outbound retry semantics must be documented:
  - payload projection must be deterministic
  - retries must preserve the same logical identifiers and references
  - producers must not recompute business decisions during retry

---
## 6.1 Contract Evolution and Rollout Playbook

1. Add the new typed payloads and subject constants in `cafe-contracts` without removing the old ones.
2. Update consumers first when feasible so they can read both versions.
3. Update producers next to emit the new version, while keeping the old version during the coexistence window if needed.
4. Compare payload projections using canonical fixtures or equivalent contract tests.
5. Announce deprecation of the old version explicitly in code comments, docs, and release notes.
6. Sunset the old version only after the documented coexistence window and consumer readiness check.

Minimum expectation:
- no silent replacement of an old contract version
- no producer-side semantic drift between old and new versions during coexistence
- deprecation and sunset dates must be explicit once production rollout starts

---

## 6.2 GitHub Actions expectations by repository type

### `cafe-cpm`
`cafe-cpm` is a service repository and should mirror `cafe-discovery` operational CI/CD conventions where appropriate.

Minimum workflow set required:
- `ci.yml`
- `conventional-commit.yml`
- `docker-rc.yml`
- `docker-release.yml`
- `release-please.yml`

### `cafe-contracts`
`cafe-contracts` is a contracts/library repository and does not need Docker image workflows.

Minimum workflow set required:
- `ci.yml`
- `conventional-commit.yml`
- release workflow appropriate for tagged contract versions
- optional release-please or equivalent versioning automation

Common requirements:
- permissions must be least-privilege
- workflows using secrets must not execute untrusted fork code paths without explicit protections
- workflow files must be reviewed like production code

---

## 7. Recommended merge order

1. PR0 — Bootstrap `cafe-contracts` repository structure
2. PR1 — Define shared `discovery.wallet.observed.v0.1` contract and exported wire vocabulary in `cafe-contracts`
3. PR2 — Bootstrap `cafe-cpm` repository structure
4. PR3 — Improve Discovery wallet observation model and adapter for CPM export
5. PR4 — Point `cafe-cpm` at `cafe-contracts` wallet observation wire types
6. PR5 — Derive current PQ posture on the Discovery observation/export side
7. PR6 — Add `PolicySelectionRequest`
8. PR7 — Add `PolicyGraphCatalog` core model
9. PR8 — Add `CryptoPolicyTemplate` model and validation
10. PR9 — Add `CryptoPolicyInstance` model
11. PR10 — Add `CryptoPolicyValidationResult` and validator
12. PR11 — Add `CryptoPolicyAssessmentResult` model
13. PR12 — Implement policy compatibility evaluation
14. PR13 — Implement ranking and policy decision output
15. PR14 — Define shared assessment-command and CPM/Remediation NATS event contracts in `cafe-contracts`
16. PR15 — Integrate CPM with explicit assessment requests over NATS
17. PR16 — Add outbound CPM events for Remediation
18. PR17 — Add CPM read APIs
19. Optional PR18 — Seed fixtures and prepare future CRUD

Note:
- PR0, PR1, PR2, and PR3 are already marked done in the tracking table below.
- Because v0.6 modeled CPM too implicitly as a consumer of raw Discovery observation events, this v0.7 document adds a correction/alignment prompt for the already completed work.
## 8. Work tracking table

Status convention:
- `todo` = not started
- `in_progress` = currently being worked on
- `done` = completed and verified
- `n/a` = intentionally not applicable for this PR

| PR | Primary repo | Scope | Implementation | Tests written | Test results | Docs: `cafe-contracts` | Docs: `cafe-cpm` | Docs: `cafe-discovery` | Docs: `cafe-deploy` | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| PR0 | `cafe-contracts` | Bootstrap `cafe-contracts` repository structure | done | done | done | done | n/a | n/a | n/a | Carried over from v0.6; still valid |
| PR1 | `cafe-contracts` | Define shared `discovery.wallet.observed.v0.1` contract and exported wire vocabulary | done | done | done | done | todo | todo | n/a | **Informational observation contract** — not the CPM trigger; see §3 explicit assessment rule |
| PR2 | `cafe-cpm` | Bootstrap `cafe-cpm` repository structure | done | done | done | todo | n/a | n/a | n/a | Carried over from v0.6 |
| PR3 | `cafe-discovery` | Improve Discovery wallet observation model and adapter for CPM export | done | done | done | n/a | n/a | done | n/a | `internal/walletobservation`, YAML `chain_id`, NATS `cafe.discovery.events.wallet.observed.v0_1` (informational); README updated. **Test results:** `go test ./...` green in repo (run locally / CI). **v0.7:** observation is not the CPM assessment trigger |
| PR4 | `cafe-cpm` | Consume shared wallet observation wire types from `cafe-contracts` | done | done | done | n/a | done | n/a | n/a | `internal/domain/walletobserved` now re-exports shared `cafe-contracts/discoverywalletobserved/v01` types/constants/errors via thin aliases; local wire-struct duplication removed. `go test ./...` green. |
| PR5 | `cafe-discovery` | Derive current PQ posture on the Discovery observation/export side | done | done | done | n/a | n/a | done | n/a | `internal/walletobservation/export.go`: deterministic posture derivation from normalized observation (`NISTLevel` mapping) with explicit rules/comments. `internal/walletobservation/export_test.go`: posture coverage for `classical_only`, `hybrid`, `full_pq`, `unknown`. **Validation:** `go test ./internal/walletobservation -count=1`, `golangci-lint run ./...`, `go vet ./...`, `govulncheck ./...` green (local run). |
| PR6 | `cafe-cpm` | Add `PolicySelectionRequest` | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/selection_request.go`: stable typed contract + explicit normalize/validate helpers; canonical multi-chain rule documented (`target_chain_ids` with `require_multichain`). `internal/domain/policy/selection_request_test.go`: JSON round-trip, defaulting, and validation coverage. **Validation:** `go test ./...` green in repo. |
| PR7 | `cafe-cpm` | Add `PolicyGraphCatalog` core model | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/graph_catalog.go`: typed catalog structures (`PolicyGraphCatalog`, `PolicyNodeDefinition`, `PolicyNodeParameterSchema`, `PolicyCompatibilityRule`) + file-backed JSON loader with load-time validation. `internal/domain/policy/testdata/*`: valid and invalid fixtures. `internal/domain/policy/graph_catalog_test.go`: parsing and invalid fixture coverage. **Validation:** `go test ./...` green in repo. |
| PR8 | `cafe-cpm` | Add `CryptoPolicyTemplate` model and validation | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/template.go`: typed `CryptoPolicyTemplate` + template constraints/metadata + file loader and catalog-backed validation (known nodes + allowed transitions). `internal/domain/policy/testdata/crypto_policy_template_*`: valid/invalid fixtures. `internal/domain/policy/template_test.go`: normalization, load, and invalid-case coverage.  |
| PR9 | `cafe-cpm` | Add `CryptoPolicyInstance` model | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/instance.go`: typed `CryptoPolicyInstance` + `PolicyScope` + typed global/per-node parameters (`NodeParameterMap` + `NodeParameterValue`) with explicit normalization and catalog-backed validation (reference rules, transition checks, node schema checks). `internal/domain/policy/testdata/crypto_policy_instance_*`: valid/invalid fixtures. `internal/domain/policy/instance_test.go`: load, serialization behavior, and validation error coverage. |
| PR10 | `cafe-cpm` | Add validation result model and validator | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/validation_result.go`: adds `CryptoPolicyValidationResult`, typed issue model (`ValidationIssue`), and `CryptoPolicyInstanceValidator` that normalizes + validates without mutating input instances. `internal/domain/policy/validation_result_test.go`: validator construction, valid/invalid/nil scenarios, and immutability coverage. **Validation:** `go test ./...`, `golangci-lint run ./...`, `govulncheck ./...` green in repo. |
| PR11 | `cafe-cpm` | Add assessment result model | done | done | done | n/a | done | n/a | n/a | `internal/domain/policy/assessment_result.go`: adds `CryptoPolicyAssessmentResult`, `AssessmentStatus`, `AssessmentFinding`, typed wallet reference, helper constructor, and explicit `Validate()` contract checks. `internal/domain/policy/assessment_result_test.go`: enum validity, constructor defaults, JSON round-trip, and invalid-case coverage. **Validation:** `go test ./...`, `golangci-lint run ./...`, `govulncheck ./...` green in repo. |
| PR12 | `cafe-cpm` | Implement compatibility evaluation | done | done | done | n/a | done | n/a | n/a | `compatibility_result.go`: `PolicyCompatibilityEvaluator` + `PolicyCompatibilityResult`; path from `node_path` or template; multichain/chain scope per v0.7. `go test`, `golangci-lint`, `govulncheck` green. |
| PR13 | `cafe-cpm` | Implement ranking and policy decision output | todo | todo | todo | n/a | todo | n/a | n/a |  |
| PR14 | `cafe-contracts` | Define shared assessment-command and CPM/Remediation NATS event contracts | todo | todo | todo | todo | todo | n/a | todo |  |
| PR15 | `cafe-cpm` + `cafe-discovery` | Integrate CPM with explicit assessment requests over NATS | todo | todo | todo | n/a | todo | todo | n/a | Keep Discovery changes minimal and integration-focused |
| PR16 | `cafe-cpm` | Publish outbound CPM events for Remediation | todo | todo | todo | n/a | todo | n/a | todo |  |
| PR17 | `cafe-cpm` | Add CPM read APIs | todo | todo | todo | n/a | todo | n/a | n/a |  |
| PR18 (optional) | `cafe-cpm` | Seed fixtures and prepare future CRUD | todo | todo | todo | n/a | todo | n/a | n/a |  |

**Remaining work (after PR0–PR12):** PR13 (`cafe-cpm` ranking and policy decision output), then **PR14** (contracts including **`policy.assessment.requested.v0.1`**), then **PR15** (CPM + Discovery integration on **explicit** assessment commands only), PR16–PR18. Narrow **§8.1 alignment patch** is optional if any code comment still implies CPM subscribes to `discovery.wallet.observed` as the main trigger.

---

## 8.1 Correction prompt for already completed PR0–PR3 work

The following alignment prompt exists because PR0, PR1, PR2, and PR3 were completed under the earlier v0.6 assumption that raw Discovery observation events might become the CPM trigger path. In v0.7 that assumption is no longer valid.

Use this prompt before implementing PR4+ if the completed work needs a corrective patch.

```text
Implement a narrow alignment patch only for the already completed PR0–PR3 work.

Goal:
Realign the already completed `cafe-contracts`, `cafe-cpm`, and `cafe-discovery` work with the v0.7 explicit-trigger architecture.

What must remain true:
- keep `cafe-contracts` as the neutral contracts repository
- keep the shared `discovery.wallet.observed.v0.1` contract
- keep Discovery-side export improvements from PR3 where they are useful
- do not introduce Discovery <-> CPM direct code coupling
- do not refactor Discovery persistence broadly

What must change conceptually:
- `discovery.wallet.observed.v0.1` must be treated as an observation/informational event, not as an implicit command that starts CPM
- CPM must not subscribe to raw Discovery observation events as the canonical assessment trigger
- the canonical asynchronous trigger must become `policy.assessment.requested.v0.1`

Required patch work:
1. Audit PR0–PR3 outputs and comments/docs for any wording or code path that implies raw Discovery observation events auto-trigger CPM.
2. Reclassify `discovery.wallet.observed.v0.1` as informational only.
3. If Discovery currently publishes that event automatically after scan completion, keep it only if useful for observability/integration, but make sure later CPM wiring does not treat it as an assessment trigger.
4. Add or update comments/tests so the contract boundary is explicit:
   - observation export exists
   - assessment requires an explicit request
5. Prepare the ground for PR14/PR15 by ensuring the future `policy.assessment.requested.v0.1` command can carry:
   - an observation snapshot using shared wire types
   - a selection request
   - request identity / correlation information
6. Do not implement the full PR14/PR15 flow in this patch unless the current PR explicitly asks for it.

Tests to add or update:
- a test or fixture note proving that `discovery.wallet.observed.v0.1` is valid as an observation event
- no test should assert that raw observation publication itself starts CPM
- if comments or docs currently say that CPM subscribes directly to observation events as its main trigger, correct them

Deliverables:
- narrow alignment patch for already completed work
- updated docs/comments/tests where needed
- no broad redesign
```

### Expected outcome of the correction patch

After this patch:
- PR0, PR1, PR2, and PR3 remain valid and useful
- the architecture is clarified so that explicit user-triggered assessment begins only at PR14/PR15
- no accidental auto-assessment behavior is baked into the already completed work

## PR0 — Bootstrap `cafe-contracts` repository structure

### Suggested PR title
`chore(contracts): bootstrap cafe-contracts repository`

### Goal
Create the neutral shared contracts repository with no business logic.

### Cursor prompt

```text
Implement PR0 only in `cafe-contracts`.

Goal:
Bootstrap `cafe-contracts` as a neutral repository for shared wire contracts.

Requirements:
- Keep this repository free of CPM, Discovery, and Remediation business logic.
- Create only the minimal standalone structure needed for shared contract development.

Expected changes:
- initialize the Go module/repository structure for `cafe-contracts`
- add package layout for versioned wire contracts
- add minimal validation helpers
- add CI and release workflow skeleton appropriate for a library/contracts repo
- add a README describing the boundaries of `cafe-contracts` vs `cafe-cpm` vs `cafe-discovery`

Deliverables:
- standalone repository skeleton
- minimal build/test success
- short architecture README

Out of scope:
- CPM policy logic
- Discovery logic
- NATS wiring
```

### Acceptance criteria
- `cafe-contracts` builds as an independent repository.
- Repository boundaries are documented.
- No service business logic is introduced.

---

## PR1 — Define shared `discovery.wallet.observed.v0.1` contract and exported wire vocabulary

### Suggested PR title
`feat(contracts): add discovery wallet observed v0.1 wire contract`

### Goal
Define the shared Discovery -> CPM wire contract and the exported wire vocabulary needed by both producer and consumer.

### Cursor prompt

```text
Implement PR1 only in `cafe-contracts`.

Goal:
Define the versioned shared `discovery.wallet.observed.v0.1` wire contract and the exported wire vocabulary it uses.

Requirements:
- This is a wire-level contract package, not a policy logic package.
- Keep only:
  - envelope and payload structs
  - exported enum/string constants
  - minimal contract validation
  - canonical fixtures/tests
- Do not add CPM business rules here.
- Do not add Discovery persistence concerns here.

Expected changes:
- define the normalized wallet/account observation wire shape
- define exported wire vocabulary for:
  - account_kind
  - current_algorithm
  - current_pq_posture
  - subject type
  - event_type / event_version constants
- add canonical fixtures and serialization tests for `discovery.wallet.observed.v0.1`

Suggested values:
- account kinds:
  - eoa
  - erc4337_smart_account
  - delegated_eoa_7702
  - contract_account
  - unknown
- algorithm IDs:
  - secp256k1_ecrecover
  - mldsa44
  - mldsa65
  - falcon512
  - hybrid_*
- posture:
  - classical_only
  - hybrid
  - full_pq
  - unknown

Deliverables:
- typed shared wire contract
- exported wire vocabulary constants/types
- canonical JSON fixtures
- focused tests for validation and serialization

Out of scope:
- Discovery implementation
- CPM policy logic
- event bus wiring
```

### Acceptance criteria
- A shared versioned wire contract exists in `cafe-contracts`.
- Discovery and CPM can depend on it without depending on each other.
- Canonical fixtures exist and are tested.

---

## PR2 — Bootstrap `cafe-cpm` repository structure

### Suggested PR title
`chore(cpm): bootstrap cafe-cpm service repository`

### Goal
Create the initial standalone repository structure for CPM without implementing business logic yet.

### Cursor prompt

```text
Implement PR2 only in `cafe-cpm`.

Goal:
Bootstrap `cafe-cpm` as its own repository.

Requirements:
- Do not move code out of `cafe-discovery` in this PR.
- Do not refactor Discovery.
- Create the minimum standalone structure needed for CPM development.

Expected changes:
- initialize the new Go service/repository structure for `cafe-cpm`
- add module/package layout consistent with the existing CAFE repos
- add minimal config loading
- add minimal application bootstrap
- add placeholders for:
  - API layer
  - policy domain models
  - NATS integration
  - persistence layer owned by CPM
- add baseline GitHub Actions workflow skeleton aligned with service-repo expectations
- add a README describing the boundaries of CPM vs Contracts vs Discovery vs Remediation

Deliverables:
- standalone repository skeleton
- minimal build/test success
- short architecture README

Out of scope:
- policy logic
- NATS wiring
- Discovery refactor
```

### Acceptance criteria
- `cafe-cpm` builds as an independent repository.
- Service boundaries are documented.
- No Discovery refactor is introduced.

---

## PR3 — Improve Discovery wallet observation model and adapter for CPM export

### Suggested PR title
`feat(discovery): deterministic wallet observation export for shared contracts`

### Goal
Improve Discovery only where it directly supports a clearer, deterministic export of wallet observations at the observation/export boundary.

### Cursor prompt

```text
Implement PR3 only in `cafe-discovery`.

Goal:
Improve the Discovery-owned wallet observation path only as needed to produce a deterministic export to the shared `discovery.wallet.observed.v0.1` wire contract from `cafe-contracts`.

Scope:
- Narrow, observation/export-boundary work only.
- Do not perform a broad refactor of Discovery internals or persistence architecture.

Requirements:
- Add or refine a Discovery-side adapter/projection that maps Discovery observation inputs to the shared versioned wire contract.
- Exported string values must match the shared wire vocabulary from `cafe-contracts`.
- `current_pq_posture` in this PR must always be `unknown` as a documented placeholder.
- `chain_ids` rules:
  - emit numeric chain IDs only when mapping is certain
  - do not invent fallback numeric values
  - do not use sentinel values such as `0`
  - if no reliable mapping exists, `chain_ids` may be an empty array

Also normalize or clarify in the export path:
- account kind
- current algorithm
- public key exposure semantics
- multichain status

Tests:
- mapping tests from representative Discovery inputs to the shared exported wire shape
- JSON shape tests against canonical fixtures
- explicit cases for empty `chain_ids` and placeholder posture

Deliverables:
- deterministic Discovery-side export adapter
- focused tests for mapping, JSON shape, chain ID rules, and posture placeholder
- brief comments documenting placeholder posture and chain ID policy

Out of scope:
- NATS publication wiring unless it fits naturally in the same touchpoints
- broad Discovery refactor
- CPM policy logic
- real `current_pq_posture` derivation
```

### Acceptance criteria
- Export is deterministic for given observation inputs.
- `current_pq_posture` is `unknown` in this PR only.
- `chain_ids` contains only confidently mapped numeric IDs.
- Discovery changes are limited to the export boundary.

---

## PR4 — Point `cafe-cpm` at shared wallet observation wire types

### Suggested PR title
`refactor(cpm): consume shared wallet observation wire types from cafe-contracts`

### Goal
Make `cafe-cpm` consume the same shared wallet observation wire types as Discovery.

### Cursor prompt

```text
Implement PR4 only in `cafe-cpm`.

Goal:
Switch `cafe-cpm` to depend on the shared `discovery.wallet.observed.v0.1` wire contract from `cafe-contracts`.

Requirements:
- Prefer type aliases or thin local wrappers for ergonomics.
- Do not copy or fork the shared wire structs silently.
- Keep wire-level validation in `cafe-contracts` minimal.
- Keep policy semantics and interpretation in `cafe-cpm`.

Deliverables:
- `cafe-cpm` imports and uses the shared wallet observation wire types
- tests updated to use canonical fixtures
- no dependency from CPM to Discovery

Out of scope:
- policy logic
- NATS wiring
- Discovery changes
```

### Acceptance criteria
- CPM uses shared wire types from `cafe-contracts`.
- No duplicated wallet-observation wire structs remain without justification.
- No CPM -> Discovery dependency exists.

---

## PR5 — Derive current PQ posture on the Discovery observation/export side

### Suggested PR title
`feat(discovery): derive current PQ posture for exported wallet observations`

### Goal
Add a deterministic posture summary close to the observed state, on the Discovery-owned side of the boundary.

### Cursor prompt

```text
Implement PR5 only in `cafe-discovery`.

Goal:
Add a derived, deterministic `current_pq_posture` field to the Discovery-owned wallet observation/export path.

Requirements:
- Derive posture only from normalized observed wallet data.
- Keep the logic explicit and small.
- Do not introduce a DSL.
- Keep the result aligned with the shared exported contract.

Expected values:
- classical_only
- hybrid
- full_pq
- unknown

Expected changes:
- implement a pure computation function
- project the derived posture into the normalized exported observation
- document the rules in code comments

Out of scope:
- policy selection request
- catalog/template/instance
- ranking
- broad Discovery refactor
```

### Acceptance criteria
- `current_pq_posture` is derived deterministically.
- Rules are easy to read and test.
- Tests cover all supported posture values.

---

## PR6 — Add `PolicySelectionRequest`
(remaining CPM-domain PRs follow the same shape as v0.5, but now assume shared contracts come from `cafe-contracts`)

### Suggested PR title
`feat(cpm): add PolicySelectionRequest for CPM selection flows`

### Cursor prompt

```text
Implement PR6 only in `cafe-cpm`.

Goal:
Introduce PolicySelectionRequest for CPM decision-making.

Requirements:
- Keep it explicit, typed, serializable, and stable.
- No DSL.
- Design it as a real contract, not a temporary internal struct.
- Document how `target_chain_ids` and `require_multichain` interact with the canonical multi-chain rules.
```

### Acceptance criteria
- `PolicySelectionRequest` is stable and serializable.
- Validation/defaulting is explicit.

---

## PR7 — Add PolicyGraphCatalog core model
### Suggested PR title
`feat(cpm): add PolicyGraphCatalog model and loader`

## PR8 — Add CryptoPolicyTemplate model and validation
### Suggested PR title
`feat(cpm): add CryptoPolicyTemplate model and validation`

## PR9 — Add CryptoPolicyInstance model
### Suggested PR title
`feat(cpm): add CryptoPolicyInstance model for concrete CPx`

## PR10 — Add validation result model and validator
### Suggested PR title
`feat(cpm): add instance validation result model and validator`

## PR11 — Add assessment result model
### Suggested PR title
`feat(cpm): add CryptoPolicyAssessmentResult model`

## PR12 — Implement compatibility evaluation
### Suggested PR title
`feat(cpm): evaluate policy compatibility against observation and request`

### Goal
Given a normalized wallet observation payload, a validated `PolicySelectionRequest`, a validated `CryptoPolicyInstance` (and optional `CryptoPolicyTemplate` when the instance only references `template_id`), compute a deterministic compatibility outcome per §3 (multi-chain rules, compatibility vs deployability states) before PR13 ranking.

### Implementation (done)
- `PolicyCompatibilityEvaluator.Evaluate(walletobserved.Payload, *PolicySelectionRequest, *CryptoPolicyInstance, *PolicyGraphCatalog, *CryptoPolicyTemplate)` returns `PolicyCompatibilityResult` with `status` ∈ {`compatible_and_deployable`, `compatible_but_not_deployable`, `incompatible`} and explainable `AssessmentFinding` codes.
- Effective node path: instance `node_path` if set; otherwise `template.NodePath` when `template_id` matches and catalog versions align.
- Checks include: monotone target-posture satisfaction (instance vs request), per-node `minimum_maturity` along the path, selection boolean/bundler/paymaster constraints, provider-mode subset, new-wallet constraint, last path node `supported_postures` vs instance `target_posture`, chain scope vs request targets vs observation `chain_ids`, and `require_multichain` coverage.
- Tests: `internal/domain/policy/compatibility_result_test.go` (fixtures + negative paths).
- No NATS, no Discovery imports, no ranking / `PolicyDecision` (PR13).

### Acceptance criteria
- Three compatibility states are distinguishable and match the normative strings used by `CryptoPolicyAssessmentResult` for the non-pending cases.
- Chain support is evaluated in this layer, not deferred to ranking.
- `go test ./...`, `golangci-lint run ./...`, `govulncheck ./...` pass in `cafe-cpm`.

## PR13 — Implement ranking and policy decision output
### Suggested PR title
`feat(cpm): rank compatible routes and emit decision output`

For PR7–PR13:
- keep the v0.5 constraints and acceptance criteria
- update imports and fixtures as needed to use shared contracts from `cafe-contracts`
- do not introduce any Discovery dependency

---

## PR14 — Define shared assessment-command and CPM/Remediation NATS event contracts

### Suggested PR title
`feat(contracts): add explicit assessment request and CPM/remediation event contracts v0.1`

### Goal
Add the shared assessment-command contract and the shared NATS event envelope/payload contracts in `cafe-contracts`.

### Cursor prompt

```text
Implement PR14 only in `cafe-contracts`.

Goal:
Add the shared NATS event envelope, the explicit `policy.assessment.requested.v0.1` command contract, and the shared CPM/Remediation event payload contracts.

Requirements:
- Keep subjects versioned.
- Keep payloads typed and serializable.
- Include fields needed for idempotent processing and replay-safe handling.
- Add a documented mapping matrix reference for the first-version events.
- Keep this repo free of business logic.

Expected event families:
- discovery.wallet.observed.v0.1
- policy.assessment.requested.v0.1
- policy.validation.completed.v0.1
- policy.instance.activated.v0.1
- policy.assessment.completed.v0.1
- policy.remediation.requested.v0.1
- remediation.plan.created.v0.1
- remediation.execution.started.v0.1
- remediation.execution.completed.v0.1
- remediation.execution.failed.v0.1

Important command semantics:
- `policy.assessment.requested.v0.1` is the canonical asynchronous trigger for CPM
- it must be usable without CPM reading Discovery persistence
- it should therefore include an embedded observation snapshot or another fully self-sufficient payload
- the wire shape for the embedded observation must reuse the shared Discovery observation contract shapes where appropriate

Deliverables:
- envelope type
- subject constants
- typed payload models
- canonical fixtures
- JSON tests

Out of scope:
- actual NATS subscriptions/publications
- CPM business logic
```

### Acceptance criteria
- Shared event contracts are packaged in `cafe-contracts`.
- Discovery, CPM, and Remediation can all depend on them without depending on each other.
- `policy.assessment.requested.v0.1` exists as an explicit command contract and is sufficient for CPM to start assessment without reading Discovery persistence.

## PR15 — Integrate CPM with explicit assessment requests over NATS

### Suggested PR title
`feat(cpm): wire explicit assessment request flow over NATS`

### Goal
Connect CPM to the explicit assessment-request flow over NATS while keeping Discovery-side changes minimal and integration-focused.

### Cursor prompt

```text
Implement PR15 only.

Goal:
Integrate CPM with the explicit assessment request flow over NATS, using shared event contracts from `cafe-contracts`.

Requirements:
- CPM must consume `policy.assessment.requested.v0.1` as its canonical asynchronous trigger.
- CPM must not use raw `discovery.wallet.observed.v0.1` publication as an implicit trigger.
- Keep the implementation idempotent by design.
- Do not refactor Discovery persistence.
- Keep Discovery changes minimal and limited to issuing the explicit request when the authenticated backend or gateway receives a user action.

Idempotence expectations:
- use inbound `event_id` as the primary duplicate-suppression key
- document the persisted or tracked idempotence mechanism used by CPM
- make replay behavior explicit in tests

Expected behavior:
- consume `policy.assessment.requested.v0.1`
- validate the embedded or referenced self-sufficient observation payload using shared contracts
- map inbound requests to the CPM assessment flow
- keep event handlers thin and delegate policy logic to CPM services
- if Discovery-side changes are required, implement only the minimal producer-side wiring to publish the explicit command after a user action

Important:
- preserve any existing informational `discovery.wallet.observed.v0.1` publication if it is still useful, but do not treat it as the assessment trigger
- avoid coupling to Discovery database or Discovery service internals

Expected changes:
- add NATS consumer wiring in CPM for `policy.assessment.requested.v0.1`
- add minimal Discovery-side publication wiring only if needed for the explicit request command
- connect inbound event handling to the assessment flow
- add focused tests for event mapping and idempotent behavior assumptions
- document any compatibility assumptions with current Discovery subjects

Deliverables:
- NATS inbound integration wiring for explicit assessment requests
- tests for inbound request behavior
- minimal documentation comments

Out of scope:
- Discovery monolith refactor
- new remediation logic
- broad eventing redesign outside the explicit request path
```

### Acceptance criteria
- CPM can consume explicit assessment requests and start assessment from them.
- Tests cover the main inbound flow, duplicate delivery, and replay handling.
- Duplicate suppression behavior is explicit and verified.
- Retry semantics are documented.
- Discovery changes, if any, remain narrow and integration-focused.
- CPM does not auto-start assessment from raw Discovery observation events.

## PR16 — Add outbound CPM events for Remediation

### Suggested PR title
`feat(cpm): publish outbound policy events for remediation workflows`

### Goal
Publish CPM outcomes for downstream Remediation without embedding remediation logic into CPM.

### Cursor prompt

```text
Implement PR16 only in `cafe-cpm`.

Goal:
Publish outbound CPM events used by Remediation and other downstream consumers, using shared event contracts from `cafe-contracts`.

Requirements:
- Do not implement remediation execution.
- Keep producers explicit and small.
- Build payloads as explicit projections of internal CPM models rather than recomputing business logic in producers.
- Preserve correlation and causation linkage back to the explicit assessment request.

Deliverables:
- outbound event publication wiring
- tests for outbound event behavior
- minimal documentation comments
```

### Acceptance criteria
- CPM publishes the expected policy events.
- `policy.remediation.requested` includes the required minimum operational fields.
- Tests cover duplicate publication behavior and deterministic retry projection.
- Outbound events preserve linkage to the originating explicit request.

## PR17 — Add CPM read APIs

### Suggested PR title
`feat(api): expose CPM read APIs for policies and decisions`

### Goal
Expose CPM read APIs for policy documents, validation, assessment, and route exploration.

### Cursor prompt

```text
Implement PR17 only in `cafe-cpm`.

Goal:
Expose CPM read APIs for policy inspection and route exploration.

Requirements:
- Reuse PolicyDecision and related models where possible.
- Keep payloads explicit and stable.
- Preserve the distinction between incompatible routes and known-but-not-deployable routes.
```

### Acceptance criteria
- Clients can inspect CPM documents and decision outputs.
- Known-but-not-deployable routes remain distinguishable from incompatible ones.

---

## Optional PR18 — Seed fixtures and prepare validation/model layer for future CRUD

### Suggested PR title
`refactor(cpm): prepare model and validation layer for future CRUD`

### Acceptance criteria
- Validation can be reused outside startup loading.
- The model is clearly CRUD-ready.
- Local fixtures exist and remain small.
- No DSL is introduced.

---

## 9. Final reviewer notes

A PR should be sent back if it introduces any of the following without explicit justification:
- a DSL
- a generic policy engine
- configurable edge parameters in `CryptoPolicyInstance`
- validation or assessment blobs embedded inside the canonical instance struct
- large unrelated refactors
- hidden business logic behind opaque scoring or expression syntax
- remediation execution logic inside CPM
- event contracts represented only as untyped dynamic maps
- loss of the distinction between incompatible and known-but-not-deployable routes
- non-normalized `policy_id` comparison in the final tie-break
- event payloads that recompute policy semantics instead of projecting internal models
- undocumented enum values introduced outside the shared contract vocabulary
- non-deterministic ranking or hidden tie-break behavior
- remediation-trigger payloads that omit required operational fields
- direct CPM dependency on Discovery database
- direct Discovery <-> CPM code dependency
- CPM wired to auto-start assessment from raw Discovery observation events
- broad Discovery monolith refactoring slipped into this workstream

---

## 10. Short implementation notes for Cursor

```text
General notes:
- Keep PRs small and reviewable.
- Use table-driven tests.
- Prefer deterministic helper functions.
- Prefer typed enums/constants over free-form strings.
- Avoid over-abstracting.
- Keep validation and assessment as separate result types.
- Keep transition semantics in catalog/template compatibility rules.
- Keep service boundaries clean:
  - Discovery observes
  - `cafe-contracts` packages shared wire contracts
  - CPM validates/selects/assesses
  - Remediation plans/executes
- Keep CPM independent from Discovery persistence.
- Improve Discovery only where it strengthens the exported observation contract.
- Build duplicate-safe event handling from explicit IDs.
- Apply the canonical multi-chain rules before ranking.
- Make rejection and ranking reasons explicit.
- Summarize what changed and what is intentionally deferred at the end of each PR.
```
