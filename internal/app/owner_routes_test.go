package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
)

func TestDraftEndpointsAreOwnerScopedAtAPILevel(t *testing.T) {
	h := newAuthedTestHandler(t)
	tokenA := mustToken(t, "user-a")
	tokenB := mustToken(t, "user-b")

	create := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/drafts", strings.NewReader(`{"id":"draft-1","payload":{"name":"draft-a"}}`))
	create.Header.Set("Authorization", "Bearer "+tokenA)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	getOwner := httptest.NewRequest(http.MethodGet, "/api/v1/cpm/drafts?id=draft-1", nil)
	getOwner.Header.Set("Authorization", "Bearer "+tokenA)
	getOwnerRes := httptest.NewRecorder()
	h.ServeHTTP(getOwnerRes, getOwner)
	if getOwnerRes.Code != http.StatusOK {
		t.Fatalf("expected owner read status 200, got %d body=%s", getOwnerRes.Code, getOwnerRes.Body.String())
	}

	getOther := httptest.NewRequest(http.MethodGet, "/api/v1/cpm/drafts?id=draft-1", nil)
	getOther.Header.Set("Authorization", "Bearer "+tokenB)
	getOtherRes := httptest.NewRecorder()
	h.ServeHTTP(getOtherRes, getOther)
	if getOtherRes.Code != http.StatusForbidden {
		t.Fatalf("expected cross-user read status 403, got %d body=%s", getOtherRes.Code, getOtherRes.Body.String())
	}

	updateOther := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/drafts", strings.NewReader(`{"id":"draft-1","payload":{"name":"hijack"}}`))
	updateOther.Header.Set("Authorization", "Bearer "+tokenB)
	updateOther.Header.Set("Content-Type", "application/json")
	updateOtherRes := httptest.NewRecorder()
	h.ServeHTTP(updateOtherRes, updateOther)
	if updateOtherRes.Code != http.StatusForbidden {
		t.Fatalf("expected cross-user update status 403, got %d body=%s", updateOtherRes.Code, updateOtherRes.Body.String())
	}
}

func TestPolicyEndpointsAreOwnerScopedAtAPILevel(t *testing.T) {
	h := newAuthedTestHandler(t)
	tokenA := mustToken(t, "user-a")
	tokenB := mustToken(t, "user-b")

	create := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/policies", strings.NewReader(`{"id":"policy-1","payload":{"mode":"strict"}}`))
	create.Header.Set("Authorization", "Bearer "+tokenA)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	getOther := httptest.NewRequest(http.MethodGet, "/api/v1/cpm/policies?id=policy-1", nil)
	getOther.Header.Set("Authorization", "Bearer "+tokenB)
	getOtherRes := httptest.NewRecorder()
	h.ServeHTTP(getOtherRes, getOther)
	if getOtherRes.Code != http.StatusForbidden {
		t.Fatalf("expected cross-user read status 403, got %d body=%s", getOtherRes.Code, getOtherRes.Body.String())
	}
}

func TestOwnerFieldsFromClientAreRejected(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "user-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/drafts", strings.NewReader(`{"id":"draft-2","owner_user_id":"evil","tenant_id":"evil","payload":{"x":1}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for owner override fields, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestMissingPrincipalFailsClosedOnOwnerEndpoints(t *testing.T) {
	h := newAuthedTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/drafts", strings.NewReader(`{"id":"draft-3","payload":{"x":1}}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without principal, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestOwnerPersistedFromPrincipal(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "user-persist")

	create := httptest.NewRequest(http.MethodPost, "/api/v1/cpm/drafts", strings.NewReader(`{"id":"draft-persist","payload":{"k":"v"}}`))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	item, _ := body["item"].(map[string]any)
	if item["OwnerUserID"] != "user-persist" {
		t.Fatalf("expected persisted owner from principal, got %#v", item["OwnerUserID"])
	}
}

func newAuthedTestHandler(t *testing.T) http.Handler {
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
	t.Cleanup(introspect.Close)
	h, err := handler("cafe-cpm", store, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return h
}

func mustToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": userID,
		"email":   userID + "@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	return token
}
