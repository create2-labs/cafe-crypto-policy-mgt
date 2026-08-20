package policy

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

var (
	// ErrPolicyDecisionRequestNil indicates a missing selection request.
	ErrPolicyDecisionRequestNil = errors.New("policy decision: policy selection request is nil")
)

// PolicyDecisionCandidate groups one candidate policy instance and its crypto policy.
// Instance remains transitional for explore until CPM-P9.
type PolicyDecisionCandidate struct {
	Instance     *CryptoPolicyInstance
	CryptoPolicy *CryptoPolicy
}

// RankedPolicy stores one compatible candidate after deterministic ranking.
type RankedPolicy struct {
	CandidateID            string                      `json:"candidate_id"`
	PolicyID               string                      `json:"policy_id"`
	CryptoPolicyInstanceID string                      `json:"crypto_policy_instance_id"`
	TemplateID             string                      `json:"template_id,omitempty"`
	RequiredPosture        vocabulary.CurrentPQPosture `json:"required_posture,omitempty"`
	ResultingPosture       vocabulary.CurrentPQPosture `json:"resulting_posture,omitempty"`
	SolutionProfileRef     SolutionProfileRef          `json:"solution_profile_ref,omitempty"`
	Maturity               string                      `json:"maturity,omitempty"`
	ClaimStatus            string                      `json:"claim_status,omitempty"`
	CompatibilityStatus    AssessmentStatus            `json:"compatibility_status"`
	CompatibilityFindings  []AssessmentFinding         `json:"compatibility_findings"`
}

// RejectedPolicy stores one incompatible candidate and explainable reasons.
type RejectedPolicy struct {
	CandidateID            string                      `json:"candidate_id"`
	PolicyID               string                      `json:"policy_id"`
	CryptoPolicyInstanceID string                      `json:"crypto_policy_instance_id"`
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
) (PolicyDecision, error) {
	if req == nil {
		return PolicyDecision{}, ErrPolicyDecisionRequestNil
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

		compatibility, err := e.CompatibilityEvaluator.Evaluate(observation, req, candidate.Instance, candidate.CryptoPolicy)
		if err != nil {
			return PolicyDecision{}, fmt.Errorf("candidate[%d] compatibility: %w", idx, err)
		}

		policyID := deriveStablePolicyID(candidate.Instance)
		profileView := resolveProfileView(e.CompatibilityEvaluator.Providers, candidate)
		if compatibility.Status == AssessmentStatusIncompatible {
			decision.RejectedCandidates = append(decision.RejectedCandidates, RejectedPolicy{
				CandidateID:            candidate.Instance.ID,
				PolicyID:               policyID,
				CryptoPolicyInstanceID: candidate.Instance.ID,
				TemplateID:             candidate.Instance.TemplateID,
				RequiredPosture:        profileView.RequiredPosture,
				ResultingPosture:       profileView.ResultingPosture,
				SolutionProfileRef:     candidate.Instance.SolutionProfileRef,
				Maturity:               profileView.Maturity,
				ClaimStatus:            profileView.ClaimStatus,
				CompatibilityStatus:    compatibility.Status,
				CompatibilityFindings:  findingsOrEmpty(compatibility.Findings),
				RejectionReasons:       compatibility.Findings,
			})
			continue
		}

		ranked := buildRankedCandidate(policyID, candidate, compatibility, profileView)
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
	candidate PolicyDecisionCandidate,
	compatibility PolicyCompatibilityResult,
	profileView candidateProfileView,
) RankedPolicy {
	return RankedPolicy{
		CandidateID:            candidate.Instance.ID,
		PolicyID:               policyID,
		CryptoPolicyInstanceID: candidate.Instance.ID,
		TemplateID:             candidate.Instance.TemplateID,
		RequiredPosture:        profileView.RequiredPosture,
		ResultingPosture:       profileView.ResultingPosture,
		SolutionProfileRef:     candidate.Instance.SolutionProfileRef,
		Maturity:               profileView.Maturity,
		ClaimStatus:            profileView.ClaimStatus,
		CompatibilityStatus:    compatibility.Status,
		CompatibilityFindings:  findingsOrEmpty(compatibility.Findings),
	}
}

func findingsOrEmpty(findings []AssessmentFinding) []AssessmentFinding {
	if findings == nil {
		return []AssessmentFinding{}
	}
	return findings
}

type candidateProfileView struct {
	RequiredPosture  vocabulary.CurrentPQPosture
	ResultingPosture vocabulary.CurrentPQPosture
	Maturity         string
	ClaimStatus      string
}

func resolveProfileView(reg *provider.Registry, candidate PolicyDecisionCandidate) candidateProfileView {
	view := candidateProfileView{
		RequiredPosture: candidate.Instance.GlobalParams.RequiredPosture,
	}
	if candidate.CryptoPolicy != nil && candidate.CryptoPolicy.RequiredPosture != "" {
		view.RequiredPosture = candidate.CryptoPolicy.RequiredPosture
	}
	if reg == nil || candidate.Instance == nil {
		return view
	}
	ref := candidate.Instance.SolutionProfileRef
	resolved, ok := reg.Lookup(provider.ProfileRef{
		ProviderID:        ref.ProviderID,
		SolutionProfileID: ref.SolutionProfileID,
		ManifestVersion:   ref.ManifestVersion,
	})
	if !ok {
		return view
	}
	view.ResultingPosture = vocabulary.CurrentPQPosture(resolved.Profile.ResultingPosture)
	view.Maturity = string(resolved.Profile.Maturity)
	view.ClaimStatus = string(resolved.Profile.ClaimStatus)
	return view
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
