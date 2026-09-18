package policy

import "errors"

var (
	// ErrSolutionProfileRefRequired indicates a missing solution profile reference.
	ErrSolutionProfileRefRequired = errors.New("instance solution_profile_ref is required")
	// ErrSolutionProfileRefProviderIDRequired indicates a missing provider_id.
	ErrSolutionProfileRefProviderIDRequired = errors.New("solution_profile_ref provider_id is required")
	// ErrSolutionProfileRefSolutionIDRequired indicates a missing solution_profile_id.
	ErrSolutionProfileRefSolutionIDRequired = errors.New("solution_profile_ref solution_profile_id is required")
	// ErrSolutionProfileRefManifestVersionRequired indicates a missing manifest_version.
	ErrSolutionProfileRefManifestVersionRequired = errors.New("solution_profile_ref manifest_version is required")
)

// SolutionProfileRef binds a catalogue Crypto Policy / persist payload to a
// Capability Provider solution profile (ADR Capability Providers).
type SolutionProfileRef struct {
	ProviderID        string `json:"provider_id"`
	SolutionProfileID string `json:"solution_profile_id"`
	ManifestVersion   string `json:"manifest_version"`
	VerificationDate  string `json:"verification_date,omitempty"`
}

// Validate checks required solution_profile_ref fields.
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
