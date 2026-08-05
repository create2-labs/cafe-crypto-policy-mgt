package policy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadCryptoPolicyInstanceFromFile_Valid(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	instance, err := LoadCryptoPolicyInstanceFromFile(filepath.Join("testdata", "crypto_policy_instance_pq_account_validation_v1.json"), catalog)
	if err != nil {
		t.Fatalf("LoadCryptoPolicyInstanceFromFile: %v", err)
	}

	if instance.ID != "cpx_pq_account_validation_v1" {
		t.Fatalf("id: got %q", instance.ID)
	}
	if !reflect.DeepEqual(instance.Scope.ChainIDs, []int64{1, 8453}) {
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
	if !reflect.DeepEqual(instance.Governance.Tags, []string{"pq_account_validation", "nicetry"}) {
		t.Fatalf("governance.tags: %#v", instance.Governance.Tags)
	}
}

func TestLoadCryptoPolicyInstanceFromFile_InvalidFixture(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	_, err = LoadCryptoPolicyInstanceFromFile(filepath.Join("testdata", "crypto_policy_instance_invalid_missing_required_param.json"), catalog)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNodeParameterRequiredMissing) {
		t.Fatalf("error = %v, want %v", err, ErrNodeParameterRequiredMissing)
	}
}

func TestCryptoPolicyInstance_Validate_ReferenceRules(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	makeBase := func() CryptoPolicyInstance {
		b := true
		return CryptoPolicyInstance{
			ID:             "cpx",
			Name:           "Policy",
			CatalogVersion: "v0.1",
			Scope: PolicyScope{
				Name: "prod",
			},
			GlobalParams: GlobalPolicyParameters{
				TargetPosture:        vocabulary.PQPostureHybrid,
				MinimumMaturity:      1,
				ApprovalMode:         ApprovalModeManual,
				KeyRotationModel:     KeyRotationNone,
				AllowedProviderModes: []ProviderMode{ProviderModeThirdParty},
			},
			NodeParameters: map[string]NodeParameterMap{
				"NODE_SIG_EIP7932": {
					"threshold": {Type: ParamTypeInt, IntValue: intPtr(2)},
					"strict":    {Type: ParamTypeBool, BoolValue: &b},
				},
			},
		}
	}

	t.Run("missing reference", func(t *testing.T) {
		in := makeBase()
		in.NodeParameters = nil
		err := in.Validate(catalog)
		if !errors.Is(err, ErrInstanceReferenceMissing) {
			t.Fatalf("error = %v, want %v", err, ErrInstanceReferenceMissing)
		}
	})

	t.Run("ambiguous reference", func(t *testing.T) {
		in := makeBase()
		in.TemplateID = "tpl_pq_account_validation_v1"
		in.NodePath = []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932"}
		in.NodeParameters = nil
		err := in.Validate(catalog)
		if !errors.Is(err, ErrInstanceReferenceAmbiguous) {
			t.Fatalf("error = %v, want %v", err, ErrInstanceReferenceAmbiguous)
		}
	})

	t.Run("node params require path", func(t *testing.T) {
		in := makeBase()
		err := in.Validate(catalog)
		if !errors.Is(err, ErrInstanceNodePathRequired) {
			t.Fatalf("error = %v, want %v", err, ErrInstanceNodePathRequired)
		}
	})
}

