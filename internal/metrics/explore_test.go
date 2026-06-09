package metrics

import (
	"testing"
)

func TestExploreNoDeployableCandidateTotal_exposedAfterIncrement(t *testing.T) {
	IncExploreNoDeployableCandidate("incompatible.chain_scope", "eoa", "discovery", "1")

	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	found := false
	for _, mf := range mfs {
		if mf.GetName() != "cpm_explore_no_deployable_candidate_total" {
			continue
		}
		found = true
		if len(mf.GetMetric()) != 1 {
			t.Fatalf("metric samples: got %d want 1", len(mf.GetMetric()))
		}
	}
	if !found {
		t.Fatal("cpm_explore_no_deployable_candidate_total not gathered after increment")
	}
}

func TestExploreNoDeployableCandidateTotal_acceptsLabels(t *testing.T) {
	if _, err := ExploreNoDeployableCandidateTotal.GetMetricWithLabelValues(
		"incompatible.chain_scope", "eoa", "discovery", "1",
	); err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
}
