package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type captureLogger struct {
	lines []string
}

func (c *captureLogger) Printf(format string, v ...any) {
	c.lines = append(c.lines, fmt.Sprintf(format, v...))
}

func TestApplyManifestLoadSignals_malformedSuggested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	// Contradicts constraints.requires_new_account=true with allow_new_wallet=false.
	raw := `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "badprov",
  "provider_name": "Bad",
  "provider_version": "0",
  "provider_maturity": "research",
  "solution_profiles": [{
    "solution_profile_id": "bad.profile",
    "display_name": "Bad",
    "maturity": "research",
    "claim_status": "declared",
    "resulting_posture": "hybrid",
    "input_requirements": {"wallet_types": ["EOA"]},
    "signature": {"scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "none"},
    "account_model": {"standard": "EOA", "execution_model": "eoa"},
    "constraints": {"requires_new_account": true, "address_continuity_supported": false},
    "suggested_user_constraints": {
      "allow_new_wallet": false,
      "address_continuity_required": false,
      "key_rotation_model": "none"
    },
    "chain_support": [{"chain_id": 11155111, "network": "sepolia", "status": "testnet_supported", "capabilities": ["deploy", "sign_userop"]}],
    "references": [{"kind": "source_repo", "url": "https://example.com", "commit": "abc"}]
  }]
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	reg, err := LoadRegistryFromFiles([]string{path})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	logger := &captureLogger{}
	n := reg.ApplyManifestLoadSignals(logger)
	if n != 1 {
		t.Fatalf("want 1 malformed, got %d", n)
	}
	got, ok := reg.Lookup(ProfileRef{ProviderID: "badprov", SolutionProfileID: "bad.profile"})
	if !ok || !got.Erroneous {
		t.Fatalf("want Erroneous profile, got ok=%v %+v", ok, got)
	}
	if len(logger.lines) != 1 || !strings.Contains(logger.lines[0], "ERROR catalogue: malformed suggested_user_constraints") {
		t.Fatalf("want ERROR catalogue log, got %#v", logger.lines)
	}
	profiles := reg.ProfilesForProvider("badprov")
	if len(profiles) != 1 || !profiles[0].Erroneous {
		t.Fatalf("ProfilesForProvider must preserve Erroneous: %+v", profiles)
	}
}

func TestApplyManifestLoadSignals_healthyNicetry(t *testing.T) {
	reg, err := LoadRegistryFromFiles([]string{filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json")})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	logger := &captureLogger{}
	n := reg.ApplyManifestLoadSignals(logger)
	if n != 0 {
		t.Fatalf("want 0 malformed on nicetry fixture, got %d logs=%v", n, logger.lines)
	}
	got, ok := reg.Lookup(ProfileRef{ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1"})
	if !ok || got.Erroneous {
		t.Fatalf("nicetry must not be erroneous: ok=%v %+v", ok, got)
	}
}
