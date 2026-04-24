package policy

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewCryptoPolicyInstanceValidator(t *testing.T) {
	t.Run("nil catalog", func(t *testing.T) {
		_, err := NewCryptoPolicyInstanceValidator(nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("valid catalog", func(t *testing.T) {
		catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
		if err != nil {
			t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
		}

		validator, err := NewCryptoPolicyInstanceValidator(catalog)
		if err != nil {
			t.Fatalf("NewCryptoPolicyInstanceValidator: %v", err)
		}
		if validator == nil {
			t.Fatal("validator is nil")
		}
	})
}

func TestCryptoPolicyInstanceValidator_Validate(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	validator, err := NewCryptoPolicyInstanceValidator(catalog)
	if err != nil {
		t.Fatalf("NewCryptoPolicyInstanceValidator: %v", err)
	}

	t.Run("valid fixture", func(t *testing.T) {
		instance, err := LoadCryptoPolicyInstanceFromFile(filepath.Join("testdata", "crypto_policy_instance_valid.json"), catalog)
		if err != nil {
			t.Fatalf("LoadCryptoPolicyInstanceFromFile: %v", err)
		}

		result := validator.Validate(instance)
		if !result.Valid {
			t.Fatalf("expected valid result, got issues=%v", result.Issues)
		}
		if result.NormalizedPolicy == nil {
			t.Fatal("normalized policy is nil")
		}
	})

	t.Run("invalid fixture", func(t *testing.T) {
		instance := &CryptoPolicyInstance{
			ID:             "cpx_invalid",
			Name:           "Invalid",
			CatalogVersion: "v0.1",
			NodePath:       []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"},
			Scope:          PolicyScope{Name: "prod"},
			GlobalParams: GlobalPolicyParameters{
				TargetPosture:   "hybrid",
				MinimumMaturity: 1,
				ApprovalMode:    ApprovalModeManual,
			},
			NodeParameters: map[string]NodeParameterMap{
				"NODE_SIG_EIP7932": {
					"strict": {Type: ParamTypeBool, BoolValue: boolPtr(true)},
				},
			},
		}

		result := validator.Validate(instance)
		if result.Valid {
			t.Fatal("expected invalid result")
		}
		if len(result.Issues) != 1 {
			t.Fatalf("issues length: got %d want 1", len(result.Issues))
		}
		if result.Issues[0].Code != ValidationIssueCodeInstanceInvalid {
			t.Fatalf("issue code: got %q", result.Issues[0].Code)
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		result := validator.Validate(nil)
		if result.Valid {
			t.Fatal("expected invalid result")
		}
		if len(result.Issues) != 1 {
			t.Fatalf("issues length: got %d want 1", len(result.Issues))
		}
		if result.Issues[0].Code != ValidationIssueCodeInstanceRequired {
			t.Fatalf("issue code: got %q", result.Issues[0].Code)
		}
	})

	t.Run("does not mutate original instance", func(t *testing.T) {
		instance := &CryptoPolicyInstance{
			ID:             "cpx_mutation_check",
			Name:           "Mutation check",
			CatalogVersion: "v0.1",
			NodePath:       []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"},
			Scope: PolicyScope{
				Name:     "prod",
				ChainIDs: []int64{8453, 1},
			},
			GlobalParams: GlobalPolicyParameters{
				TargetPosture:        "hybrid",
				MinimumMaturity:      1,
				ApprovalMode:         ApprovalModeManual,
				AllowedProviderModes: []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged},
			},
			NodeParameters: map[string]NodeParameterMap{
				"NODE_SIG_EIP7932": {
					"threshold": {Type: ParamTypeInt, IntValue: intPtr(2)},
					"strict":    {Type: ParamTypeBool, BoolValue: boolPtr(true)},
					"profile":   {Type: ParamTypeEnum, StringValue: "hybrid"},
				},
			},
		}

		before := *instance
		before.Scope.ChainIDs = append([]int64(nil), instance.Scope.ChainIDs...)
		before.GlobalParams.AllowedProviderModes = append([]ProviderMode(nil), instance.GlobalParams.AllowedProviderModes...)

		result := validator.Validate(instance)
		if !result.Valid {
			t.Fatalf("expected valid result, got issues=%v", result.Issues)
		}

		if !reflect.DeepEqual(instance.Scope.ChainIDs, before.Scope.ChainIDs) {
			t.Fatalf("instance scope.chain_ids mutated: before=%v after=%v", before.Scope.ChainIDs, instance.Scope.ChainIDs)
		}
		if !reflect.DeepEqual(instance.GlobalParams.AllowedProviderModes, before.GlobalParams.AllowedProviderModes) {
			t.Fatalf("instance global_parameters.allowed_provider_modes mutated: before=%v after=%v", before.GlobalParams.AllowedProviderModes, instance.GlobalParams.AllowedProviderModes)
		}
		if reflect.DeepEqual(result.NormalizedPolicy.Scope.ChainIDs, instance.Scope.ChainIDs) {
			t.Fatalf("expected normalized policy to differ from original ordering")
		}
	})
}

func TestCryptoPolicyInstanceValidator_ValidateNilReceiver(t *testing.T) {
	var validator *CryptoPolicyInstanceValidator
	result := validator.Validate(&CryptoPolicyInstance{})
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues length: got %d want 1", len(result.Issues))
	}
	if result.Issues[0].Code != ValidationIssueCodeValidatorConfig {
		t.Fatalf("issue code: got %q", result.Issues[0].Code)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
