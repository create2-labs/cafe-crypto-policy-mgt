package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

var (
	// ErrTemplateIDRequired indicates a missing template identifier.
	ErrTemplateIDRequired = errors.New("template id is required")
	// ErrTemplateNameRequired indicates a missing template name.
	ErrTemplateNameRequired = errors.New("template name is required")
	// ErrTemplateVersionRequired indicates a missing template version.
	ErrTemplateVersionRequired = errors.New("template version is required")
	// ErrTemplateCatalogVersionRequired indicates a missing catalog version reference.
	ErrTemplateCatalogVersionRequired = errors.New("template catalog_version is required")
	// ErrTemplateRequiredPostureRequired indicates a missing required posture.
	ErrTemplateRequiredPostureRequired = errors.New("template required_posture is required")
	// ErrTemplateRequiredPostureInvalid indicates an invalid required posture.
	ErrTemplateRequiredPostureInvalid = errors.New("template required_posture is invalid")
	// ErrTemplateNodeUnknown indicates a node not present in the catalog (legacy path only).
	ErrTemplateNodeUnknown = errors.New("template node_path references unknown node")
	// ErrTemplateTransitionInvalid indicates a disallowed transition in node_path (legacy path only).
	ErrTemplateTransitionInvalid = errors.New("template node_path transition is not allowed by catalog rules")
	// ErrTemplateMinMaturityRange indicates a maturity value outside [1, 5].
	ErrTemplateMinMaturityRange = errors.New("template constraints minimum_maturity must be between 1 and 5")
	// ErrTemplateChainIDInvalid indicates non-positive target chain IDs.
	ErrTemplateChainIDInvalid = errors.New("template constraints target_chain_ids must contain only positive values")
	// ErrTemplateMultichainRequiresTargets indicates invalid multichain constraints.
	ErrTemplateMultichainRequiresTargets = errors.New("template constraints require_multichain requires at least two target_chain_ids when target_chain_ids is specified")
	// ErrTemplateMetadataPostureMismatch indicates inconsistent posture metadata.
	ErrTemplateMetadataPostureMismatch = errors.New("template metadata required_posture must match template required_posture")
	// ErrTemplateSelectionPostureMismatch indicates inconsistent default selection posture.
	ErrTemplateSelectionPostureMismatch = errors.New("template default_selection target_posture must match template required_posture")
)

// CryptoPolicyTemplate defines a reusable CAFE intention (required posture + defaults).
// node_path is legacy/non-normative (CPM-P4b); solution profiles carry concrete technique.
type CryptoPolicyTemplate struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Version         string                      `json:"version"`
	CatalogVersion  string                      `json:"catalog_version"`
	Description     string                      `json:"description,omitempty"`
	RequiredPosture vocabulary.CurrentPQPosture `json:"required_posture"`
	NodePath        []string                    `json:"node_path,omitempty"` // legacy; not required
	Defaults        PolicySelectionRequest      `json:"default_selection"`
	Constraints     TemplateConstraints         `json:"constraints,omitempty"`
	Metadata        TemplateMetadata            `json:"metadata,omitempty"`
}

// TemplateConstraints captures template-level restrictions independent from
// runtime observation compatibility checks.
type TemplateConstraints struct {
	MinimumMaturity   int     `json:"minimum_maturity,omitempty"`
	RequireMultichain bool    `json:"require_multichain"`
	TargetChainIDs    []int64 `json:"target_chain_ids,omitempty"`
}

// TemplateMetadata stores optional documentation and discoverability attributes.
type TemplateMetadata struct {
	OwnerTeam       string                      `json:"owner_team,omitempty"`
	RequiredPosture vocabulary.CurrentPQPosture `json:"required_posture,omitempty"`
	Tags            []string                    `json:"tags,omitempty"`
}

// LoadCryptoPolicyTemplateFromFile reads, decodes, normalizes, and validates a
// template. catalog is still accepted for legacy node_path validation when present.
func LoadCryptoPolicyTemplateFromFile(path string, catalog *PolicyGraphCatalog) (*CryptoPolicyTemplate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}
	var tpl CryptoPolicyTemplate
	if err := json.Unmarshal(raw, &tpl); err != nil {
		return nil, fmt.Errorf("decode template file: %w", err)
	}
	if err := tpl.NormalizeAndValidate(catalog); err != nil {
		return nil, err
	}
	return &tpl, nil
}

