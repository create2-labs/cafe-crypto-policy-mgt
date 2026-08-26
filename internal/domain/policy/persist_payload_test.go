package policy

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func validPersistPayload() CryptoPolicyPersistPayload {
	return CryptoPolicyPersistPayload{
		SchemaVersion:   CryptoPolicySchemaVersionV02,
		PolicyKind:      "wallet_migration_policy",
		CryptoPolicyID:  "cpm_pq_account_validation_v1",
		RequiredPosture: "hybrid",
		UserConstraints: provider.UserConstraints{
			AllowNewWallet: true, AddressContinuityRequired: false, KeyRotationModel: provider.KeyRotationPerUserOp,
		},
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		AcceptedProviderSnapshot: AcceptedProviderSnapshot{
			Maturity: provider.MaturityResearch, ClaimStatus: provider.ClaimDeclared, ResultingPosture: "hybrid",
			InputRequirements: provider.InputRequirements{WalletTypes: []string{"EOA"}, RequiresWalletControlProof: true},
			Signature:         provider.SignatureProfile{Scheme: "FORS+C", Family: "hash_based", KeyRotationModel: provider.KeyRotationPerUserOp},
			AccountModel:      provider.AccountModel{Standard: "ERC-4337", ExecutionModel: "erc4337_bundler", RequiresBundler: true},
			Constraints:       provider.ProfileConstraints{RequiresNewAccount: true, RequiresLocalSignerState: true},
			ChainSupportUsed: SnapshotChainSupport{
				ChainID: 11155111, Status: provider.ChainStatusTestnetSupported,
				Capabilities: []string{provider.CapabilityDeploy, provider.CapabilitySignUserOp, provider.CapabilityRotateSigner},
			},
			References: []provider.Reference{
				{Kind: provider.ReferenceKindSourceRepo, URL: "https://example.com/a", Commit: "abc123deadbeef"},
				{Kind: provider.ReferenceKindProtocolSpec, URL: "https://example.com/b", Version: "v0.1.0-test"},
			},
			AcceptedFindings: []string{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState},
		},
	}
}

func persistObsEOA() provider.HardObservation {
	return provider.HardObservation{AccountKind: "eoa", ChainIDs: []int64{11155111}}
}

func TestValidateForPersist_acceptsPinnedSnapshot(t *testing.T) {
	p := validPersistPayload()
	if err := p.ValidateForPersist(persistObsEOA()); err != nil {
		t.Fatalf("ValidateForPersist: %v", err)
	}
}

// CPM-P7: persist gate accepts the shipped Nicetry fixture once refs are real pins
// (complements the synthetic pinned case above / P10).
func TestValidateForPersist_acceptsPinnedNicetryFixtureRefs(t *testing.T) {
	m, err := provider.LoadProviderManifestFromFile(filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"))
	if err != nil {
		t.Fatalf("load nicetry fixture: %v", err)
	}
	if m.HasUnpinnedReferences() {
		t.Fatal("nicetry fixture must be pinned for CPM-P7")
	}
	profile := m.SolutionProfiles[0]
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.References = append([]provider.Reference(nil), profile.References...)
	if err := p.ValidateForPersist(persistObsEOA()); err != nil {
		t.Fatalf("ValidateForPersist with nicetry fixture refs: %v", err)
	}
}

func TestValidateForPersist_rejectsUnpinnedAndEmptyRefs(t *testing.T) {
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.References[0].Commit = provider.UnpinnedPendingFixture
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("unpinned: got %v", err)
	}
	p = validPersistPayload()
	p.AcceptedProviderSnapshot.References = nil
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("empty: got %v", err)
	}
}

func TestValidateForPersist_rejectsSoftFindingsPlannedSchema(t *testing.T) {
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.AcceptedFindings = []string{provider.FindingCodeRequiresBundler}
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrProviderSoftFindingsRequired) {
		t.Fatalf("soft: got %v", err)
	}
	p = validPersistPayload()
	p.AcceptedProviderSnapshot.ChainSupportUsed.Status = provider.ChainStatusPlanned
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrProviderChainPlanned) {
		t.Fatalf("planned: got %v", err)
	}
	p = validPersistPayload()
	p.SchemaVersion = "cafe.crypto_policy.v0.1"
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("schema: got %v", err)
	}
}

func TestValidateForPersist_rejectsCoucheBKO(t *testing.T) {
	p := validPersistPayload()
	p.UserConstraints.AddressContinuityRequired = true
	err := p.ValidateForPersist(persistObsEOA())
	if !errors.Is(err, ErrProviderUserConstraintsIncompatible) {
		t.Fatalf("couche B: got %v", err)
	}
	if got := UserConstraintsIncompatibleFindingCode(err); got != provider.FindingCodeContinuity {
		t.Fatalf("finding code: got %q want %q", got, provider.FindingCodeContinuity)
	}
}

