package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadReadStoreAndRoutes(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CatalogPath: fixturePath("policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			fixturePath("crypto_policy_template_valid.json"),
		},
		InstancePaths: []string{
			fixturePath("crypto_policy_instance_valid.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/policies/catalog"},
		{method: http.MethodGet, path: "/api/v1/policies/templates"},
		{method: http.MethodGet, path: "/api/v1/policies/instances"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status: got %d want %d", tc.method, tc.path, rec.Code, http.StatusOK)
		}
	}
}

func TestDecisionExplorePreservesDeployabilityDistinction(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CatalogPath: fixturePath("policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			fixturePath("crypto_policy_template_valid.json"),
		},
		InstancePaths: []string{
			fixturePath("crypto_policy_instance_valid.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int{1, 8453},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
		"selection_request": map[string]any{
			"target_posture":              string(vocabulary.PQPostureHybrid),
			"target_chain_ids":            []int64{1, 8453},
			"require_multichain":          true,
			"allow_new_wallet":            false,
			"address_continuity_required": true,
			"key_rotation_required":       true,
			"recovery_required":           true,
			"minimum_maturity":            1,
			"approval_mode":               "manual",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	assertStatus := func(expected string) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", bytes.NewReader(raw))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var response struct {
			Decision struct {
				RankedCandidates []struct {
					CompatibilityStatus string `json:"compatibility_status"`
				} `json:"ranked_candidates"`
			} `json:"decision"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(response.Decision.RankedCandidates) != 1 {
			t.Fatalf("ranked_candidates length: got %d want 1", len(response.Decision.RankedCandidates))
		}
		if got := response.Decision.RankedCandidates[0].CompatibilityStatus; got != expected {
			t.Fatalf("compatibility_status: got %q want %q", got, expected)
		}
	}

	assertStatus("compatible_and_deployable")

	// Remove concrete chain scope to force known-but-not-deployable classification.
	store.instances[0].Scope.ChainIDs = nil
	assertStatus("compatible_but_not_deployable")
}

func TestDecisionExplore_optionA_policy_context(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CatalogPath: fixturePath("policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			fixturePath("crypto_policy_template_valid.json"),
		},
		InstancePaths: []string{
			fixturePath("crypto_policy_instance_valid.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := map[string]any{
		"scan_id": "705c9704-9428-45e0-882d-fae4cb9d2a0b",
		"policy_context": map[string]any{
			"scan_id":            "705c9704-9428-45e0-882d-fae4cb9d2a0b",
			"wallet_address":     "0x0802b015613ef6701192811e595e085a9c560caf",
			"wallet_type":        "EOA",
			"chain_ids":          []int{1},
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-05-11T10:27:10.187512Z",
			"status":             "completed",
		},
		"selection_request": map[string]any{
			"target_posture":              string(vocabulary.PQPostureHybrid),
			"target_chain_ids":            []int64{1},
			"require_multichain":          false,
			"allow_new_wallet":            false,
			"address_continuity_required": true,
			"key_rotation_required":       true,
			"recovery_required":           true,
			"minimum_maturity":            1,
			"approval_mode":               "manual",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Decision struct {
			SelectedPolicyID string `json:"selected_policy_id"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Decision.SelectedPolicyID == "" {
		t.Fatalf("expected selected_policy_id, got empty body=%s", rec.Body.String())
	}
}

func fixturePath(name string) string {
	return filepath.Join("..", "domain", "policy", "testdata", name)
}
