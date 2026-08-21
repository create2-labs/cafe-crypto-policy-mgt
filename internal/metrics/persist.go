package metrics

import "github.com/prometheus/client_golang/prometheus"

// PersistUserConstraintsIncompatibleTotal counts persist gates that fail couche B
// (ADR §7.2.1 family 2 / CPM-P11b): scan-compatible provider rejected by user_constraints.
var PersistUserConstraintsIncompatibleTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "cpm_persist_user_constraints_incompatible_total",
		Help: "Persist requests rejected because user_constraints fail couche B against the accepted provider snapshot.",
	},
)

func init() {
	registry.MustRegister(PersistUserConstraintsIncompatibleTotal)
}

// IncPersistUserConstraintsIncompatible increments the couche B persist signal counter.
func IncPersistUserConstraintsIncompatible() {
	PersistUserConstraintsIncompatibleTotal.Inc()
}
