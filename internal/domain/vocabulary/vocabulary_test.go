package vocabulary

import "testing"

func TestAccountKind_IsValid(t *testing.T) {
	for _, k := range []AccountKind{
		AccountKindEOA,
		AccountKindERC4337SmartAccount,
		AccountKindDelegatedEOA7702,
		AccountKindContractAccount,
		AccountKindUnknown,
	} {
		if !k.IsValid() {
			t.Fatalf("expected valid: %q", k)
		}
	}
	if AccountKind("not_a_kind").IsValid() {
		t.Fatal("expected invalid account kind")
	}
}

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

func TestIsValidAlgorithmID(t *testing.T) {
	cases := []struct {
		id    string
		valid bool
	}{
		{"secp256k1_ecrecover", true},
		{"mldsa44", true},
		{"mldsa65", true},
		{"falcon512", true},
		{"hybrid_ecdsa_mldsa", true},
		{"hybrid_", false},
		{"", false},
		{"unknown_algo", false},
	}
	for _, tc := range cases {
		if got := IsValidAlgorithmID(tc.id); got != tc.valid {
			t.Fatalf("IsValidAlgorithmID(%q) = %v, want %v", tc.id, got, tc.valid)
		}
	}
}

func TestSubjectType_IsValid(t *testing.T) {
	if !SubjectTypeWallet.IsValid() {
		t.Fatal("wallet should be valid")
	}
	if SubjectType("other").IsValid() {
		t.Fatal("unexpected subject valid")
	}
}
