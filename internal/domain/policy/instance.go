package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

var (
	// ErrInstanceIDRequired indicates a missing instance identifier.
	ErrInstanceIDRequired = errors.New("instance id is required")
	// ErrInstanceNameRequired indicates a missing instance name.
	ErrInstanceNameRequired = errors.New("instance name is required")
	// ErrInstanceCatalogVersionRequired indicates a missing catalog version.
	ErrInstanceCatalogVersionRequired = errors.New("instance catalog_version is required")
	// ErrInstanceTemplateIDRequired indicates a missing template identifier.
	ErrInstanceTemplateIDRequired = errors.New("instance template_id is required")
	// ErrScopeNameRequired indicates missing scope name.
	ErrScopeNameRequired = errors.New("scope name is required")
	// ErrScopeChainIDInvalid indicates non-positive chain IDs in scope.
	ErrScopeChainIDInvalid = errors.New("scope chain_ids must contain only positive values")
	// ErrScopeMultichainRequiresTargets indicates invalid multichain scope constraints.
	ErrScopeMultichainRequiresTargets = errors.New("scope require_multichain requires at least two chain_ids when chain_ids is specified")
	// ErrGlobalRequiredPostureInvalid indicates invalid global required posture.
	ErrGlobalRequiredPostureInvalid = errors.New("global parameters required_posture is invalid")
	// ErrGlobalMaturityRange indicates maturity outside [1,5].
	ErrGlobalMaturityRange = errors.New("global parameters minimum_maturity must be between 1 and 5")
	// ErrGlobalApprovalModeInvalid indicates invalid approval mode.
	ErrGlobalApprovalModeInvalid = errors.New("global parameters approval_mode is invalid")
	// ErrGlobalProviderModeInvalid indicates invalid provider mode.
	ErrGlobalProviderModeInvalid = errors.New("global parameters allowed_provider_modes contains an invalid value")
	// ErrGlobalKeyRotationModelInvalid indicates invalid key rotation model.
	ErrGlobalKeyRotationModelInvalid = errors.New("global parameters key_rotation_model is invalid")
	// ErrSolutionProfileRefRequired indicates a missing solution profile reference.
	ErrSolutionProfileRefRequired = errors.New("instance solution_profile_ref is required")
	// ErrSolutionProfileRefProviderIDRequired indicates a missing provider_id.
	ErrSolutionProfileRefProviderIDRequired = errors.New("solution_profile_ref provider_id is required")
	// ErrSolutionProfileRefSolutionIDRequired indicates a missing solution_profile_id.
	ErrSolutionProfileRefSolutionIDRequired = errors.New("solution_profile_ref solution_profile_id is required")
	// ErrSolutionProfileRefManifestVersionRequired indicates a missing manifest_version.
	ErrSolutionProfileRefManifestVersionRequired = errors.New("solution_profile_ref manifest_version is required")
)

// CryptoPolicyInstance is the concrete, scope-bound policy document used by CPM.
type CryptoPolicyInstance struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	CatalogVersion     string                 `json:"catalog_version"`
	TemplateID         string                 `json:"template_id"`
	SolutionProfileRef SolutionProfileRef     `json:"solution_profile_ref"`
	Scope              PolicyScope            `json:"scope"`
	GlobalParams       GlobalPolicyParameters `json:"global_parameters"`
	Governance         GovernanceMetadata     `json:"governance,omitempty"`
}

// SolutionProfileRef binds a catalogue instance to a Capability Provider solution
// profile (ADR Capability Providers). Explore resolves the ref via the provider
// registry and applies ADR §7 hard checks (CPM-P4).
type SolutionProfileRef struct {
	ProviderID        string `json:"provider_id"`
	SolutionProfileID string `json:"solution_profile_id"`
	ManifestVersion   string `json:"manifest_version"`
	VerificationDate  string `json:"verification_date,omitempty"`
}

// PolicyScope identifies where the instance applies.
type PolicyScope struct {
	Name              string   `json:"name"`
	SubjectType       string   `json:"subject_type,omitempty"`
	SubjectIDs        []string `json:"subject_ids,omitempty"`
	ChainIDs          []int64  `json:"chain_ids,omitempty"`
	RequireMultichain bool     `json:"require_multichain"`
}

// GlobalPolicyParameters stores typed global constraints/defaults.
type GlobalPolicyParameters struct {
	RequiredPosture           vocabulary.CurrentPQPosture `json:"required_posture"`
	MinimumMaturity           int                         `json:"minimum_maturity"`
	ApprovalMode              ApprovalMode                `json:"approval_mode"`
	AllowResearch             bool                        `json:"allow_research"`
	AllowNewWallet            bool                        `json:"allow_new_wallet"`
	AddressContinuityRequired bool                        `json:"address_continuity_required"`
	KeyRotationModel          KeyRotationModel            `json:"key_rotation_model"`
	RecoveryRequired          bool                        `json:"recovery_required"`
	AllowedProviderModes      []ProviderMode              `json:"allowed_provider_modes,omitempty"`
	RequireBundlerAvailable   bool                        `json:"require_bundler_available"`
	RequirePaymasterAvailable bool                        `json:"require_paymaster_available"`
}

