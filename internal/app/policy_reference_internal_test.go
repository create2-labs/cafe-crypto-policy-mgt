//go:build dev

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
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func policyRefTestReadStore(t *testing.T) *api.ReadStore {
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
	return store
}

func TestPolicyReferenceInternalForbiddenWithoutValidServiceToken(t *testing.T) {
	store := policyRefTestReadStore(t)
	owner := persistence.NewOwnerScopedStore()
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		PolicyReferenceInternalServiceToken: "correct-internal-token",
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := `{"scan_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"u1","tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceScan, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPolicyReferenceInternalUnavailableWhenTokenNotConfigured(t *testing.T) {
	store := policyRefTestReadStore(t)
	owner := persistence.NewOwnerScopedStore()
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := testHandler(store, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		PolicyReferenceInternalServiceToken: "",
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := `{"scan_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"u1","tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceScan, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer any-token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPolicyReferenceInternalReferencedAndCount(t *testing.T) {
	store := policyRefTestReadStore(t)
	owner := persistence.NewOwnerScopedStore()
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "tenant-a"}
	scanID := "550e8400-e29b-41d4-a716-446655440000"
	if _, err := owner.SavePolicy(principal, "pol-a", scanID, map[string]any{"x": 1}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := owner.SavePolicy(principal, "pol-b", scanID, map[string]any{"x": 2}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := owner.SavePolicy(principal, "pol-c", "660e8400-e29b-41d4-a716-446655440001", nil); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	const svcToken = "svc-token-policy-ref"
	h, err := testHandler(store, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		PolicyReferenceInternalServiceToken: svcToken,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := `{"scan_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"u1","tenant_id":"tenant-a"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceScan, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+svcToken)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var out struct {
		Referenced bool `json:"referenced"`
		Count      int  `json:"count"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Referenced || out.Count != 2 {
		t.Fatalf("expected referenced=true count=2, got %+v", out)
	}

	bodyNoRef := `{"scan_id":"00000000-0000-0000-0000-000000000099","user_id":"u1","tenant_id":"tenant-a"}`
	req2 := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceScan, strings.NewReader(bodyNoRef))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+svcToken)
	res2 := httptest.NewRecorder()
	h.ServeHTTP(res2, req2)
	var out2 struct {
		Referenced bool `json:"referenced"`
		Count      int  `json:"count"`
	}
	if err := json.Unmarshal(res2.Body.Bytes(), &out2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out2.Referenced || out2.Count != 0 {
		t.Fatalf("expected referenced=false count=0 for scan without policies, got referenced=%v count=%d", out2.Referenced, out2.Count)
	}
}

func TestPublicGETPoliciesByScanIDAgreesWithInternalReferenceCheck(t *testing.T) {
	owner := persistence.NewOwnerScopedStore()
	read := policyRefTestReadStore(t)
	scanID := "550e8400-e29b-41d4-a716-446655440000"
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "tenant-a"}
	if _, err := owner.SavePolicy(principal, "pol-1", scanID, map[string]any{"x": 1}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := owner.SavePolicy(principal, "pol-2", scanID, map[string]any{"x": 2}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	scanAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	t.Cleanup(scanAuth.Close)
	const svcToken = "svc-ref-agree"
	h, err := testHandler(read, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		ScanAuthorizationURL:                scanAuth.URL,
		ScanAuthorizationTimeoutSec:         2,
		PolicyReferenceInternalServiceToken: svcToken,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id":   "u1",
		"tenant_id": "tenant-a",
		"email":     "u1@example.com",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	getReq := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?scan_id="+scanID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRes := httptest.NewRecorder()
	h.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GET policies by scan_id: want 200, got %d body=%s", getRes.Code, getRes.Body.String())
	}
	var listBody struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(getRes.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listBody.Total != 2 || len(listBody.Items) != 2 {
		t.Fatalf("want 2 policies in list, got total=%d len(items)=%d", listBody.Total, len(listBody.Items))
	}

	internalBody := `{"scan_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"u1","tenant_id":"tenant-a"}`
	inReq := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceScan, strings.NewReader(internalBody))
	inReq.Header.Set("Content-Type", "application/json")
	inReq.Header.Set("Authorization", "Bearer "+svcToken)
	inRes := httptest.NewRecorder()
	h.ServeHTTP(inRes, inReq)
	if inRes.Code != http.StatusOK {
		t.Fatalf("internal: want 200, got %d body=%s", inRes.Code, inRes.Body.String())
	}
	var inOut struct {
		Referenced bool `json:"referenced"`
		Count      int  `json:"count"`
	}
	if err := json.Unmarshal(inRes.Body.Bytes(), &inOut); err != nil {
		t.Fatalf("decode internal: %v", err)
	}
	if !inOut.Referenced || inOut.Count != len(listBody.Items) {
		t.Fatalf("internal count=%d referenced=%v vs public len=%d", inOut.Count, inOut.Referenced, len(listBody.Items))
	}
}
