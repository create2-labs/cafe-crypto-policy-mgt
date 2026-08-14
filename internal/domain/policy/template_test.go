package policy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadCryptoPolicyTemplateFromFile_Valid(t *testing.T) {
	tpl, err := LoadCryptoPolicyTemplateFromFile(filepath.Join("testdata", "crypto_policy_template_pq_account_validation_v1.json"), nil)
	if err != nil {
		t.Fatalf("LoadCryptoPolicyTemplateFromFile: %v", err)
	}

	if tpl.ID != "tpl_pq_account_validation_v1" {
		t.Fatalf("id: got %q", tpl.ID)
	}
	if tpl.RequiredPosture != vocabulary.PQPostureHybrid {
		t.Fatalf("required_posture: got %q", tpl.RequiredPosture)
	}
	if len(tpl.NodePath) != 0 {
		t.Fatalf("node_path should be empty (non-normative), got %#v", tpl.NodePath)
	}
	if !reflect.DeepEqual(tpl.Constraints.TargetChainIDs, []int64{1, 8453}) {
		t.Fatalf("constraints.target_chain_ids: %#v", tpl.Constraints.TargetChainIDs)
	}
	if tpl.Constraints.MinimumMaturity != 1 {
		t.Fatalf("constraints.minimum_maturity: got %d want 1", tpl.Constraints.MinimumMaturity)
	}
	if tpl.Defaults.MinimumMaturity != 1 {
		t.Fatalf("default_selection.minimum_maturity: got %d want 1", tpl.Defaults.MinimumMaturity)
	}
	if tpl.Defaults.TargetPosture != vocabulary.PQPostureHybrid {
		t.Fatalf("default_selection.target_posture: got %q", tpl.Defaults.TargetPosture)
	}
	if !reflect.DeepEqual(tpl.Metadata.Tags, []string{"pq_account_validation", "capability_provider"}) {
		t.Fatalf("metadata.tags: %#v", tpl.Metadata.Tags)
	}
}

func TestLoadCryptoPolicyTemplateFromFile_InvalidFixtures(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	tests := []struct {
		name    string
		fixture string
		wantErr error
	}{
		{
			name:    "unknown node",
			fixture: "crypto_policy_template_invalid_unknown_node.json",
			wantErr: ErrTemplateNodeUnknown,
		},
		{
			name:    "invalid transition",
			fixture: "crypto_policy_template_invalid_transition.json",
			wantErr: ErrTemplateTransitionInvalid,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadCryptoPolicyTemplateFromFile(filepath.Join("testdata", tc.fixture), catalog)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestCryptoPolicyTemplate_Validate_MetadataPostureMismatch(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	tpl := &CryptoPolicyTemplate{
		ID:              "tpl_mismatch",
		Name:            "Mismatch template",
		Version:         "v0.1",
		CatalogVersion:  "v0.1",
		RequiredPosture: vocabulary.PQPostureHybrid,
		Defaults: PolicySelectionRequest{
			TargetPosture:   vocabulary.PQPostureHybrid,
			MinimumMaturity: 1,
			ApprovalMode:    ApprovalModeManual,
		},
		Constraints: TemplateConstraints{
			MinimumMaturity: 1,
		},
		Metadata: TemplateMetadata{
			RequiredPosture: vocabulary.PQPostureFullPQ,
		},
	}

	if err := tpl.Validate(catalog); !errors.Is(err, ErrTemplateMetadataPostureMismatch) {
		t.Fatalf("error = %v, want %v", err, ErrTemplateMetadataPostureMismatch)
	}
}
