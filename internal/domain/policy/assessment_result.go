package policy

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

// AssessmentFinding captures one explainable compatibility/assessment signal.
type AssessmentFinding struct {
	Code     string                    `json:"code"`
	Message  string                    `json:"message,omitempty"`
	Severity AssessmentFindingSeverity `json:"severity"`
	Field    string                    `json:"field,omitempty"`
	Details  map[string]string         `json:"details,omitempty"`
}
