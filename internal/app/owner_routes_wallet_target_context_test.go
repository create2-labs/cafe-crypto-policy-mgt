//go:build dev

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func TestGETWalletTargetContextRequiresAddress(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "wtc-query-user")

	req := httptest.NewRequest(http.MethodGet, cpmroutes.WalletTargetContext, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGETWalletTargetContextExistsWithPolicy(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "wtc-exists-user")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	post := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	post.Header.Set("Authorization", "Bearer "+token)
	post.Header.Set("Content-Type", "application/json")
	pr := httptest.NewRecorder()
	h.ServeHTTP(pr, post)
	if pr.Code != http.StatusOK {
		t.Fatalf("persist: %d %s", pr.Code, pr.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, cpmroutes.WalletTargetContext+"?target_address="+policyPersistTestWallet, nil)
	get.Header.Set("Authorization", "Bearer "+token)
	gr := httptest.NewRecorder()
	h.ServeHTTP(gr, get)
	if gr.Code != http.StatusOK {
		t.Fatalf("context get: %d %s", gr.Code, gr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(gr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["exists"] != true {
		t.Fatalf("exists = %v", out["exists"])
	}
	if pc, ok := out["policy_count"].(float64); !ok || pc < 1 {
		t.Fatalf("policy_count = %v", out["policy_count"])
	}
}

func TestWalletTargetContextStoreCountViaCreatePolicy(t *testing.T) {
	store := persistence.NewOwnerScopedStore()
	principal := authz.Principal{UserID: "u1", Subject: "u1", TenantID: ""}
	_, err := store.CreatePolicy(principal, persistence.CreatePolicyInput{
		ScanID:                  policyPersistTestScanID,
		WalletAddress:           policyPersistTestWallet,
		ChainID:                 1,
		Payload:                 map[string]any{"k": "v"},
		PayloadSHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		WalletControlVerifiedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	counts, err := store.CountActiveWalletCPMContext(principal, policyPersistTestWallet)
	if err != nil {
		t.Fatal(err)
	}
	if !counts.Exists || counts.PolicyCount < 1 {
		t.Fatalf("counts = %+v", counts)
	}
}
