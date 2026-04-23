package policy

import (
	"errors"
	"fmt"
	"slices"

	"github.com/create2-labs/cafe-cpm/internal/domain/vocabulary"
)

// ApprovalMode defines the operator approval behavior for the selected policy path.
type ApprovalMode string

const (
	ApprovalModeAuto   ApprovalMode = "auto"
	ApprovalModeManual ApprovalMode = "manual"
)

// ProviderMode constrains where remediation operations can be executed.
type ProviderMode string

const (
	ProviderModeThirdParty  ProviderMode = "third_party"
	ProviderModeUserManaged ProviderMode = "user_managed"
)

const (
	defaultMinimumMaturity = 1
)

var (
	ErrTargetPostureRequired        = errors.New("target_posture is required")
	ErrTargetPostureInvalid         = errors.New("target_posture is invalid")
	ErrTargetChainIDInvalid         = errors.New("target_chain_ids must contain only positive values")
	ErrMultichainRequiresTwoTargets = errors.New("require_multichain requires at least two target_chain_ids when target_chain_ids is specified")
	ErrMinimumMaturityRange         = errors.New("minimum_maturity must be between 1 and 5")
	ErrApprovalModeInvalid          = errors.New("approval_mode is invalid")
	ErrProviderModeInvalid          = errors.New("allowed_provider_modes contains an invalid value")
)

// PolicySelectionRequest is the stable, serializable contract that drives CPM policy selection.
//
// Canonical multi-chain semantics:
//   - target_chain_ids defines the required target deployment scope.
//   - require_multichain forces a multi-chain deployable outcome.
//   - when require_multichain=true and target_chain_ids is provided, at least two target chains are required.
type PolicySelectionRequest struct {
	TargetPosture             vocabulary.CurrentPQPosture `json:"target_posture"`
	TargetChainIDs            []int64                     `json:"target_chain_ids,omitempty"`
	RequireMultichain         bool                        `json:"require_multichain"`
	AllowNewWallet            bool                        `json:"allow_new_wallet"`
	AddressContinuityRequired bool                        `json:"address_continuity_required"`
	KeyRotationRequired       bool                        `json:"key_rotation_required"`
	RecoveryRequired          bool                        `json:"recovery_required"`
	MinimumMaturity           int                         `json:"minimum_maturity"`
	AllowResearch             bool                        `json:"allow_research"`
	AllowedProviderModes      []ProviderMode              `json:"allowed_provider_modes,omitempty"`
	PreferredFamilies         []string                    `json:"preferred_families,omitempty"`
	PreferredProviders        []string                    `json:"preferred_providers,omitempty"`
	RequireBundlerAvailable   bool                        `json:"require_bundler_available"`
	RequirePaymasterAvailable bool                        `json:"require_paymaster_available"`
	ApprovalMode              ApprovalMode                `json:"approval_mode"`
}

// Normalize applies explicit defaults and stable canonicalization for deterministic behavior.
func (r *PolicySelectionRequest) Normalize() {
	if r == nil {
		return
	}
	if r.MinimumMaturity == 0 {
		r.MinimumMaturity = defaultMinimumMaturity
	}
	if r.ApprovalMode == "" {
		r.ApprovalMode = ApprovalModeManual
	}

	r.TargetChainIDs = normalizeChainIDs(r.TargetChainIDs)
	r.AllowedProviderModes = normalizeProviderModes(r.AllowedProviderModes)
	r.PreferredFamilies = normalizeStringsPreserveOrder(r.PreferredFamilies)
	r.PreferredProviders = normalizeStringsPreserveOrder(r.PreferredProviders)
}

// Validate checks request consistency after normalization.
func (r *PolicySelectionRequest) Validate() error {
	if r == nil {
		return errors.New("policy selection request is nil")
	}
	if r.TargetPosture == "" {
		return ErrTargetPostureRequired
	}
	if !r.TargetPosture.IsValid() {
		return fmt.Errorf("%w: %q", ErrTargetPostureInvalid, r.TargetPosture)
	}

	for _, id := range r.TargetChainIDs {
		if id <= 0 {
			return fmt.Errorf("%w: %d", ErrTargetChainIDInvalid, id)
		}
	}
	if r.RequireMultichain && len(r.TargetChainIDs) > 0 && len(r.TargetChainIDs) < 2 {
		return ErrMultichainRequiresTwoTargets
	}
	if r.MinimumMaturity < 1 || r.MinimumMaturity > 5 {
		return fmt.Errorf("%w: %d", ErrMinimumMaturityRange, r.MinimumMaturity)
	}
	if !isValidApprovalMode(r.ApprovalMode) {
		return fmt.Errorf("%w: %q", ErrApprovalModeInvalid, r.ApprovalMode)
	}
	for _, mode := range r.AllowedProviderModes {
		if !isValidProviderMode(mode) {
			return fmt.Errorf("%w: %q", ErrProviderModeInvalid, mode)
		}
	}
	return nil
}

// NormalizeAndValidate applies deterministic defaulting/canonicalization then validates.
func (r *PolicySelectionRequest) NormalizeAndValidate() error {
	r.Normalize()
	return r.Validate()
}

func normalizeChainIDs(in []int64) []int64 {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, id := range in {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func normalizeProviderModes(in []ProviderMode) []ProviderMode {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[ProviderMode]struct{}, len(in))
	out := make([]ProviderMode, 0, len(in))
	for _, mode := range in {
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}

func normalizeStringsPreserveOrder(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isValidApprovalMode(mode ApprovalMode) bool {
	switch mode {
	case ApprovalModeAuto, ApprovalModeManual:
		return true
	default:
		return false
	}
}

func isValidProviderMode(mode ProviderMode) bool {
	switch mode {
	case ProviderModeThirdParty, ProviderModeUserManaged:
		return true
	default:
		return false
	}
}
