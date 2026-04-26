package policy

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/create2-labs/cafe-cpm/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-cpm/internal/domain/walletobserved"
)

var (
	// ErrPolicyDecisionRequestNil indicates a missing selection request.
	ErrPolicyDecisionRequestNil = errors.New("policy decision: policy selection request is nil")
	// ErrPolicyDecisionCatalogNil indicates a missing policy graph catalog.
	ErrPolicyDecisionCatalogNil = errors.New("policy decision: policy graph catalog is nil")
)

// PolicyDecisionCandidate groups one candidate policy instance and its optional template.
type PolicyDecisionCandidate struct {
	Instance *CryptoPolicyInstance
	Template *CryptoPolicyTemplate
}

// RankedPolicy stores one compatible candidate after deterministic ranking.
type RankedPolicy struct {
	PolicyID                 string              `json:"policy_id"`
	CryptoPolicyInstanceID   string              `json:"crypto_policy_instance_id"`
	TemplateID               string              `json:"template_id,omitempty"`
	CompatibilityStatus      AssessmentStatus    `json:"compatibility_status"`
	CompatibilityFindings    []AssessmentFinding `json:"compatibility_findings,omitempty"`
	TargetPostureAlignment   int                 `json:"target_posture_alignment"`
	MaturityScore            int                 `json:"maturity_score"`
	ChainCoverageScore       int                 `json:"chain_coverage_score"`
	AddressContinuityMatched bool                `json:"address_continuity_matched"`
	AvoidsNewWalletCreation  bool                `json:"avoids_new_wallet_creation"`
	RankingReasons           []string            `json:"ranking_reasons,omitempty"`
}

// RejectedPolicy stores one incompatible candidate and explainable reasons.
type RejectedPolicy struct {
	PolicyID               string              `json:"policy_id"`
	CryptoPolicyInstanceID string              `json:"crypto_policy_instance_id"`
	TemplateID             string              `json:"template_id,omitempty"`
	CompatibilityStatus    AssessmentStatus    `json:"compatibility_status"`
	RejectionReasons       []AssessmentFinding `json:"rejection_reasons,omitempty"`
}

// ObservedWalletSummary is a compact decision-facing projection of wallet input.
type ObservedWalletSummary struct {
	ChainIDs []int64 `json:"chain_ids,omitempty"`
}

// RequestSummary is a compact projection of the selection request used for ranking.
type RequestSummary struct {
	TargetPosture             vocabulary.CurrentPQPosture `json:"target_posture"`
	TargetChainIDs            []int64                     `json:"target_chain_ids,omitempty"`
	RequireMultichain         bool                        `json:"require_multichain"`
	AllowNewWallet            bool                        `json:"allow_new_wallet"`
	AddressContinuityRequired bool                        `json:"address_continuity_required"`
	MinimumMaturity           int                         `json:"minimum_maturity"`
}

// PolicyDecision is the explainable first-version output from ranking.
type PolicyDecision struct {
	ObservedWalletSummary    ObservedWalletSummary `json:"observed_wallet_summary"`
	RequestSummary           RequestSummary        `json:"request_summary"`
	SelectedPolicyID         string                `json:"selected_policy_id,omitempty"`
	SelectedPolicyInstanceID string                `json:"selected_policy_instance_id,omitempty"`
	RankedCandidates         []RankedPolicy        `json:"ranked_candidates,omitempty"`
	RejectedCandidates       []RejectedPolicy      `json:"rejected_candidates,omitempty"`
	Warnings                 []string              `json:"warnings,omitempty"`
}

// PolicyDecisionEvaluator evaluates compatibility and applies deterministic PR13 ranking.
type PolicyDecisionEvaluator struct {
	CompatibilityEvaluator PolicyCompatibilityEvaluator
}

// Evaluate computes compatibility for each candidate, rejects incompatible routes,
// and ranks the remaining routes using the first-version deterministic order.
func (e PolicyDecisionEvaluator) Evaluate(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	candidates []PolicyDecisionCandidate,
	catalog *PolicyGraphCatalog,
) (PolicyDecision, error) {
	if req == nil {
		return PolicyDecision{}, ErrPolicyDecisionRequestNil
	}
	if catalog == nil {
		return PolicyDecision{}, ErrPolicyDecisionCatalogNil
	}
	if err := req.NormalizeAndValidate(); err != nil {
		return PolicyDecision{}, err
	}

	decision := PolicyDecision{
		ObservedWalletSummary: ObservedWalletSummary{
			ChainIDs: normalizeChainIDs(observation.ChainIDs),
		},
		RequestSummary: RequestSummary{
			TargetPosture:             req.TargetPosture,
			TargetChainIDs:            req.TargetChainIDs,
			RequireMultichain:         req.RequireMultichain,
			AllowNewWallet:            req.AllowNewWallet,
			AddressContinuityRequired: req.AddressContinuityRequired,
			MinimumMaturity:           req.MinimumMaturity,
		},
		RankedCandidates:   make([]RankedPolicy, 0),
		RejectedCandidates: make([]RejectedPolicy, 0),
		Warnings:           make([]string, 0),
	}

	for idx, candidate := range candidates {
		if candidate.Instance == nil {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("candidate[%d]: missing instance", idx))
			continue
		}

		compatibility, err := e.CompatibilityEvaluator.Evaluate(observation, req, candidate.Instance, catalog, candidate.Template)
		if err != nil {
			return PolicyDecision{}, fmt.Errorf("candidate[%d] compatibility: %w", idx, err)
		}

		policyID := deriveStablePolicyID(candidate.Instance)
		if compatibility.Status == AssessmentStatusIncompatible {
			decision.RejectedCandidates = append(decision.RejectedCandidates, RejectedPolicy{
				PolicyID:               policyID,
				CryptoPolicyInstanceID: candidate.Instance.ID,
				TemplateID:             candidate.Instance.TemplateID,
				CompatibilityStatus:    compatibility.Status,
				RejectionReasons:       compatibility.Findings,
			})
			continue
		}

		ranked := buildRankedCandidate(policyID, observation, req, candidate, compatibility, catalog)
		decision.RankedCandidates = append(decision.RankedCandidates, ranked)
	}

	sort.SliceStable(decision.RankedCandidates, func(i, j int) bool {
		return compareRanked(decision.RankedCandidates[i], decision.RankedCandidates[j]) < 0
	})

	if len(decision.RankedCandidates) > 0 {
		decision.SelectedPolicyID = decision.RankedCandidates[0].PolicyID
		decision.SelectedPolicyInstanceID = decision.RankedCandidates[0].CryptoPolicyInstanceID
	}

	sort.SliceStable(decision.RejectedCandidates, func(i, j int) bool {
		return decision.RejectedCandidates[i].PolicyID < decision.RejectedCandidates[j].PolicyID
	})

	return decision, nil
}

