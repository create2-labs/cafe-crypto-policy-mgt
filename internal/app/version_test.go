package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/version"
)

func TestHandlerVersionReturnsJSON(t *testing.T) {
	t.Setenv("APP_VERSION", "v1.2.3-test")
	t.Cleanup(func() { _ = os.Unsetenv("APP_VERSION") })

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

	h, err := testHandler(store, nil, authConfig{Required: false})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Version, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var body version.Response
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if body.Version != "v1.2.3-test" {
		t.Fatalf("version = %q, want v1.2.3-test", body.Version)
	}
}

func TestVersionRoutePublicWhenAuthRequired(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer introspect.Close()

	t.Setenv("APP_VERSION", "v0.0.0")
	t.Cleanup(func() { _ = os.Unsetenv("APP_VERSION") })

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler, err := withAuthentication(next, authConfig{
		Required:             true,
		SessionValidationURL: introspect.URL,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Version, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected public version route pass-through, code=%d called=%v", rec.Code, called)
	}
}
