package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

const optionAContractScanID = "705c9704-9428-45e0-882d-fae4cb9d2a0b"

func TestOptionAContract_persistBindingDiscoveryWithoutScanIDReturns400(t *testing.T) {
	h := newAuthedTestHandler(t)
	tok := mustToken(t, "user-discovery-persist")
	body := `{"id":"policy-no-scan","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	// RD-P5: unsigned legacy body is rejected (missing wallet_address / signature).
	if res.Code != http.StatusBadRequest && res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 400/403 body=%s", res.Code, res.Body.String())
	}
}

func TestOptionAContract_unsignedLegacyPOSTPoliciesRejected(t *testing.T) {
	h := newAuthedTestHandlerWithScanAuthz(t)
	tok := mustToken(t, "user-discovery-persist-ok")
	body := `{"id":"policy-with-scan","scan_id":"` + optionAContractScanID + `","binding":"discovery","payload":{"mode":"strict"}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("legacy unsigned POST /policies must not succeed after RD-P5, body=%s", res.Body.String())
	}
}

func newAuthedTestHandlerWithScanAuthz(t *testing.T) http.Handler {
	t.Helper()
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		CryptoPolicyPaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_pq_account_validation_v1.json"),
		},
		InstancePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_instance_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		},

	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	t.Cleanup(scanAuthz.Close)
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ScanAuthorizationURL:        scanAuthz.URL,
		ScanAuthorizationTimeoutSec: 2,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return h
}

func TestOptionAContract_assessmentPolicyContextForbiddenReturns400(t *testing.T) {
	h := newAssessmentTestHandler(t, assessmentHarness{})
	tok := mustAssessmentToken(t)
	body := map[string]any{
		"scan_id":           optionAContractScanID,
		"policy_context":    map[string]any{"wallet_address": "0x1"},
		"crypto_policy_id": "cpm_pq_account_validation_v1",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesAssessmentRequest, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", res.Code, res.Body.String())
	}
}
