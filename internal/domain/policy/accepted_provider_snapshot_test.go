package policy

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func TestBuildAcceptedProviderSnapshot_nicetrySepolia(t *testing.T) {
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
		ChainID:          11155111,
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
	if snap.ChainSupportUsed.ChainID.Int64() != 11155111 || snap.ChainSupportUsed.Status != provider.ChainStatusTestnetSupported {
		t.Fatalf("chain: %+v", snap.ChainSupportUsed)
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
		t.Fatalf("registry snapshot must pass persist gates: %v", err)
	}
}

func TestBuildAcceptedProviderSnapshot_rejectsPlannedAndUnknownChain(t *testing.T) {
	reg := mustLoadNicetryRegistry(t)
	cp := &CryptoPolicy{ID: "cp", Name: "n", Version: "1", RequiredPosture: "hybrid", AllowedProviders: []string{"nicetry"}}
	ref := SolutionProfileRef{ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08"}

	_, err := BuildAcceptedProviderSnapshot(BuildAcceptedProviderSnapshotInput{
		CryptoPolicy: cp, SolutionProfileRef: ref, ChainID: 999999, Registry: reg,
	})
	if !errors.Is(err, ErrSnapshotAssistChainNotSupported) {
		t.Fatalf("unknown chain: got %v", err)
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
		ChainID:  11155111,
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
		ChainID:          11155111,
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
		ChainID:  11155111,
		Registry: reg,
	})
	if !errors.Is(err, ErrSnapshotAssistProfileNotFound) {
		t.Fatalf("got %v", err)
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
