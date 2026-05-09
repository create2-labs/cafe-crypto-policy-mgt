package app

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

func TestAuthErrorPayloadContract(t *testing.T) {
	introspectAccepted := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspectAccepted.Close()
	introspectUnavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer introspectUnavailable.Close()

	scanForbidden := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/scan-deny/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer scanForbidden.Close()

	cases := []struct {
		name       string
		method     string
		target     string
		body       string
		authHeader string
		cfg        authConfig
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing token",
			method:     http.MethodGet,
			target:     "/api/v1/policies/catalog",
			cfg:        authConfig{Required: true, SessionValidationURL: introspectUnavailable.URL},
			wantStatus: http.StatusUnauthorized,
			wantCode:   authCodeUnauthenticated,
		},
		{
			name:       "malformed token",
			method:     http.MethodGet,
			target:     "/api/v1/policies/catalog",
			authHeader: "Bearer malformed",
			cfg:        authConfig{Required: true, SessionValidationURL: introspectUnavailable.URL},
			wantStatus: http.StatusUnauthorized,
			wantCode:   authCodeUnauthenticated,
		},
		{
			name:       "validation unavailable",
			method:     http.MethodGet,
			target:     "/api/v1/policies/catalog",
			authHeader: "Bearer " + mustTokenEnvelope(t, "user-1"),
			cfg:        authConfig{Required: true, SessionValidationURL: introspectUnavailable.URL},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   authCodeValidationUnavailable,
		},
		{
			name:       "scan id malformed",
			method:     http.MethodPost,
			target:     "/api/v1/policies/decisions/explore",
			body:       `{"scan_id":42}`,
			authHeader: "Bearer " + mustTokenEnvelope(t, "user-1"),
			cfg: authConfig{
				Required:             true,
				SessionValidationURL: introspectAccepted.URL,
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   authCodeScanIDMalformed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := mustAuthHandler(t, tc.cfg)
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d body=%s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			assertAuthErrorPayload(t, rec, tc.wantCode)
		})
	}

	t.Run("scan forbidden", func(t *testing.T) {
		introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
		defer introspect.Close()
		h := mustAuthHandler(t, authConfig{
			Required:             true,
			SessionValidationURL: introspect.URL,
			ScanAuthorizationURL: scanForbidden.URL,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(`{"scan_id":"scan-deny"}`))
		req.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertAuthErrorPayload(t, rec, authCodeScanForbidden)
	})
}

func TestAuthRequestIDHeaderAndPayload(t *testing.T) {
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusUnauthorized})
	defer introspect.Close()
	h := mustAuthHandler(t, authConfig{Required: true, SessionValidationURL: introspect.URL})

	t.Run("propagates incoming request id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
		req.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
		req.Header.Set("X-Request-Id", "rid-from-client")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("X-Request-Id") != "rid-from-client" {
			t.Fatalf("expected response request id to match input")
		}
		assertAuthErrorPayloadWithRequestID(t, rec, authCodeUnauthenticated, "rid-from-client")
	})

	t.Run("generates missing request id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
		req.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var payload map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
		rid, _ := payload["request_id"].(string)
		if strings.TrimSpace(rid) == "" {
			t.Fatalf("expected generated request_id")
		}
		if rec.Header().Get("X-Request-Id") != rid {
			t.Fatalf("expected response header request id to match payload")
		}
	})

	t.Run("sanitizes invalid incoming request id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
		req.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
		req.Header.Set("X-Request-Id", "bad\nid")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var payload map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
		rid, _ := payload["request_id"].(string)
		if strings.TrimSpace(rid) == "" {
			t.Fatalf("expected generated sanitized request id")
		}
		if rid == "bad\nid" {
			t.Fatalf("expected incoming invalid request id to be replaced")
		}
		if rec.Header().Get("X-Request-Id") != rid {
			t.Fatalf("expected response header request id to match payload")
		}
	})
}

