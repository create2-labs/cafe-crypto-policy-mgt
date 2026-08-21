package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestDecisionExplore_v02_rejectsLegacyAndReturnsScanCompatible(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			providerManifestFixturePath(),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	legacy := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{11155111},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-08-03T12:00:00Z",
		},
		"selection_request": map[string]any{"target_posture": "hybrid"},
	}
	raw, _ := json.Marshal(legacy)
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("legacy status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}

	okBody := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{11155111},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-08-03T12:00:00Z",
		},
	}
	raw, _ = json.Marshal(okBody)
	req = httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", rec.Code, rec.Body.String())
	}
	var wrapped map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapped); err != nil {
		t.Fatalf("decode: %v", err)
	}
	decision, _ := wrapped["decision"].(map[string]any)
	providers, ok := decision["scan_compatible_providers"].([]any)
	if !ok {
		t.Fatalf("missing scan_compatible_providers in %#v", decision)
	}
	if len(providers) != 1 {
		t.Fatalf("want 1 scan-compatible provider, got %d", len(providers))
	}
	if _, ok := decision["ranked_candidates"]; ok {
		t.Fatal("ranked_candidates must not appear on public explore JSON")
	}
}

func TestDecisionExplore_v02_mainnetPlannedRejected(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{providerManifestFixturePath()},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{1},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-08-03T12:00:00Z",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var wrapped struct {
		Decision struct {
			ScanCompatibleProviders []any `json:"scan_compatible_providers"`
			RejectedCandidates      []any `json:"rejected_candidates"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapped); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(wrapped.Decision.ScanCompatibleProviders) != 0 {
		t.Fatalf("mainnet planned must not be scan-compatible")
	}
	if len(wrapped.Decision.RejectedCandidates) == 0 {
		t.Fatal("expected rejected_candidates for mainnet planned")
	}
}