// GovernanceMetadata stores optional instance governance attributes.
type GovernanceMetadata struct {
	OwnerTeam      string   `json:"owner_team,omitempty"`
	ChangeTicketID string   `json:"change_ticket_id,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// LoadCryptoPolicyInstanceFromFile reads, decodes, normalizes, and validates an
// instance.
func LoadCryptoPolicyInstanceFromFile(path string) (*CryptoPolicyInstance, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read instance file: %w", err)
	}
	var in CryptoPolicyInstance
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("decode instance file: %w", err)
	}
	if err := in.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	return &in, nil
}

// Normalize applies deterministic canonicalization/defaulting for stable behavior.
func (i *CryptoPolicyInstance) Normalize() {
	if i == nil {
		return
	}

	i.SolutionProfileRef.ProviderID = strings.TrimSpace(i.SolutionProfileRef.ProviderID)
	i.SolutionProfileRef.SolutionProfileID = strings.TrimSpace(i.SolutionProfileRef.SolutionProfileID)
	i.SolutionProfileRef.ManifestVersion = strings.TrimSpace(i.SolutionProfileRef.ManifestVersion)
	i.SolutionProfileRef.VerificationDate = strings.TrimSpace(i.SolutionProfileRef.VerificationDate)
	i.Scope.ChainIDs = normalizeChainIDs(i.Scope.ChainIDs)
	i.Scope.SubjectIDs = normalizeStringsPreserveOrder(i.Scope.SubjectIDs)
	i.GlobalParams.AllowedProviderModes = normalizeProviderModes(i.GlobalParams.AllowedProviderModes)
	i.Governance.Tags = normalizeStringsPreserveOrder(i.Governance.Tags)

	if i.GlobalParams.MinimumMaturity == 0 {
		i.GlobalParams.MinimumMaturity = defaultMinimumMaturity
	}
	if i.GlobalParams.ApprovalMode == "" {
		i.GlobalParams.ApprovalMode = ApprovalModeManual
	}
	if i.GlobalParams.KeyRotationModel == "" {
		i.GlobalParams.KeyRotationModel = KeyRotationNone
	}
}

// Validate checks instance consistency.
func (i *CryptoPolicyInstance) Validate() error {
	if i == nil {
		return errors.New("instance is nil")
	}
	if i.ID == "" {
		return ErrInstanceIDRequired
	}
	if i.Name == "" {
		return ErrInstanceNameRequired
	}
	if i.CatalogVersion == "" {
		return ErrInstanceCatalogVersionRequired
	}
	if i.TemplateID == "" {
		return ErrInstanceTemplateIDRequired
	}
	if err := i.SolutionProfileRef.Validate(); err != nil {
		return err
	}

	if err := i.Scope.Validate(); err != nil {
		return fmt.Errorf("scope: %w", err)
	}
	if err := i.GlobalParams.Validate(); err != nil {
		return fmt.Errorf("global_parameters: %w", err)
	}
	return nil
}

// Validate checks solution_profile_ref identity fields.
func (r SolutionProfileRef) Validate() error {
	if r.ProviderID == "" && r.SolutionProfileID == "" && r.ManifestVersion == "" && r.VerificationDate == "" {
		return ErrSolutionProfileRefRequired
	}
	if r.ProviderID == "" {
		return ErrSolutionProfileRefProviderIDRequired
	}
	if r.SolutionProfileID == "" {
		return ErrSolutionProfileRefSolutionIDRequired
	}
	if r.ManifestVersion == "" {
		return ErrSolutionProfileRefManifestVersionRequired
	}
	return nil
}

// Validate checks scope consistency.
func (s *PolicyScope) Validate() error {
	if s == nil {
		return errors.New("scope is nil")
	}
	if s.Name == "" {
		return ErrScopeNameRequired
	}
	for _, chainID := range s.ChainIDs {
		if chainID <= 0 {
			return fmt.Errorf("%w: %d", ErrScopeChainIDInvalid, chainID)
		}
	}
	if s.RequireMultichain && len(s.ChainIDs) > 0 && len(s.ChainIDs) < 2 {
		return ErrScopeMultichainRequiresTargets
	}
	return nil
}

// Validate checks global parameter constraints.
func (p *GlobalPolicyParameters) Validate() error {
	if p == nil {
		return errors.New("global parameters are nil")
	}
	if p.RequiredPosture == "" || !p.RequiredPosture.IsValid() || p.RequiredPosture == vocabulary.PQPostureUnknown {
		return fmt.Errorf("%w: %q", ErrGlobalRequiredPostureInvalid, p.RequiredPosture)
	}
	if p.MinimumMaturity < 1 || p.MinimumMaturity > 5 {
		return fmt.Errorf("%w: %d", ErrGlobalMaturityRange, p.MinimumMaturity)
	}
	if !isValidApprovalMode(p.ApprovalMode) {
		return fmt.Errorf("%w: %q", ErrGlobalApprovalModeInvalid, p.ApprovalMode)
	}
	if !isValidKeyRotationModel(p.KeyRotationModel) {
		return fmt.Errorf("%w: %q", ErrGlobalKeyRotationModelInvalid, p.KeyRotationModel)
	}
	for _, mode := range p.AllowedProviderModes {
		if !isValidProviderMode(mode) {
			return fmt.Errorf("%w: %q", ErrGlobalProviderModeInvalid, mode)
		}
	}
	return nil
}

// NormalizeAndValidate applies deterministic canonicalization then validation.
func (i *CryptoPolicyInstance) NormalizeAndValidate() error {
	i.Normalize()
	return i.Validate()
}