// Normalize applies explicit canonicalization/defaulting for stable behavior.
func (t *CryptoPolicyTemplate) Normalize() {
	if t == nil {
		return
	}

	t.Constraints.TargetChainIDs = normalizeChainIDs(t.Constraints.TargetChainIDs)
	t.Metadata.Tags = normalizeStringsPreserveOrder(t.Metadata.Tags)
	t.Defaults.TargetChainIDs = normalizeChainIDs(t.Defaults.TargetChainIDs)
	t.Defaults.AllowedProviderModes = normalizeProviderModes(t.Defaults.AllowedProviderModes)
	t.Defaults.PreferredFamilies = normalizeStringsPreserveOrder(t.Defaults.PreferredFamilies)
	t.Defaults.PreferredProviders = normalizeStringsPreserveOrder(t.Defaults.PreferredProviders)
	t.NodePath = normalizeStringsPreserveOrder(t.NodePath)

	if t.Constraints.MinimumMaturity == 0 {
		t.Constraints.MinimumMaturity = defaultMinimumMaturity
	}
	if t.Defaults.MinimumMaturity == 0 {
		t.Defaults.MinimumMaturity = t.Constraints.MinimumMaturity
	}
	if t.Defaults.TargetPosture == "" {
		t.Defaults.TargetPosture = t.RequiredPosture
	}
	if t.Defaults.ApprovalMode == "" {
		t.Defaults.ApprovalMode = ApprovalModeManual
	}
	if t.Defaults.KeyRotationModel == "" {
		t.Defaults.KeyRotationModel = KeyRotationNone
	}
}

// Validate ensures template integrity. When node_path is empty, graph rules are skipped.
func (t *CryptoPolicyTemplate) Validate(catalog *PolicyGraphCatalog) error {
	if t == nil {
		return errors.New("template is nil")
	}
	if t.ID == "" {
		return ErrTemplateIDRequired
	}
	if t.Name == "" {
		return ErrTemplateNameRequired
	}
	if t.Version == "" {
		return ErrTemplateVersionRequired
	}
	if t.CatalogVersion == "" {
		return ErrTemplateCatalogVersionRequired
	}
	if t.RequiredPosture == "" {
		return ErrTemplateRequiredPostureRequired
	}
	if !t.RequiredPosture.IsValid() || t.RequiredPosture == vocabulary.PQPostureUnknown {
		return fmt.Errorf("%w: %q", ErrTemplateRequiredPostureInvalid, t.RequiredPosture)
	}

	if len(t.NodePath) > 0 {
		if catalog == nil {
			return errors.New("catalog is nil")
		}
		for _, nodeID := range t.NodePath {
			if _, ok := catalog.Nodes[nodeID]; !ok {
				return fmt.Errorf("%w: %q", ErrTemplateNodeUnknown, nodeID)
			}
		}
		for i := 0; i < len(t.NodePath)-1; i++ {
			from, to := t.NodePath[i], t.NodePath[i+1]
			if !catalog.IsTransitionAllowed(from, to) {
				return fmt.Errorf("%w: %q -> %q", ErrTemplateTransitionInvalid, from, to)
			}
		}
	}

	for _, id := range t.Constraints.TargetChainIDs {
		if id <= 0 {
			return fmt.Errorf("%w: %d", ErrTemplateChainIDInvalid, id)
		}
	}
	if t.Constraints.RequireMultichain && len(t.Constraints.TargetChainIDs) > 0 && len(t.Constraints.TargetChainIDs) < 2 {
		return ErrTemplateMultichainRequiresTargets
	}
	if t.Constraints.MinimumMaturity < 1 || t.Constraints.MinimumMaturity > 5 {
		return fmt.Errorf("%w: %d", ErrTemplateMinMaturityRange, t.Constraints.MinimumMaturity)
	}

	if t.Metadata.RequiredPosture != "" && t.Metadata.RequiredPosture != t.RequiredPosture {
		return fmt.Errorf("%w: metadata=%q template=%q", ErrTemplateMetadataPostureMismatch, t.Metadata.RequiredPosture, t.RequiredPosture)
	}
	if t.Defaults.TargetPosture != t.RequiredPosture {
		return fmt.Errorf("%w: default=%q template=%q", ErrTemplateSelectionPostureMismatch, t.Defaults.TargetPosture, t.RequiredPosture)
	}
	if err := t.Defaults.Validate(); err != nil {
		return fmt.Errorf("template default_selection: %w", err)
	}
	return nil
}

// NormalizeAndValidate applies normalization then validates the template.
func (t *CryptoPolicyTemplate) NormalizeAndValidate(catalog *PolicyGraphCatalog) error {
	t.Normalize()
	return t.Validate(catalog)
}