func TestCryptoPolicyInstance_Validate_NodeSchemaAndTransitions(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	makeBase := func() CryptoPolicyInstance {
		requiredBool := true
		return CryptoPolicyInstance{
			ID:             "cpx_schema",
			Name:           "Schema checks",
			CatalogVersion: "v0.1",
			NodePath:       []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"},
			SolutionProfileRef: SolutionProfileRef{
				ProviderID:        "nicetry",
				SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
				ManifestVersion:   "2026-08",
			},
			Scope: PolicyScope{
				Name:     "prod",
				ChainIDs: []int64{8453, 1},
			},
			GlobalParams: GlobalPolicyParameters{
				TargetPosture:        vocabulary.PQPostureHybrid,
				MinimumMaturity:      1,
				ApprovalMode:         ApprovalModeManual,
				KeyRotationModel:     KeyRotationNone,
				AllowedProviderModes: []ProviderMode{ProviderModeThirdParty},
			},
			NodeParameters: map[string]NodeParameterMap{
				"NODE_SIG_EIP7932": {
					"threshold": {Type: ParamTypeInt, IntValue: intPtr(2)},
					"strict":    {Type: ParamTypeBool, BoolValue: &requiredBool},
					"profile":   {Type: ParamTypeEnum, StringValue: "hybrid"},
				},
			},
		}
	}

	t.Run("valid", func(t *testing.T) {
		in := makeBase()
		if err := in.NormalizeAndValidate(catalog); err != nil {
			t.Fatalf("NormalizeAndValidate: %v", err)
		}
	})

	t.Run("unknown node in path", func(t *testing.T) {
		in := makeBase()
		in.NodePath = []string{"NODE_EOA_ENTRY", "NODE_UNKNOWN"}
		err := in.Validate(catalog)
		if !errors.Is(err, ErrInstanceNodeUnknown) {
			t.Fatalf("error = %v, want %v", err, ErrInstanceNodeUnknown)
		}
	})

	t.Run("invalid transition", func(t *testing.T) {
		in := makeBase()
		in.NodePath = []string{"NODE_TARGET_HYBRID", "NODE_EOA_ENTRY"}
		err := in.Validate(catalog)
		if !errors.Is(err, ErrInstanceTransitionInvalid) {
			t.Fatalf("error = %v, want %v", err, ErrInstanceTransitionInvalid)
		}
	})

	t.Run("unknown node params node", func(t *testing.T) {
		in := makeBase()
		in.NodeParameters["NODE_UNKNOWN"] = NodeParameterMap{
			"x": {Type: ParamTypeString, StringValue: "abc"},
		}
		err := in.Validate(catalog)
		if !errors.Is(err, ErrNodeParameterNodeUnknown) {
			t.Fatalf("error = %v, want %v", err, ErrNodeParameterNodeUnknown)
		}
	})

	t.Run("unknown param", func(t *testing.T) {
		in := makeBase()
		in.NodeParameters["NODE_SIG_EIP7932"]["unexpected"] = NodeParameterValue{Type: ParamTypeString, StringValue: "x"}
		err := in.Validate(catalog)
		if !errors.Is(err, ErrNodeParameterUnknown) {
			t.Fatalf("error = %v, want %v", err, ErrNodeParameterUnknown)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		in := makeBase()
		in.NodeParameters["NODE_SIG_EIP7932"]["threshold"] = NodeParameterValue{Type: ParamTypeString, StringValue: "2"}
		err := in.Validate(catalog)
		if !errors.Is(err, ErrNodeParameterTypeMismatch) {
			t.Fatalf("error = %v, want %v", err, ErrNodeParameterTypeMismatch)
		}
	})
}

func TestCryptoPolicyInstance_Validate_SolutionProfileRef(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	base := CryptoPolicyInstance{
		ID:             "cpx_ref",
		Name:           "Ref checks",
		CatalogVersion: "v0.1",
		TemplateID:     "tpl_pq_account_validation_v1",
		Scope:          PolicyScope{Name: "catalogue"},
		GlobalParams: GlobalPolicyParameters{
			TargetPosture:    vocabulary.PQPostureHybrid,
			MinimumMaturity:  1,
			ApprovalMode:     ApprovalModeManual,
			KeyRotationModel: KeyRotationNone,
		},
	}

	t.Run("missing", func(t *testing.T) {
		in := base
		err := in.Validate(catalog)
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
		err := in.Validate(catalog)
		if !errors.Is(err, ErrSolutionProfileRefProviderIDRequired) {
			t.Fatalf("error = %v, want %v", err, ErrSolutionProfileRefProviderIDRequired)
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
		if err := in.NormalizeAndValidate(catalog); err != nil {
			t.Fatalf("NormalizeAndValidate: %v", err)
		}
	})
}

func intPtr(v int64) *int64 {
	return &v
}
