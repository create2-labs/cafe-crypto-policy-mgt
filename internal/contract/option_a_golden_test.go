package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

// Golden mapping: Discovery v1 WalletScanDetail → explore policy_context (A2 §3.1).
func TestOptionA_DiscoveryV1DetailMapsToGoldenExplorePolicyContext(t *testing.T) {
	detail := loadJSONFixture(t, "discovery_v1_wallet_scan_detail.json")
	golden := loadJSONFixture(t, "option_a_explore_policy_context_golden.json")
	got := policyContextFromDiscoveryV1Detail(detail)
	gotRaw, _ := json.Marshal(got)
	wantRaw, _ := json.Marshal(golden)
	if string(gotRaw) != string(wantRaw) {
		t.Fatalf("policy_context mismatch:\ngot  %s\nwant %s", gotRaw, wantRaw)
	}
}

func TestOptionA_GoldenExplorePolicyContextReturns200(t *testing.T) {
	store := mustLoadReadStore(t)
	mux := http.NewServeMux()
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	policyContext := loadJSONFixture(t, "option_a_explore_policy_context_golden.json")
	body := map[string]any{
		"scan_id":           optionAScanID,
		"policy_context":    policyContext,
		"selection_request": validOptionASelectionRequest(),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("explore status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestOptionA_DiscoveryV1DetailObservationPayload(t *testing.T) {
	pl, err := api.ObservationPayloadFromDiscoveryWalletScanDetail(contractFixture(t, "discovery_v1_wallet_scan_detail.json"))
	if err != nil {
		t.Fatalf("ObservationPayloadFromDiscoveryWalletScanDetail: %v", err)
	}
	if pl.CurrentPQPosture != "hybrid" {
		t.Fatalf("current_pq_posture = %q, want hybrid (pq_ready/hybrid mapping)", pl.CurrentPQPosture)
	}
	if len(pl.ChainIDs) != 1 || pl.ChainIDs[0] != 1 {
		t.Fatalf("chain_ids = %v, want [1]", pl.ChainIDs)
	}
}
