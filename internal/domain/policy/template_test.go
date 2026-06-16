package policy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadCryptoPolicyTemplateFromFile_Valid(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	tpl, err := LoadCryptoPolicyTemplateFromFile(filepath.Join("testdata", "crypto_policy_template_valid.json"), catalog)
	if err != nil {
		t.Fatalf("LoadCryptoPolicyTemplateFromFile: %v", err)
	}

	if tpl.ID != "tpl_hybrid_baseline" {
		t.Fatalf("id: got %q", tpl.ID)
	}
	if tpl.TargetPosture != vocabulary.PQPostureHybrid {
		t.Fatalf("target_posture: got %q", tpl.TargetPosture)
	}
	if !reflect.DeepEqual(tpl.NodePath, []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"}) {
		t.Fatalf("node_path: %#v", tpl.NodePath)
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
	if !reflect.DeepEqual(tpl.Metadata.Tags, []string{"baseline", "hybrid"}) {
		t.Fatalf("metadata.tags: %#v", tpl.Metadata.Tags)
	}
}

func TestLoadCryptoPolicyTemplateFromFile_PqReadyProgressive(t *testing.T) {
	catalog, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	tpl, err := LoadCryptoPolicyTemplateFromFile(
		filepath.Join("testdata", "crypto_policy_template_pq_ready_progressive.json"),
		catalog,
	)
	if err != nil {
		t.Fatalf("LoadCryptoPolicyTemplateFromFile pq_ready_progressive: %v", err)
	}
	if tpl.ID != "tpl_pq_ready_progressive" {
		t.Fatalf("id: got %q", tpl.ID)
	}
	if tpl.Name != "PQ-ready progressive path" {
		t.Fatalf("name: got %q", tpl.Name)
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
		ID:             "tpl_mismatch",
		Name:           "Mismatch template",
		Version:        "v0.1",
		CatalogVersion: "v0.1",
		TargetPosture:  vocabulary.PQPostureHybrid,
		NodePath:       []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"},
		Defaults: PolicySelectionRequest{
			TargetPosture:   vocabulary.PQPostureHybrid,
			MinimumMaturity: 1,
			ApprovalMode:    ApprovalModeManual,
		},
		Constraints: TemplateConstraints{
			MinimumMaturity: 1,
		},
		Metadata: TemplateMetadata{
			TargetPosture: vocabulary.PQPostureFullPQ,
		},
	}

	if err := tpl.Validate(catalog); !errors.Is(err, ErrTemplateMetadataPostureMismatch) {
		t.Fatalf("error = %v, want %v", err, ErrTemplateMetadataPostureMismatch)
	}
}
