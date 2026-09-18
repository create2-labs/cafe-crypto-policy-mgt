package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func TestDeriveCompatibleNetworks_MultichainExcludesPlanned(t *testing.T) {
	cp := mustLoadCryptoPolicy(t, "crypto_policy_pq_account_validation_v1.json")
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}

	got := DeriveCompatibleNetworks(cp, reg)
	if len(got) < 2 {
		t.Fatalf("multichain fixture: want several deployable networks, got %d (%+v)", len(got), got)
	}
	for _, n := range got {
		if n.Status == provider.ChainStatusPlanned {
			t.Fatalf("planned must be absent: %+v", n)
		}
		if n.ChainID <= 0 || n.Network == "" {
			t.Fatalf("incomplete network fact: %+v", n)
		}
	}
	// Deterministic ascending chain_id order.
	for i := 1; i < len(got); i++ {
		if got[i].ChainID <= got[i-1].ChainID {
			t.Fatalf("not sorted by chain_id: %+v", got)
		}
	}
	wantIDs := []int64{1, 10, 137, 8453, 42161, 84532, 421614, 11155111, 11155420}
	if len(got) != len(wantIDs) {
		t.Fatalf("network count: got %d want %d (%+v)", len(got), len(wantIDs), got)
	}
	for i, id := range wantIDs {
		if got[i].ChainID != id {
			t.Fatalf("chain_id[%d]: got %d want %d", i, got[i].ChainID, id)
		}
	}
}

func TestDeriveCompatibleNetworks_SepoliaOnlyHistoricalExcludesPlanned(t *testing.T) {
	cp := mustLoadCryptoPolicy(t, "crypto_policy_pq_account_validation_v1.json")
	dir := t.TempDir()
	path := filepath.Join(dir, "provider_manifest_nicetry_sepolia_only.json")
	const sepoliaOnly = `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "nicetry",
  "provider_name": "NiceTry",
  "provider_version": "2026-08",
  "provider_maturity": "research",
  "solution_profiles": [{
    "solution_profile_id": "nicetry.fors_c.erc4337.v0_1",
    "display_name": "NiceTry FORS+C ERC-4337 Smart Account",
    "maturity": "research",
    "claim_status": "declared",
    "resulting_posture": "hybrid",
    "input_requirements": { "wallet_types": ["EOA"], "requires_wallet_control_proof": true },
    "signature": { "scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop" },
    "account_model": {
      "standard": "ERC-4337",
      "execution_model": "erc4337_bundler",
      "requires_bundler": true,
      "requires_entrypoint": true,
      "entrypoint_versions": ["0.7"]
    },
    "constraints": {
      "requires_new_account": true,
      "address_continuity_supported": false,
      "requires_local_signer_state": true
    },
    "chain_support": [
      { "chain_id": 11155111, "network": "sepolia", "status": "testnet_supported", "capabilities": ["deploy", "sign_userop", "rotate_signer"] },
      { "chain_id": 1, "network": "ethereum-mainnet", "status": "planned", "capabilities": [] }
    ]
  }]
}`
	if err := os.WriteFile(path, []byte(sepoliaOnly), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	reg, err := provider.LoadRegistryFromFiles([]string{path})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}

	got := DeriveCompatibleNetworks(cp, reg)
	if len(got) != 1 {
		t.Fatalf("sepolia-only: want 1 network, got %d (%+v)", len(got), got)
	}
	if got[0].ChainID != 11155111 || got[0].Network != "sepolia" || got[0].Status != provider.ChainStatusTestnetSupported {
		t.Fatalf("unexpected network: %+v", got[0])
	}
}

func TestDeriveCompatibleNetworks_UnknownProviderEmpty(t *testing.T) {
	cp := &CryptoPolicy{
		ID:               "x",
		Name:             "x",
		Version:          "1",
		RequiredPosture:  "hybrid",
		AllowedProviders: []string{"does-not-exist"},
	}
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	got := DeriveCompatibleNetworks(cp, reg)
	if got == nil || len(got) != 0 {
		t.Fatalf("want empty slice, got %#v", got)
	}
}

func TestCatalogItemFromCryptoPolicy_IncludesDerivedField(t *testing.T) {
	cp := mustLoadCryptoPolicy(t, "crypto_policy_pq_account_validation_v1.json")
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	item := CatalogItemFromCryptoPolicy(cp, reg)
	if item.ID != cp.ID || item.Name != cp.Name {
		t.Fatalf("identity mismatch: %+v vs %+v", item, cp)
	}
	if len(item.CompatibleNetworks) == 0 {
		t.Fatal("compatible_networks must be populated from registry")
	}
}

func mustLoadCryptoPolicy(t *testing.T, name string) *CryptoPolicy {
	t.Helper()
	cp, err := LoadCryptoPolicyFromFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("LoadCryptoPolicyFromFile(%s): %v", name, err)
	}
	return cp
}
