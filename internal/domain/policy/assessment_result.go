package policy

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrAssessmentResultIDRequired indicates a missing assessment identifier.
	ErrAssessmentResultIDRequired = errors.New("assessment_result id is required")
	// ErrAssessmentResultInstanceIDRequired indicates a missing policy instance reference.
	ErrAssessmentResultInstanceIDRequired = errors.New("assessment_result crypto_policy_instance_id is required")
	// ErrAssessmentResultWalletAddressRequired indicates a missing wallet address reference.
	ErrAssessmentResultWalletAddressRequired = errors.New("assessment_result wallet_ref.address is required")
	// ErrAssessmentResultStatusInvalid indicates an unknown assessment status.
	ErrAssessmentResultStatusInvalid = errors.New("assessment_result status is invalid")
	// ErrAssessmentResultEvaluatedAtRequired indicates a missing evaluation timestamp.
	ErrAssessmentResultEvaluatedAtRequired = errors.New("assessment_result evaluated_at is required")
	// ErrAssessmentFindingCodeRequired indicates a missing finding code.
	ErrAssessmentFindingCodeRequired = errors.New("assessment_result finding code is required")
	// ErrAssessmentFindingSeverityInvalid indicates an unknown finding severity.
	ErrAssessmentFindingSeverityInvalid = errors.New("assessment_result finding severity is invalid")
)

// AssessmentStatus describes the high-level outcome for policy assessment.
type AssessmentStatus string

const (
	// AssessmentStatusPending indicates assessment is not finalized yet.
	AssessmentStatusPending AssessmentStatus = "pending"
	// AssessmentStatusCompatibleAndDeployable indicates assessment passed and deployment can proceed.
	AssessmentStatusCompatibleAndDeployable AssessmentStatus = "compatible_and_deployable"
	// AssessmentStatusCompatibleButNotDeployable indicates compatibility exists but deployability constraints remain.
	AssessmentStatusCompatibleButNotDeployable AssessmentStatus = "compatible_but_not_deployable"
	// AssessmentStatusIncompatible indicates the evaluated route is incompatible.
	AssessmentStatusIncompatible AssessmentStatus = "incompatible"
	// AssessmentStatusError indicates assessment failed due to processing/runtime error.
	AssessmentStatusError AssessmentStatus = "error"
)

// IsValid reports whether the status value is part of the stable contract.
func (s AssessmentStatus) IsValid() bool {
	switch s {
	case AssessmentStatusPending,
		AssessmentStatusCompatibleAndDeployable,
		AssessmentStatusCompatibleButNotDeployable,
		AssessmentStatusIncompatible,
		AssessmentStatusError:
		return true
	default:
		return false
	}
}

// AssessmentFindingSeverity classifies finding importance.
type AssessmentFindingSeverity string

const (
	// AssessmentFindingSeverityInfo is purely informational.
	AssessmentFindingSeverityInfo AssessmentFindingSeverity = "info"
	// AssessmentFindingSeverityWarning indicates non-blocking caution.
	AssessmentFindingSeverityWarning AssessmentFindingSeverity = "warning"
	// AssessmentFindingSeverityBlocking indicates a hard blocker.
	AssessmentFindingSeverityBlocking AssessmentFindingSeverity = "blocking"
)

// IsValid reports whether the severity value is part of the stable contract.
func (s AssessmentFindingSeverity) IsValid() bool {
	switch s {
	case AssessmentFindingSeverityInfo,
		AssessmentFindingSeverityWarning,
		AssessmentFindingSeverityBlocking:
		return true
	default:
		return false
	}
}

// AssessmentWalletReference identifies the wallet used as assessment input.
type AssessmentWalletReference struct {
	Address string `json:"address"`
	ChainID int64  `json:"chain_id,omitempty"`
}

// AssessmentFinding captures one explainable compatibility/assessment signal.
type AssessmentFinding struct {
	Code     string                    `json:"code"`
	Message  string                    `json:"message,omitempty"`
	Severity AssessmentFindingSeverity `json:"severity"`
	Field    string                    `json:"field,omitempty"`
	Details  map[string]string         `json:"details,omitempty"`
}

// CryptoPolicyAssessmentResult is the stable, serializable policy assessment output.
type CryptoPolicyAssessmentResult struct {
	ID                     string                    `json:"id"`
	CryptoPolicyInstanceID string                    `json:"crypto_policy_instance_id"`
	TemplateID             string                    `json:"template_id,omitempty"`
	WalletRef              AssessmentWalletReference `json:"wallet_ref"`
	Status                 AssessmentStatus          `json:"status"`
	EvaluatedAt            time.Time                 `json:"evaluated_at"`
	Findings               []AssessmentFinding       `json:"findings,omitempty"`
	Warnings               []string                  `json:"warnings,omitempty"`
	CatalogVersion         string                    `json:"catalog_version,omitempty"`
	TemplateVersion        string                    `json:"template_version,omitempty"`
	CorrelationID          string                    `json:"correlation_id,omitempty"`
}

// NewCryptoPolicyAssessmentResult builds a stable result envelope with explicit defaults.
func NewCryptoPolicyAssessmentResult(
	assessmentID, instanceID string,
	walletRef AssessmentWalletReference,
	status AssessmentStatus,
) CryptoPolicyAssessmentResult {
	return CryptoPolicyAssessmentResult{
		ID:                     assessmentID,
		CryptoPolicyInstanceID: instanceID,
		WalletRef:              walletRef,
		Status:                 status,
		EvaluatedAt:            time.Now().UTC(),
		Findings:               make([]AssessmentFinding, 0),
		Warnings:               make([]string, 0),
	}
}

// Validate enforces required fields and stable enum values.
func (r *CryptoPolicyAssessmentResult) Validate() error {
	if r == nil {
		return errors.New("assessment_result is nil")
	}
	if r.ID == "" {
		return ErrAssessmentResultIDRequired
	}
	if r.CryptoPolicyInstanceID == "" {
		return ErrAssessmentResultInstanceIDRequired
	}
	if r.WalletRef.Address == "" {
		return ErrAssessmentResultWalletAddressRequired
	}
	if !r.Status.IsValid() {
		return fmt.Errorf("%w: %q", ErrAssessmentResultStatusInvalid, r.Status)
	}
	if r.EvaluatedAt.IsZero() {
		return ErrAssessmentResultEvaluatedAtRequired
	}
	for idx, finding := range r.Findings {
		if finding.Code == "" {
			return fmt.Errorf("%w: index %d", ErrAssessmentFindingCodeRequired, idx)
		}
		if !finding.Severity.IsValid() {
			return fmt.Errorf("%w: index %d (%q)", ErrAssessmentFindingSeverityInvalid, idx, finding.Severity)
		}
	}
	return nil
}
