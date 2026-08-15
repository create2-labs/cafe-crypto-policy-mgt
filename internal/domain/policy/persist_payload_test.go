package policy

import (
	"errors"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func validPersistPayload() CryptoPolicyPersistPayload {
	return CryptoPolicyPersistPayload{
		SchemaVersion:   CryptoPolicySchemaVersionV02,
		PolicyKind:      "wallet_migration_policy",
		TemplateID:      "tpl_pq_account_validation_v1",
		RequiredPosture: "hybrid",
		SolutionProfileRef: SolutionProfileRef{
			ProviderID: "nicetry", SolutionProfileID: "nicetry.fors_c.erc4337.v0_1", ManifestVersion: "2026-08",
		},
		AcceptedProviderSnapshot: AcceptedProviderSnapshot{
			Maturity: provider.MaturityResearch, ClaimStatus: provider.ClaimDeclared, ResultingPosture: "hybrid",
			Signature:        provider.SignatureProfile{Scheme: "FORS+C", Family: "hash_based", KeyRotationModel: provider.KeyRotationPerUserOp},
			AccountModel:     provider.AccountModel{Standard: "ERC-4337", ExecutionModel: "erc4337_bundler", RequiresBundler: true},
			Constraints:      provider.ProfileConstraints{RequiresNewAccount: true, RequiresLocalSignerState: true},
			ChainSupportUsed: SnapshotChainSupport{ChainID: 11155111, Status: provider.ChainStatusTestnetSupported},
			References: []provider.Reference{
				{Kind: provider.ReferenceKindSourceRepo, URL: "https://example.com/a", Commit: "abc123deadbeef"},
				{Kind: provider.ReferenceKindProtocolSpec, URL: "https://example.com/b", Version: "v0.1.0-test"},
			},
			AcceptedFindings: []string{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState},
		},
	}
}

func TestValidateForPersist_acceptsPinnedSnapshot(t *testing.T) {
	p := validPersistPayload()
	if err := p.ValidateForPersist(); err != nil {
		t.Fatalf("ValidateForPersist: %v", err)
	}
}

func TestValidateForPersist_rejectsUnpinnedAndEmptyRefs(t *testing.T) {
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.References[0].Commit = provider.UnpinnedPendingFixture
	if err := p.ValidateForPersist(); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("unpinned: got %v", err)
	}
	p = validPersistPayload()
	p.AcceptedProviderSnapshot.References = nil
	if err := p.ValidateForPersist(); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("empty: got %v", err)
	}
}

func TestValidateForPersist_rejectsSoftFindingsPlannedSchema(t *testing.T) {
	p := validPersistPayload()
	p.AcceptedProviderSnapshot.AcceptedFindings = []string{provider.FindingCodeRequiresBundler}
	if err := p.ValidateForPersist(); !errors.Is(err, ErrProviderSoftFindingsRequired) {
		t.Fatalf("soft: got %v", err)
	}
	p = validPersistPayload()
	p.AcceptedProviderSnapshot.ChainSupportUsed.Status = provider.ChainStatusPlanned
	if err := p.ValidateForPersist(); !errors.Is(err, ErrProviderChainPlanned) {
		t.Fatalf("planned: got %v", err)
	}
	p = validPersistPayload()
	p.SchemaVersion = "cafe.crypto_policy.v0.1"
	if err := p.ValidateForPersist(); !errors.Is(err, ErrCryptoPolicyPayloadInvalid) {
		t.Fatalf("schema: got %v", err)
	}
}

func TestValidateDraftPayloadForPersist_mapRoundTrip(t *testing.T) {
	raw := map[string]any{
		"schema_version":   CryptoPolicySchemaVersionV02,
		"template_id":      "tpl_pq_account_validation_v1",
		"required_posture": "hybrid",
		"solution_profile_ref": map[string]any{
			"provider_id": "nicetry", "solution_profile_id": "nicetry.fors_c.erc4337.v0_1", "manifest_version": "2026-08",
		},
		"accepted_provider_snapshot": map[string]any{
			"maturity": "research", "claim_status": "declared", "resulting_posture": "hybrid",
			"signature":          map[string]any{"scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop"},
			"account_model":      map[string]any{"standard": "ERC-4337", "execution_model": "erc4337_bundler", "requires_bundler": true},
			"constraints":        map[string]any{"requires_local_signer_state": true},
			"chain_support_used": map[string]any{"chain_id": float64(11155111), "status": "testnet_supported"},
			"references": []any{
				map[string]any{"kind": "source_repo", "url": "https://example.com/a", "commit": "deadbeef"},
			},
			"accepted_findings": []any{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState},
		},
		"policy_context": map[string]any{"wallet_type": "eoa"},
	}
	if err := ValidateDraftPayloadForPersist(raw); err != nil {
		t.Fatalf("ok: %v", err)
	}
	raw["accepted_provider_snapshot"].(map[string]any)["references"] = []any{
		map[string]any{"kind": "source_repo", "url": "https://example.com/a", "commit": provider.UnpinnedPendingFixture},
	}
	if err := ValidateDraftPayloadForPersist(raw); !errors.Is(err, ErrProviderRefsUnpinned) {
		t.Fatalf("want unpinned, got %v", err)
	}
}
