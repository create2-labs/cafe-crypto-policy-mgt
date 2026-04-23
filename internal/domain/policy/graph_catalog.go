package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/create2-labs/cafe-cpm/internal/domain/vocabulary"
)

type PolicyNodeKind string

const (
	NodeKindAccount  PolicyNodeKind = "account"
	NodeKindSig      PolicyNodeKind = "signature"
	NodeKindVerifier PolicyNodeKind = "verifier"
	NodeKindTarget   PolicyNodeKind = "target_posture"
)

type PolicyParamType string

const (
	ParamTypeString PolicyParamType = "string"
	ParamTypeBool   PolicyParamType = "bool"
	ParamTypeInt    PolicyParamType = "int"
	ParamTypeEnum   PolicyParamType = "enum"
)

var (
	ErrCatalogVersionRequired = errors.New("catalog version is required")
	ErrCatalogNodesRequired   = errors.New("catalog must define at least one node")
	ErrCatalogNodeIDRequired  = errors.New("node id is required")
	ErrCatalogNodeIDMismatch  = errors.New("node id must match map key")
	ErrCatalogNodeKindInvalid = errors.New("node kind is invalid")
	ErrCatalogMaturityRange   = errors.New("node maturity must be between 1 and 5")
	ErrCatalogParamName       = errors.New("parameter name is required")
	ErrCatalogParamType       = errors.New("parameter type is invalid")
	ErrCatalogParamEnumValues = errors.New("enum parameter requires at least one allowed value")
	ErrCatalogRuleNodeUnknown = errors.New("compatibility rule references unknown node")
)

// PolicyGraphCatalog is the authoritative source for available policy graph nodes
// and admissible transitions between them.
type PolicyGraphCatalog struct {
	Version            string                          `json:"version"`
	Nodes              map[string]PolicyNodeDefinition `json:"nodes"`
	CompatibilityRules []PolicyCompatibilityRule       `json:"compatibility_rules"`
}

type PolicyNodeDefinition struct {
	ID                string                        `json:"id"`
	Kind              PolicyNodeKind                `json:"kind"`
	DisplayName       string                        `json:"display_name"`
	ParameterSchemas  []PolicyNodeParameterSchema   `json:"parameter_schemas,omitempty"`
	SupportedPostures []vocabulary.CurrentPQPosture `json:"supported_postures,omitempty"`
	Maturity          int                           `json:"maturity"`
}

type PolicyNodeParameterSchema struct {
	Name          string          `json:"name"`
	Type          PolicyParamType `json:"type"`
	Required      bool            `json:"required"`
	AllowedValues []string        `json:"allowed_values,omitempty"`
}

// PolicyCompatibilityRule captures typed transition semantics between consecutive nodes.
type PolicyCompatibilityRule struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason,omitempty"`
}

func LoadPolicyGraphCatalogFromFile(path string) (*PolicyGraphCatalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog file: %w", err)
	}
	var catalog PolicyGraphCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("decode catalog file: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func (c *PolicyGraphCatalog) Validate() error {
	if c == nil {
		return errors.New("catalog is nil")
	}
	if c.Version == "" {
		return ErrCatalogVersionRequired
	}
	if len(c.Nodes) == 0 {
		return ErrCatalogNodesRequired
	}

	for key, n := range c.Nodes {
		if n.ID == "" {
			return fmt.Errorf("%w: key=%q", ErrCatalogNodeIDRequired, key)
		}
		if n.ID != key {
			return fmt.Errorf("%w: key=%q id=%q", ErrCatalogNodeIDMismatch, key, n.ID)
		}
		if !isValidNodeKind(n.Kind) {
			return fmt.Errorf("%w: node=%q kind=%q", ErrCatalogNodeKindInvalid, n.ID, n.Kind)
		}
		if n.Maturity < 1 || n.Maturity > 5 {
			return fmt.Errorf("%w: node=%q maturity=%d", ErrCatalogMaturityRange, n.ID, n.Maturity)
		}
		for _, p := range n.ParameterSchemas {
			if p.Name == "" {
				return fmt.Errorf("%w: node=%q", ErrCatalogParamName, n.ID)
			}
			if !isValidParamType(p.Type) {
				return fmt.Errorf("%w: node=%q param=%q type=%q", ErrCatalogParamType, n.ID, p.Name, p.Type)
			}
			if p.Type == ParamTypeEnum && len(p.AllowedValues) == 0 {
				return fmt.Errorf("%w: node=%q param=%q", ErrCatalogParamEnumValues, n.ID, p.Name)
			}
		}
		for _, posture := range n.SupportedPostures {
			if !posture.IsValid() {
				return fmt.Errorf("node=%q has invalid supported posture %q", n.ID, posture)
			}
		}
	}
	for _, r := range c.CompatibilityRules {
		if _, ok := c.Nodes[r.FromNodeID]; !ok {
			return fmt.Errorf("%w: from_node_id=%q", ErrCatalogRuleNodeUnknown, r.FromNodeID)
		}
		if _, ok := c.Nodes[r.ToNodeID]; !ok {
			return fmt.Errorf("%w: to_node_id=%q", ErrCatalogRuleNodeUnknown, r.ToNodeID)
		}
	}
	return nil
}

func (c *PolicyGraphCatalog) IsTransitionAllowed(fromNodeID, toNodeID string) bool {
	for _, r := range c.CompatibilityRules {
		if r.FromNodeID == fromNodeID && r.ToNodeID == toNodeID {
			return r.Allowed
		}
	}
	return false
}

func (c *PolicyGraphCatalog) NodeIDs() []string {
	ids := make([]string, 0, len(c.Nodes))
	for id := range c.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func isValidNodeKind(kind PolicyNodeKind) bool {
	switch kind {
	case NodeKindAccount, NodeKindSig, NodeKindVerifier, NodeKindTarget:
		return true
	default:
		return false
	}
}

func isValidParamType(t PolicyParamType) bool {
	switch t {
	case ParamTypeString, ParamTypeBool, ParamTypeInt, ParamTypeEnum:
		return true
	default:
		return false
	}
}
