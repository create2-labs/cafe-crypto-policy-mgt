package app

import (
	"encoding/json"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
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

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-1","payload":{"name":"draft-a"}}`))
	create.Header.Set("Authorization", "Bearer "+tokenA)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	getOwner := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id=draft-1", nil)
	getOwner.Header.Set("Authorization", "Bearer "+tokenA)
	getOwnerRes := httptest.NewRecorder()
	h.ServeHTTP(getOwnerRes, getOwner)
	if getOwnerRes.Code != http.StatusOK {
		t.Fatalf("expected owner read status 200, got %d body=%s", getOwnerRes.Code, getOwnerRes.Body.String())
	}

	getOther := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id=draft-1", nil)
	getOther.Header.Set("Authorization", "Bearer "+tokenB)
	getOtherRes := httptest.NewRecorder()
	h.ServeHTTP(getOtherRes, getOther)
	if getOtherRes.Code != http.StatusForbidden {
		t.Fatalf("expected cross-user read status 403, got %d body=%s", getOtherRes.Code, getOtherRes.Body.String())
	}

	updateOther := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-1","payload":{"name":"hijack"}}`))
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

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(`{"id":"policy-1","binding":"fixture","payload":{"mode":"strict"}}`))
	create.Header.Set("Authorization", "Bearer "+tokenA)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	getOther := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id=policy-1", nil)
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

	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-2","owner_user_id":"evil","tenant_id":"evil","payload":{"x":1}}`))
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
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-3","payload":{"x":1}}`))
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

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-persist","payload":{"k":"v"}}`))
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

func TestGETPoliciesMutuallyExclusiveQueryReturns400(t *testing.T) {
	h := newAuthedTestHandler(t)
	tok := mustToken(t, "u1")
	scan := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id=policy-x&scan_id="+scan, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPOSTPolicyDiscoveryRequiresScanUUID(t *testing.T) {
	h := newAuthedTestHandler(t)
	tok := mustToken(t, "user-discovery")
	body := `{"id":"p-no-scan","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without scan_id, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDELETEPolicy204And404(t *testing.T) {
	h := newAuthedTestHandler(t)
	tokA := mustToken(t, "owner-a")
	tokB := mustToken(t, "owner-b")
	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(`{"id":"to-del","binding":"fixture","payload":{"k":1}}`))
	create.Header.Set("Authorization", "Bearer "+tokA)
	create.Header.Set("Content-Type", "application/json")
	cr := httptest.NewRecorder()
	h.ServeHTTP(cr, create)
	if cr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", cr.Code, cr.Body.String())
	}
	del := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id=to-del", nil)
	del.Header.Set("Authorization", "Bearer "+tokA)
	dr := httptest.NewRecorder()
	h.ServeHTTP(dr, del)
	if dr.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", dr.Code)
	}
	del404 := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id=to-del", nil)
	del404.Header.Set("Authorization", "Bearer "+tokA)
	d404 := httptest.NewRecorder()
	h.ServeHTTP(d404, del404)
	if d404.Code != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d body=%s", d404.Code, d404.Body.String())
	}
	delOther := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id=ghost", nil)
	delOther.Header.Set("Authorization", "Bearer "+tokB)
	do := httptest.NewRecorder()
	h.ServeHTTP(do, delOther)
	if do.Code != http.StatusNotFound {
		t.Fatalf("delete missing as other user: expected 404, got %d", do.Code)
	}

	tokC := mustToken(t, "owner-c")
	createC := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(`{"id":"owned-by-c","binding":"fixture","payload":{"x":1}}`))
	createC.Header.Set("Authorization", "Bearer "+tokC)
	createC.Header.Set("Content-Type", "application/json")
	cc := httptest.NewRecorder()
	h.ServeHTTP(cc, createC)
	if cc.Code != http.StatusOK {
		t.Fatalf("create C: %d %s", cc.Code, cc.Body.String())
	}
	delCross := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id=owned-by-c", nil)
	delCross.Header.Set("Authorization", "Bearer "+tokA)
	dx := httptest.NewRecorder()
	h.ServeHTTP(dx, delCross)
	if dx.Code != http.StatusNotFound {
		t.Fatalf("delete other owner's policy: expected 404, got %d body=%s", dx.Code, dx.Body.String())
	}
}

func TestDELETEDraft204And404(t *testing.T) {
	h := newAuthedTestHandler(t)
	tokA := mustToken(t, "owner-a")
	tokB := mustToken(t, "owner-b")
	create := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-del","payload":{"name":"x"}}`))
	create.Header.Set("Authorization", "Bearer "+tokA)
	create.Header.Set("Content-Type", "application/json")
	cr := httptest.NewRecorder()
	h.ServeHTTP(cr, create)
	if cr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", cr.Code, cr.Body.String())
	}
	del := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id=draft-del", nil)
	del.Header.Set("Authorization", "Bearer "+tokA)
	dr := httptest.NewRecorder()
	h.ServeHTTP(dr, del)
	if dr.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", dr.Code)
	}
	del404 := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id=draft-del", nil)
	del404.Header.Set("Authorization", "Bearer "+tokA)
	d404 := httptest.NewRecorder()
	h.ServeHTTP(d404, del404)
	if d404.Code != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d body=%s", d404.Code, d404.Body.String())
	}
	delOther := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id=ghost-draft", nil)
	delOther.Header.Set("Authorization", "Bearer "+tokB)
	do := httptest.NewRecorder()
	h.ServeHTTP(do, delOther)
	if do.Code != http.StatusNotFound {
		t.Fatalf("delete missing as other user: expected 404, got %d", do.Code)
	}

	tokC := mustToken(t, "owner-c")
	createC := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-owned-by-c","payload":{"y":1}}`))
	createC.Header.Set("Authorization", "Bearer "+tokC)
	createC.Header.Set("Content-Type", "application/json")
	cc := httptest.NewRecorder()
	h.ServeHTTP(cc, createC)
	if cc.Code != http.StatusOK {
		t.Fatalf("create C: %d %s", cc.Code, cc.Body.String())
	}
	delCross := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id=draft-owned-by-c", nil)
	delCross.Header.Set("Authorization", "Bearer "+tokA)
	dx := httptest.NewRecorder()
	h.ServeHTTP(dx, delCross)
	if dx.Code != http.StatusNotFound {
		t.Fatalf("delete other owner's draft: expected 404, got %d body=%s", dx.Code, dx.Body.String())
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
