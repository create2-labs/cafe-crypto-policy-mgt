package policy

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-cpm/internal/domain/vocabulary"
)

func TestPolicySelectionRequest_NormalizeAndValidate_OK(t *testing.T) {
	req := &PolicySelectionRequest{
		TargetPosture:             vocabulary.PQPostureHybrid,
		TargetChainIDs:            []int64{8453, 1, 8453},
		RequireMultichain:         true,
		AllowNewWallet:            true,
		MinimumMaturity:           0, // defaulted
		AllowedProviderModes:      []ProviderMode{ProviderModeThirdParty, ProviderModeThirdParty, ProviderModeUserManaged},
		PreferredFamilies:         []string{"smart_account", "smart_account", ""},
		PreferredProviders:        []string{"provider_a", "provider_a", "provider_b"},
		RequireBundlerAvailable:   true,
		RequirePaymasterAvailable: true,
		ApprovalMode:              "",
	}

	if err := req.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate: %v", err)
	}

	if req.MinimumMaturity != 1 {
		t.Fatalf("minimum_maturity: got %d want 1", req.MinimumMaturity)
	}
	if req.ApprovalMode != ApprovalModeManual {
		t.Fatalf("approval_mode: got %q want %q", req.ApprovalMode, ApprovalModeManual)
	}
	if !reflect.DeepEqual(req.TargetChainIDs, []int64{1, 8453}) {
		t.Fatalf("target_chain_ids: %#v", req.TargetChainIDs)
	}
	if !reflect.DeepEqual(req.AllowedProviderModes, []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged}) {
		t.Fatalf("allowed_provider_modes: %#v", req.AllowedProviderModes)
	}
	if !reflect.DeepEqual(req.PreferredFamilies, []string{"smart_account"}) {
		t.Fatalf("preferred_families: %#v", req.PreferredFamilies)
	}
	if !reflect.DeepEqual(req.PreferredProviders, []string{"provider_a", "provider_b"}) {
		t.Fatalf("preferred_providers: %#v", req.PreferredProviders)
	}
}

func TestPolicySelectionRequest_JSONRoundTrip(t *testing.T) {
	in := PolicySelectionRequest{
		TargetPosture:             vocabulary.PQPostureHybrid,
		TargetChainIDs:            []int64{1, 8453},
		RequireMultichain:         true,
		AllowNewWallet:            false,
		AddressContinuityRequired: true,
		KeyRotationRequired:       true,
		RecoveryRequired:          true,
		MinimumMaturity:           2,
		AllowResearch:             true,
		AllowedProviderModes:      []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged},
		PreferredFamilies:         []string{"smart_account"},
		PreferredProviders:        []string{"provider_a"},
		RequireBundlerAvailable:   true,
		RequirePaymasterAvailable: false,
		ApprovalMode:              ApprovalModeAuto,
	}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out PolicySelectionRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch:\n%+v\nvs\n%+v", in, out)
	}
}

func TestPolicySelectionRequest_Validate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		req     PolicySelectionRequest
		wantErr error
	}{
		{
			name: "missing target posture",
			req: PolicySelectionRequest{
				MinimumMaturity: 1,
				ApprovalMode:    ApprovalModeManual,
			},
			wantErr: ErrTargetPostureRequired,
		},
		{
			name: "invalid target posture",
			req: PolicySelectionRequest{
				TargetPosture:   vocabulary.CurrentPQPosture("invalid"),
				MinimumMaturity: 1,
				ApprovalMode:    ApprovalModeManual,
			},
			wantErr: ErrTargetPostureInvalid,
		},
		{
			name: "invalid chain id",
			req: PolicySelectionRequest{
				TargetPosture:   vocabulary.PQPostureHybrid,
				TargetChainIDs:  []int64{1, 0},
				MinimumMaturity: 1,
				ApprovalMode:    ApprovalModeManual,
			},
			wantErr: ErrTargetChainIDInvalid,
		},
		{
			name: "multichain requires at least two target chains when explicit target set is provided",
			req: PolicySelectionRequest{
				TargetPosture:     vocabulary.PQPostureHybrid,
				TargetChainIDs:    []int64{1},
				RequireMultichain: true,
				MinimumMaturity:   1,
				ApprovalMode:      ApprovalModeManual,
			},
			wantErr: ErrMultichainRequiresTwoTargets,
		},
		{
			name: "invalid maturity",
			req: PolicySelectionRequest{
				TargetPosture:   vocabulary.PQPostureHybrid,
				MinimumMaturity: 6,
				ApprovalMode:    ApprovalModeManual,
			},
			wantErr: ErrMinimumMaturityRange,
		},
		{
			name: "invalid approval mode",
			req: PolicySelectionRequest{
				TargetPosture:   vocabulary.PQPostureHybrid,
				MinimumMaturity: 1,
				ApprovalMode:    ApprovalMode("invalid"),
			},
			wantErr: ErrApprovalModeInvalid,
		},
		{
			name: "invalid provider mode",
			req: PolicySelectionRequest{
				TargetPosture:        vocabulary.PQPostureHybrid,
				MinimumMaturity:      1,
				ApprovalMode:         ApprovalModeManual,
				AllowedProviderModes: []ProviderMode{ProviderMode("invalid")},
			},
			wantErr: ErrProviderModeInvalid,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
