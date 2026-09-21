package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func TestDeriveAllowedProviderSummaries_NicetryAndNicetry2DistinctSchemes(t *testing.T) {
	cp := mustLoadCryptoPolicy(t, "crypto_policy_pq_account_validation_v1.json")
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry2_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}

	got := DeriveAllowedProviderSummaries(cp, reg)
	if len(got) != 2 {
		t.Fatalf("want 2 summaries (nicetry + nicetry2), got %d (%+v)", len(got), got)
	}
	if got[0].ProviderID != "nicetry" || got[1].ProviderID != "nicetry2" {
		t.Fatalf("order must follow allowed_providers: %+v", got)
	}
	if got[0].Signature.Scheme != "FORS+C" || got[0].Signature.Family != "hash_based" {
		t.Fatalf("nicetry signature: %+v", got[0].Signature)
	}
	if got[1].Signature.Scheme != "MLDSA" || got[1].Signature.Family != "hash_based" {
		t.Fatalf("nicetry2 signature: %+v", got[1].Signature)
	}
	for _, row := range got {
		if len(row.Networks) == 0 {
			t.Fatalf("%s: networks must be non-empty", row.ProviderID)
		}
		for _, n := range row.Networks {
			if n.Status == provider.ChainStatusPlanned {
				t.Fatalf("%s: planned must be absent: %+v", row.ProviderID, n)
			}
			if n.ChainID <= 0 || n.Network == "" {
				t.Fatalf("%s: incomplete network: %+v", row.ProviderID, n)
			}
		}
		for i := 1; i < len(row.Networks); i++ {
			if row.Networks[i].ChainID <= row.Networks[i-1].ChainID {
				t.Fatalf("%s: networks not sorted: %+v", row.ProviderID, row.Networks)
			}
		}
	}
}

func TestDeriveAllowedProviderSummaries_ExcludesPlannedFromRowNetworks(t *testing.T) {
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
      { "chain_id": 11155111, "network": "sepolia", "status": "testnet_supported", "capabilities": ["deploy"] },
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

	got := DeriveAllowedProviderSummaries(cp, reg)
	if len(got) != 1 || got[0].ProviderID != "nicetry" {
		t.Fatalf("want nicetry only (nicetry2 unknown), got %+v", got)
	}
	if len(got[0].Networks) != 1 || got[0].Networks[0].ChainID != 11155111 {
		t.Fatalf("want Sepolia-only networks, got %+v", got[0].Networks)
	}
}

func TestDeriveAllowedProviderSummaries_PrefersPostureCompatibleProfile(t *testing.T) {
	cp := &CryptoPolicy{
		ID:               "x",
		Name:             "x",
		Version:          "1",
		RequiredPosture:  "hybrid",
		AllowedProviders: []string{"multiprof"},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "multiprof.json")
	const multi = `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "multiprof",
  "provider_name": "Multi",
  "provider_version": "1",
  "provider_maturity": "research",
  "solution_profiles": [
    {
      "solution_profile_id": "multiprof.classical.v0_1",
      "display_name": "Classical",
      "maturity": "research",
      "claim_status": "declared",
      "resulting_posture": "classical_only",
      "input_requirements": { "wallet_types": ["EOA"], "requires_wallet_control_proof": true },
      "signature": { "scheme": "ECDSA", "family": "elliptic_curve", "key_rotation_model": "per_userop" },
      "account_model": {
        "standard": "EOA",
        "execution_model": "eoa",
        "requires_bundler": false,
        "requires_entrypoint": false
      },
      "constraints": {
        "requires_new_account": false,
        "address_continuity_supported": true,
        "requires_local_signer_state": false
      },
      "chain_support": [
        { "chain_id": 1, "network": "ethereum-mainnet", "status": "production_supported", "capabilities": ["sign"] }
      ]
    },
    {
      "solution_profile_id": "multiprof.hybrid.v0_1",
      "display_name": "Hybrid",
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
        { "chain_id": 11155111, "network": "sepolia", "status": "testnet_supported", "capabilities": ["deploy"] }
      ]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(multi), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	reg, err := provider.LoadRegistryFromFiles([]string{path})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}

	got := DeriveAllowedProviderSummaries(cp, reg)
	if len(got) != 1 {
		t.Fatalf("want 1 summary, got %+v", got)
	}
	if got[0].Signature.Scheme != "FORS+C" {
		t.Fatalf("want hybrid posture profile (FORS+C), got %+v", got[0])
	}
	if len(got[0].Networks) != 1 || got[0].Networks[0].ChainID != 11155111 {
		t.Fatalf("want hybrid profile network, got %+v", got[0].Networks)
	}
}

func TestDeriveAllowedProviderSummaries_UnknownProviderSkipped(t *testing.T) {
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
	got := DeriveAllowedProviderSummaries(cp, reg)
	if got == nil || len(got) != 0 {
		t.Fatalf("want empty slice, got %#v", got)
	}
}
