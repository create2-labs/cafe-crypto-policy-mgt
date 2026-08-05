package policy

import (
	"errors"
	"fmt"
	"slices"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

var (
	// ErrCompatibilityRequestNil indicates a nil policy selection request.
	ErrCompatibilityRequestNil = errors.New("compatibility: policy selection request is nil")
	// ErrCompatibilityInstanceNil indicates a nil policy instance.
	ErrCompatibilityInstanceNil = errors.New("compatibility: crypto policy instance is nil")
	// ErrCompatibilityCatalogNil indicates a nil graph catalog.
	ErrCompatibilityCatalogNil = errors.New("compatibility: policy graph catalog is nil")
	// ErrCompatibilityNodePathRequired indicates a missing resolvable node path.
	ErrCompatibilityNodePathRequired = errors.New("compatibility: instance needs node_path or a matching template for node_path resolution")
	// ErrCompatibilityTemplateMismatch indicates template id / catalog mismatch.
	ErrCompatibilityTemplateMismatch = errors.New("compatibility: template id does not match provided template or catalog version differs")
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
type PolicyCompatibilityEvaluator struct{}

// Evaluate runs deterministic compatibility rules from the CPM v0.7 workstream.
// tpl must be provided when the instance only references template_id and does
// not materialize node_path, so the effective path can be resolved.
func (PolicyCompatibilityEvaluator) Evaluate(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
	catalog *PolicyGraphCatalog,
	tpl *CryptoPolicyTemplate,
) (PolicyCompatibilityResult, error) {
	if req == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityRequestNil
	}
	if inst == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityInstanceNil
	}
	if catalog == nil {
		return PolicyCompatibilityResult{}, ErrCompatibilityCatalogNil
	}
	if err := req.NormalizeAndValidate(); err != nil {
		return PolicyCompatibilityResult{}, err
	}
	if err := inst.NormalizeAndValidate(catalog); err != nil {
		return PolicyCompatibilityResult{}, err
	}
	if tpl != nil {
		if err := tpl.NormalizeAndValidate(catalog); err != nil {
			return PolicyCompatibilityResult{}, err
		}
	}
	path, err := effectiveNodePath(inst, tpl, catalog)
	if err != nil {
		return PolicyCompatibilityResult{}, err
	}
	return evaluateWithPath(observation, req, inst, catalog, path), nil
}

func effectiveNodePath(
	inst *CryptoPolicyInstance,
	tpl *CryptoPolicyTemplate,
	catalog *PolicyGraphCatalog,
) ([]string, error) {
	if len(inst.NodePath) > 0 {
		if inst.TemplateID != "" && tpl != nil && inst.TemplateID != tpl.ID {
			return nil, fmt.Errorf("%w: instance template_id %q != template %q", ErrCompatibilityTemplateMismatch, inst.TemplateID, tpl.ID)
		}
		return inst.NodePath, nil
	}
	if inst.TemplateID == "" {
		return nil, ErrCompatibilityNodePathRequired
	}
	if tpl == nil {
		return nil, ErrCompatibilityNodePathRequired
	}
	if tpl.ID != inst.TemplateID {
		return nil, fmt.Errorf("%w: want template %q", ErrCompatibilityTemplateMismatch, inst.TemplateID)
	}
	if tpl.CatalogVersion != inst.CatalogVersion {
		return nil, fmt.Errorf("%w: instance catalog %q template catalog %q", ErrCompatibilityTemplateMismatch, inst.CatalogVersion, tpl.CatalogVersion)
	}
	if catalog != nil && len(tpl.NodePath) > 0 {
		for _, id := range tpl.NodePath {
			if _, ok := catalog.Nodes[id]; !ok {
				return nil, fmt.Errorf("template node_path: %w: %q", ErrInstanceNodeUnknown, id)
			}
		}
	}
	if len(tpl.NodePath) == 0 {
		return nil, ErrCompatibilityNodePathRequired
	}
	return tpl.NodePath, nil
}

// postureRank maps PQ posture to a monotonic strength for request satisfaction.
// unknown is -1 (cannot satisfy a concrete request by itself).
func postureRank(p vocabulary.CurrentPQPosture) int {
	switch p {
	case vocabulary.PQPostureClassicalOnly:
		return 0
	case vocabulary.PQPostureHybrid:
		return 1
	case vocabulary.PQPostureFullPQ:
		return 2
	case vocabulary.PQPostureUnknown:
		return -1
	default:
		return -1
	}
}

