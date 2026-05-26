package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func TestPolicyWalletTargetReferenceInternalExists(t *testing.T) {
	store := policyRefTestReadStore(t)
	owner := persistence.NewOwnerScopedStore()
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "tenant-a"}
	addr := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	if _, err := owner.SavePolicy(principal, "pol-a", "550e8400-e29b-41d4-a716-446655440000", map[string]any{
		"policy_context": map[string]any{"target_address": addr},
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := handlerWithOwnerStore("cafe-cpm", store, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		PolicyReferenceInternalServiceToken: "correct-internal-token",
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := `{"target_address":"` + addr + `","user_id":"u1","tenant_id":"tenant-a"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceWalletTarget, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct-internal-token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["exists"] != true {
		t.Fatalf("exists = %v, want true", out["exists"])
	}
	if int(out["policy_count"].(float64)) != 1 {
		t.Fatalf("policy_count = %v", out["policy_count"])
	}
}

func TestPolicyWalletTargetReferenceInternalFalseWhenRemoved(t *testing.T) {
	store := policyRefTestReadStore(t)
	owner := persistence.NewOwnerScopedStore()
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "tenant-a"}
	addr := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	if _, err := owner.SaveDraft(principal, "draft-a", "550e8400-e29b-41d4-a716-446655440000", map[string]any{
		"policy_context": map[string]any{
			"wallet_address": addr,
		},
	}); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	defer introspect.Close()
	h, err := handlerWithOwnerStore("cafe-cpm", store, owner, authConfig{
		Required:                            true,
		SessionValidationURL:                introspect.URL,
		SessionValidationTimeoutSec:         3,
		ClockSkewSec:                        30,
		PolicyReferenceInternalServiceToken: "correct-internal-token",
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	other := "0x0000000000000000000000000000000000000001"
	body := `{"target_address":"` + other + `","user_id":"u1","tenant_id":"tenant-a"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.InternalPolicyReferenceWalletTarget, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct-internal-token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["exists"] != false {
		t.Fatalf("exists = %v, want false", out["exists"])
	}
}
