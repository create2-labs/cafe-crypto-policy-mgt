package policy

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func mustLoadFixtures(t *testing.T) (*CryptoPolicy, *CryptoPolicyInstance) {
	t.Helper()
	cp, err := LoadCryptoPolicyFromFile(testdataPath(t, "crypto_policy_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("crypto policy: %v", err)
	}
	inst, err := LoadCryptoPolicyInstanceFromFile(testdataPath(t, "crypto_policy_instance_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	return cp, inst
}

func mustLoadProviderRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("provider registry: %v", err)
	}
	return reg
}

func baseObservation() walletobserved.Payload {
	return walletobserved.Payload{
		ChainIDs:         []int64{11155111},
		AccountKind:      "eoa",
		CurrentAlgorithm: "secp256k1_ecrecover",
		PublicKeyExposed: true,
		IsMultichain:     false,
		ObservedAt:       time.Date(2026, 4, 17, 9, 59, 58, 0, time.UTC),
		CurrentPQPosture: string(vocabulary.PQPostureClassicalOnly),
	}
}

func baseSelection() PolicySelectionRequest {
	return PolicySelectionRequest{
		TargetPosture:             vocabulary.PQPostureHybrid,
		TargetChainIDs:            []int64{11155111},
		RequireMultichain:         false,
		AllowNewWallet:            true,
		AddressContinuityRequired: false,
		KeyRotationModel:          KeyRotationPerUserOp,
		RecoveryRequired:          true,
		MinimumMaturity:           1,
		AllowedProviderModes:      []ProviderMode{ProviderModeThirdParty, ProviderModeUserManaged},
		RequireBundlerAvailable:   true,
		RequirePaymasterAvailable: false,
		ApprovalMode:              ApprovalModeManual,
	}
}

func TestPolicyCompatibilityEvaluator_happyPath(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleAndDeployable {
		t.Fatalf("status: got %q findings %+v", res.Status, res.Findings)
	}
	assertSoftFindings(t, res.Findings)
}

func TestPolicyCompatibilityEvaluator_requiredPostureMustEqualResultingPosture(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	tpl.RequiredPosture = vocabulary.PQPostureFullPQ
	obs := baseObservation()
	req := baseSelection()
	req.TargetPosture = vocabulary.PQPostureFullPQ
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible {
		t.Fatalf("want incompatible, got %q %+v", res.Status, res.Findings)
	}
	if len(res.Findings) == 0 || res.Findings[0].Code != provider.FindingCodePosture {
		t.Fatalf("want %s, got %+v", provider.FindingCodePosture, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_compatButNotDeployable(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	inst = shallowCopyInstance(inst)
	inst.Scope.ChainIDs = nil
	obs := baseObservation()
	req := baseSelection()
	req.TargetChainIDs = nil
	req.RequireMultichain = false
	if err := inst.NormalizeAndValidate(); err != nil {
		t.Fatalf("instance: %v", err)
	}
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleButNotDeployable {
		t.Fatalf("want not deployable, got %q %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_incompatibleChainScope(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	inst = shallowCopyInstance(inst)
	inst.Scope.ChainIDs = []int64{1}
	inst.Scope.RequireMultichain = false
	if err := inst.NormalizeAndValidate(); err != nil {
		t.Fatalf("instance: %v", err)
	}

	obs := baseObservation()
	req := baseSelection()

	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
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
	tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	obs.ChainIDs = nil
	req := baseSelection()
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible {
		t.Fatalf("want incompatible, got %q %+v", res.Status, res.Findings)
	}
}

func TestPolicyCompatibilityEvaluator_requiresTemplate(t *testing.T) {
	_, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	_, err := ev.Evaluate(obs, &req, inst, nil)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestPolicyCompatibilityEvaluator_missingRegistryFailsClosed(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	obs := baseObservation()
	req := baseSelection()
	var ev PolicyCompatibilityEvaluator
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible ||
		len(res.Findings) == 0 ||
		res.Findings[0].Code != provider.FindingCodeUnresolved {
		t.Fatalf("want fail-closed unresolved, got %+v", res)
	}
}

func TestPolicyCompatibilityEvaluator_unresolvedProfileFailsClosed(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	inst.SolutionProfileRef.ManifestVersion = "missing-version"
	obs := baseObservation()
	req := baseSelection()
	ev := PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}
	res, err := ev.Evaluate(obs, &req, inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusIncompatible ||
		len(res.Findings) == 0 ||
		res.Findings[0].Code != provider.FindingCodeUnresolved {
		t.Fatalf("want unresolved rejection, got %+v", res)
	}
}

func TestPolicyCompatibilityEvaluator_providerProfileIsConstraintAuthority(t *testing.T) {
	tpl, inst := mustLoadFixtures(t)
	inst.GlobalParams.AllowNewWallet = false
	inst.GlobalParams.AddressContinuityRequired = true
	inst.GlobalParams.KeyRotationModel = KeyRotationNone

	res, err := (PolicyCompatibilityEvaluator{Providers: mustLoadProviderRegistry(t)}).
		Evaluate(baseObservation(), ptrSelection(baseSelection()), inst, tpl)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Status != AssessmentStatusCompatibleAndDeployable {
		t.Fatalf("legacy global provider copies must not override profile: %+v", res)
	}
}

func assertSoftFindings(t *testing.T, findings []AssessmentFinding) {
	t.Helper()
	got := map[string]AssessmentFinding{}
	for _, f := range findings {
		got[f.Code] = f
		if f.Code == "requires_wallet_control_proof" {
			t.Fatal("requires_wallet_control_proof must not appear on explore findings")
		}
	}
	for _, code := range []string{provider.FindingCodeRequiresBundler, provider.FindingCodeRequiresLocalSignerState} {
		f, ok := got[code]
		if !ok {
			t.Fatalf("missing soft finding %q in %+v", code, findings)
		}
		if f.Severity != AssessmentFindingSeverityWarning {
			t.Fatalf("%s severity: got %q want warning", code, f.Severity)
		}
	}
}

func ptrSelection(req PolicySelectionRequest) *PolicySelectionRequest {
	return &req
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
