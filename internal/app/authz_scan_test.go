package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestExtractScanIDsForAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"scan-123","policy_context":{"scan_id":"scan-123"}}`))
	scanIDs, authErr, status := extractScanIDsForAuthorization(req)
	if authErr.Code != "" || status != http.StatusOK {
		t.Fatalf("expected no error, got code=%q status=%d", authErr.Code, status)
	}
	if len(scanIDs) != 1 || scanIDs[0] != "scan-123" {
		t.Fatalf("expected [scan-123], got %#v", scanIDs)
	}
	// Body must remain readable by downstream handler.
	var payload map[string]any
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("expected restored body, decode failed: %v", err)
	}
}

type extractScanIDsCase struct {
	name    string
	body    string
	want    []string
	wantErr bool
}

func TestExtractScanIDsForAuthorizationVariants(t *testing.T) {
	cases := []extractScanIDsCase{
		{name: "top-level scan_id", body: `{"scan_id":"scan-1"}`, want: []string{"scan-1"}},
		{name: "no scan id", body: `{"name":"x"}`, want: nil},
		{name: "empty string scan_id ignored", body: `{"scan_id":"","id":"pol-1","payload":{}}`, want: nil},
		{name: "mismatch top vs policy_context scan_id", body: `{"scan_id":"scan-a","policy_context":{"scan_id":"scan-b"}}`, wantErr: true},
		{name: "malformed scan_id type", body: `{"scan_id":42}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExtractScanIDsForAuthorization(t, tc)
		})
	}
}

func assertExtractScanIDsForAuthorization(t *testing.T, tc extractScanIDsCase) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(tc.body))
	scanIDs, authErr, status := extractScanIDsForAuthorization(req)
	if tc.wantErr {
		if authErr.Code == "" || status != http.StatusBadRequest {
			t.Fatalf("expected 400 with error, got code=%q status=%d ids=%#v", authErr.Code, status, scanIDs)
		}
		return
	}
	if authErr.Code != "" || status != http.StatusOK {
		t.Fatalf("expected no error, got code=%q status=%d", authErr.Code, status)
	}
	if len(tc.want) != len(scanIDs) {
		t.Fatalf("expected ids %#v, got %#v", tc.want, scanIDs)
	}
	for i := range tc.want {
		if tc.want[i] != scanIDs[i] {
			t.Fatalf("expected ids %#v, got %#v", tc.want, scanIDs)
		}
	}
}

func TestExtractScanIDsForAuthorizationPoliciesPostUsesScanID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(
		`{"wallet_address":"0xee387b44819eb54d7fff026a18229421738a8a24","chain_id":1,"scan_id":"550e8400-e29b-41d4-a716-446655440000","payload":{}}`,
	))
	scanIDs, authErr, status := extractScanIDsForAuthorization(req)
	if authErr.Code != "" || status != http.StatusOK {
		t.Fatalf("expected no scan auth error, got code=%q status=%d", authErr.Code, status)
	}
	if len(scanIDs) != 1 {
		t.Fatalf("expected scan id from persist body, got %#v", scanIDs)
	}
}

func TestExtractScanIDsForAuthorizationGETPolicies(t *testing.T) {
	scan := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?scan_id="+scan, nil)
	ids, authErr, status := extractScanIDsForAuthorization(req)
	if authErr.Code != "" || status != http.StatusOK {
		t.Fatalf("expected ok, got code=%q status=%d", authErr.Code, status)
	}
	if len(ids) != 1 || ids[0] != strings.ToLower(scan) {
		t.Fatalf("expected normalized scan id, got %#v", ids)
	}

	reqBoth := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id=a&scan_id="+scan, nil)
	_, ae2, st2 := extractScanIDsForAuthorization(reqBoth)
	if ae2.Code == "" || st2 != http.StatusBadRequest {
		t.Fatalf("expected 400 conflict, got code=%q status=%d", ae2.Code, st2)
	}
}

func scanAuthzMappingHandler(gotUserID, gotTenantID *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gotUserID.Store(r.Header.Get("X-User-Id"))
		gotTenantID.Store(r.Header.Get("X-Tenant-Id"))
		switch {
		case strings.Contains(r.URL.Path, "allow"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"allowed":true}`))
		case strings.Contains(r.URL.Path, "deny"):
			w.WriteHeader(http.StatusForbidden)
		case strings.Contains(r.URL.Path, "notfound"):
			w.WriteHeader(http.StatusNotFound)
		case strings.Contains(r.URL.Path, "bad200"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"allowed":false}`))
		case strings.Contains(r.URL.Path, "timeout"):
			time.Sleep(2 * time.Second)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func TestAuthorizeScanAccessMappings(t *testing.T) {
	var gotUserID, gotTenantID atomic.Value
	server := httptest.NewServer(scanAuthzMappingHandler(&gotUserID, &gotTenantID))
	defer server.Close()

	cfg := authConfig{
		ScanAuthorizationURL:        server.URL,
		ScanAuthorizationTimeoutSec: 1,
	}
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "tenant-a"}
	ctx := t.Context()

	cases := []struct {
		name       string
		scanID     string
		wantStatus int
		wantCode   bool // true when errPayload.Code must be non-empty
	}{
		{name: "allow", scanID: "allow", wantStatus: http.StatusOK},
		{name: "deny", scanID: "deny", wantStatus: http.StatusForbidden, wantCode: true},
		{name: "notfound", scanID: "notfound", wantStatus: http.StatusForbidden, wantCode: true},
		{name: "bad200", scanID: "bad200", wantStatus: http.StatusForbidden, wantCode: true},
		{name: "unavailable", scanID: "unavailable", wantStatus: http.StatusServiceUnavailable, wantCode: true},
		{name: "timeout", scanID: "timeout", wantStatus: http.StatusServiceUnavailable, wantCode: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errPayload, status, _ := authorizeScanAccess(ctx, principal, tc.scanID, cfg, "rid")
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d code=%q", status, tc.wantStatus, errPayload.Code)
			}
			if tc.wantCode && errPayload.Code == "" {
				t.Fatalf("expected non-empty error code, got status=%d", status)
			}
			if !tc.wantCode && errPayload.Code != "" {
				t.Fatalf("expected no error code, got %q status=%d", errPayload.Code, status)
			}
		})
	}
	if gotUserID.Load() != "u1" || gotTenantID.Load() != "tenant-a" {
		t.Fatalf("expected principal-derived headers, got user=%v tenant=%v", gotUserID.Load(), gotTenantID.Load())
	}
}

