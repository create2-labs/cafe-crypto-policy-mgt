package policy

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadPolicyGraphCatalogFromFile_Valid(t *testing.T) {
	path := filepath.Join("testdata", "policy_graph_catalog_valid.json")
	cat, err := LoadPolicyGraphCatalogFromFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyGraphCatalogFromFile: %v", err)
	}

	if cat.Version != "v0.1" {
		t.Fatalf("version: got %q", cat.Version)
	}
	if len(cat.Nodes) != 3 {
		t.Fatalf("nodes: got %d", len(cat.Nodes))
	}
	if !cat.IsTransitionAllowed("NODE_EOA_ENTRY", "NODE_SIG_EIP7932") {
		t.Fatal("expected NODE_EOA_ENTRY -> NODE_SIG_EIP7932 to be allowed")
	}
	if cat.IsTransitionAllowed("NODE_SIG_EIP7932", "NODE_EOA_ENTRY") {
		t.Fatal("expected NODE_SIG_EIP7932 -> NODE_EOA_ENTRY to be disallowed")
	}
}

func TestLoadPolicyGraphCatalogFromFile_InvalidFixtures(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		wantErr error
	}{
		{
			name:    "missing version",
			fixture: "policy_graph_catalog_invalid_missing_version.json",
			wantErr: ErrCatalogVersionRequired,
		},
		{
			name:    "unknown rule node",
			fixture: "policy_graph_catalog_invalid_unknown_rule_node.json",
			wantErr: ErrCatalogRuleNodeUnknown,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadPolicyGraphCatalogFromFile(filepath.Join("testdata", tc.fixture))
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
