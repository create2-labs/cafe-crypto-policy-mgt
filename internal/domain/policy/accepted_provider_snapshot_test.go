package policy

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func TestBuildAcceptedProviderSnapshot_nicetryMultichain(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{
		ID:               "cpm_pq_account_validation_v1",
		Name:             "PQ account validation",
		Version:          "1",
		RequiredPosture:  "hybrid",
		AllowedProviders: []string{"nicetry"},
	}

	got, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp,
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		AcceptedFindings: nil, // required soft findings must be auto-merged
		Registry:         reg,
	})
	if err != nil {
		t.Fatalf("BuildAcceptedProviderSnapshot: %v", err)
	}
	if got.SolutionProfileRef.ProviderID != "nicetry" || got.SolutionProfileRef.ManifestVersion != "2026-08" {
		t.Fatalf("ref: %+v", got.SolutionProfileRef)
	}
	snap := got.AcceptedProviderSnapshot
	if snap.Maturity != provider.MaturityResearch || snap.ClaimStatus != provider.ClaimDeclared {
		t.Fatalf("maturity/claim: %+v", snap)
	}
	if len(snap.ChainSupportUsed) < 2 {
		t.Fatalf("expected multi-chain pin, got %#v", snap.ChainSupportUsed)
	}
	// Sorted by chain_id; first should be ethereum mainnet (1).
	if snap.ChainSupportUsed[0].ChainID.Int64() != 1 {
		t.Fatalf("first chain_id want 1, got %+v", snap.ChainSupportUsed[0])
	}
	for i := 1; i < len(snap.ChainSupportUsed); i++ {
		if snap.ChainSupportUsed[i].ChainID.Int64() <= snap.ChainSupportUsed[i-1].ChainID.Int64() {
			t.Fatalf("chain_support_used not sorted: %#v", snap.ChainSupportUsed)
		}
		if snap.ChainSupportUsed[i].Status == provider.ChainStatusPlanned {
			t.Fatalf("planned must not appear: %#v", snap.ChainSupportUsed[i])
		}
	}
	var hasSepolia bool
	for _, cs := range snap.ChainSupportUsed {
		if cs.ChainID.Int64() == 11155111 {
			hasSepolia = true
			if cs.Status != provider.ChainStatusTestnetSupported {
				t.Fatalf("sepolia status: %+v", cs)
			}
		}
	}
	if !hasSepolia {
		t.Fatalf("missing sepolia in %#v", snap.ChainSupportUsed)
	}
	if len(snap.AcceptedFindings) != 2 ||
		snap.AcceptedFindings[0] != provider.FindingCodeRequiresBundler ||
		snap.AcceptedFindings[1] != provider.FindingCodeRequiresLocalSignerState {
		t.Fatalf("findings: %#v", snap.AcceptedFindings)
	}
	if len(snap.References) == 0 || snap.References[0].IsUnpinned() {
		t.Fatalf("refs must be pinned from registry: %#v", snap.References)
	}
	if !snap.AccountModel.RequiresBundler || !snap.Constraints.RequiresLocalSignerState {
		t.Fatalf("flags: account=%+v constraints=%+v", snap.AccountModel, snap.Constraints)
	}
	if err := (&CryptoPolicyPersistPayload{
		SchemaVersion:            CryptoPolicySchemaVersionV02,
		CryptoPolicyID:           cp.ID,
		RequiredPosture:          cp.RequiredPosture,
		UserConstraints:          provider.UserConstraints{AllowNewWallet: true, KeyRotationModel: provider.KeyRotationPerUserOp},
		SolutionProfileRef:       got.SolutionProfileRef,
		AcceptedProviderSnapshot: snap,
		AcceptedFindings:         snap.AcceptedFindings,
	}).ValidateForPersist(persistObsEOA()); err != nil {
		t.Fatalf("registry snapshot must pass persist gates (constrained): %v", err)
	}
	if err := (&CryptoPolicyPersistPayload{
		SchemaVersion:            CryptoPolicySchemaVersionV02,
		CryptoPolicyID:           cp.ID,
		RequiredPosture:          cp.RequiredPosture,
		UserConstraints:          provider.UserConstraints{AllowNewWallet: true, KeyRotationModel: provider.KeyRotationPerUserOp},
		SolutionProfileRef:       got.SolutionProfileRef,
		AcceptedProviderSnapshot: snap,
		AcceptedFindings:         snap.AcceptedFindings,
	}).ValidateForPersist(provider.HardObservation{AccountKind: "eoa", ChainIDs: nil}); err != nil {
		t.Fatalf("registry snapshot must pass persist gates (greenfield): %v", err)
	}
}

