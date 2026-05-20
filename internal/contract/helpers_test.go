package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
)

const optionAScanID = "705c9704-9428-45e0-882d-fae4cb9d2a0b"

func contractFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func loadJSONFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(contractFixture(t, name), &doc); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	return doc
}

func mustLoadReadStore(t *testing.T) *api.ReadStore {
	t.Helper()
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		CatalogPath: filepath.Join("..", "domain", "policy", "testdata", "policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_template_valid.json"),
		},
		InstancePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_instance_valid.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	return store
}

func validOptionASelectionRequest() map[string]any {
	return map[string]any{
		"target_posture":              "hybrid",
		"target_chain_ids":            []int64{1},
		"require_multichain":          false,
		"allow_new_wallet":            false,
		"address_continuity_required": true,
		"key_rotation_required":       true,
		"recovery_required":           true,
		"minimum_maturity":            1,
		"approval_mode":               "manual",
	}
}

// policyContextFromDiscoveryV1Detail builds the explore policy_context envelope per A2 §3.1
// (scan_id + status + result copied from GET …/wallets/scans/{scan_id}).
func policyContextFromDiscoveryV1Detail(detail map[string]any) map[string]any {
	out := map[string]any{
		"scan_id": detail["scan_id"],
		"status":  detail["status"],
	}
	if res, ok := detail["result"]; ok {
		out["result"] = res
	}
	return out
}
