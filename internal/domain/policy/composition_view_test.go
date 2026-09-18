package policy

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

func TestDeriveCompositionView_nicetryExpectedResultFacts(t *testing.T) {
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	resolved, ok := reg.Lookup(provider.ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
	})
	if !ok {
		t.Fatal("missing nicetry profile")
	}

	got := DeriveCompositionView(&resolved.Profile)
	if got.AccountModel.Standard != "ERC-4337" {
		t.Fatalf("standard: %q", got.AccountModel.Standard)
	}
	if got.AccountModel.ExecutionModel != "erc4337_bundler" {
		t.Fatalf("execution_model: %q", got.AccountModel.ExecutionModel)
	}
	if !got.AccountModel.RequiresBundler || !got.AccountModel.RequiresEntrypoint {
		t.Fatalf("bundler/entrypoint flags: %+v", got.AccountModel)
	}
	if len(got.AccountModel.EntrypointVersions) != 1 || got.AccountModel.EntrypointVersions[0] != "0.7" {
		t.Fatalf("entrypoint_versions: %+v", got.AccountModel.EntrypointVersions)
	}
	if got.Signature.Scheme != "FORS+C" || got.Signature.Family != "hash_based" {
		t.Fatalf("signature: %+v", got.Signature)
	}
	if got.Signature.KeyRotationModel != "per_userop" {
		t.Fatalf("key_rotation_model: %q", got.Signature.KeyRotationModel)
	}
	if !got.Constraints.RequiresNewAccount || got.Constraints.AddressContinuitySupported || !got.Constraints.RequiresLocalSignerState {
		t.Fatalf("constraints: %+v", got.Constraints)
	}

	// Defensive copy: mutating the view must not mutate the registry profile.
	got.AccountModel.EntrypointVersions[0] = "mutated"
	if resolved.Profile.AccountModel.EntrypointVersions[0] != "0.7" {
		t.Fatal("DeriveCompositionView must copy entrypoint_versions")
	}
}

func TestDeriveCompositionView_nilProfile(t *testing.T) {
	got := DeriveCompositionView(nil)
	if got.AccountModel.Standard != "" || got.Signature.Scheme != "" {
		t.Fatalf("nil profile should yield zero view: %+v", got)
	}
}

func TestEvaluateExploreCoucheA_compositionViewOnCompatible(t *testing.T) {
	reg := mustLoadExploreProviderRegistry(t)
	cp := &CryptoPolicy{
		ID:               "cpm_pq_account_validation_v1",
		RequiredPosture:  vocabulary.PQPostureHybrid,
		AllowedProviders: []string{"nicetry"},
	}
	obs := walletobserved.Payload{AccountKind: "eoa", ChainIDs: []int64{11155111}}

	decision, err := (ExploreCoucheAEvaluator{Providers: reg}).EvaluateExploreCoucheA(obs, cp)
	if err != nil {
		t.Fatalf("EvaluateExploreCoucheA: %v", err)
	}
	if len(decision.RankedCandidates) != 1 {
		t.Fatalf("scan_compatible: got %d", len(decision.RankedCandidates))
	}
	comp := decision.RankedCandidates[0].Composition
	if comp == nil {
		t.Fatal("composition must be present on scan_compatible_providers")
	}
	if comp.AccountModel.Standard != "ERC-4337" || !comp.AccountModel.RequiresBundler {
		t.Fatalf("account_model: %+v", comp.AccountModel)
	}
	if comp.Signature.Scheme != "FORS+C" || comp.Signature.KeyRotationModel != "per_userop" {
		t.Fatalf("signature: %+v", comp.Signature)
	}
	if !comp.Constraints.RequiresNewAccount || !comp.Constraints.RequiresLocalSignerState {
		t.Fatalf("constraints: %+v", comp.Constraints)
	}

	raw, err := json.Marshal(decision.RankedCandidates[0])
	if err != nil {
		t.Fatalf("marshal candidate: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal wire: %v", err)
	}
	composition, ok := wire["composition"].(map[string]any)
	if !ok {
		t.Fatalf("composition missing on wire: %s", raw)
	}
	for _, forbidden := range []string{"chain_support", "references", "risk_notes", "input_requirements", "display_name"} {
		if _, present := composition[forbidden]; present {
			t.Fatalf("composition must not leak admin manifest field %q: %s", forbidden, raw)
		}
		if _, present := wire[forbidden]; present {
			t.Fatalf("ranked candidate must not expose admin manifest field %q: %s", forbidden, raw)
		}
	}
}

func TestEvaluateExploreCoucheA_rejectedHasNoComposition(t *testing.T) {
	reg := mustLoadExploreProviderRegistry(t)
	cp := &CryptoPolicy{
		ID:               "cpm_pq_account_validation_v1",
		RequiredPosture:  vocabulary.PQPostureHybrid,
		AllowedProviders: []string{"nicetry"},
	}
	obs := walletobserved.Payload{AccountKind: "eoa", ChainIDs: []int64{56}}

	decision, err := (ExploreCoucheAEvaluator{Providers: reg}).EvaluateExploreCoucheA(obs, cp)
	if err != nil {
		t.Fatalf("EvaluateExploreCoucheA: %v", err)
	}
	if len(decision.RankedCandidates) != 0 {
		t.Fatalf("want empty compatible, got %d", len(decision.RankedCandidates))
	}
	if len(decision.RejectedCandidates) != 1 {
		t.Fatalf("rejected: %d", len(decision.RejectedCandidates))
	}
	raw, err := json.Marshal(decision.RejectedCandidates[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := wire["composition"]; ok {
		t.Fatalf("rejected candidates must not carry composition: %s", raw)
	}
}