func TestBuildAcceptedProviderSnapshot_rejectsNoDeployableChains(t *testing.T) {
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Mutate in-memory profile to planned-only.
	resolved, ok := reg.Lookup(provider.ProfileRef{
		ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
	})
	if !ok || resolved == nil {
		t.Fatal("lookup")
	}
	for i := range resolved.Profile.ChainSupport {
		resolved.Profile.ChainSupport[i].Status = provider.ChainStatusPlanned
	}

	cp := &CryptoPolicy{ID: "cp", Name: "n", Version: "1", RequiredPosture: "hybrid", AllowedProviders: []string{"nicetry"}}
	ref := SolutionProfileRef{ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08"}
	_, err = BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp, SolutionProfileRef: ref, Registry: reg,
	})
	if !errors.Is(err, ErrSnapshotAssistChainNotSupported) {
		t.Fatalf("planned-only: got %v", err)
	}
}

func TestBuildAcceptedProviderSnapshot_rejectsProviderNotAllowed(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{ID: "cp", Name: "n", Version: "1", RequiredPosture: "hybrid", AllowedProviders: []string{"other"}}
	_, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp,
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		Registry: reg,
	})
	if !errors.Is(err, ErrSnapshotAssistProviderNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildAcceptedProviderSnapshot_rejectsUnknownFinding(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{ID: "cp", Name: "n", Version: "1", RequiredPosture: "hybrid", AllowedProviders: []string{"nicetry"}}
	_, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp,
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		AcceptedFindings: []string{"not_a_real_finding"},
		Registry:         reg,
	})
	if !errors.Is(err, ErrSnapshotAssistFindingsUnknown) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildAcceptedProviderSnapshot_rejectsMissingProfile(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{ID: "cp", Name: "n", Version: "1", RequiredPosture: "hybrid", AllowedProviders: []string{"nicetry"}}
	_, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp,
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "missing.profile", ManifestVersion: "2026-08",
		},
		Registry: reg,
	})
	if !errors.Is(err, ErrSnapshotAssistProfileNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateForPersist_rejectsConstrainedNoOverlap(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{
		ID: "cpm_pq_account_validation_v1", Name: "n", Version: "1",
		RequiredPosture: "hybrid", AllowedProviders: []string{"nicetry"},
	}
	got, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp,
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		Registry: reg,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := &CryptoPolicyPersistPayload{
		SchemaVersion:            CryptoPolicySchemaVersionV02,
		CryptoPolicyID:           cp.ID,
		RequiredPosture:          cp.RequiredPosture,
		UserConstraints:          provider.UserConstraints{AllowNewWallet: true, KeyRotationModel: provider.KeyRotationPerUserOp},
		SolutionProfileRef:       got.SolutionProfileRef,
		AcceptedProviderSnapshot: got.AcceptedProviderSnapshot,
		AcceptedFindings:         got.AcceptedProviderSnapshot.AcceptedFindings,
	}
	// BSC (56) is not in nicetry deployable set → constrained couche A fail.
	if err := p.ValidateForPersist(provider.HardObservation{AccountKind: "eoa", ChainIDs: []int64{56}}); !errors.Is(err, ErrProviderScanCompatFailed) {
		t.Fatalf("want scan compat fail for no overlap, got %v", err)
	}
}

func mustLoadNicetryRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return reg
}
