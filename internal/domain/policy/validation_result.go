package policy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ValidationSeverity is the normalized issue severity level.
type ValidationSeverity string

const (
	// ValidationSeverityError marks a blocking validation issue.
	ValidationSeverityError ValidationSeverity = "error"
)

// ValidationIssueCode is a stable machine-readable issue identifier.
type ValidationIssueCode string

const (
	// ValidationIssueCodeValidatorConfig indicates validator setup issues.
	ValidationIssueCodeValidatorConfig ValidationIssueCode = "validator_config"
	// ValidationIssueCodeInstanceRequired indicates a missing instance input.
	ValidationIssueCodeInstanceRequired ValidationIssueCode = "instance_required"
	// ValidationIssueCodeInstanceInvalid indicates instance validation failure.
	ValidationIssueCodeInstanceInvalid ValidationIssueCode = "instance_invalid"
)

// ValidationIssue captures one concrete validation failure.
type ValidationIssue struct {
	Code     ValidationIssueCode `json:"code"`
	Severity ValidationSeverity  `json:"severity"`
	Message  string              `json:"message"`
}

// CryptoPolicyValidationResult is the stable output model for instance validation.
type CryptoPolicyValidationResult struct {
	InstanceID       string                `json:"instance_id,omitempty"`
	CatalogVersion   string                `json:"catalog_version,omitempty"`
	Valid            bool                  `json:"valid"`
	Issues           []ValidationIssue     `json:"issues,omitempty"`
	NormalizedPolicy *CryptoPolicyInstance `json:"normalized_policy,omitempty"`
}

// CryptoPolicyInstanceValidator validates policy instances against a catalog and
// returns typed, serializable result data.
type CryptoPolicyInstanceValidator struct {
	catalog *PolicyGraphCatalog
}

// NewCryptoPolicyInstanceValidator builds a validator with explicit config checks.
func NewCryptoPolicyInstanceValidator(catalog *PolicyGraphCatalog) (*CryptoPolicyInstanceValidator, error) {
	if catalog == nil {
		return nil, errors.New("validator catalog is nil")
	}
	return &CryptoPolicyInstanceValidator{catalog: catalog}, nil
}

// Validate normalizes and validates an instance and returns a typed result.
func (v *CryptoPolicyInstanceValidator) Validate(instance *CryptoPolicyInstance) CryptoPolicyValidationResult {
	result := CryptoPolicyValidationResult{Valid: false}

	if v == nil || v.catalog == nil {
		result.Issues = []ValidationIssue{{
			Code:     ValidationIssueCodeValidatorConfig,
			Severity: ValidationSeverityError,
			Message:  "validator catalog is nil",
		}}
		return result
	}

	if instance == nil {
		result.Issues = []ValidationIssue{{
			Code:     ValidationIssueCodeInstanceRequired,
			Severity: ValidationSeverityError,
			Message:  "instance is nil",
		}}
		return result
	}

	result.InstanceID = instance.ID
	result.CatalogVersion = instance.CatalogVersion

	normalized, err := cloneCryptoPolicyInstance(instance)
	if err != nil {
		result.Issues = []ValidationIssue{{
			Code:     ValidationIssueCodeInstanceInvalid,
			Severity: ValidationSeverityError,
			Message:  fmt.Sprintf("clone instance for validation: %v", err),
		}}
		return result
	}

	if err := normalized.NormalizeAndValidate(v.catalog); err != nil {
		result.Issues = []ValidationIssue{{
			Code:     ValidationIssueCodeInstanceInvalid,
			Severity: ValidationSeverityError,
			Message:  err.Error(),
		}}
		return result
	}

	result.Valid = true
	result.NormalizedPolicy = normalized
	return result
}

func cloneCryptoPolicyInstance(in *CryptoPolicyInstance) (*CryptoPolicyInstance, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	var out CryptoPolicyInstance
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
