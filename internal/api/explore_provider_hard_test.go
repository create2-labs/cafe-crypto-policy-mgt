package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestDecisionExplore_providerHard_sepoliaRankedMainnetRejected(t *testing.T) {
	store, err := LoadReadStore(ReadStoreOptions{
		TemplatePaths: []string{
			fixturePath("crypto_policy_template_pq_account_validation_v1.json"),
		},
		InstancePaths: []string{
			fixturePath("crypto_policy_instance_pq_account_validation_v1.json"),
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
					CompatibilityFindings []struct {
						Code string `json:"code"`
					} `json:"compatibility_findings"`
				} `json:"rejected_candidates"`
			} `json:"decision"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, c := range response.Decision.RejectedCandidates {
			for _, r := range c.CompatibilityFindings {
				codes = append(codes, r.Code)
			}
		}
		return len(response.Decision.RankedCandidates), len(response.Decision.RejectedCandidates), codes
	}

	ranked, rejected, _ := explore(11155111)
	if ranked != 1 || rejected != 0 {
		t.Fatalf("sepolia: ranked=%d rejected=%d", ranked, rejected)
	}

	// Structured explore payload (CPM-P4b): posture + solution_profile_ref, no graph arrays.
	body := map[string]any{
		"policy_context": map[string]any{
			"wallet_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"wallet_type":        "eoa",
			"chain_ids":          []int64{11155111},
			"current_algorithm":  "secp256k1_ecrecover",
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-08-03T12:00:00Z",
		},
		"selection_request": map[string]any{
			"target_posture":              string(vocabulary.PQPostureHybrid),
			"target_chain_ids":            []int64{11155111},
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
	var structured struct {
		Decision struct {
			RankedCandidates []map[string]any `json:"ranked_candidates"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &structured); err != nil {
		t.Fatalf("decode structured: %v", err)
	}
	if len(structured.Decision.RankedCandidates) != 1 {
		t.Fatalf("structured ranked len: %d", len(structured.Decision.RankedCandidates))
	}
	c := structured.Decision.RankedCandidates[0]
	for _, absent := range []string{
		"nodeInstances", "graphEdges", "edgeIds", "node_path",
		"target_posture_alignment", "maturity_score", "chain_coverage_score",
	} {
		if _, ok := c[absent]; ok {
			t.Fatalf("ranked candidate must not expose %q", absent)
		}
	}
	if c["candidate_id"] != "cpx_pq_account_validation_v1" {
		t.Fatalf("candidate_id: %#v", c["candidate_id"])
	}
	if c["template_id"] != "tpl_pq_account_validation_v1" {
		t.Fatalf("template_id: %#v", c["template_id"])
	}
	if c["required_posture"] != "hybrid" || c["resulting_posture"] != "hybrid" {
		t.Fatalf("postures: required=%v resulting=%v", c["required_posture"], c["resulting_posture"])
	}
	if findings, ok := c["compatibility_findings"].([]any); !ok || len(findings) != 0 {
		t.Fatalf("compatibility_findings: %#v", c["compatibility_findings"])
	}
	if c["maturity"] != "research" || c["claim_status"] != "declared" {
		t.Fatalf("maturity/claim: %#v %#v", c["maturity"], c["claim_status"])
	}
	ref, _ := c["solution_profile_ref"].(map[string]any)
	if ref["provider_id"] != "nicetry" ||
		ref["solution_profile_id"] != "nicetry.fors_c.erc4337.v0_1" ||
		ref["manifest_version"] != "2026-08" {
		t.Fatalf("solution_profile_ref: %#v", c["solution_profile_ref"])
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
