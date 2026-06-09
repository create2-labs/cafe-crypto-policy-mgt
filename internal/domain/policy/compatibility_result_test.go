package policy

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func mustLoadFixtures(t *testing.T) (*PolicyGraphCatalog, *CryptoPolicyTemplate, *CryptoPolicyInstance) {
	t.Helper()
	cat, err := LoadPolicyGraphCatalogFromFile(testdataPath(t, "policy_graph_catalog_valid.json"))
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	tpl, err := LoadCryptoPolicyTemplateFromFile(testdataPath(t, "crypto_policy_template_valid.json"), cat)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	inst, err := LoadCryptoPolicyInstanceFromFile(testdataPath(t, "crypto_policy_instance_valid.json"), cat)
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	return cat, tpl, inst
}

func baseObservation() walletobserved.Payload {
	return walletobserved.Payload{
		ChainIDs:         []int64{1, 8453},
		AccountKind:      "eoa",
		CurrentAlgorithm: "secp256k1_ecrecover",
		PublicKeyExposed: true,
		IsMultichain:     true,
		ObservedAt:       time.Date(2026, 4, 17, 9, 59, 58, 0, time.UTC),
		CurrentPQPosture: string(vocabulary.PQPostureClassicalOnly),
	}
}

func baseSelection() PolicySelectionRequest {
	return PolicySelectionRequest{
		TargetPosture:             vocabulary.PQPostureHybrid,
		TargetChainIDs:            []int64{1, 8453},
		RequireMultichain:         true,
		AllowNewWallet:            true,
		AddressContinuityRequired: true,
		KeyRotationRequired:       true,
		RecoveryRequired:          true,
		MinimumMaturity:           1,
		AllowedProviderModes:      []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged},
		RequireBundlerAvailable:   true,
		RequirePaymasterAvailable: false,
		ApprovalMode:              ApprovalModeManual,
	}
}

func TestPolicyCompatibilityEvaluator_happyPath(t *testing.T) {
	cat, tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleAndDeployable {
		t.Fatalf("status: got %q findings %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_incompatibleTargetPosture(t *testing.T) {
	cat, tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	req.TargetPosture = vocabulary.PQPostureFullPQ
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible {
		t.Fatalf("want incompatible, got %q %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_compatButNotDeployable(t *testing.T) {
	cat, _, inst := mustLoadFixtures(t)
	inst = shallowCopyInstance(inst)
	inst.TemplateID = ""
	inst.NodePath = []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"}
	inst.Scope.ChainIDs = nil
	obs := baseObservation()
	req := baseSelection()
	req.TargetChainIDs = nil
	req.RequireMultichain = false
	if err := inst.NormalizeAndValidate(cat); err != nil {
		t.Fatalf("instance: %v", err)
	}
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleButNotDeployable {
		t.Fatalf("want not deployable, got %q %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_incompatibleChainScope(t *testing.T) {
	cat, tpl, inst := mustLoadFixtures(t)
	inst = shallowCopyInstance(inst)
	inst.Scope.ChainIDs = []int64{1, 3, 5}
	if err := inst.NormalizeAndValidate(cat); err != nil {
		t.Fatalf("instance: %v", err)
	}

	obs := baseObservation()
	obs.ChainIDs = []int64{1, 2, 5}
	req := baseSelection()
	req.TargetChainIDs = []int64{1, 2, 5}

	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible {
		t.Fatalf("want incompatible, got %q %+v", res.Status, res.Findings)
	}
	found := false
	for _, finding := range res.Findings {
		if finding.Code == "incompatible.chain_scope" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected incompatible.chain_scope finding, got %+v", res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_chainNotObserved(t *testing.T) {
	cat, tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	obs.ChainIDs = []int64{1}
	req := baseSelection()
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible {
		t.Fatalf("want incompatible, got %q %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_requiresTemplateForPath(t *testing.T) {
	cat, _, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	var ev PolicyCompatibilityEvaluator
	// Instance uses template_id only; nil template must error.
	_, err := ev.Evaluate(obs, &req, inst, cat, nil)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestEffectiveNodePath_explicitPath(t *testing.T) {
	cat, _, inst := mustLoadFixtures(t)
	inst = shallowCopyInstance(inst)
	inst.NodePath = []string{"NODE_EOA_ENTRY", "NODE_SIG_EIP7932", "NODE_TARGET_HYBRID"}
	inst.TemplateID = ""
	if err := inst.NormalizeAndValidate(cat); err != nil {
		t.Fatalf("instance: %v", err)
	}
	obs := baseObservation()
	req := baseSelection()
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, cat, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleAndDeployable {
		t.Fatalf("status: got %q", res.Status)
	}
}

func shallowCopyInstance(i *CryptoPolicyInstance) *CryptoPolicyInstance {
	if i == nil {
		return nil
	}
	out := *i
	out.Scope = i.Scope
	out.GlobalParams = i.GlobalParams
	return &out
}
