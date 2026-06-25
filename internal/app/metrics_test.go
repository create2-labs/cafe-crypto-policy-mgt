package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/metrics"
)

func TestHandlerMetricsExposesExploreCounter(t *testing.T) {
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

	metrics.IncExploreNoDeployableCandidate("incompatible.chain_scope", "eoa", "discovery", "1")

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Metrics, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "cpm_explore_no_deployable_candidate_total") {
		t.Fatalf("metrics body missing explore counter: %s", body)
	}
}
