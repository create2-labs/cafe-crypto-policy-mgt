package provider

import (
	"path/filepath"
	"testing"
)

func TestEvaluateScanCompatibility_table(t *testing.T) {
	reg, err := LoadRegistryFromFiles([]string{filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json")})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	resolved, ok := reg.Lookup(ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
	})
	if !ok {
		t.Fatal("missing profile")
	}
	profile := resolved.Profile

	tests := []struct {
		name            string
		obs             HardObservation
		requiredPosture string
		mutate          func(*SolutionProfile)
		wantCode        string
	}{
		{
			name:            "sepolia_ok",
			obs:             HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}},
			requiredPosture: "hybrid",
			wantCode:        "",
		},
		{
			name:            "mainnet_planned_reject",
			obs:             HardObservation{AccountKind: "eoa", ChainIDs: []int64{1}},
			requiredPosture: "hybrid",
			wantCode:        FindingCodeChain,
		},
		{
			name:            "non_eoa_reject",
			obs:             HardObservation{AccountKind: "erc4337_smart_account", ChainIDs: []int64{11155111}},
			requiredPosture: "hybrid",
			wantCode:        FindingCodeWalletType,
		},
		{
			name:            "posture_mismatch",
			obs:             HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}},
			requiredPosture: "full_pq",
			wantCode:        FindingCodePosture,
		},
		{
			name:            "rotate_signer_missing",
			obs:             HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}},
			requiredPosture: "hybrid",
			mutate: func(p *SolutionProfile) {
				cs := make([]ChainSupport, len(p.ChainSupport))
				copy(cs, p.ChainSupport)
				for i := range cs {
					if cs[i].ChainID == 11155111 {
						cs[i].Capabilities = []string{CapabilityDeploy, CapabilitySignUserOp}
					}
				}
				p.ChainSupport = cs
			},
			wantCode: FindingCodeCapability,
		},
		{
			name:            "couche_b_fields_ignored",
			obs:             HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}},
			requiredPosture: "hybrid",
			wantCode:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := profile
			if tc.mutate != nil {
				tc.mutate(&p)
			}
			findings := EvaluateScanCompatibility(tc.obs, tc.requiredPosture, &p)
			if tc.wantCode == "" {
				if len(findings) != 0 {
					t.Fatalf("want hard pass, got %+v", findings)
				}
				return
			}
			found := false
			for _, f := range findings {
				if f.Code == tc.wantCode {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("want code %q in findings %+v", tc.wantCode, findings)
			}
		})
	}
}

func TestValidateSuggestedUserConstraints(t *testing.T) {
	base := validBase().SolutionProfiles[0]
	base.Constraints = ProfileConstraints{
		RequiresNewAccount:         true,
		AddressContinuitySupported: false,
	}
	base.SuggestedUserConstraints = &SuggestedUserConstraints{
		AllowNewWallet:            true,
		AddressContinuityRequired: false,
		KeyRotationModel:          KeyRotationPerUserOp,
	}
	if err := ValidateSuggestedUserConstraints(&base); err != nil {
		t.Fatalf("valid suggestion: %v", err)
	}

	bad := base
	bad.SuggestedUserConstraints = &SuggestedUserConstraints{
		AllowNewWallet:            false,
		AddressContinuityRequired: false,
		KeyRotationModel:          KeyRotationPerUserOp,
	}
	if err := ValidateSuggestedUserConstraints(&bad); err == nil {
		t.Fatal("expected contradiction for allow_new_wallet=false")
	}

	badCont := base
	badCont.SuggestedUserConstraints = &SuggestedUserConstraints{
		AllowNewWallet:            true,
		AddressContinuityRequired: true,
		KeyRotationModel:          KeyRotationPerUserOp,
	}
	if err := ValidateSuggestedUserConstraints(&badCont); err == nil {
		t.Fatal("expected contradiction for continuity")
	}
}

func TestLoadProviderManifest_suggestedUserConstraints(t *testing.T) {
	m, err := LoadProviderManifestFromFile(filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	s := m.SolutionProfiles[0].SuggestedUserConstraints
	if s == nil {
		t.Fatal("expected suggested_user_constraints on nicetry fixture")
	}
	if !s.AllowNewWallet || s.AddressContinuityRequired || s.KeyRotationModel != KeyRotationPerUserOp {
		t.Fatalf("unexpected suggestion: %+v", s)
	}
	if err := ValidateSuggestedUserConstraints(&m.SolutionProfiles[0]); err != nil {
		t.Fatalf("fixture suggestion must be consistent: %v", err)
	}
}
