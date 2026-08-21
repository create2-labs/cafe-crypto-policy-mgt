package metrics

import "github.com/prometheus/client_golang/prometheus"

// CataloguePostureOrphanTotal counts Crypto Policies with posture-only orphanage
// detected at catalogue load (ADR §7.2.1 / CPM-P11a).
var CataloguePostureOrphanTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "cpm_catalogue_posture_orphan_total",
		Help: "Crypto Policies with no posture-compatible allowed provider at catalogue load.",
	},
)

// CatalogueMalformedManifestTotal counts solution profiles whose
// suggested_user_constraints contradict constraints at catalogue load (CPM-P11a).
var CatalogueMalformedManifestTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "cpm_catalogue_malformed_manifest_total",
		Help: "Solution profiles with contradictory suggested_user_constraints at catalogue load.",
	},
)

func init() {
	registry.MustRegister(CataloguePostureOrphanTotal)
	registry.MustRegister(CatalogueMalformedManifestTotal)
}

// AddCataloguePostureOrphans increments the posture-orphan counter by n (no-op if n <= 0).
func AddCataloguePostureOrphans(n int) {
	if n > 0 {
		CataloguePostureOrphanTotal.Add(float64(n))
	}
}

// AddCatalogueMalformedManifests increments the malformed-manifest counter by n (no-op if n <= 0).
func AddCatalogueMalformedManifests(n int) {
	if n > 0 {
		CatalogueMalformedManifestTotal.Add(float64(n))
	}
}