func evaluateWithPath(
	observation walletobserved.Payload,
	req *PolicySelectionRequest,
	inst *CryptoPolicyInstance,
	catalog *PolicyGraphCatalog,
	nodePath []string,
) PolicyCompatibilityResult {
	var findings []AssessmentFinding
	add := func(code, msg string, sev AssessmentFindingSeverity) {
		findings = append(findings, AssessmentFinding{Code: code, Message: msg, Severity: sev})
	}

	// Policy must reach at least the posture requested for this assessment.
	if rp, ip := postureRank(req.TargetPosture), postureRank(inst.GlobalParams.TargetPosture); ip < rp {
		add("incompatible.target_posture", "instance target_posture does not satisfy selection target_posture", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}
	if inst.GlobalParams.TargetPosture == vocabulary.PQPostureUnknown {
		add("incompatible.target_posture", "instance target_posture is unknown", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}

	for _, nodeID := range nodePath {
		n := catalog.Nodes[nodeID]
		if n.Maturity < req.MinimumMaturity {
			add("incompatible.maturity", fmt.Sprintf("node %q below requested minimum_maturity", nodeID), AssessmentFindingSeverityBlocking)
			return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
		}
	}

	if req.KeyRotationModel != inst.GlobalParams.KeyRotationModel {
		add("incompatible.constraint", "key_rotation_model not met by instance", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}
	if req.AddressContinuityRequired && !inst.GlobalParams.AddressContinuityRequired {
		add("incompatible.constraint", "address_continuity_required not met by instance", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}
	if req.RecoveryRequired && !inst.GlobalParams.RecoveryRequired {
		add("incompatible.constraint", "recovery_required not met by instance", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}
	if req.RequireBundlerAvailable && !inst.GlobalParams.RequireBundlerAvailable {
		add("incompatible.constraint", "require_bundler_available not met by instance", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}
	if req.RequirePaymasterAvailable && !inst.GlobalParams.RequirePaymasterAvailable {
		add("incompatible.constraint", "require_paymaster_available not met by instance", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}

	if !req.AllowNewWallet && inst.GlobalParams.AllowNewWallet {
		add("incompatible.new_wallet", "instance allows new wallet but selection disallows it", AssessmentFindingSeverityBlocking)
		return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
	}

	if len(req.AllowedProviderModes) > 0 {
		allowed := make(map[ProviderMode]struct{}, len(req.AllowedProviderModes))
		for _, m := range req.AllowedProviderModes {
			allowed[m] = struct{}{}
		}
		for _, m := range inst.GlobalParams.AllowedProviderModes {
			if _, ok := allowed[m]; !ok {
				add("incompatible.provider_mode", fmt.Sprintf("instance provider mode %q not allowed by selection", m), AssessmentFindingSeverityBlocking)
				return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
			}
		}
	}

	// Target (last) node must advertise the instance global target posture when constrained.
	if len(nodePath) > 0 {
		lastID := nodePath[len(nodePath)-1]
		last := catalog.Nodes[lastID]
		if len(last.SupportedPostures) > 0 && !slices.Contains(last.SupportedPostures, inst.GlobalParams.TargetPosture) {
			add("incompatible.path_posture", "last path node does not support instance target_posture", AssessmentFindingSeverityBlocking)
			return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
		}
	}

	obsChains := normalizeChainSet(observation.ChainIDs)
	scope := inst.Scope.ChainIDs
	T := req.TargetChainIDs
	P := scope

	if len(P) == 0 {
		return PolicyCompatibilityResult{Status: AssessmentStatusCompatibleButNotDeployable, Findings: []AssessmentFinding{{
			Code:     "compatible_but_not_deployable",
			Message:  "instance scope has no chain_ids; route is not deployable to concrete chains",
			Severity: AssessmentFindingSeverityInfo,
		}}}
	}

	if len(T) > 0 {
		for _, c := range T {
			if !slices.Contains(P, c) {
				add("incompatible.chain_scope", fmt.Sprintf("target_chain_id %d not covered by instance scope", c), AssessmentFindingSeverityBlocking)
				return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
			}
			if !obsChains[c] {
				add("incompatible.chain_not_observed", fmt.Sprintf("target_chain_id %d not present in wallet observation", c), AssessmentFindingSeverityBlocking)
				return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
			}
		}
	} else {
		// No explicit request targets: every scope chain must still be observed for deployability.
		for _, c := range P {
			if !obsChains[c] {
				add("incompatible.chain_not_observed", fmt.Sprintf("scope chain_id %d not present in wallet observation", c), AssessmentFindingSeverityBlocking)
				return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
			}
		}
	}

	if req.RequireMultichain {
		var candidateChains []int64
		if len(T) > 0 {
			candidateChains = T
		} else {
			candidateChains = P
		}
		seenID := make(map[int64]struct{})
		matching := 0
		for _, c := range candidateChains {
			if _, dup := seenID[c]; dup {
				continue
			}
			seenID[c] = struct{}{}
			if slices.Contains(P, c) && obsChains[c] {
				matching++
			}
		}
		if matching < 2 {
			add("incompatible.multichain", "require_multichain not satisfied: fewer than two matching observed chains in scope", AssessmentFindingSeverityBlocking)
			return PolicyCompatibilityResult{Status: AssessmentStatusIncompatible, Findings: findings}
		}
	}

	return PolicyCompatibilityResult{Status: AssessmentStatusCompatibleAndDeployable, Findings: findings}
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
