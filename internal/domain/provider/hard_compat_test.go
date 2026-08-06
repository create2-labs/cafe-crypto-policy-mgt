package provider

import (
	"path/filepath"
	"testing"
)

func TestRegistry_LookupNicetry(t *testing.T) {
	reg, err := LoadRegistryFromFiles([]string{filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json")})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	got, ok := reg.Lookup(ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
		ManifestVersion:   "2026-08",
	})
	if !ok || got == nil {
		t.Fatal("expected nicetry profile")
	}
	if got.Profile.Signature.KeyRotationModel != KeyRotationPerUserOp {
		t.Fatalf("rotation: got %q", got.Profile.Signature.KeyRotationModel)
	}
	if _, ok := reg.Lookup(ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
		ManifestVersion:   "wrong",
	}); ok {
		t.Fatal("expected version mismatch to miss")
	}
}

func TestEvaluateHardCompatibility_table(t *testing.T) {
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
	profile := &resolved.Profile

	baseObs := HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}}
	baseReq := HardSelectionRequest{
		TargetChainIDs:            []int64{11155111},
		AllowNewWallet:            true,
		AddressContinuityRequired: false,
		KeyRotationModel:          string(KeyRotationPerUserOp),
	}

	tests := []struct {
		name     string
		obs      HardObservation
		req      HardSelectionRequest
		wantCode string // empty => hard pass
	}{
		{name: "sepolia_ok", obs: baseObs, req: baseReq, wantCode: ""},
		{
			name: "mainnet_planned_reject",
			obs:  HardObservation{AccountKind: "eoa", ChainIDs: []int64{1}},
			req: HardSelectionRequest{
				TargetChainIDs:   []int64{1},
				AllowNewWallet:   true,
				KeyRotationModel: string(KeyRotationPerUserOp),
			},
			wantCode: FindingCodeChain,
		},
		{
			name: "continuity_reject",
			obs:  baseObs,
			req: HardSelectionRequest{
				TargetChainIDs:            []int64{11155111},
				AllowNewWallet:            true,
				AddressContinuityRequired: true,
				KeyRotationModel:          string(KeyRotationPerUserOp),
			},
			wantCode: FindingCodeContinuity,
		},
		{
			name: "rotation_none_reject",
			obs:  baseObs,
			req: HardSelectionRequest{
				TargetChainIDs:   []int64{11155111},
				AllowNewWallet:   true,
				KeyRotationModel: string(KeyRotationNone),
			},
			wantCode: FindingCodeRotation,
		},
		{
			name: "new_wallet_disallowed_reject",
			obs:  baseObs,
			req: HardSelectionRequest{
				TargetChainIDs:   []int64{11155111},
				AllowNewWallet:   false,
				KeyRotationModel: string(KeyRotationPerUserOp),
			},
			wantCode: FindingCodeNewWallet,
		},
		{
			name: "non_eoa_reject",
			obs:  HardObservation{AccountKind: "erc4337_smart_account", ChainIDs: []int64{11155111}},
			req:  baseReq,
			wantCode: FindingCodeWalletType,
		},
		{
			name: "unknown_chain_reject",
			obs:  HardObservation{AccountKind: "eoa", ChainIDs: []int64{8453}},
			req: HardSelectionRequest{
				TargetChainIDs:   []int64{8453},
				AllowNewWallet:   true,
				KeyRotationModel: string(KeyRotationPerUserOp),
			},
			wantCode: FindingCodeChain,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := EvaluateHardCompatibility(tc.obs, tc.req, profile)
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
