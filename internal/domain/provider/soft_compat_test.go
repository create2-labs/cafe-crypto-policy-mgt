package provider

import (
	"path/filepath"
	"testing"
)

func TestEvaluateSoftFindings_nicetryFixture(t *testing.T) {
	reg, err := LoadRegistryFromFiles([]string{filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json")})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	resolved, ok := reg.Lookup(ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
	})
	if !ok {
		t.Fatal("missing nicetry profile")
	}

	findings := EvaluateSoftFindings(&resolved.Profile)
	if len(findings) != 2 {
		t.Fatalf("want 2 soft findings, got %+v", findings)
	}
	if findings[0].Code != FindingCodeRequiresBundler {
		t.Fatalf("first: got %q", findings[0].Code)
	}
	if findings[1].Code != FindingCodeRequiresLocalSignerState {
		t.Fatalf("second: got %q", findings[1].Code)
	}
	for _, f := range findings {
		if f.Code == "requires_wallet_control_proof" {
			t.Fatal("wallet-control proof must not be an explore finding")
		}
	}
}

func TestEvaluateSoftFindings_onlyWhenFlagsTrue(t *testing.T) {
	tests := []struct {
		name     string
		bundler  bool
		signer   bool
		wantCode []string
	}{
		{name: "none", wantCode: nil},
		{name: "bundler_only", bundler: true, wantCode: []string{FindingCodeRequiresBundler}},
		{name: "signer_only", signer: true, wantCode: []string{FindingCodeRequiresLocalSignerState}},
		{
			name:     "both",
			bundler:  true,
			signer:   true,
			wantCode: []string{FindingCodeRequiresBundler, FindingCodeRequiresLocalSignerState},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := &SolutionProfile{}
			profile.AccountModel.RequiresBundler = tc.bundler
			profile.Constraints.RequiresLocalSignerState = tc.signer
			got := EvaluateSoftFindings(profile)
			if len(got) != len(tc.wantCode) {
				t.Fatalf("len: got %d want %d (%+v)", len(got), len(tc.wantCode), got)
			}
			for i, code := range tc.wantCode {
				if got[i].Code != code {
					t.Fatalf("[%d]: got %q want %q", i, got[i].Code, code)
				}
			}
		})
	}
}

func TestEvaluateSoftFindings_nilProfile(t *testing.T) {
	if got := EvaluateSoftFindings(nil); got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
}