func buildRankedCandidate(
	policyID string,
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	candidate PolicyDecisionCandidate,
	compatibility PolicyCompatibilityResult,
	catalog *PolicyGraphCatalog,
) RankedPolicy {
	path, err := effectiveNodePath(candidate.Instance, candidate.Template, catalog)
	if err != nil {
		// Compatibility passed, so this should never happen. Keep an explicit fallback.
		path = nil
	}

	r := RankedPolicy{
		PolicyID:                 policyID,
		CryptoPolicyInstanceID:   candidate.Instance.ID,
		TemplateID:               candidate.Instance.TemplateID,
		CompatibilityStatus:      compatibility.Status,
		CompatibilityFindings:    compatibility.Findings,
		TargetPostureAlignment:   computeTargetPostureAlignment(req, candidate.Instance),
		MaturityScore:            computeMaturityScore(path, candidate.Instance, catalog),
		ChainCoverageScore:       computeChainCoverageScore(observation, req, candidate.Instance),
		AddressContinuityMatched: candidate.Instance.GlobalParams.AddressContinuityRequired == req.AddressContinuityRequired,
		AvoidsNewWalletCreation:  !candidate.Instance.GlobalParams.AllowNewWallet,
		RankingReasons: []string{
			"compatible route retained after incompatibility filtering",
		},
	}

	if r.AddressContinuityMatched {
		r.RankingReasons = append(r.RankingReasons, "address continuity preference matched request")
	}
	if req.AllowNewWallet && r.AvoidsNewWalletCreation {
		r.RankingReasons = append(r.RankingReasons, "prefers route that avoids new wallet creation")
	}

	return r
}

func compareRanked(a, b RankedPolicy) int {
	// 2) Better alignment first: lower delta to requested posture wins.
	if a.TargetPostureAlignment != b.TargetPostureAlignment {
		return cmpIntAsc(a.TargetPostureAlignment, b.TargetPostureAlignment)
	}
	// 3) Higher maturity wins.
	if a.MaturityScore != b.MaturityScore {
		return cmpIntDesc(a.MaturityScore, b.MaturityScore)
	}
	// 4) Better chain coverage wins.
	if a.ChainCoverageScore != b.ChainCoverageScore {
		return cmpIntDesc(a.ChainCoverageScore, b.ChainCoverageScore)
	}
	// 5) Better address continuity requirement satisfaction wins.
	if a.AddressContinuityMatched != b.AddressContinuityMatched {
		if a.AddressContinuityMatched {
			return -1
		}
		return 1
	}
	// 6) Prefer avoiding new wallet creation when request allows choice.
	if a.AvoidsNewWalletCreation != b.AvoidsNewWalletCreation {
		if a.AvoidsNewWalletCreation {
			return -1
		}
		return 1
	}
	// 7) Final stable tie-break on normalized policy_id.
	if a.PolicyID < b.PolicyID {
		return -1
	}
	if a.PolicyID > b.PolicyID {
		return 1
	}
	return 0
}

func computeTargetPostureAlignment(req *PolicySelectionRequest, inst *CryptoPolicyInstance) int {
	return postureRank(inst.GlobalParams.TargetPosture) - postureRank(req.TargetPosture)
}

func computeMaturityScore(path []string, inst *CryptoPolicyInstance, catalog *PolicyGraphCatalog) int {
	if len(path) == 0 || catalog == nil {
		return inst.GlobalParams.MinimumMaturity
	}
	score := 0
	for idx, nodeID := range path {
		node := catalog.Nodes[nodeID]
		if idx == 0 || node.Maturity < score {
			score = node.Maturity
		}
	}
	return score
}

func computeChainCoverageScore(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
) int {
	target := req.TargetChainIDs
	if len(target) == 0 {
		target = observation.ChainIDs
	}

	targetSet := normalizeChainSet(target)
	scopeSet := normalizeChainSet(inst.Scope.ChainIDs)

	score := 0
	for id := range targetSet {
		if scopeSet[id] {
			score++
		}
	}
	return score
}

func deriveStablePolicyID(inst *CryptoPolicyInstance) string {
	if inst == nil {
		return ""
	}
	// No dedicated policy_id exists yet in the instance model, so PR13 derives
	// the stable lexical tie-break identifier from the normalized instance id.
	return normalizeASCIIUpper(inst.ID)
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

func cmpIntAsc(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpIntDesc(a, b int) int {
	return cmpIntAsc(b, a)
}
