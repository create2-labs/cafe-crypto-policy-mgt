package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestDecisionExplore_providerHard_sepoliaRankedMainnetRejected(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		CatalogPath: fixturePath("policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			fixturePath("crypto_policy_template_pq_account_validation_v1.json"),
		},
		InstancePaths: []string{
			fixturePath("crypto_policy_instance_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	explore := func(chainID int64) (ranked, rejected int, codes []string) {
		t.Helper()
		body := map[string]any{
			"policy_context": map[string]any{
				"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
				"wallet_type":        "eoa",
				"chain_ids":          []int64{chainID},
				"current_algorithm":  "secp256k1_ecrecover",
				"current_pq_posture": "classical_only",
				"scanned_at":         "2026-08-03T12:00:00Z",
			},
			"selection_request": map[string]any{
				"target_posture":              string(vocabulary.PQPostureHybrid),
				"target_chain_ids":            []int64{chainID},
				"require_multichain":          false,
				"allow_new_wallet":            true,
				"address_continuity_required": false,
				"key_rotation_model":          "per_userop",
				"recovery_required":           true,
				"minimum_maturity":            1,
				"approval_mode":               "manual",
				"allowed_provider_modes":      []string{"third_party", "user_managed"},
				"require_bundler_available":   true,
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
				RankedCandidates   []any `json:"ranked_candidates"`
				RejectedCandidates []struct {
					RejectionReasons []struct {
						Code string `json:"code"`
					} `json:"rejection_reasons"`
				} `json:"rejected_candidates"`
			} `json:"decision"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, c := range response.Decision.RejectedCandidates {
			for _, r := range c.RejectionReasons {
				codes = append(codes, r.Code)
			}
		}
		return len(response.Decision.RankedCandidates), len(response.Decision.RejectedCandidates), codes
	}

	ranked, rejected, _ := explore(11155111)
	if ranked != 1 || rejected != 0 {
		t.Fatalf("sepolia: ranked=%d rejected=%d", ranked, rejected)
	}
	ranked, rejected, codes := explore(1)
	if ranked != 0 || rejected == 0 {
		t.Fatalf("mainnet: ranked=%d rejected=%d", ranked, rejected)
	}
	found := false
	for _, c := range codes {
		if c == provider.FindingCodeChain {
			found = true
		}
	}
	if !found {
		t.Fatalf("want %s in %v", provider.FindingCodeChain, codes)
	}
}
