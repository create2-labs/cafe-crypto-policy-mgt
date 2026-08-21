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
		CryptoPolicyPaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_pq_account_validation_v1.json"),
		},
		InstancePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_instance_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		},

	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	return store
}

func validOptionACryptoPolicyID() string {
	return "cpm_pq_account_validation_v1"
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
