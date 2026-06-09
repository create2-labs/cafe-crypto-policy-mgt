package metrics

import "github.com/prometheus/client_golang/prometheus"

var registry = prometheus.NewRegistry()

// Registry exposes the CPM application metrics registry (IMM-OPS-1).
func Registry() prometheus.Gatherer {
	return registry
}

// ExploreNoDeployableCandidateTotal counts explore responses that return HTTP 200 with
// no ranked deployable candidate and at least one rejected candidate (IMM-OPS-1).
var ExploreNoDeployableCandidateTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "cpm_explore_no_deployable_candidate_total",
		Help: "Explore responses with no deployable candidate and non-empty rejected_candidates.",
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
