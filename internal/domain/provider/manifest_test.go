package provider

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadProviderManifestFromFile_NicetryFixture(t *testing.T) {
	m, err := LoadProviderManifestFromFile(filepath.Join("testdata", "provider_manifest_nicetry_v0_1.json"))
	if err != nil {
		t.Fatalf("LoadProviderManifestFromFile: %v", err)
	}
	if m.SchemaVersion != SchemaVersionV01 || m.ProviderID != "nicetry" {
		t.Fatalf("identity: schema=%q id=%q", m.SchemaVersion, m.ProviderID)
	}
	if len(m.SolutionProfiles) != 1 {
		t.Fatalf("solution_profiles len: %d", len(m.SolutionProfiles))
	}
	p := m.SolutionProfiles[0]
	if p.SolutionProfileID != "nicetry.fors_c.erc4337.v0_1" {
		t.Fatalf("solution_profile_id: %q", p.SolutionProfileID)
	}
	if p.ClaimStatus != ClaimDeclared || p.Signature.KeyRotationModel != KeyRotationPerUserOp {
		t.Fatalf("claim=%q rotation=%q", p.ClaimStatus, p.Signature.KeyRotationModel)
	}
	if p.ResultingPosture != "hybrid" {
		t.Fatalf("resulting_posture: got %q", p.ResultingPosture)
	}
	if p.Signature.Scheme != "FORS+C" || p.Signature.Family != "hash_based" {
		t.Fatalf("signature: scheme=%q family=%q", p.Signature.Scheme, p.Signature.Family)
	}
	if len(p.References) != 2 {
		t.Fatalf("references len: %d", len(p.References))
	}
	// CPM-P7: real upstream pins (NiceTry main HEAD; Ephemeral-Keys-Protocol main HEAD — no release tags).
	const wantNiceTryCommit = "40a1286d18dee2a92631da82a52e484fa9a3628c"
	const wantProtocolVersion = "ac140c71d400449adec18884c4fd3373592292f3"
	if p.References[0].Commit != wantNiceTryCommit || p.References[1].Version != wantProtocolVersion {
		t.Fatalf("refs not pinned: %#v", p.References)
	}
	if m.HasUnpinnedReferences() {
		t.Fatal("expected pinned Nicetry fixture (no unpinned_pending_fixture)")
	}
}

func validBase() ProviderManifest {
	return ProviderManifest{
		SchemaVersion:    SchemaVersionV01,
		ProviderID:       "nicetry",
		ProviderName:     "NiceTry",
		ProviderVersion:  "2026-08",
		ProviderMaturity: MaturityResearch,
		SolutionProfiles: []SolutionProfile{{
			SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
			DisplayName:       "NiceTry FORS+C ERC-4337 Smart Account",
			Maturity:          MaturityResearch,
			ClaimStatus:       ClaimDeclared,
			ResultingPosture:  "hybrid",
			InputRequirements: InputRequirements{WalletTypes: []string{"EOA"}},
			Signature: SignatureProfile{
				Scheme: "FORS+C", Family: "hash_based", KeyRotationModel: KeyRotationPerUserOp,
			},
			AccountModel: AccountModel{Standard: "ERC-4337", ExecutionModel: "erc4337_bundler"},
			ChainSupport: []ChainSupport{{ChainID: 11155111, Network: "sepolia", Status: ChainStatusTestnetSupported}},
		}},
	}
}

func TestProviderManifest_RejectIncompleteAndUnknownClaim(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ProviderManifest)
		wantErr error
	}{
		{"unknown claim", func(m *ProviderManifest) { m.SolutionProfiles[0].ClaimStatus = "not_a_claim" }, ErrClaimStatusInvalid},
		{"missing claim", func(m *ProviderManifest) { m.SolutionProfiles[0].ClaimStatus = "" }, ErrInvalidManifest},
		{"no profiles", func(m *ProviderManifest) { m.SolutionProfiles = nil }, ErrInvalidManifest},
		{"bad schema", func(m *ProviderManifest) { m.SchemaVersion = "v9" }, ErrInvalidManifest},
		{"bad rotation", func(m *ProviderManifest) { m.SolutionProfiles[0].Signature.KeyRotationModel = "weekly" }, ErrInvalidManifest},
		{"missing resulting_posture", func(m *ProviderManifest) { m.SolutionProfiles[0].ResultingPosture = "" }, ErrInvalidManifest},
		{"bad resulting_posture", func(m *ProviderManifest) { m.SolutionProfiles[0].ResultingPosture = "pq_hash_based" }, ErrInvalidManifest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validBase()
			tc.mutate(&m)
			err := m.NormalizeAndValidate()
			if err == nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v want %v", err, tc.wantErr)
			}
		})
	}
}

func TestReference_IsUnpinned(t *testing.T) {
	if !(Reference{Commit: UnpinnedPendingFixture}).IsUnpinned() ||
		!(Reference{Version: UnpinnedPendingFixture}).IsUnpinned() ||
		!(Reference{}).IsUnpinned() {
		t.Fatal("placeholders/empty should be unpinned")
	}
	if (Reference{Commit: "abc"}).IsUnpinned() || (Reference{Version: "v1"}).IsUnpinned() {
		t.Fatal("pinned refs should not be unpinned")
	}
}
