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
)

func TestExtractScanIDsForAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(`{"scan_id":"scan-123"}`))
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

func TestExtractScanIDsForAuthorizationVariants(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		want    []string
		wantErr bool
	}{
		{name: "top-level scan_id", body: `{"scan_id":"scan-1"}`, want: []string{"scan-1"}},
		{name: "draft scan_id only", body: `{"draft":{"scan_id":"scan-2"}}`, want: []string{"scan-2"}},
		{name: "no scan id", body: `{"draft":{"name":"x"}}`, want: nil},
		{name: "empty string scan_id ignored", body: `{"scan_id":"","id":"draft-1","payload":{}}`, want: nil},
		{name: "mismatch top vs draft scan_id", body: `{"scan_id":"scan-a","draft":{"scan_id":"scan-b"}}`, wantErr: true},
		{name: "malformed scan_id type", body: `{"scan_id":42}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(tc.body))
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
		})
	}
}

func TestAuthorizeScanAccessMappings(t *testing.T) {
	principal := authz.Principal{UserID: "u1", Subject: "u1"}
	var gotUserID atomic.Value
	var gotTenantID atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cfg := authConfig{
		ScanAuthorizationURL:        server.URL,
		ScanAuthorizationTimeoutSec: 1,
	}
	principal.TenantID = "tenant-a"
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "allow", cfg, "rid"); errPayload.Code != "" || status != http.StatusOK {
		t.Fatalf("expected allow, got code=%q status=%d", errPayload.Code, status)
	}
	if gotUserID.Load() != "u1" || gotTenantID.Load() != "tenant-a" {
		t.Fatalf("expected principal-derived headers, got user=%v tenant=%v", gotUserID.Load(), gotTenantID.Load())
	}
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "deny", cfg, "rid"); status != http.StatusForbidden || errPayload.Code == "" {
		t.Fatalf("expected forbidden mapping, got code=%q status=%d", errPayload.Code, status)
	}
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "notfound", cfg, "rid"); status != http.StatusForbidden || errPayload.Code == "" {
		t.Fatalf("expected notfound->forbidden mapping, got code=%q status=%d", errPayload.Code, status)
	}
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "bad200", cfg, "rid"); status != http.StatusForbidden || errPayload.Code == "" {
		t.Fatalf("expected explicit allowed=false to map forbidden, got code=%q status=%d", errPayload.Code, status)
	}
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "unavailable", cfg, "rid"); status != http.StatusServiceUnavailable || errPayload.Code == "" {
		t.Fatalf("expected unavailable mapping, got code=%q status=%d", errPayload.Code, status)
	}
	if errPayload, status, _ := authorizeScanAccess(t.Context(), principal, "timeout", cfg, "rid"); status != http.StatusServiceUnavailable || errPayload.Code == "" {
		t.Fatalf("expected timeout mapping, got code=%q status=%d", errPayload.Code, status)
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(`{"scan_id":"scan-123"}`))
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
