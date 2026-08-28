package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func testReadStore(t *testing.T) *ReadStore {
	t.Helper()
	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{providerManifestFixturePath()},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	return store
}

func TestLoadReadStoreAndRoutes(t *testing.T) {
	store := testReadStore(t)

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: cpmroutes.CryptoPolicies},
		{method: http.MethodGet, path: cpmroutes.CryptoPolicies + "/cpm_pq_account_validation_v1"},
		{method: http.MethodGet, path: cpmroutes.Providers},
		{method: http.MethodGet, path: cpmroutes.Providers + "/nicetry"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status: got %d want %d body=%s", tc.method, tc.path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	for _, legacy := range []string{
		"/api/cpm/v1/policies/catalog",
		"/api/cpm/v1/policies/templates",
		"/api/cpm/v1/policies/instances",
	} {
		req := httptest.NewRequest(http.MethodGet, legacy, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status: got %d want %d (legacy route must be absent)", legacy, rec.Code, http.StatusNotFound)
		}
	}
}

func TestLoadReadStore_CatalogueIntentionOnly(t *testing.T) {
	store := testReadStore(t)
	if len(store.cryptoPolicies) != 1 {
		t.Fatalf("cryptoPolicies: got %d want 1", len(store.cryptoPolicies))
	}
	cp := store.cryptoPolicies[0]
	if cp.ID != "cpm_pq_account_validation_v1" {
		t.Fatalf("crypto policy id: got %q", cp.ID)
	}
	if len(cp.AllowedProviders) != 1 || cp.AllowedProviders[0] != "nicetry" {
		t.Fatalf("allowed_providers: %#v", cp.AllowedProviders)
	}
	items := store.providers.List()
	if len(items) != 1 || items[0].ProviderID != "nicetry" {
		t.Fatalf("providers: %#v", items)
	}
}

func TestDecisionExplore_v02_sepoliaScanCompatibleProviders(t *testing.T) {
	store := testReadStore(t)

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{11155111},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Decision struct {
			RequestSummary struct {
				CryptoPolicyID string `json:"crypto_policy_id"`
			} `json:"request_summary"`
			ScanCompatibleProviders []struct {
				CandidateID              string `json:"candidate_id"`
				SuggestedUserConstraints *struct {
					AllowNewWallet bool `json:"allow_new_wallet"`
				} `json:"suggested_user_constraints"`
				SolutionProfileRef struct {
					ProviderID string `json:"provider_id"`
				} `json:"solution_profile_ref"`
			} `json:"scan_compatible_providers"`
			RejectedCandidates []any    `json:"rejected_candidates"`
			Warnings           []string `json:"warnings"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Decision.RequestSummary.CryptoPolicyID != "cpm_pq_account_validation_v1" {
		t.Fatalf("crypto_policy_id: got %q", response.Decision.RequestSummary.CryptoPolicyID)
	}
	if len(response.Decision.ScanCompatibleProviders) != 1 {
		t.Fatalf("P9b: want 1 scan_compatible_providers, got %d body=%s", len(response.Decision.ScanCompatibleProviders), rec.Body.String())
	}
	got := response.Decision.ScanCompatibleProviders[0]
	if got.SolutionProfileRef.ProviderID != "nicetry" {
		t.Fatalf("provider: %+v", got.SolutionProfileRef)
	}
	if got.SuggestedUserConstraints == nil || !got.SuggestedUserConstraints.AllowNewWallet {
		t.Fatalf("suggested_user_constraints: %+v", got.SuggestedUserConstraints)
	}
	for _, w := range response.Decision.Warnings {
		if strings.Contains(w, "degraded") {
			t.Fatalf("degraded warning must be gone after P9b: %q", w)
		}
	}
}

func TestDecisionExplore_legacySelectionRequestRejected(t *testing.T) {
	store := testReadStore(t)
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{11155111},
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
		"selection_request": map[string]any{
			"target_posture": string(vocabulary.PQPostureHybrid),
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestDecisionExplore_optionA_policy_context(t *testing.T) {
	store := testReadStore(t)

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"scan_id":          "705c9704-9428-45e0-882d-fae4cb9d2a0b",
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"scan_id":            "705c9704-9428-45e0-882d-fae4cb9d2a0b",
			"wallet_address":     "0x0802b015613ef6701192811e595e085a9c560caf",
			"wallet_type":        "EOA",
			"chain_ids":          []int64{11155111},
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-05-11T10:27:10.187512Z",
			"status":             "completed",
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
		t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestDecisionExplore_discoveryV1WalletScanDetailEnvelope(t *testing.T) {
	store := testReadStore(t)

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"scan_id":          "705c9704-9428-45e0-882d-fae4cb9d2a0b",
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"scan_id": "705c9704-9428-45e0-882d-fae4cb9d2a0b",
			"status":  "completed",
			"result": map[string]any{
				"target_address":     "0x0802b015613ef6701192811e595e085a9c560caf",
				"chain_ids":          []int64{1},
				"wallet_type":        "eoa",
				"current_pq_posture": "hybrid",
				"algorithm":          "ECDSA-secp256k1",
				"observations":       []any{},
			},
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
}

func TestDecisionExplore_targetAddressFlatPolicyContext(t *testing.T) {
	store := testReadStore(t)

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"policy_context": map[string]any{
			"scan_id":            "705c9704-9428-45e0-882d-fae4cb9d2a0b",
			"target_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "smart_account",
			"chain_ids":          []int64{1},
			"current_pq_posture": "pq_ready",
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
}

func TestDecisionExplore_unknownCryptoPolicyID(t *testing.T) {
	store := testReadStore(t)
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}
	body := map[string]any{
		"crypto_policy_id": "does_not_exist",
		"policy_context": map[string]any{
			"wallet_type":        "eoa",
			"chain_ids":          []int64{1},
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func fixturePath(name string) string {
	return filepath.Join("..", "domain", "policy", "testdata", name)
}

func providerManifestFixturePath() string {
	return filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json")
}
