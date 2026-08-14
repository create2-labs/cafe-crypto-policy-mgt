package policy

import (
	"errors"
	"fmt"
	"slices"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

var (
	// ErrCompatibilityRequestNil indicates a nil policy selection request.
	ErrCompatibilityRequestNil = errors.New("compatibility: policy selection request is nil")
	// ErrCompatibilityInstanceNil indicates a nil policy instance.
	ErrCompatibilityInstanceNil = errors.New("compatibility: crypto policy instance is nil")
	// ErrCompatibilityTemplateRequired indicates a candidate without its normative template.
	ErrCompatibilityTemplateRequired = errors.New("compatibility: matching template is required")
	// ErrCompatibilityTemplateMismatch indicates a template id mismatch.
	ErrCompatibilityTemplateMismatch = errors.New("compatibility: instance template_id does not match provided template")
)

// PolicyCompatibilityResult is the first-version, explainable output of
// compatibility-only evaluation (before ranking; see PR12).
// Status uses the same string values as CryptoPolicyAssessmentResult for the
// three main outcomes, excluding pending/error.
type PolicyCompatibilityResult struct {
	Status   AssessmentStatus    `json:"status"`
	Findings []AssessmentFinding `json:"findings,omitempty"`
}

// PolicyCompatibilityEvaluator evaluates a single policy instance against a
// normalized wallet observation and a selection request.
type PolicyCompatibilityEvaluator struct {
	// Providers resolves instance solution_profile_ref for ADR §7 hard checks.
	// Evaluation fails closed when the registry or exact profile reference is unavailable.
	Providers *provider.Registry
}

// Evaluate runs deterministic compatibility rules for a provider candidate.
func (e PolicyCompatibilityEvaluator) Evaluate(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
	tpl *CryptoPolicyTemplate,
) (PolicyCompatibilityResult, error) {
	if req == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityRequestNil
	}
	if inst == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityInstanceNil
	}
	if err := req.NormalizeAndValidate(); err != nil {
		return PolicyCompatibilityResult{}, err
	}
	if err := inst.NormalizeAndValidate(nil); err != nil {
		return PolicyCompatibilityResult{}, err
	}
	if tpl == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityTemplateRequired
	}
	if tpl.ID != inst.TemplateID {
		return PolicyCompatibilityResult{}, fmt.Errorf("%w: instance template_id %q != template %q", ErrCompatibilityTemplateMismatch, inst.TemplateID, tpl.ID)
	}
	if err := tpl.NormalizeAndValidate(nil); err != nil {
		return PolicyCompatibilityResult{}, err
	}
	return e.evaluateProviderCandidate(observation, req, inst, tpl), nil
}

func (e PolicyCompatibilityEvaluator) evaluateProviderCandidate(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
	tpl *CryptoPolicyTemplate,
) PolicyCompatibilityResult {
	obsChains := normalizeChainSet(observation.ChainIDs)

	checks := []func() *PolicyCompatibilityResult{
		func() *PolicyCompatibilityResult { return checkPostureCompatibility(req, tpl) },
		func() *PolicyCompatibilityResult { return e.checkProviderCompatibility(observation, req, inst, tpl) },
		func() *PolicyCompatibilityResult { return checkRequiredCapabilities(req, inst) },
		func() *PolicyCompatibilityResult { return checkProviderModes(req, inst) },
		func() *PolicyCompatibilityResult {
			return checkChainCompatibility(req.TargetChainIDs, inst.Scope.ChainIDs, obsChains)
		},
		func() *PolicyCompatibilityResult {
			return checkMultichainCompatibility(req, inst.Scope.ChainIDs, obsChains)
		},
	}
	for _, check := range checks {
		if result := check(); result != nil {
			return *result
		}
	}

	return PolicyCompatibilityResult{Status: AssessmentStatusCompatibleAndDeployable}
}

func checkPostureCompatibility(req *PolicySelectionRequest, tpl *CryptoPolicyTemplate) *PolicyCompatibilityResult {
	if req.TargetPosture == tpl.RequiredPosture {
		return nil
	}
	return incompatibleResult(fieldFinding(
		provider.FindingCodePosture,
		fmt.Sprintf("selection target_posture %q does not equal template required_posture %q", req.TargetPosture, tpl.RequiredPosture),
		"required_posture",
	))
}

// checkProviderCompatibility performs the ADR §7 provider hard checks.
// Soft findings are CPM-P5.
func (e PolicyCompatibilityEvaluator) checkProviderCompatibility(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
	tpl *CryptoPolicyTemplate,
) *PolicyCompatibilityResult {
	if e.Providers == nil {
		return incompatibleResult(fieldFinding(provider.FindingCodeUnresolved, "provider registry is unavailable", "solution_profile_ref"))
	}

	ref := provider.ProfileRef{
		ProviderID:        inst.SolutionProfileRef.ProviderID,
		SolutionProfileID: inst.SolutionProfileRef.SolutionProfileID,
		ManifestVersion:   inst.SolutionProfileRef.ManifestVersion,
	}
	resolved, ok := e.Providers.Lookup(ref)
	if !ok {
		return incompatibleResult(fieldFinding(
			provider.FindingCodeUnresolved,
			fmt.Sprintf("solution_profile_ref %s/%s (manifest_version %q) not found in provider registry",
				ref.ProviderID, ref.SolutionProfileID, ref.ManifestVersion),
			"solution_profile_ref",
		))
	}

	hard := provider.EvaluateHardCompatibility(
		provider.HardObservation{AccountKind: observation.AccountKind, ChainIDs: observation.ChainIDs},
		provider.HardSelectionRequest{
			RequiredPosture:           string(tpl.RequiredPosture),
			TargetChainIDs:            req.TargetChainIDs,
			AllowNewWallet:            req.AllowNewWallet,
			AddressContinuityRequired: req.AddressContinuityRequired,
			KeyRotationModel:          string(req.KeyRotationModel),
		},
		&resolved.Profile,
	)
	if len(hard) == 0 {
		return nil
	}

	findings := make([]AssessmentFinding, 0, len(hard))
	for _, finding := range hard {
		findings = append(findings, fieldFinding(finding.Code, finding.Message, finding.Field))
	}
	return incompatibleResult(findings...)
}