func TestValidateForPersist_rejectsCoucheAKO(t *testing.T) {
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.ChainSupportUsed.Capabilities = []string{provider.CapabilityDeploy}
	if err := p.ValidateForPersist(persistObsEOA()); !errors.Is(err, ErrProviderScanCompatFailed) {
		t.Fatalf("couche A: got %v", err)
	}
}

func TestValidatePayloadForPersist_mapRoundTrip(t *testing.T) {
	raw := map[string]any{
		"schema_version":   CryptoPolicySchemaVersionV02,
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"required_posture": "hybrid",
		"user_constraints": map[string]any{
			"allow_new_wallet": true, "address_continuity_required": false, "key_rotation_model": "per_userop",
		},
		"solution_profile_ref": map[string]any{
			"provider_id": "nicetry", "solution_profile_id": "nicetry.fors_c.erc4337.v0_1", "manifest_version": "2026-08",
		},
		"accepted_provider_snapshot": map[string]any{
			"maturity": "research", "claim_status": "declared", "resulting_posture": "hybrid",
			"input_requirements": map[string]any{"wallet_types": []any{"EOA"}},
			"signature":          map[string]any{"scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop"},
			"account_model":      map[string]any{"standard": "ERC-4337", "execution_model": "erc4337_bundler", "requires_bundler": true},
			"constraints":        map[string]any{"requires_new_account": true, "requires_local_signer_state": true},
			"chain_support_used": map[string]any{
				"chain_id": float64(11155111), "status": "testnet_supported",
				"capabilities": []any{"deploy", "sign_userop", "rotate_signer"},
			},
			"references": []any{
				map[string]any{"kind": "source_repo", "url": "https://example.com/a", "commit": "deadbeef"},
			},
			"accepted_findings": []any{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState},
		},
		"policy_context": map[string]any{"wallet_type": "eoa", "chain_ids": []any{float64(11155111)}},
	}
	if err := ValidatePayloadForPersist(raw); err != nil {
		t.Fatalf("ok: %v", err)
	}
	raw["accepted_provider_snapshot"].(map[string]any)["references"] = []any{
		map[string]any{"kind": "source_repo", "url": "https://example.com/a", "commit": provider.UnpinnedPendingFixture},
	}
	if err := ValidatePayloadForPersist(raw); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("want unpinned, got %v", err)
	}
}

func TestValidatePayloadForPersist_rejectsLegacy(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{
			"schema_version":   CryptoPolicySchemaVersionV02,
			"crypto_policy_id": "cpm_pq_account_validation_v1",
			"required_posture": "hybrid",
			"user_constraints": map[string]any{
				"allow_new_wallet": true, "address_continuity_required": false, "key_rotation_model": "per_userop",
			},
			"solution_profile_ref": map[string]any{
				"provider_id": "nicetry", "solution_profile_id": "nicetry.fors_c.erc4337.v0_1", "manifest_version": "2026-08",
			},
			"accepted_provider_snapshot": map[string]any{
				"maturity": "research", "claim_status": "declared", "resulting_posture": "hybrid",
				"input_requirements": map[string]any{"wallet_types": []any{"EOA"}},
				"signature":          map[string]any{"scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop"},
				"account_model":      map[string]any{"requires_bundler": true},
				"constraints":        map[string]any{"requires_new_account": true, "requires_local_signer_state": true},
				"chain_support_used": map[string]any{
					"chain_id": float64(11155111), "status": "testnet_supported",
					"capabilities": []any{"deploy", "sign_userop", "rotate_signer"},
				},
				"references": []any{
					map[string]any{"kind": "source_repo", "url": "https://example.com/a", "commit": "deadbeef"},
				},
				"accepted_findings": []any{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState},
			},
			"policy_context": map[string]any{"wallet_type": "eoa"},
		}
	}

	withTemplate := base()
	withTemplate["template_id"] = "tpl_legacy"
	if err := ValidatePayloadForPersist(withTemplate); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("template_id: %v", err)
	}

	missingUC := base()
	delete(missingUC, "user_constraints")
	if err := ValidatePayloadForPersist(missingUC); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("missing user_constraints: %v", err)
	}

	missingCP := base()
	delete(missingCP, "crypto_policy_id")
	if err := ValidatePayloadForPersist(missingCP); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("missing crypto_policy_id: %v", err)
	}

	withSelection := base()
	withSelection["selection_request"] = map[string]any{"allow_new_wallet": true}
	if err := ValidatePayloadForPersist(withSelection); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("selection_request: %v", err)
	}
}
