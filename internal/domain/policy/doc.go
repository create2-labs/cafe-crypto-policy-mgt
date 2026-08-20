// Package policy contains CPM policy domain models and services.
//
// PolicySelectionRequest is the stable input contract for policy selection.
// CryptoPolicy is the catalogue intention (required_posture + allowed_providers).
// CryptoPolicyInstance remains an internal explore/persist transitional type until
// CPM-P9/P10; it is not part of the public catalogue.
// AssessmentStatus and AssessmentFinding model compatibility/deployability signals.
// PolicyCompatibilityEvaluator and PolicyCompatibilityResult implement
// observation+request+instance compatibility classification (PR12) before
// ranking (PR13).
// PolicyDecisionEvaluator builds deterministic ranked/rejected candidate output
// and selected policy decision from compatibility results (PR13).
// CryptoPolicyPersistPayload / ValidateDraftPayloadForPersist enforce ADR §9
// persist gates (schema v0.2, soft findings listed, pinned provider refs).
package policy
