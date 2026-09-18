// Package policy contains CPM policy domain models and services.
//
// PolicySelectionRequest is the stable input contract for policy selection.
// CryptoPolicy is the catalogue intention (required_posture + allowed_providers).
// CryptoPolicyCatalogItem / CompatibleNetwork expose derived catalogue facts
// (compatible_networks from registry chain_support, status ≠ planned).
// CryptoPolicyInstance / PolicySelectionRequest / PolicyCompatibilityEvaluator /
// PolicyDecisionEvaluator remain as legacy unit-test helpers (*_legacy_test.go)
// for pre-P9b ranking semantics; explore HTTP uses ExploreCoucheAEvaluator.
// AssessmentStatus and AssessmentFinding model compatibility/deployability signals.
// CryptoPolicyPersistPayload / ValidatePayloadForPersist enforce ADR §9
// persist gates (schema v0.2, crypto_policy_id, user_constraints, couche A+B
// replay against accepted snapshot, soft findings listed, pinned provider refs).
package policy