func TestWithAuthenticationAllowsRequestWhenScanAuthzAllowed(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()

	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"scan-123","crypto_policy_id":"cpm_pq_account_validation_v1","policy_context":{"scan_id":"scan-123","wallet_type":"EOA","current_pq_posture":"classical_only","chain_ids":[1],"scanned_at":"2026-01-01T00:00:00Z"}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-Id", "fake-client-header")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if !called {
		t.Fatalf("expected next handler to be called")
	}
}

func TestWithAuthentication_RDP6_W2AllowsLatestCompletedEvenIfNewestRowFailed(t *testing.T) {
	const latestScanID = "705c9704-9428-45e0-882d-fae4cb9d2a0b"
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()
	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/discovery/v1/wallets/scans/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"` + latestScanID + `","status":"completed","result":{"target_address":"0xabc"}}`))
		case r.URL.Path == "/discovery/v1/wallets/scans" && r.URL.Query().Get("latest") == "true":
			// Newest row may be failed/pending; W2 only requires latest completed.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"scan_id":"` + latestScanID + `","status":"completed"}],"total":1,"limit":1,"offset":0}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer discovery.Close()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"`+latestScanID+`","crypto_policy_id":"cpm_pq_account_validation_v1","policy_context":{"scan_id":"`+latestScanID+`","wallet_type":"EOA","current_pq_posture":"classical_only","chain_ids":[1],"scanned_at":"2026-01-01T00:00:00Z"}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("want 200 explore on latest completed (W2), got %d body=%s", res.Code, res.Body.String())
	}
	if !called {
		t.Fatalf("expected next handler to be called")
	}
}

func TestWithAuthentication_RDP6_W2RejectsHistoricalScanID(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()
	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/discovery/v1/wallets/scans/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","status":"completed","result":{"target_address":"0xabc"}}`))
		case r.URL.Path == "/discovery/v1/wallets/scans" && r.URL.Query().Get("latest") == "true":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"scan_id":"11111111-1111-4111-8111-111111111111","status":"completed"}],"total":1,"limit":1,"offset":0}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer discovery.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","policy_context":{"wallet_address":"0xabc","wallet_type":"eoa","chain_ids":[1]}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "SCAN_NOT_LATEST") {
		t.Fatalf("want 422 SCAN_NOT_LATEST, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestWithAuthentication_RDP6_ExploreDiscoveryUnavailable(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()
	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/discovery/v1/wallets/scans/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","status":"completed","result":{"target_address":"0xabc"}}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer discovery.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","policy_context":{"wallet_address":"0xabc","wallet_type":"eoa","chain_ids":[1]}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable || !strings.Contains(res.Body.String(), "DISCOVERY_UNAVAILABLE") {
		t.Fatalf("want 503 DISCOVERY_UNAVAILABLE, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestWithAuthentication_IMM10_ExploreRejectsTLSScanIDAsNotFound(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()
	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/discovery/v1/wallets/scans/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","scan_family":"tls","status":"completed","result":{"target_address":"https://example.com"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer discovery.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","crypto_policy_id":"cpm_pq_account_validation_v1","policy_context":{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","wallet_type":"EOA","current_pq_posture":"classical_only","chain_ids":[1],"scanned_at":"2026-01-01T00:00:00Z"}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), `"error":"not_found"`) {
		t.Fatalf("want 404 not_found, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestWithAuthentication_IMM10_PersistRejectsTLSScanIDAsNotFound(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer introspect.Close()
	authzServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer authzServer.Close()
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/discovery/v1/wallets/scans/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","scan_family":"tls","status":"completed","result":{"target_address":"https://example.com"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer discovery.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ScanAuthorizationURL:        authzServer.URL,
		ScanAuthorizationTimeoutSec: 1,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesDecisionsExplore, strings.NewReader(`{"scan_id":"705c9704-9428-45e0-882d-fae4cb9d2a0b","policy_context":{"wallet_address":"0xabc","wallet_type":"eoa","chain_ids":[1]}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), `"error":"not_found"`) {
		t.Fatalf("want 404 not_found, got %d body=%s", res.Code, res.Body.String())
	}
}
