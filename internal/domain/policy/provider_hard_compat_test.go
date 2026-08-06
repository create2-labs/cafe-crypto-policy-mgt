package policy

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

func TestPolicyDecisionEvaluator_providerHard_rankedVsRejected(t *testing.T) {
	cat, tpl, inst := mustLoadFixtures(t)
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	ev := PolicyDecisionEvaluator{
		CompatibilityEvaluator: PolicyCompatibilityEvaluator{Providers: reg},
	}
	obs := walletobserved.Payload{
		ChainIDs:         []int64{11155111},
		AccountKind:      "eoa",
		CurrentAlgorithm: "secp256k1_ecrecover",
		PublicKeyExposed: true,
		ObservedAt:       time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
		CurrentPQPosture: string(vocabulary.PQPostureClassicalOnly),
	}
	okReq := PolicySelectionRequest{
		TargetPosture:             vocabulary.PQPostureHybrid,
		TargetChainIDs:            []int64{11155111},
		AllowNewWallet:            true,
		AddressContinuityRequired: false,
		KeyRotationModel:          KeyRotationPerUserOp,
		RecoveryRequired:          true,
		MinimumMaturity:           1,
		AllowedProviderModes:      []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged},
		RequireBundlerAvailable:   true,
		ApprovalMode:              ApprovalModeManual,
	}
	decision, err := ev.Evaluate(obs, &okReq, []PolicyDecisionCandidate{{Instance: inst, Template: tpl}}, cat)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(decision.RankedCandidates) != 1 || len(decision.RejectedCandidates) != 0 {
		t.Fatalf("want ranked, got ranked=%d rejected=%d", len(decision.RankedCandidates), len(decision.RejectedCandidates))
	}

	bad := okReq
	bad.KeyRotationModel = KeyRotationNone
	decision, err = ev.Evaluate(obs, &bad, []PolicyDecisionCandidate{{Instance: inst, Template: tpl}}, cat)
	if err != nil {
		t.Fatalf("Evaluate reject: %v", err)
	}
	if len(decision.RankedCandidates) != 0 || len(decision.RejectedCandidates) != 1 {
		t.Fatalf("want rejected, got ranked=%d rejected=%d", len(decision.RankedCandidates), len(decision.RejectedCandidates))
	}
	if decision.RejectedCandidates[0].RejectionReasons[0].Code != provider.FindingCodeRotation {
		t.Fatalf("rejection: %+v", decision.RejectedCandidates[0].RejectionReasons)
	}
}
