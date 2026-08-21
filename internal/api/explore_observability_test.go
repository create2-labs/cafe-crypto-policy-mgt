package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func TestDominantExploreRejectionCode_prefersChainScope(t *testing.T) {
	rejected := []policy.RejectedPolicy{
		{
			RejectionReasons: []policy.AssessmentFinding{
				{Code: "incompatible.required_posture", Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
		{
			RejectionReasons: []policy.AssessmentFinding{
				{Code: rejectionCodeChainScope, Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
	}
	if got := dominantExploreRejectionCode(rejected); got != rejectionCodeChainScope {
		t.Fatalf("dominant code: got %q want %q", got, rejectionCodeChainScope)
	}
}

func TestDominantExploreRejectionCode_stableFallback(t *testing.T) {
	rejected := []policy.RejectedPolicy{
		{
			RejectionReasons: []policy.AssessmentFinding{
				{Code: "incompatible.multichain", Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
		{
			RejectionReasons: []policy.AssessmentFinding{
				{Code: "incompatible.maturity", Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
	}
	if got := dominantExploreRejectionCode(rejected); got != "incompatible.maturity" {
		t.Fatalf("dominant code: got %q want incompatible.maturity", got)
	}
}

func TestBucketMissingChainCount_chainScopeMinimum(t *testing.T) {
	rejected := []policy.RejectedPolicy{
		{
			CryptoPolicyInstanceID: "far",
			RejectionReasons: []policy.AssessmentFinding{
				{Code: rejectionCodeChainScope, Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
		{
			CryptoPolicyInstanceID: "close",
			RejectionReasons: []policy.AssessmentFinding{
				{Code: rejectionCodeChainScope, Severity: policy.AssessmentFindingSeverityBlocking},
			},
		},
	}
	scopes := map[string][]int64{
		"far":   {1, 3, 5},
		"close": {1, 2, 3},
	}
	target := []int64{1, 2, 5}
	if got := bucketMissingChainCount(target, rejected, scopes, rejectionCodeChainScope); got != "1" {
		t.Fatalf("missing_chain_count bucket: got %q want 1", got)
	}
}

func TestBucketMissingChainCount_nonChainScopeDominantIsUnknown(t *testing.T) {
	got := bucketMissingChainCount(
		[]int64{1, 2, 5},
		[]policy.RejectedPolicy{{
			RejectionReasons: []policy.AssessmentFinding{
				{Code: "incompatible.required_posture", Severity: policy.AssessmentFindingSeverityBlocking},
			},
		}},
		map[string][]int64{"inst": {1, 2, 5}},
		"incompatible.required_posture",
	)
	if got != exploreLabelUnknown {
		t.Fatalf("missing_chain_count bucket: got %q want unknown", got)
	}
}

func TestExploreObservability_emitsMetricAndLogForNoCandidate(t *testing.T) {
	metrics := &testExploreMetrics{}
	logger := &testExploreLogger{}
	restore := setExploreObservabilityForTest(exploreObservability{
		logger:  logger,
		metrics: metrics,
	})
	defer restore()

	decision := policy.PolicyDecision{
		ObservedWalletSummary: policy.ObservedWalletSummary{ChainIDs: []int64{1, 2, 5}},
		RequestSummary: policy.RequestSummary{
			TargetChainIDs: []int64{1, 2, 5},
		},
		RejectedCandidates: []policy.RejectedPolicy{{
			CryptoPolicyInstanceID: "cpx_pq_account_validation_v1",
			TemplateID:             "tpl_pq_account_validation_v1",
			RejectionReasons: []policy.AssessmentFinding{{
				Code:     rejectionCodeChainScope,
				Severity: policy.AssessmentFindingSeverityBlocking,
			}},
		}},
	}
	req := &decisionExploreRequest{
		ScanID: "scan-123",
		PolicyContext: &walletPolicyContextWire{
			ScanID:        "scan-123",
			WalletAddress: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			WalletType:    "EOA",
			ChainIDs:      []int64{1, 2, 5},
		},
	}
	scopes := map[string][]int64{"cpx_pq_account_validation_v1": {1, 3, 5}}
	httpReq := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, nil)
	httpReq.Header.Set("X-Request-Id", "req-ops-1")

	exploreObs.recordNoDeployableCandidate(httpReq, req, decision, scopes)

	if len(metrics.increments) != 1 {
		t.Fatalf("metric increments: got %d want 1", len(metrics.increments))
	}
	inc := metrics.increments[0]
	if inc.RejectionCode != rejectionCodeChainScope {
		t.Fatalf("rejection_code label: got %q", inc.RejectionCode)
	}
	if inc.WalletType != "eoa" {
		t.Fatalf("wallet_type label: got %q want eoa", inc.WalletType)
	}
	if inc.Binding != exploreBindingDiscovery {
		t.Fatalf("binding label: got %q want discovery", inc.Binding)
	}
	if inc.MissingChainCount != "1" {
		t.Fatalf("missing_chain_count label: got %q want 1", inc.MissingChainCount)
	}
	if len(logger.lines) != 1 {
		t.Fatalf("log lines: got %d want 1", len(logger.lines))
	}
	line := logger.lines[0]
	if !strings.Contains(line, "event="+strconvQuote(exploreNoDeployableEventName)) {
		t.Fatalf("log missing event name: %s", line)
	}
	if !strings.Contains(line, "adr_signal="+strconvQuote(adrSignalNoScanCompatibleProviders)) {
		t.Fatalf("log missing adr_signal: %s", line)
	}
	if strings.Contains(line, "runtime.no_provider_after_user_constraints") {
		t.Fatalf("explore signal must be distinct from persist couche B: %s", line)
	}
	rawAddr := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	if strings.Contains(line, rawAddr) {
		t.Fatalf("log must not contain raw wallet address: %s", line)
	}
	norm, err := persistence.NormalizeWalletTargetAddress(rawAddr)
	if err != nil {
		t.Fatalf("normalize address: %v", err)
	}
	sum := sha256.Sum256([]byte(norm))
	wantHash := hex.EncodeToString(sum[:])[:16]
	if !strings.Contains(line, "wallet_address_hash="+strconvQuote(wantHash)) {
		t.Fatalf("log missing wallet hash %q: %s", wantHash, line)
	}
	if strings.Contains(line, "scan_id=") && strings.Contains(line, strconvQuote("scan-123")) {
		// scan_id allowed in logs
	} else {
		t.Fatalf("log missing scan_id: %s", line)
	}
	for _, forbidden := range []string{" tenant_id=", " owner_id=", " wallet_address=\"0x"} {
		if strings.Contains(line, forbidden) {
			t.Fatalf("log must not contain forbidden field %q: %s", forbidden, line)
		}
	}
}

func TestExploreObservability_skipsWhenCandidateSelected(t *testing.T) {
	metrics := &testExploreMetrics{}
	restore := setExploreObservabilityForTest(exploreObservability{metrics: metrics})
	defer restore()

	decision := policy.PolicyDecision{
		RankedCandidates: []policy.RankedPolicy{{PolicyID: "CP001"}},
		RejectedCandidates: []policy.RejectedPolicy{{
			RejectionReasons: []policy.AssessmentFinding{{
				Code: rejectionCodeChainScope, Severity: policy.AssessmentFindingSeverityBlocking,
			}},
		}},
	}
	exploreObs.recordNoDeployableCandidate(nil, nil, decision, nil)
	if len(metrics.increments) != 0 {
		t.Fatalf("expected no metric increment, got %d", len(metrics.increments))
	}
}

func TestExploreObservability_skipsWhenRejectedEmpty(t *testing.T) {
	metrics := &testExploreMetrics{}
	restore := setExploreObservabilityForTest(exploreObservability{metrics: metrics})
	defer restore()

	decision := policy.PolicyDecision{}
	exploreObs.recordNoDeployableCandidate(nil, nil, decision, nil)
	if len(metrics.increments) != 0 {
		t.Fatalf("expected no metric increment, got %d", len(metrics.increments))
	}
}

func TestDecisionExplore_noDeployableCandidateObservabilityIntegration(t *testing.T) {
	// Mainnet-only scan → Nicetry planned → empty scan_compatible + rejected → observability fires.
	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{providerManifestFixturePath()},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}

	metrics := &testExploreMetrics{}
	restore := setExploreObservabilityForTest(exploreObservability{metrics: metrics})
	defer restore()

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"scan_id":          "705c9704-9428-45e0-882d-fae4cb9d2a0b",
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{1},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Decision struct {
			ScanCompatibleProviders []any `json:"scan_compatible_providers"`
			RejectedCandidates      []any `json:"rejected_candidates"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.Decision.ScanCompatibleProviders) != 0 {
		t.Fatalf("expected empty scan_compatible_providers for mainnet planned")
	}
	if len(response.Decision.RejectedCandidates) == 0 {
		t.Fatal("expected rejected_candidates for mainnet planned")
	}
	if len(metrics.increments) != 1 {
		t.Fatalf("want 1 no-deployable metric, got %d", len(metrics.increments))
	}
}

func strconvQuote(s string) string {
	return `"` + s + `"`
}
