package app

import (
	"encoding/base64"
	"encoding/json"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
)

func TestHandlerHealthz(t *testing.T) {
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

	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.Healthz, nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if got := res.Body.String(); got != "cafe-cpm ok" {
		t.Fatalf("expected body %q, got %q", "cafe-cpm ok", got)
	}
}

func TestHandlerHealthPathNotRegistered(t *testing.T) {
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

	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}
}

func TestHandlerRequiresAuthForBusinessRoutes(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if payload["code"] == nil {
		t.Fatalf("expected error code in response body: %s", res.Body.String())
	}
}

func TestHandlerRejectsMalformedBearerHeader(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Token abc")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestHandlerAcceptsValidBearerToken(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
}

func TestHandlerRejectsExpiredDiscoveryToken(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(-10 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, res.Code, res.Body.String())
	}
}

func TestHandlerRejectsDiscoveryDeniedToken(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusUnauthorized})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, res.Code, res.Body.String())
	}
}

func TestHandlerRejectsMissingEdDSASignature(t *testing.T) {
	assertTokenRejected(t, []string{"ML-DSA-65"}, http.StatusUnauthorized)
}

func TestHandlerRejectsMissingMLDSASignature(t *testing.T) {
	assertTokenRejected(t, []string{"EdDSA"}, http.StatusUnauthorized)
}

func TestHandlerReturns503OnValidationTimeout(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{
		status: http.StatusOK,
		delay:  2 * time.Second,
	})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusServiceUnavailable, res.Code, res.Body.String())
	}
}

func TestHandlerReturns503OnValidation5xx(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusInternalServerError})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusServiceUnavailable, res.Code, res.Body.String())
	}
}

func TestHandlerRejectsValidationSuccessButMissingUserID(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{
		status: http.StatusOK,
		body:   `{"accepted":true,"claims":{"email":"x@example.com","exp":9999999999}}`,
	})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, res.Code, res.Body.String())
	}
}

func TestHandlerPropagatesRequestIDToDiscoveryValidation(t *testing.T) {
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
	var rid atomic.Value
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{
		status:     http.StatusOK,
		captureRID: &rid,
	})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-Id", "rid-123")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	got, _ := rid.Load().(string)
	if got != "rid-123" {
		t.Fatalf("expected request id propagation to discovery, got %q", got)
	}
}

func TestHandlerFailsClosedWhenScanIDPresentButAuthzNotConfigured(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(
		http.MethodPost,
		cpmroutes.PoliciesDecisionsExplore,
		strings.NewReader(`{"scan_id":"scan-123","policy_context":{"scan_id":"scan-123","wallet_type":"EOA","chain_ids":[1],"current_pq_posture":"classical_only","scanned_at":"2026-01-01T00:00:00Z"},"selection_request":{"target_posture":"hybrid","target_chain_ids":[1],"require_multichain":false,"allow_new_wallet":false,"address_continuity_required":true,"key_rotation_required":true,"recovery_required":true,"minimum_maturity":1,"approval_mode":"manual"}}`),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusServiceUnavailable, res.Code, res.Body.String())
	}
}

func TestHandlerContinuesWhenAuthzNotConfiguredAndNoScanID(t *testing.T) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(
		http.MethodPost,
		cpmroutes.PoliciesDecisionsExplore,
		strings.NewReader(`{"policy_context":{"wallet_type":"EOA","chain_ids":[1],"current_pq_posture":"classical_only","scanned_at":"2026-01-01T00:00:00Z"},"selection_request":{"target_posture":"hybrid","target_chain_ids":[1],"require_multichain":false,"allow_new_wallet":false,"address_continuity_required":true,"key_rotation_required":true,"recovery_required":true,"minimum_maturity":1,"approval_mode":"manual"}}`),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
}

type discoveryValidationTestConfig struct {
	status     int
	body       string
	delay      time.Duration
	captureRID *atomic.Value
}

func newDiscoveryValidationServer(t *testing.T, cfg discoveryValidationTestConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.delay > 0 {
			time.Sleep(cfg.delay)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected json content-type, got %q", r.Header.Get("Content-Type"))
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["token"] == "" {
			t.Fatalf("expected non-empty token in introspection payload")
		}
		if cfg.captureRID != nil {
			cfg.captureRID.Store(r.Header.Get("X-Request-Id"))
		}
		w.WriteHeader(cfg.status)
		if cfg.body != "" {
			_, _ = w.Write([]byte(cfg.body))
		}
	}))
}

func assertTokenRejected(t *testing.T, algorithms []string, expectedStatus int) {
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
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, algorithms)
	if err != nil {
		t.Fatalf("make token envelope: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, cpmroutes.PoliciesCatalog, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d body=%s", expectedStatus, res.Code, res.Body.String())
	}
}

func makeDiscoveryHybridToken(claims map[string]any) (string, error) {
	payloadRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadRaw)

	protectedEdRaw, err := json.Marshal(map[string]any{
		"alg": "EdDSA",
		"typ": "JWT",
		"kid": "ed25519-cafe-1",
	})
	if err != nil {
		return "", err
	}
	protectedPqcRaw, err := json.Marshal(map[string]any{
		"alg": "ML-DSA-65",
		"typ": "JWT",
		"kid": "mldsa65-cafe-1",
	})
	if err != nil {
		return "", err
	}
	envelopeRaw, err := json.Marshal(map[string]any{
		"payload": payloadB64,
		"signatures": []map[string]string{
			{
				"protected": base64.RawURLEncoding.EncodeToString(protectedEdRaw),
				"signature": base64.RawURLEncoding.EncodeToString([]byte("ed-signature-placeholder")),
			},
			{
				"protected": base64.RawURLEncoding.EncodeToString(protectedPqcRaw),
				"signature": base64.RawURLEncoding.EncodeToString([]byte("pqc-signature-placeholder")),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(envelopeRaw), nil
}
