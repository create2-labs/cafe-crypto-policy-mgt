package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
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
	// ErrInstanceReferenceMissing indicates no template or path was supplied.
	ErrInstanceReferenceMissing = errors.New("instance must define template_id or node_path")
	// ErrInstanceReferenceAmbiguous indicates both template and explicit path were supplied.
	ErrInstanceReferenceAmbiguous = errors.New("instance cannot define both template_id and node_path")
	// ErrInstanceNodePathRequired indicates missing node path when node parameters are provided.
	ErrInstanceNodePathRequired = errors.New("instance node_path is required when node_parameters are provided")
	// ErrInstanceNodeUnknown indicates node path references unknown nodes.
	ErrInstanceNodeUnknown = errors.New("instance node_path references unknown node")
	// ErrInstanceTransitionInvalid indicates a transition not allowed by catalog rules.
	ErrInstanceTransitionInvalid = errors.New("instance node_path transition is not allowed by catalog rules")
	// ErrScopeNameRequired indicates missing scope name.
	ErrScopeNameRequired = errors.New("scope name is required")
	// ErrScopeChainIDInvalid indicates non-positive chain IDs in scope.
	ErrScopeChainIDInvalid = errors.New("scope chain_ids must contain only positive values")
	// ErrScopeMultichainRequiresTargets indicates invalid multichain scope constraints.
	ErrScopeMultichainRequiresTargets = errors.New("scope require_multichain requires at least two chain_ids when chain_ids is specified")
	// ErrGlobalTargetPostureInvalid indicates invalid global target posture.
	ErrGlobalTargetPostureInvalid = errors.New("global parameters target_posture is invalid")
	// ErrGlobalMaturityRange indicates maturity outside [1,5].
	ErrGlobalMaturityRange = errors.New("global parameters minimum_maturity must be between 1 and 5")
	// ErrGlobalApprovalModeInvalid indicates invalid approval mode.
	ErrGlobalApprovalModeInvalid = errors.New("global parameters approval_mode is invalid")
	// ErrGlobalProviderModeInvalid indicates invalid provider mode.
	ErrGlobalProviderModeInvalid = errors.New("global parameters allowed_provider_modes contains an invalid value")
	// ErrNodeParameterNodeUnknown indicates parameter map references unknown node.
	ErrNodeParameterNodeUnknown = errors.New("node_parameters references unknown node")
	// ErrNodeParameterNodeNotInPath indicates parameter map references node not selected in path.
	ErrNodeParameterNodeNotInPath = errors.New("node_parameters references a node that is not part of node_path")
	// ErrNodeParameterUnknown indicates unknown parameter for a node.
	ErrNodeParameterUnknown = errors.New("node parameter is not defined in catalog schema")
	// ErrNodeParameterRequiredMissing indicates missing required schema parameter.
	ErrNodeParameterRequiredMissing = errors.New("required node parameter is missing")
	// ErrNodeParameterTypeMismatch indicates node parameter type mismatch.
	ErrNodeParameterTypeMismatch = errors.New("node parameter value does not match catalog schema type")
	// ErrNodeParameterEnumValueInvalid indicates enum value outside schema.
	ErrNodeParameterEnumValueInvalid = errors.New("node parameter enum value is not allowed by catalog schema")
)

// CryptoPolicyInstance is the concrete, scope-bound policy document used by CPM.
// It intentionally has no edge-level configurable parameters; transition semantics
// are enforced by catalog/template compatibility rules.
type CryptoPolicyInstance struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	CatalogVersion string                      `json:"catalog_version"`
	TemplateID     string                      `json:"template_id,omitempty"`
	NodePath       []string                    `json:"node_path,omitempty"`
	Scope          PolicyScope                 `json:"scope"`
	GlobalParams   GlobalPolicyParameters      `json:"global_parameters"`
	NodeParameters map[string]NodeParameterMap `json:"node_parameters,omitempty"`
	Governance     GovernanceMetadata          `json:"governance,omitempty"`
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
	TargetPosture             vocabulary.CurrentPQPosture `json:"target_posture"`
	MinimumMaturity           int                         `json:"minimum_maturity"`
	ApprovalMode              ApprovalMode                `json:"approval_mode"`
	AllowResearch             bool                        `json:"allow_research"`
	AllowNewWallet            bool                        `json:"allow_new_wallet"`
	AddressContinuityRequired bool                        `json:"address_continuity_required"`
	KeyRotationRequired       bool                        `json:"key_rotation_required"`
	RecoveryRequired          bool                        `json:"recovery_required"`
	AllowedProviderModes      []ProviderMode              `json:"allowed_provider_modes,omitempty"`
	RequireBundlerAvailable   bool                        `json:"require_bundler_available"`
	RequirePaymasterAvailable bool                        `json:"require_paymaster_available"`
}

