package policy

import (
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestPolicyDecisionEvaluator_SelectsBestCompatibleAndKeepsRejected(t *testing.T) {
	catalog, tpl, base := mustLoadFixtures(t)
	req := baseSelection()
	obs := baseObservation()

	best := shallowCopyInstance(base)
	best.ID = "cp002"
	best.GlobalParams.MinimumMaturity = 4
	best.GlobalParams.AllowNewWallet = false
	if err := best.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("best instance: %v", err)
	}

	second := shallowCopyInstance(base)
	second.ID = "cp001"
	second.GlobalParams.MinimumMaturity = 2
	second.GlobalParams.AllowNewWallet = true
	if err := second.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("second instance: %v", err)
	}

	rejected := shallowCopyInstance(base)
	rejected.ID = "cp003"
	rejected.GlobalParams.RequiredPosture = vocabulary.PQPostureClassicalOnly
	if err := rejected.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("rejected instance: %v", err)
	}

	decision, err := (PolicyDecisionEvaluator{}).Evaluate(
		obs,
		&req,
		[]PolicyDecisionCandidate{
			{Instance: second, Template: tpl},
			{Instance: rejected, Template: tpl},
			{Instance: best, Template: tpl},
		},
		catalog,
	)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if got, want := decision.SelectedPolicyID, "CP002"; got != want {
		t.Fatalf("selected_policy_id: got %q want %q", got, want)
	}
	if got, want := len(decision.RankedCandidates), 2; got != want {
		t.Fatalf("ranked_candidates: got %d want %d", got, want)
	}
	if got, want := len(decision.RejectedCandidates), 1; got != want {
		t.Fatalf("rejected_candidates: got %d want %d", got, want)
	}
	if decision.RejectedCandidates[0].PolicyID != "CP003" {
		t.Fatalf("rejected policy id: got %q", decision.RejectedCandidates[0].PolicyID)
	}
}

func TestPolicyDecisionEvaluator_DeterministicNormalizedPolicyIDTieBreak(t *testing.T) {
	catalog, tpl, base := mustLoadFixtures(t)
	req := baseSelection()
	obs := baseObservation()

	first := shallowCopyInstance(base)
	first.ID = "cp010"
	if err := first.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("first instance: %v", err)
	}

	second := shallowCopyInstance(base)
	second.ID = "CP002"
	if err := second.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("second instance: %v", err)
	}

	decision, err := (PolicyDecisionEvaluator{}).Evaluate(
		obs,
		&req,
		[]PolicyDecisionCandidate{
			{Instance: first, Template: tpl},
			{Instance: second, Template: tpl},
		},
		catalog,
	)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if got, want := decision.SelectedPolicyID, "CP002"; got != want {
		t.Fatalf("selected_policy_id: got %q want %q", got, want)
	}
	if got, want := decision.RankedCandidates[0].PolicyID, "CP002"; got != want {
		t.Fatalf("first ranked policy_id: got %q want %q", got, want)
	}
	if got, want := decision.RankedCandidates[1].PolicyID, "CP010"; got != want {
		t.Fatalf("second ranked policy_id: got %q want %q", got, want)
	}
}

func TestPolicyDecisionEvaluator_PrefersBetterTargetPostureAlignment(t *testing.T) {
	catalog, tpl, base := mustLoadFixtures(t)
	req := baseSelection()
	obs := baseObservation()

	exact := shallowCopyInstance(base)
	exact.ID = "cp100"
	exact.GlobalParams.RequiredPosture = vocabulary.PQPostureHybrid
	if err := exact.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("exact instance: %v", err)
	}

	over := shallowCopyInstance(base)
	over.ID = "cp101"
	over.GlobalParams.RequiredPosture = vocabulary.PQPostureFullPQ
	if err := over.NormalizeAndValidate(catalog); err != nil {
		t.Fatalf("over instance: %v", err)
	}

	decision, err := (PolicyDecisionEvaluator{}).Evaluate(
		obs,
		&req,
		[]PolicyDecisionCandidate{
			{Instance: over, Template: tpl},
			{Instance: exact, Template: tpl},
		},
		catalog,
	)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if got, want := decision.SelectedPolicyID, "CP100"; got != want {
		t.Fatalf("selected_policy_id: got %q want %q", got, want)
	}
}
