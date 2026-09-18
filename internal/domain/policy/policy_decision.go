package policy

import (
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

// RankedPolicy stores one compatible candidate after deterministic ranking.
type RankedPolicy struct {
	CandidateID              string                             `json:"candidate_id"`
	PolicyID                 string                             `json:"policy_id"`
	CryptoPolicyInstanceID   string                             `json:"crypto_policy_instance_id,omitempty"`
	TemplateID               string                             `json:"template_id,omitempty"`
	RequiredPosture          vocabulary.CurrentPQPosture        `json:"required_posture,omitempty"`
	ResultingPosture         vocabulary.CurrentPQPosture        `json:"resulting_posture,omitempty"`
	SolutionProfileRef       SolutionProfileRef                 `json:"solution_profile_ref,omitempty"`
	Maturity                 string                             `json:"maturity,omitempty"`
	ClaimStatus              string                             `json:"claim_status,omitempty"`
	CompatibilityStatus      AssessmentStatus                   `json:"compatibility_status"`
	CompatibilityFindings    []AssessmentFinding                `json:"compatibility_findings"`
	SuggestedUserConstraints *provider.SuggestedUserConstraints `json:"suggested_user_constraints,omitempty"`
	// Composition is the explore product view for Expected result (CFB-P3).
	// Present on catalogue-driven explore v0.2 scan_compatible_providers.
	Composition *CompositionView `json:"composition,omitempty"`
}

// RejectedPolicy stores one incompatible candidate and explainable reasons.
type RejectedPolicy struct {
	CandidateID            string                      `json:"candidate_id"`
	PolicyID               string                      `json:"policy_id"`
	CryptoPolicyInstanceID string                      `json:"crypto_policy_instance_id,omitempty"`
	TemplateID             string                      `json:"template_id,omitempty"`
	RequiredPosture        vocabulary.CurrentPQPosture `json:"required_posture,omitempty"`
	ResultingPosture       vocabulary.CurrentPQPosture `json:"resulting_posture,omitempty"`
	SolutionProfileRef     SolutionProfileRef          `json:"solution_profile_ref,omitempty"`
	Maturity               string                      `json:"maturity,omitempty"`
	ClaimStatus            string                      `json:"claim_status,omitempty"`
	CompatibilityStatus    AssessmentStatus            `json:"compatibility_status"`
	CompatibilityFindings  []AssessmentFinding         `json:"compatibility_findings"`
	RejectionReasons       []AssessmentFinding         `json:"-"`
}

// ObservedWalletSummary is a compact decision-facing projection of wallet input.
type ObservedWalletSummary struct {
	ChainIDs []int64 `json:"chain_ids,omitempty"`
}

// RequestSummary is a compact projection of explore/assessment input.
// Explore HTTP v0.2 carries crypto_policy_id; Target* / couche-B fields remain for the
// transitional PolicyDecisionEvaluator path (persist couche B → CPM-P10).
type RequestSummary struct {
	CryptoPolicyID            string                      `json:"crypto_policy_id,omitempty"`
	TargetPosture             vocabulary.CurrentPQPosture `json:"target_posture,omitempty"`
	TargetChainIDs            []int64                     `json:"target_chain_ids,omitempty"`
	RequireMultichain         bool                        `json:"require_multichain,omitempty"`
	AllowNewWallet            bool                        `json:"allow_new_wallet,omitempty"`
	AddressContinuityRequired bool                        `json:"address_continuity_required,omitempty"`
	MinimumMaturity           int                         `json:"minimum_maturity,omitempty"`
}

// PolicyDecision is the explainable first-version output from ranking.
// JSON public key for compatible candidates is scan_compatible_providers (ADR amendement / P9a).
// Go type RankedPolicy may remain internal (ADR §14 Q9).
type PolicyDecision struct {
	ObservedWalletSummary    ObservedWalletSummary `json:"observed_wallet_summary"`
	RequestSummary           RequestSummary        `json:"request_summary"`
	SelectedPolicyID         string                `json:"selected_policy_id,omitempty"`
	SelectedPolicyInstanceID string                `json:"selected_policy_instance_id,omitempty"`
	RankedCandidates         []RankedPolicy        `json:"scan_compatible_providers"`
	RejectedCandidates       []RejectedPolicy      `json:"rejected_candidates,omitempty"`
	Warnings                 []string              `json:"warnings,omitempty"`
}

func findingsOrEmpty(findings []AssessmentFinding) []AssessmentFinding {
	if findings == nil {
		return []AssessmentFinding{}
	}
	return findings
}

func compareRanked(a, b RankedPolicy) int {
	// P4b: compatibility is a hard filter; ranked candidates use only a stable ID.
	if a.CandidateID < b.CandidateID {
		return -1
	}
	if a.CandidateID > b.CandidateID {
		return 1
	}
	return 0
}

func normalizeASCIIUpper(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToUpper(value)

	var b strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch > 127 {
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