// NodeParameterMap is the typed per-node parameter assignment.
type NodeParameterMap map[string]NodeParameterValue

// NodeParameterValue is a typed container for node parameter values.
type NodeParameterValue struct {
	Type        PolicyParamType `json:"type"`
	StringValue string          `json:"string_value,omitempty"`
	BoolValue   *bool           `json:"bool_value,omitempty"`
	IntValue    *int64          `json:"int_value,omitempty"`
}

// GovernanceMetadata stores optional instance governance attributes.
type GovernanceMetadata struct {
	OwnerTeam      string   `json:"owner_team,omitempty"`
	ChangeTicketID string   `json:"change_ticket_id,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// LoadCryptoPolicyInstanceFromFile reads, decodes, normalizes, and validates an
// instance against the provided graph catalog.
func LoadCryptoPolicyInstanceFromFile(path string, catalog *PolicyGraphCatalog) (*CryptoPolicyInstance, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read instance file: %w", err)
	}
	var in CryptoPolicyInstance
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("decode instance file: %w", err)
	}
	if err := in.NormalizeAndValidate(catalog); err != nil {
		return nil, err
	}
	return &in, nil
}

// Normalize applies deterministic canonicalization/defaulting for stable behavior.
func (i *CryptoPolicyInstance) Normalize() {
	if i == nil {
		return
	}

	i.NodePath = normalizeStringsPreserveOrder(i.NodePath)
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
}

// Validate checks instance consistency and catalog/schema compatibility.
func (i *CryptoPolicyInstance) Validate(catalog *PolicyGraphCatalog) error {
	if i == nil {
		return errors.New("instance is nil")
	}
	if catalog == nil {
		return errors.New("catalog is nil")
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
	if len(i.NodeParameters) > 0 && len(i.NodePath) == 0 {
		return ErrInstanceNodePathRequired
	}
	if i.TemplateID == "" && len(i.NodePath) == 0 {
		return ErrInstanceReferenceMissing
	}
	if i.TemplateID != "" && len(i.NodePath) > 0 {
		return ErrInstanceReferenceAmbiguous
	}

	if err := i.Scope.Validate(); err != nil {
		return fmt.Errorf("scope: %w", err)
	}
	if err := i.GlobalParams.Validate(); err != nil {
		return fmt.Errorf("global_parameters: %w", err)
	}

	if len(i.NodePath) > 0 {
		for _, nodeID := range i.NodePath {
			if _, ok := catalog.Nodes[nodeID]; !ok {
				return fmt.Errorf("%w: %q", ErrInstanceNodeUnknown, nodeID)
			}
		}
		for idx := 0; idx < len(i.NodePath)-1; idx++ {
			from, to := i.NodePath[idx], i.NodePath[idx+1]
			if !catalog.IsTransitionAllowed(from, to) {
				return fmt.Errorf("%w: %q -> %q", ErrInstanceTransitionInvalid, from, to)
			}
		}
	}

	if err := i.validateNodeParameters(catalog); err != nil {
		return err
	}
	return nil
}

func (i *CryptoPolicyInstance) validateNodeParameters(catalog *PolicyGraphCatalog) error {
	if len(i.NodeParameters) == 0 {
		return nil
	}

	nodePathSet := make(map[string]struct{}, len(i.NodePath))
	for _, nodeID := range i.NodePath {
		nodePathSet[nodeID] = struct{}{}
	}

	for nodeID, params := range i.NodeParameters {
		nodeDef, ok := catalog.Nodes[nodeID]
		if !ok {
			return fmt.Errorf("%w: %q", ErrNodeParameterNodeUnknown, nodeID)
		}
		if len(i.NodePath) > 0 {
			if _, inPath := nodePathSet[nodeID]; !inPath {
				return fmt.Errorf("%w: %q", ErrNodeParameterNodeNotInPath, nodeID)
			}
		}

		paramSchemaByName := make(map[string]PolicyNodeParameterSchema, len(nodeDef.ParameterSchemas))
		for _, schema := range nodeDef.ParameterSchemas {
			paramSchemaByName[schema.Name] = schema
		}

		for paramName, value := range params {
			schema, exists := paramSchemaByName[paramName]
			if !exists {
				return fmt.Errorf("%w: node=%q param=%q", ErrNodeParameterUnknown, nodeID, paramName)
			}
			if err := value.ValidateAgainstSchema(schema); err != nil {
				return fmt.Errorf("node=%q param=%q: %w", nodeID, paramName, err)
			}
		}

		for _, schema := range nodeDef.ParameterSchemas {
			if !schema.Required {
				continue
			}
			if _, exists := params[schema.Name]; !exists {
				return fmt.Errorf("%w: node=%q param=%q", ErrNodeParameterRequiredMissing, nodeID, schema.Name)
			}
		}
	}

	return nil
}

// ValidateAgainstSchema validates a typed value against a node parameter schema.
func (v NodeParameterValue) ValidateAgainstSchema(schema PolicyNodeParameterSchema) error {
	if v.Type != schema.Type {
		return fmt.Errorf("%w: got=%q want=%q", ErrNodeParameterTypeMismatch, v.Type, schema.Type)
	}
	switch schema.Type {
	case ParamTypeString:
		if v.BoolValue != nil || v.IntValue != nil {
			return fmt.Errorf("%w: string value cannot include bool/int payload", ErrNodeParameterTypeMismatch)
		}
	case ParamTypeBool:
		if v.BoolValue == nil || v.IntValue != nil || v.StringValue != "" {
			return fmt.Errorf("%w: bool value must use bool_value only", ErrNodeParameterTypeMismatch)
		}
	case ParamTypeInt:
		if v.IntValue == nil || v.BoolValue != nil || v.StringValue != "" {
			return fmt.Errorf("%w: int value must use int_value only", ErrNodeParameterTypeMismatch)
		}
	case ParamTypeEnum:
		if v.BoolValue != nil || v.IntValue != nil {
			return fmt.Errorf("%w: enum value must use string_value only", ErrNodeParameterTypeMismatch)
		}
		if len(schema.AllowedValues) == 0 {
			return fmt.Errorf("%w: schema has no allowed values", ErrNodeParameterEnumValueInvalid)
		}
		normalized := make([]string, len(schema.AllowedValues))
		copy(normalized, schema.AllowedValues)
		for idx := range normalized {
			normalized[idx] = strings.TrimSpace(normalized[idx])
		}
		if !slices.Contains(normalized, v.StringValue) {
			return fmt.Errorf("%w: %q", ErrNodeParameterEnumValueInvalid, v.StringValue)
		}
	default:
		return fmt.Errorf("%w: unsupported schema type %q", ErrNodeParameterTypeMismatch, schema.Type)
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
	if p.TargetPosture == "" || !p.TargetPosture.IsValid() {
		return fmt.Errorf("%w: %q", ErrGlobalTargetPostureInvalid, p.TargetPosture)
	}
	if p.MinimumMaturity < 1 || p.MinimumMaturity > 5 {
		return fmt.Errorf("%w: %d", ErrGlobalMaturityRange, p.MinimumMaturity)
	}
	if !isValidApprovalMode(p.ApprovalMode) {
		return fmt.Errorf("%w: %q", ErrGlobalApprovalModeInvalid, p.ApprovalMode)
	}
	for _, mode := range p.AllowedProviderModes {
		if !isValidProviderMode(mode) {
			return fmt.Errorf("%w: %q", ErrGlobalProviderModeInvalid, mode)
		}
	}
	return nil
}

// NormalizeAndValidate applies deterministic canonicalization then validation.
func (i *CryptoPolicyInstance) NormalizeAndValidate(catalog *PolicyGraphCatalog) error {
	i.Normalize()
	return i.Validate(catalog)
}
