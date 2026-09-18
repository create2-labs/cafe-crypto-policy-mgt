package policy

import (
	"errors"
	"fmt"
	"sort"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

var (
	// ErrPolicyDecisionRequestNil indicates a missing selection request.
	ErrPolicyDecisionRequestNil = errors.New("policy decision: policy selection request is nil")
)

// PolicyDecisionCandidate groups one legacy policy instance and its crypto policy.
type PolicyDecisionCandidate struct {
	Instance     *CryptoPolicyInstance
	CryptoPolicy *CryptoPolicy
}

// PolicyDecisionEvaluator evaluates compatibility and applies deterministic PR13 ranking.
// Explore HTTP v0.2 uses ExploreCoucheAEvaluator instead (CPM-P9b).
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


func deriveStablePolicyID(inst *CryptoPolicyInstance) string {
	if inst == nil {
		return ""
	}
	// No dedicated policy_id exists yet in the instance model, so PR13 derives
	// the stable lexical tie-break identifier from the normalized instance id.
	return normalizeASCIIUpper(inst.ID)
}

