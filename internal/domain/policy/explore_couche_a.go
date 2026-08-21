package policy

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

// AssessmentStatusErroneous marks a solution profile with a malformed
// suggested_user_constraints block (ADR §6.2 rule 9 / CPM-P9b).
const AssessmentStatusErroneous AssessmentStatus = "erroneous"

// ExploreCoucheAEvaluator matches providers from a Crypto Policy against a scan
// observation using ADR couche A only (no user_constraints / couche B).
type ExploreCoucheAEvaluator struct {
	Providers *provider.Registry
	Logger    exploreCoucheALogger // optional; defaults to log.Default()
}

type exploreCoucheALogger interface {
	Printf(format string, v ...any)
}

// EvaluateExploreCoucheA builds scan_compatible_providers / rejected_candidates
// for HTTP explore v0.2 (CPM-P9b). Couche B fields must not influence membership.
func (e ExploreCoucheAEvaluator) EvaluateExploreCoucheA(
	observation walletobserved.Payload,
	cp *CryptoPolicy,
) (PolicyDecision, error) {
	if cp == nil {
		return PolicyDecision{}, fmt.Errorf("crypto policy is nil")
	}

	decision := PolicyDecision{
		ObservedWalletSummary: ObservedWalletSummary{
			ChainIDs: normalizeChainIDs(observation.ChainIDs),
		},
		RequestSummary: RequestSummary{
			CryptoPolicyID: cp.ID,
		},
		RankedCandidates:   make([]RankedPolicy, 0),
		RejectedCandidates: make([]RejectedPolicy, 0),
		Warnings:           make([]string, 0),
	}

	reg := e.Providers
	logger := e.Logger
	if logger == nil {
		logger = log.Default()
	}

	obs := provider.HardObservation{
		AccountKind: observation.AccountKind,
		ChainIDs:    observation.ChainIDs,
	}
	requiredPosture := string(cp.RequiredPosture)

	for _, providerID := range cp.AllowedProviders {
		providerID = strings.TrimSpace(providerID)
		if providerID == "" {
			continue
		}
		profiles := []*provider.ResolvedProfile{}
		if reg != nil {
			profiles = reg.ProfilesForProvider(providerID)
		}
		if len(profiles) == 0 {
			decision.RejectedCandidates = append(decision.RejectedCandidates, RejectedPolicy{
				CandidateID:         providerID,
				PolicyID:            normalizeASCIIUpper(cp.ID),
				RequiredPosture:     cp.RequiredPosture,
				SolutionProfileRef:  SolutionProfileRef{ProviderID: providerID},
				CompatibilityStatus: AssessmentStatusIncompatible,
				CompatibilityFindings: findingsOrEmpty([]AssessmentFinding{
					fieldFinding(provider.FindingCodeUnresolved, fmt.Sprintf("provider %q not found in registry", providerID), "allowed_providers"),
				}),
				RejectionReasons: []AssessmentFinding{
					fieldFinding(provider.FindingCodeUnresolved, fmt.Sprintf("provider %q not found in registry", providerID), "allowed_providers"),
				},
			})
			continue
		}

		for _, resolved := range profiles {
			if resolved == nil {
				continue
			}
			entry := buildExploreCandidate(cp, resolved)

			if err := provider.ValidateSuggestedUserConstraints(&resolved.Profile); err != nil {
				logger.Printf(
					"ERROR explore: erroneous suggested_user_constraints provider=%s profile=%s crypto_policy_id=%s err=%v",
					resolved.ProviderID, resolved.Profile.SolutionProfileID, cp.ID, err,
				)
				finding := AssessmentFinding{
					Code:     provider.FindingCodeErroneousSuggested,
					Message:  err.Error(),
					Severity: AssessmentFindingSeverityBlocking,
					Field:    "suggested_user_constraints",
				}
				entry.CompatibilityStatus = AssessmentStatusErroneous
				entry.CompatibilityFindings = findingsOrEmpty([]AssessmentFinding{finding})
				entry.RejectionReasons = []AssessmentFinding{finding}
				decision.RejectedCandidates = append(decision.RejectedCandidates, entry)
				continue
			}

			hard := provider.EvaluateScanCompatibility(obs, requiredPosture, &resolved.Profile)
			if len(hard) > 0 {
				findings := make([]AssessmentFinding, 0, len(hard))
				for _, f := range hard {
					findings = append(findings, fieldFinding(f.Code, f.Message, f.Field))
				}
				entry.CompatibilityStatus = AssessmentStatusIncompatible
				entry.CompatibilityFindings = findingsOrEmpty(findings)
				entry.RejectionReasons = findings
				decision.RejectedCandidates = append(decision.RejectedCandidates, entry)
				continue
			}

			ranked := RankedPolicy{
				CandidateID:              entry.CandidateID,
				PolicyID:                 entry.PolicyID,
				RequiredPosture:          entry.RequiredPosture,
				ResultingPosture:         entry.ResultingPosture,
				SolutionProfileRef:       entry.SolutionProfileRef,
				Maturity:                 entry.Maturity,
				ClaimStatus:              entry.ClaimStatus,
				CompatibilityStatus:      AssessmentStatusCompatibleAndDeployable,
				CompatibilityFindings:    softFindingsForProfile(&resolved.Profile),
				SuggestedUserConstraints: cloneSuggested(resolved.Profile.SuggestedUserConstraints),
			}
			decision.RankedCandidates = append(decision.RankedCandidates, ranked)
		}
	}

	sort.SliceStable(decision.RankedCandidates, func(i, j int) bool {
		return compareRanked(decision.RankedCandidates[i], decision.RankedCandidates[j]) < 0
	})
	sort.SliceStable(decision.RejectedCandidates, func(i, j int) bool {
		return decision.RejectedCandidates[i].CandidateID < decision.RejectedCandidates[j].CandidateID
	})

	if len(decision.RankedCandidates) > 0 {
		decision.SelectedPolicyID = decision.RankedCandidates[0].PolicyID
	}

	return decision, nil
}

func buildExploreCandidate(cp *CryptoPolicy, resolved *provider.ResolvedProfile) RejectedPolicy {
	ref := SolutionProfileRef{
		ProviderID:        resolved.ProviderID,
		SolutionProfileID: resolved.Profile.SolutionProfileID,
		ManifestVersion:   resolved.ProviderVersion,
	}
	candidateID := resolved.ProviderID + "/" + resolved.Profile.SolutionProfileID
	return RejectedPolicy{
		CandidateID:        candidateID,
		PolicyID:           normalizeASCIIUpper(cp.ID),
		RequiredPosture:    cp.RequiredPosture,
		ResultingPosture:   vocabulary.CurrentPQPosture(resolved.Profile.ResultingPosture),
		SolutionProfileRef: ref,
		Maturity:           string(resolved.Profile.Maturity),
		ClaimStatus:        string(resolved.Profile.ClaimStatus),
	}
}

func softFindingsForProfile(profile *provider.SolutionProfile) []AssessmentFinding {
	soft := provider.EvaluateSoftFindings(profile)
	findings := make([]AssessmentFinding, 0, len(soft))
	for _, finding := range soft {
		findings = append(findings, AssessmentFinding{
			Code:     finding.Code,
			Message:  finding.Message,
			Severity: AssessmentFindingSeverityWarning,
			Field:    finding.Field,
		})
	}
	return findingsOrEmpty(findings)
}

func cloneSuggested(in *provider.SuggestedUserConstraints) *provider.SuggestedUserConstraints {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}
