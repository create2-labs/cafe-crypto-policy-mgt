// Package policy contains CPM policy domain models and services.
//
// PolicySelectionRequest is the stable input contract for policy selection.
// CryptoPolicyTemplate defines reusable CAFE intentions (required posture + defaults).
// CryptoPolicyInstance defines the concrete scope-bound policy document bound to a
// Capability Provider solution profile.
// AssessmentStatus and AssessmentFinding model compatibility/deployability signals.
// PolicyCompatibilityEvaluator and PolicyCompatibilityResult implement
// observation+request+instance compatibility classification (PR12) before
// ranking (PR13).
// PolicyDecisionEvaluator builds deterministic ranked/rejected candidate output
// and selected policy decision from compatibility results (PR13).
// CryptoPolicyPersistPayload / ValidateDraftPayloadForPersist enforce ADR §9
// persist gates (schema v0.2, soft findings listed, pinned provider refs).
package policy
