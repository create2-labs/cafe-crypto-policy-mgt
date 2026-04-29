package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

	h, err := handler("cafe-cpm", store)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
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

	h, err := handler("cafe-cpm", store)
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
