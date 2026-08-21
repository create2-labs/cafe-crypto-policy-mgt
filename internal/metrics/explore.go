package metrics

import "github.com/prometheus/client_golang/prometheus"

var registry = prometheus.NewRegistry()

// Registry exposes the CPM application metrics registry (IMM-OPS-1).
func Registry() prometheus.Gatherer {
	return registry
}

// ExploreNoDeployableCandidateTotal counts explore responses that return HTTP 200 with
// no scan-compatible provider and at least one rejected candidate
// (IMM-OPS-1 / ADR §7.2.1 family 2 runtime.no_scan_compatible / CPM-P11b).
var ExploreNoDeployableCandidateTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "cpm_explore_no_deployable_candidate_total",
		Help: "Explore responses with empty scan_compatible_providers and non-empty rejected_candidates (ADR runtime no-scan-compatible signal).",
	},
	[]string{"rejection_code", "wallet_type", "binding", "missing_chain_count"},
)

func init() {
	registry.MustRegister(ExploreNoDeployableCandidateTotal)
}

// IncExploreNoDeployableCandidate increments the explore no-candidate counter once per event.
func IncExploreNoDeployableCandidate(rejectionCode, walletType, binding, missingChainCount string) {
	ExploreNoDeployableCandidateTotal.WithLabelValues(
		rejectionCode,
		walletType,
		binding,
		missingChainCount,
	).Inc()
}
