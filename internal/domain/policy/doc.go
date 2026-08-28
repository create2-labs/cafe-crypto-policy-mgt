// Package policy contains CPM policy domain models and services.
//
// PolicySelectionRequest is the stable input contract for policy selection.
// CryptoPolicy is the catalogue intention (required_posture + allowed_providers).
// CryptoPolicyInstance remains for legacy domain unit tests only (PolicyCompatibilityEvaluator).
// AssessmentStatus and AssessmentFinding model compatibility/deployability signals.
// PolicyCompatibilityEvaluator and PolicyCompatibilityResult implement
// observation+request+instance compatibility classification (PR12) before
// ranking (PR13).
// PolicyDecisionEvaluator builds deterministic ranked/rejected candidate output
// and selected policy decision from compatibility results (PR13).
// CryptoPolicyPersistPayload / ValidatePayloadForPersist enforce ADR §9
// persist gates (schema v0.2, crypto_policy_id, user_constraints, couche A+B
// replay against accepted snapshot, soft findings listed, pinned provider refs).
package policy
