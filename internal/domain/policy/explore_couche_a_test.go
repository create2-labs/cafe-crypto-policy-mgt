package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

func TestEvaluateExploreCoucheA_sepoliaScanCompatible(t *testing.T) {
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
		t.Fatalf("scan_compatible: got %d want 1 (%+v rejected)", len(decision.RankedCandidates), decision.RejectedCandidates)
	}
	got := decision.RankedCandidates[0]
	if got.SuggestedUserConstraints == nil || !got.SuggestedUserConstraints.AllowNewWallet {
		t.Fatalf("suggested_user_constraints: %+v", got.SuggestedUserConstraints)
	}
	if got.SolutionProfileRef.ProviderID != "nicetry" {
		t.Fatalf("ref: %+v", got.SolutionProfileRef)
	}
	if got.CompatibilityStatus != AssessmentStatusCompatibleAndDeployable {
		t.Fatalf("status: %s", got.CompatibilityStatus)
	}
	softCodes := map[string]bool{}
	for _, f := range got.CompatibilityFindings {
		softCodes[f.Code] = true
	}
	if !softCodes["requires_bundler"] || !softCodes["requires_local_signer_state"] {
		t.Fatalf("soft findings missing: %+v", got.CompatibilityFindings)
	}
}

func TestEvaluateExploreCoucheA_mainnetPlannedRejected(t *testing.T) {
	reg := mustLoadExploreProviderRegistry(t)
	cp := &CryptoPolicy{
		ID:               "cpm_pq_account_validation_v1",
		RequiredPosture:  vocabulary.PQPostureHybrid,
		AllowedProviders: []string{"nicetry"},
	}
	obs := walletobserved.Payload{AccountKind: "eoa", ChainIDs: []int64{1}}

	decision, err := (ExploreCoucheAEvaluator{Providers: reg}).EvaluateExploreCoucheA(obs, cp)
	if err != nil {
		t.Fatalf("EvaluateExploreCoucheA: %v", err)
	}
	if len(decision.RankedCandidates) != 0 {
		t.Fatalf("want empty scan_compatible, got %d", len(decision.RankedCandidates))
	}
	if len(decision.RejectedCandidates) != 1 {
		t.Fatalf("rejected: %d", len(decision.RejectedCandidates))
	}
	if decision.RejectedCandidates[0].CompatibilityFindings[0].Code != provider.FindingCodeChain {
		t.Fatalf("want chain finding, got %+v", decision.RejectedCandidates[0].CompatibilityFindings)
	}
}

func TestEvaluateExploreCoucheA_erroneousSuggested(t *testing.T) {
	reg := mustLoadExploreProviderRegistry(t)
	resolved, ok := reg.Lookup(provider.ProfileRef{
		ProviderID:        "nicetry",
		SolutionProfileID: "nicetry.fors_c.erc4337.v0_1",
	})
	if !ok {
		t.Fatal("missing nicetry")
	}
	profile := resolved.Profile
	profile.SuggestedUserConstraints = &provider.SuggestedUserConstraints{
		AllowNewWallet:            false,
		AddressContinuityRequired: false,
		KeyRotationModel:          provider.KeyRotationPerUserOp,
	}
	manifest := provider.ProviderManifest{
		SchemaVersion:    provider.SchemaVersionV01,
		ProviderID:       "nicetry",
		ProviderName:     "NiceTry",
		ProviderVersion:  "2026-08",
		ProviderMaturity: provider.MaturityResearch,
		SolutionProfiles: []provider.SolutionProfile{profile},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	synthetic, err := provider.LoadRegistryFromFiles([]string{path})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}

	logger := &recordingLogger{}
	cp := &CryptoPolicy{
		ID:               "cpm_pq_account_validation_v1",
		RequiredPosture:  vocabulary.PQPostureHybrid,
		AllowedProviders: []string{"nicetry"},
	}
	obs := walletobserved.Payload{AccountKind: "eoa", ChainIDs: []int64{11155111}}
	decision, err := (ExploreCoucheAEvaluator{Providers: synthetic, Logger: logger}).EvaluateExploreCoucheA(obs, cp)
	if err != nil {
		t.Fatalf("EvaluateExploreCoucheA: %v", err)
	}
	if len(decision.RankedCandidates) != 0 {
		t.Fatal("erroneous profile must not be scan-compatible")
	}
	if len(decision.RejectedCandidates) != 1 || decision.RejectedCandidates[0].CompatibilityStatus != AssessmentStatusErroneous {
		t.Fatalf("want erroneous rejected, got %+v", decision.RejectedCandidates)
	}
	if len(logger.lines) == 0 || !strings.Contains(logger.lines[0], "erroneous suggested_user_constraints") {
		t.Fatalf("want error log, got %#v", logger.lines)
	}
}

func TestEvaluateExploreCoucheA_coucheBDoesNotInfluence(t *testing.T) {
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
		t.Fatalf("couche B must not filter scan_compatible, got %d", len(decision.RankedCandidates))
	}
}

func mustLoadExploreProviderRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	return reg
}

type recordingLogger struct {
	lines []string
}

func (l *recordingLogger) Printf(format string, v ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, v...))
}
