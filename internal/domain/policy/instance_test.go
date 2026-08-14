package policy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadCryptoPolicyInstanceFromFile_Valid(t *testing.T) {
	instance, err := LoadCryptoPolicyInstanceFromFile(filepath.Join("testdata", "crypto_policy_instance_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("LoadCryptoPolicyInstanceFromFile: %v", err)
	}

	if instance.ID != "cpx_pq_account_validation_v1" {
		t.Fatalf("id: got %q", instance.ID)
	}
	if !reflect.DeepEqual(instance.Scope.ChainIDs, []int64{1, 8453, 11155111}) {
		t.Fatalf("scope.chain_ids: %#v", instance.Scope.ChainIDs)
	}
	if instance.GlobalParams.MinimumMaturity != 1 {
		t.Fatalf("global_parameters.minimum_maturity: got %d want 1", instance.GlobalParams.MinimumMaturity)
	}
	if instance.GlobalParams.ApprovalMode != ApprovalModeManual {
		t.Fatalf("global_parameters.approval_mode: got %q want %q", instance.GlobalParams.ApprovalMode, ApprovalModeManual)
	}
	if instance.SolutionProfileRef.ProviderID != "nicetry" {
		t.Fatalf("solution_profile_ref.provider_id: got %q", instance.SolutionProfileRef.ProviderID)
	}
	if instance.SolutionProfileRef.SolutionProfileID != "nicetry.fors_c.erc4337.v0_1" {
		t.Fatalf("solution_profile_ref.solution_profile_id: got %q", instance.SolutionProfileRef.SolutionProfileID)
	}
	if instance.GlobalParams.RequiredPosture != vocabulary.PQPostureHybrid {
		t.Fatalf("global_parameters.required_posture: got %q", instance.GlobalParams.RequiredPosture)
	}
	if !instance.GlobalParams.AllowNewWallet {
		t.Fatal("Nicetry requires_new_account=true requires allow_new_wallet=true")
	}
	if instance.GlobalParams.AddressContinuityRequired {
		t.Fatal("Nicetry address_continuity_supported=false requires address_continuity_required=false")
	}
	if !reflect.DeepEqual(instance.Governance.Tags, []string{"pq_account_validation", "nicetry"}) {
		t.Fatalf("governance.tags: %#v", instance.Governance.Tags)
	}
}

func TestCryptoPolicyInstance_Validate_TemplateIDRequired(t *testing.T) {
	in := CryptoPolicyInstance{
		ID:             "cpx",
		Name:           "Policy",
		CatalogVersion: "v0.1",
		Scope: PolicyScope{
			Name: "prod",
		},
		GlobalParams: GlobalPolicyParameters{
			RequiredPosture:      vocabulary.PQPostureHybrid,
			MinimumMaturity:      1,
			ApprovalMode:         ApprovalModeManual,
			KeyRotationModel:     KeyRotationNone,
			AllowedProviderModes: []ProviderMode{ProviderModeThirdParty},
		},
		SolutionProfileRef: SolutionProfileRef{
			ProviderID:        "nicetry",
			SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
			ManifestVersion:   "2026-08",
		},
	}

	if err := in.Validate(); !errors.Is(err, ErrInstanceTemplateIDRequired) {
		t.Fatalf("error = %v, want %v", err, ErrInstanceTemplateIDRequired)
	}
}

func TestCryptoPolicyInstance_Validate_SolutionProfileRef(t *testing.T) {
	base := CryptoPolicyInstance{
		ID:             "cpx_ref",
		Name:           "Ref checks",
		CatalogVersion: "v0.1",
		TemplateID:     "tpl_pq_account_validation_v1",
		Scope:          PolicyScope{Name: "catalogue"},
		GlobalParams: GlobalPolicyParameters{
			RequiredPosture:  vocabulary.PQPostureHybrid,
			MinimumMaturity:  1,
			ApprovalMode:     ApprovalModeManual,
			KeyRotationModel: KeyRotationNone,
		},
	}

	t.Run("missing", func(t *testing.T) {
		in := base
		err := in.Validate()
		if !errors.Is(err, ErrSolutionProfileRefRequired) {
			t.Fatalf("error = %v, want %v", err, ErrSolutionProfileRefRequired)
		}
	})

	t.Run("missing provider_id", func(t *testing.T) {
		in := base
		in.SolutionProfileRef = SolutionProfileRef{
			SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
			ManifestVersion:   "2026-08",
		}
		err := in.Validate()
		if !errors.Is(err, ErrSolutionProfileRefProviderIDRequired) {
			t.Fatalf("error = %v, want %v", err, ErrSolutionProfileRefProviderIDRequired)
		}
	})

	t.Run("whitespace manifest_version", func(t *testing.T) {
		in := base
		in.SolutionProfileRef = SolutionProfileRef{
			ProviderID:        "nicetry",
			SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
			ManifestVersion:   " \t ",
		}
		err := in.NormalizeAndValidate()
		if !errors.Is(err, ErrSolutionProfileRefManifestVersionRequired) {
			t.Fatalf("error = %v, want %v", err, ErrSolutionProfileRefManifestVersionRequired)
		}
	})

	t.Run("valid", func(t *testing.T) {
		in := base
		in.SolutionProfileRef = SolutionProfileRef{
			ProviderID:        "nicetry",
			SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
			ManifestVersion:   "2026-08",
			VerificationDate:  "2026-08-03",
		}
		if err := in.NormalizeAndValidate(); err != nil {
			t.Fatalf("NormalizeAndValidate: %v", err)
		}
	})
}
