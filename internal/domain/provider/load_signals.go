package provider

import (
	"log"
)

// loadSignalsLogger is the minimal logger used for catalogue load signals (CPM-P11a).
type loadSignalsLogger interface {
	Printf(format string, v ...any)
}

// ApplyManifestLoadSignals marks profiles whose suggested_user_constraints
// contradict constraints / signature (ADR §7.2.1 / §6.2 rule 9). Load stays
// permissive: contradictions are logged and flagged Erroneous, not rejected.
// Returns the number of profiles marked erroneous.
func (r *Registry) ApplyManifestLoadSignals(logger loadSignalsLogger) int {
	if r == nil || r.byKey == nil {
		return 0
	}
	if logger == nil {
		logger = log.Default()
	}
	n := 0
	for _, resolved := range r.byKey {
		if resolved == nil {
			continue
		}
		if err := ValidateSuggestedUserConstraints(&resolved.Profile); err != nil {
			resolved.Erroneous = true
			n++
			logger.Printf(
				"ERROR catalogue: malformed suggested_user_constraints provider=%s profile=%s err=%v",
				resolved.ProviderID, resolved.Profile.SolutionProfileID, err,
			)
		}
	}
	return n
}
