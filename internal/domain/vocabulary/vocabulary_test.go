package vocabulary

import "testing"

func TestCurrentPQPosture_IsValid(t *testing.T) {
	for _, p := range []CurrentPQPosture{
		PQPostureClassicalOnly,
		PQPostureHybrid,
		PQPostureFullPQ,
		PQPostureUnknown,
	} {
		if !p.IsValid() {
			t.Fatalf("expected valid: %q", p)
		}
	}
	if CurrentPQPosture("invalid").IsValid() {
		t.Fatal("expected invalid posture")
	}
}
