package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func testReadStore(t *testing.T) *ReadStore {
	t.Helper()
	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			providerManifestFixturePath(),
			providerManifestNicetry2FixturePath(),
		},
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
	if len(cp.AllowedProviders) != 2 || cp.AllowedProviders[0] != "nicetry" || cp.AllowedProviders[1] != "nicetry2" {
		t.Fatalf("allowed_providers: %#v", cp.AllowedProviders)
	}
	items := store.providers.List()
	if len(items) != 2 {
		t.Fatalf("providers: %#v", items)
	}
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ProviderID] = true
	}
	if !ids["nicetry"] || !ids["nicetry2"] {
		t.Fatalf("providers missing nicetry/nicetry2: %#v", items)
	}
}

func TestCryptoPolicies_CompatibleNetworks_ListAndGet(t *testing.T) {
	store := testReadStore(t)
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies, nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Items []struct {
			ID                 string `json:"id"`
			CompatibleNetworks []struct {
				ChainID int64  `json:"chain_id"`
				Network string `json:"network"`
				Status  string `json:"status"`
			} `json:"compatible_networks"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("list items: got %d", len(listResp.Items))
	}
	assertCompatibleNetworksMultichain(t, listResp.Items[0].CompatibleNetworks)

	getReq := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies+"/cpm_pq_account_validation_v1", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status: got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var getResp struct {
		ID                 string `json:"id"`
		CompatibleNetworks []struct {
			ChainID int64  `json:"chain_id"`
			Network string `json:"network"`
			Status  string `json:"status"`
		} `json:"compatible_networks"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if getResp.ID != "cpm_pq_account_validation_v1" {
		t.Fatalf("get id: %q", getResp.ID)
	}
	assertCompatibleNetworksMultichain(t, getResp.CompatibleNetworks)
}

func TestCryptoPolicies_AllowedProviderSummaries_ListAndGet(t *testing.T) {
	store := testReadStore(t)
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies, nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Items []struct {
			ID                       string                      `json:"id"`
			AllowedProviderSummaries []allowedProviderSummaryDTO `json:"allowed_provider_summaries"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("list items: got %d", len(listResp.Items))
	}
	assertAllowedProviderSummariesNicetryPair(t, listResp.Items[0].AllowedProviderSummaries)

	getReq := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies+"/cpm_pq_account_validation_v1", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status: got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var getResp struct {
		ID                       string                      `json:"id"`
		AllowedProviderSummaries []allowedProviderSummaryDTO `json:"allowed_provider_summaries"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if getResp.ID != "cpm_pq_account_validation_v1" {
		t.Fatalf("get id: %q", getResp.ID)
	}
	assertAllowedProviderSummariesNicetryPair(t, getResp.AllowedProviderSummaries)
}

type allowedProviderSummaryDTO struct {
	ProviderID string `json:"provider_id"`
	Signature  struct {
		Scheme string `json:"scheme"`
		Family string `json:"family"`
	} `json:"signature"`
	Networks []struct {
		ChainID int64  `json:"chain_id"`
		Network string `json:"network"`
		Status  string `json:"status"`
	} `json:"networks"`
}

func assertAllowedProviderSummariesNicetryPair(t *testing.T, rows []allowedProviderSummaryDTO) {
	t.Helper()
	if len(rows) != 2 {
		t.Fatalf("want nicetry + nicetry2, got %+v", rows)
	}
	if rows[0].ProviderID != "nicetry" || rows[0].Signature.Scheme != "FORS+C" {
		t.Fatalf("nicetry row: %+v", rows[0])
	}
	if rows[1].ProviderID != "nicetry2" || rows[1].Signature.Scheme != "MLDSA" {
		t.Fatalf("nicetry2 row: %+v", rows[1])
	}
	for _, row := range rows {
		if row.Signature.Family == "" {
			t.Fatalf("%s: empty family", row.ProviderID)
		}
		if len(row.Networks) == 0 {
			t.Fatalf("%s: empty networks", row.ProviderID)
		}
		for _, n := range row.Networks {
			if n.Status == "planned" {
				t.Fatalf("%s: planned present: %+v", row.ProviderID, n)
			}
		}
	}
}

func TestCryptoPolicies_CompatibleNetworks_SepoliaOnlyHistorical(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "nicetry_sepolia_only.json")
	const sepoliaOnly = `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "nicetry",
  "provider_name": "NiceTry",
  "provider_version": "2026-08",
  "provider_maturity": "research",
  "solution_profiles": [{
    "solution_profile_id": "nicetry.fors_c.erc4337.v0_1",
    "display_name": "NiceTry FORS+C ERC-4337 Smart Account",
    "maturity": "research",
    "claim_status": "declared",
    "resulting_posture": "hybrid",
    "input_requirements": { "wallet_types": ["EOA"], "requires_wallet_control_proof": true },
    "signature": { "scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop" },
    "account_model": {
      "standard": "ERC-4337",
      "execution_model": "erc4337_bundler",
      "requires_bundler": true,
      "requires_entrypoint": true,
      "entrypoint_versions": ["0.7"]
    },
    "constraints": {
      "requires_new_account": true,
      "address_continuity_supported": false,
      "requires_local_signer_state": true
    },
    "chain_support": [
      { "chain_id": 11155111, "network": "sepolia", "status": "testnet_supported", "capabilities": ["deploy"] },
      { "chain_id": 1, "network": "ethereum-mainnet", "status": "planned", "capabilities": [] }
    ]
  }]
}`
	if err := os.WriteFile(manifestPath, []byte(sepoliaOnly), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths:     []string{fixturePath("crypto_policy_pq_account_validation_v1.json")},
		ProviderManifestPaths: []string{manifestPath},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies+"/cpm_pq_account_validation_v1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		CompatibleNetworks []struct {
			ChainID int64  `json:"chain_id"`
			Status  string `json:"status"`
		} `json:"compatible_networks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.CompatibleNetworks) != 1 || resp.CompatibleNetworks[0].ChainID != 11155111 {
		t.Fatalf("want Sepolia-only, got %+v", resp.CompatibleNetworks)
	}
	if resp.CompatibleNetworks[0].Status == "planned" {
		t.Fatal("planned must not appear")
	}
}

func assertCompatibleNetworksMultichain(t *testing.T, networks []struct {
	ChainID int64  `json:"chain_id"`
	Network string `json:"network"`
	Status  string `json:"status"`
}) {
	t.Helper()
	if len(networks) < 2 {
		t.Fatalf("want multichain networks, got %+v", networks)
	}
	seen := map[int64]struct{}{}
	for _, n := range networks {
		if n.Status == "planned" {
			t.Fatalf("planned present: %+v", n)
		}
		if n.ChainID <= 0 || n.Network == "" {
			t.Fatalf("incomplete: %+v", n)
		}
		if _, ok := seen[n.ChainID]; ok {
			t.Fatalf("duplicate chain_id %d", n.ChainID)
		}
		seen[n.ChainID] = struct{}{}
	}
	if _, ok := seen[11155111]; !ok {
		t.Fatalf("missing sepolia: %+v", networks)
	}
}

func TestCryptoPolicies_omitsPostureOrphanFromProductMatching(t *testing.T) {
	dir := t.TempDir()
	orphanPath := filepath.Join(dir, "cp_orphan.json")
	plannedCPPath := filepath.Join(dir, "cp_planned_only.json")
	plannedManifestPath := filepath.Join(dir, "planned_only.json")
	const orphanCP = `{
  "id": "cp_orphan",
  "name": "Orphan",
  "version": "v0.1",
  "required_posture": "full_pq",
  "allowed_providers": ["nicetry"]
}`
	const plannedCP = `{
  "id": "cp_planned_only",
  "name": "Planned only",
  "version": "v0.1",
  "required_posture": "hybrid",
  "allowed_providers": ["plannedonly"]
}`
	const plannedManifest = `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "plannedonly",
  "provider_name": "Planned Only",
  "provider_version": "2026-08",
  "provider_maturity": "research",
  "solution_profiles": [{
    "solution_profile_id": "plannedonly.profile.v0_1",
    "display_name": "Planned only",
    "maturity": "research",
    "claim_status": "declared",
    "resulting_posture": "hybrid",
    "input_requirements": { "wallet_types": ["EOA"], "requires_wallet_control_proof": true },
    "signature": { "scheme": "FORS+C", "family": "hash_based", "key_rotation_model": "per_userop" },
    "account_model": {
      "standard": "ERC-4337",
      "execution_model": "erc4337_bundler",
      "requires_bundler": true,
      "requires_entrypoint": true,
      "entrypoint_versions": ["0.7"]
    },
    "constraints": {
      "requires_new_account": true,
      "address_continuity_supported": false,
      "requires_local_signer_state": true
    },
    "chain_support": [
      { "chain_id": 1, "network": "ethereum-mainnet", "status": "planned", "capabilities": [] }
    ]
  }]
}`
	for path, body := range map[string]string{
		orphanPath:          orphanCP,
		plannedCPPath:       plannedCP,
		plannedManifestPath: plannedManifest,
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	store, err := LoadReadStore(ReadStoreOptions{
		CryptoPolicyPaths: []string{
			fixturePath("crypto_policy_pq_account_validation_v1.json"),
			orphanPath,
			plannedCPPath,
		},
		ProviderManifestPaths: []string{
			providerManifestFixturePath(),
			plannedManifestPath,
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	if len(store.cryptoPolicies) != 3 {
		t.Fatalf("orphan must stay loaded, got %d policies", len(store.cryptoPolicies))
	}
	logger := &orphanageCaptureLogger{}
	if n := policy.CheckPostureOrphanage(store.cryptoPolicies, store.providers, logger); n != 1 {
		t.Fatalf("want 1 admin orphan signal, got %d logs=%v", n, logger.lines)
	}
	if len(logger.lines) != 1 || !strings.Contains(logger.lines[0], "WARN catalogue: posture orphanage") || !strings.Contains(logger.lines[0], "cp_orphan") {
		t.Fatalf("admin signal: %#v", logger.lines)
	}

	mux := http.NewServeMux()
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies, nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	ids := map[string]bool{}
	for _, item := range listResp.Items {
		ids[item.ID] = true
	}
	if ids["cp_orphan"] {
		t.Fatalf("orphan listed: %+v", listResp.Items)
	}
	if !ids["cpm_pq_account_validation_v1"] || !ids["cp_planned_only"] {
		t.Fatalf("usable and planned-only must stay listed: %+v", listResp.Items)
	}
	if len(listResp.Items) != 2 {
		t.Fatalf("list items: got %d", len(listResp.Items))
	}

	getOrphan := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies+"/cp_orphan", nil)
	getOrphanRec := httptest.NewRecorder()
	mux.ServeHTTP(getOrphanRec, getOrphan)
	if getOrphanRec.Code != http.StatusNotFound {
		t.Fatalf("get orphan status: got %d body=%s", getOrphanRec.Code, getOrphanRec.Body.String())
	}
	if !strings.Contains(getOrphanRec.Body.String(), "crypto policy is not offered") || strings.Contains(getOrphanRec.Body.String(), "runtime.no_scan_compatible") {
		t.Fatalf("get orphan body: %s", getOrphanRec.Body.String())
	}

	getPlanned := httptest.NewRequest(http.MethodGet, cpmroutes.CryptoPolicies+"/cp_planned_only", nil)
	getPlannedRec := httptest.NewRecorder()
	mux.ServeHTTP(getPlannedRec, getPlanned)
	if getPlannedRec.Code != http.StatusOK {
		t.Fatalf("planned-only get: got %d body=%s", getPlannedRec.Code, getPlannedRec.Body.String())
	}

	metrics := &testExploreMetrics{}
	restore := setExploreObservabilityForTest(exploreObservability{metrics: metrics})
	defer restore()
	exploreBody := map[string]any{
		"crypto_policy_id": "cp_orphan",
		"policy_context": map[string]any{
			"wallet_type":        "eoa",
			"chain_ids":          []int64{1},
			"current_pq_posture": "classical_only",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
	}
	raw, _ := json.Marshal(exploreBody)
	exploreReq := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	exploreRec := httptest.NewRecorder()
	mux.ServeHTTP(exploreRec, exploreReq)
	if exploreRec.Code != http.StatusBadRequest {
		t.Fatalf("explore orphan status: got %d body=%s", exploreRec.Code, exploreRec.Body.String())
	}
	if !strings.Contains(exploreRec.Body.String(), "crypto policy is not offered") ||
		strings.Contains(exploreRec.Body.String(), "runtime.no_scan_compatible") ||
		strings.Contains(exploreRec.Body.String(), "scan_compatible_providers") {
		t.Fatalf("explore orphan body: %s", exploreRec.Body.String())
	}
	if len(metrics.increments) != 0 {
		t.Fatalf("explore must not emit no_deployable for an orphan: %+v", metrics.increments)
	}

	assistBody := `{
		"crypto_policy_id": "cp_orphan",
		"solution_profile_ref": {
			"provider_id": "nicetry",
			"solution_profile_id": "nicetry.fors_c.erc4337.v0_1",
			"manifest_version": "2026-08"
		}
	}`
	assistReq := httptest.NewRequest(http.MethodPost, cpmroutes.AcceptedProviderSnapshots, strings.NewReader(assistBody))
	assistRec := httptest.NewRecorder()
	mux.ServeHTTP(assistRec, assistReq)
	if assistRec.Code != http.StatusBadRequest {
		t.Fatalf("assist orphan status: got %d body=%s", assistRec.Code, assistRec.Body.String())
	}
	if !strings.Contains(assistRec.Body.String(), "crypto policy is not offered") ||
		strings.Contains(assistRec.Body.String(), "accepted_provider_snapshot") ||
		strings.Contains(assistRec.Body.String(), "runtime.no_scan_compatible") {
		t.Fatalf("assist orphan body: %s", assistRec.Body.String())
	}
}

type orphanageCaptureLogger struct {
	lines []string
}

func (c *orphanageCaptureLogger) Printf(format string, v ...any) {
	c.lines = append(c.lines, fmt.Sprintf(format, v...))
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
				Composition *struct {
					AccountModel struct {
						Standard        string   `json:"standard"`
						RequiresBundler bool     `json:"requires_bundler"`
						EntrypointVers  []string `json:"entrypoint_versions"`
					} `json:"account_model"`
					Signature struct {
						Scheme           string `json:"scheme"`
						KeyRotationModel string `json:"key_rotation_model"`
					} `json:"signature"`
					Constraints struct {
						RequiresNewAccount       bool `json:"requires_new_account"`
						RequiresLocalSignerState bool `json:"requires_local_signer_state"`
					} `json:"constraints"`
				} `json:"composition"`
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
	if len(response.Decision.ScanCompatibleProviders) != 2 {
		t.Fatalf("P9b: want 2 scan_compatible_providers, got %d body=%s", len(response.Decision.ScanCompatibleProviders), rec.Body.String())
	}
	byProvider := map[string]struct {
		CandidateID              string `json:"candidate_id"`
		SuggestedUserConstraints *struct {
			AllowNewWallet bool `json:"allow_new_wallet"`
		} `json:"suggested_user_constraints"`
		SolutionProfileRef struct {
			ProviderID string `json:"provider_id"`
		} `json:"solution_profile_ref"`
		Composition *struct {
			AccountModel struct {
				Standard        string   `json:"standard"`
				RequiresBundler bool     `json:"requires_bundler"`
				EntrypointVers  []string `json:"entrypoint_versions"`
			} `json:"account_model"`
			Signature struct {
				Scheme           string `json:"scheme"`
				KeyRotationModel string `json:"key_rotation_model"`
			} `json:"signature"`
			Constraints struct {
				RequiresNewAccount       bool `json:"requires_new_account"`
				RequiresLocalSignerState bool `json:"requires_local_signer_state"`
			} `json:"constraints"`
		} `json:"composition"`
	}{}
	for _, c := range response.Decision.ScanCompatibleProviders {
		byProvider[c.SolutionProfileRef.ProviderID] = c
	}
	if _, ok := byProvider["nicetry"]; !ok {
		t.Fatal("missing nicetry candidate")
	}
	if _, ok := byProvider["nicetry2"]; !ok {
		t.Fatal("missing nicetry2 candidate")
	}
	got := byProvider["nicetry"]
	if got.SuggestedUserConstraints == nil || !got.SuggestedUserConstraints.AllowNewWallet {
		t.Fatalf("suggested_user_constraints: %+v", got.SuggestedUserConstraints)
	}
	if got.Composition == nil {
		t.Fatal("CFB-P3: composition must be present on scan_compatible_providers")
	}
	if got.Composition.AccountModel.Standard != "ERC-4337" || !got.Composition.AccountModel.RequiresBundler {
		t.Fatalf("composition.account_model: %+v", got.Composition.AccountModel)
	}
	if got.Composition.Signature.Scheme != "FORS+C" || got.Composition.Signature.KeyRotationModel != "per_userop" {
		t.Fatalf("composition.signature: %+v", got.Composition.Signature)
	}
	if !got.Composition.Constraints.RequiresNewAccount || !got.Composition.Constraints.RequiresLocalSignerState {
		t.Fatalf("composition.constraints: %+v", got.Composition.Constraints)
	}
	nicetry2 := byProvider["nicetry2"]
	if nicetry2.Composition == nil || nicetry2.Composition.Signature.Scheme != "MLDSA" {
		t.Fatalf("nicetry2 composition.signature: %+v", nicetry2.Composition)
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

func providerManifestNicetry2FixturePath() string {
	return filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry2_v0_1.json")
}
