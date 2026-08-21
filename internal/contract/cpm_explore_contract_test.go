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

func TestCPMExplore_optionAValidReturns200(t *testing.T) {
	store := mustLoadReadStore(t)
	mux := http.NewServeMux()
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	detail := loadJSONFixture(t, "discovery_v1_wallet_scan_detail.json")
	body := map[string]any{
		"scan_id":           optionAScanID,
		"policy_context":    policyContextFromDiscoveryV1Detail(detail),
		"crypto_policy_id": validOptionACryptoPolicyID(),
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestCPMExplore_invalidPolicyContextReturns400(t *testing.T) {
	store := mustLoadReadStore(t)
	mux := http.NewServeMux()
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing policy_context",
			body: map[string]any{"crypto_policy_id": validOptionACryptoPolicyID()},
		},
		{
			name: "invalid pq posture",
			body: map[string]any{
				"policy_context": map[string]any{
					"wallet_type":        "eoa",
					"chain_ids":          []int64{1},
					"current_pq_posture": "not-a-real-posture",
				},
				"crypto_policy_id": validOptionACryptoPolicyID(),
			},
		},
		{
			name: "invalid scanned_at",
			body: map[string]any{
				"policy_context": map[string]any{
					"wallet_type":        "eoa",
					"chain_ids":          []int64{1},
					"current_pq_posture": "hybrid",
					"scanned_at":         "not-rfc3339",
				},
				"crypto_policy_id": validOptionACryptoPolicyID(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, bytes.NewReader(raw))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