func TestAuthLogsDoNotLeakSensitiveData(t *testing.T) {
	var logs bytes.Buffer
	obs := newAuthObservability()
	obs.logger = log.New(&logs, "", 0)
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h := mustAuthHandler(t, authConfig{
		Required:             true,
		SessionValidationURL: introspect.URL,
		Observability:        obs,
	})

	rawToken := mustTokenEnvelope(t, "user@example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken+"raw-fragment")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
	output := logs.String()
	if strings.Contains(output, "Authorization") || strings.Contains(output, rawToken) || strings.Contains(output, "@example.com") {
		t.Fatalf("sensitive data leaked in logs: %s", output)
	}
}

func TestAuthMetricsCounters(t *testing.T) {
	counter := newAuthDecisionCounter()
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/deny/"):
			w.WriteHeader(http.StatusForbidden)
		case strings.Contains(r.URL.Path, "/slow/"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"allowed":true}`))
		}
	}))
	defer scanAuthz.Close()

	obs := newAuthObservability()
	obs.metrics = counter
	h := mustAuthHandler(t, authConfig{
		Required:             true,
		SessionValidationURL: introspect.URL,
		ScanAuthorizationURL: scanAuthz.URL,
		Observability:        obs,
	})

	successReq := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
	successReq.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
	h.ServeHTTP(httptest.NewRecorder(), successReq)

	missingTokenReq := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
	h.ServeHTTP(httptest.NewRecorder(), missingTokenReq)

	denyReq := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(`{"scan_id":"deny"}`))
	denyReq.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
	h.ServeHTTP(httptest.NewRecorder(), denyReq)

	unavailableReq := httptest.NewRequest(http.MethodPost, "/api/v1/policies/decisions/explore", strings.NewReader(`{"scan_id":"slow"}`))
	unavailableReq.Header.Set("Authorization", "Bearer "+mustTokenEnvelope(t, "user-1"))
	h.ServeHTTP(httptest.NewRecorder(), unavailableReq)

	if counter.Count(authCategoryAuthn, "allowed", authCodeOK, authz.RouteClassAuthenticated) == 0 {
		t.Fatalf("expected authn allowed metric increment")
	}
	if counter.Count(authCategoryAuthn, "denied", authCodeUnauthenticated, authz.RouteClassAuthenticated) == 0 {
		t.Fatalf("expected authn denied metric increment")
	}
	if counter.Count(authCategoryScanAuth, "denied", authCodeScanForbidden, authz.RouteClassAuthenticated) == 0 {
		t.Fatalf("expected scan denied metric increment")
	}
	if counter.Count(authCategoryScanAuth, "unavailable", authCodeScanUnavailable, authz.RouteClassAuthenticated) == 0 {
		t.Fatalf("expected scan unavailable metric increment")
	}
}

func mustAuthHandler(t *testing.T, cfg authConfig) http.Handler {
	t.Helper()
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		CatalogPath: filepath.Join("..", "domain", "policy", "testdata", "policy_graph_catalog_valid.json"),
		TemplatePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_template_valid.json"),
		},
		InstancePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_instance_valid.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	cfg.ClockSkewSec = 30
	h, err := handler("cafe-cpm", store, cfg)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return h
}

func mustTokenEnvelope(t *testing.T, userID string) string {
	t.Helper()
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": userID,
		"email":   userID + "@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	return token
}

func assertAuthErrorPayload(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode auth error payload: %v body=%s", err, rec.Body.String())
	}
	if payload["code"] != wantCode {
		t.Fatalf("expected code %q got %v", wantCode, payload["code"])
	}
	if _, ok := payload["message"].(string); !ok {
		t.Fatalf("expected message field")
	}
	if _, ok := payload["details"].(map[string]any); !ok {
		t.Fatalf("expected details object")
	}
	if rid, _ := payload["request_id"].(string); strings.TrimSpace(rid) == "" {
		t.Fatalf("expected non-empty request_id")
	}
}

func assertAuthErrorPayloadWithRequestID(t *testing.T, rec *httptest.ResponseRecorder, wantCode string, requestID string) {
	t.Helper()
	assertAuthErrorPayload(t, rec, wantCode)
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if payload["request_id"] != requestID {
		t.Fatalf("expected request_id %q got %v", requestID, payload["request_id"])
	}
}