func checkRequiredCapabilities(req *PolicySelectionRequest, inst *CryptoPolicyInstance) *PolicyCompatibilityResult {
	switch {
	case req.RecoveryRequired && !inst.GlobalParams.RecoveryRequired:
		return incompatibleResult(blockingFinding("incompatible.constraint", "recovery_required not met by instance"))
	case req.RequireBundlerAvailable && !inst.GlobalParams.RequireBundlerAvailable:
		return incompatibleResult(blockingFinding("incompatible.constraint", "require_bundler_available not met by instance"))
	case req.RequirePaymasterAvailable && !inst.GlobalParams.RequirePaymasterAvailable:
		return incompatibleResult(blockingFinding("incompatible.constraint", "require_paymaster_available not met by instance"))
	default:
		return nil
	}
}

func checkProviderModes(req *PolicySelectionRequest, inst *CryptoPolicyInstance) *PolicyCompatibilityResult {
	if len(req.AllowedProviderModes) == 0 {
		return nil
	}
	allowed := make(map[ProviderMode]struct{}, len(req.AllowedProviderModes))
	for _, mode := range req.AllowedProviderModes {
		allowed[mode] = struct{}{}
	}
	for _, mode := range inst.GlobalParams.AllowedProviderModes {
		if _, ok := allowed[mode]; !ok {
			return incompatibleResult(blockingFinding(
				"incompatible.provider_mode",
				fmt.Sprintf("instance provider mode %q not allowed by selection", mode),
			))
		}
	}
	return nil
}

func checkChainCompatibility(targets, scope []int64, observed map[int64]bool) *PolicyCompatibilityResult {
	if len(scope) == 0 {
		return &PolicyCompatibilityResult{
			Status: AssessmentStatusCompatibleButNotDeployable,
			Findings: []AssessmentFinding{{
				Code:     "compatible_but_not_deployable",
				Message:  "instance scope has no chain_ids; route is not deployable to concrete chains",
				Severity: AssessmentFindingSeverityInfo,
			}},
		}
	}
	if len(targets) == 0 {
		return checkObservedScopeChains(scope, observed)
	}
	for _, chainID := range targets {
		if !slices.Contains(scope, chainID) {
			return incompatibleResult(blockingFinding(
				"incompatible.chain_scope",
				fmt.Sprintf("target_chain_id %d not covered by instance scope", chainID),
			))
		}
		if !observed[chainID] {
			return incompatibleResult(blockingFinding(
				"incompatible.chain_not_observed",
				fmt.Sprintf("target_chain_id %d not present in wallet observation", chainID),
			))
		}
	}
	return nil
}

func checkObservedScopeChains(scope []int64, observed map[int64]bool) *PolicyCompatibilityResult {
	for _, chainID := range scope {
		if !observed[chainID] {
			return incompatibleResult(blockingFinding(
				"incompatible.chain_not_observed",
				fmt.Sprintf("scope chain_id %d not present in wallet observation", chainID),
			))
		}
	}
	return nil
}

func checkMultichainCompatibility(
	req *PolicySelectionRequest,
	scope []int64,
	observed map[int64]bool,
) *PolicyCompatibilityResult {
	if !req.RequireMultichain {
		return nil
	}
	candidates := req.TargetChainIDs
	if len(candidates) == 0 {
		candidates = scope
	}
	if countMatchingChains(candidates, scope, observed) >= 2 {
		return nil
	}
	return incompatibleResult(blockingFinding(
		"incompatible.multichain",
		"require_multichain not satisfied: fewer than two matching observed chains in scope",
	))
}

func countMatchingChains(candidates, scope []int64, observed map[int64]bool) int {
	seen := make(map[int64]struct{}, len(candidates))
	matching := 0
	for _, chainID := range candidates {
		if _, duplicate := seen[chainID]; duplicate {
			continue
		}
		seen[chainID] = struct{}{}
		if slices.Contains(scope, chainID) && observed[chainID] {
			matching++
		}
	}
	return matching
}

func blockingFinding(code, message string) AssessmentFinding {
	return AssessmentFinding{Code: code, Message: message, Severity: AssessmentFindingSeverityBlocking}
}

func fieldFinding(code, message, field string) AssessmentFinding {
	finding := blockingFinding(code, message)
	finding.Field = field
	return finding
}

func incompatibleResult(findings ...AssessmentFinding) *PolicyCompatibilityResult {
	return &PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
}

func normalizeChainSet(ids []int64) map[int64]bool {
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id > 0 {
			m[id] = true
		}
	}
	return m
}
