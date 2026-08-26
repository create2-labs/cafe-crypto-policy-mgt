//go:build dev

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func TestDraftRoutesRemovedAfterRDP5(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "user-a")
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(`{"id":"draft-1","payload":{"x":1}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound && res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("draft POST must be gone, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPolicyEndpointsAreOwnerScopedAtAPILevel(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	tokenA := mustToken(t, "user-a")
	tokenB := mustToken(t, "user-b")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	create.Header.Set("Authorization", "Bearer "+tokenA)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}
	var resp policyPersistResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	getOther := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id="+resp.PolicyID, nil)
	getOther.Header.Set("Authorization", "Bearer "+tokenB)
	getOtherRes := httptest.NewRecorder()
	h.ServeHTTP(getOtherRes, getOther)
	if getOtherRes.Code != http.StatusForbidden && getOtherRes.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user read deny, got %d body=%s", getOtherRes.Code, getOtherRes.Body.String())
	}
}

func TestMissingPrincipalFailsClosedOnOwnerEndpoints(t *testing.T) {
	h := newAuthedTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id=x", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without principal, got %d body=%s", res.Code, res.Body.String())
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

func TestPOSTPolicyRequiresSignedBody(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	tok := mustToken(t, "user-discovery")
	body := `{"wallet_address":"` + policyPersistTestWallet + `","chain_id":1,"scan_id":"` + policyPersistTestScanID + `","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden && res.Code != http.StatusBadRequest {
		t.Fatalf("expected fail without signature, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDELETEPolicy204And404(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	tokA := mustToken(t, "owner-a")
	tokB := mustToken(t, "owner-b")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	create.Header.Set("Authorization", "Bearer "+tokA)
	create.Header.Set("Content-Type", "application/json")
	cr := httptest.NewRecorder()
	h.ServeHTTP(cr, create)
	if cr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", cr.Code, cr.Body.String())
	}
	var resp policyPersistResponse
	if err := json.Unmarshal(cr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	del := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id="+resp.PolicyID+"&reason=test", nil)
	del.Header.Set("Authorization", "Bearer "+tokA)
	dr := httptest.NewRecorder()
	h.ServeHTTP(dr, del)
	if dr.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", dr.Code)
	}
	del404 := httptest.NewRequest(http.MethodDelete, cpmroutes.Policies+"?id="+resp.PolicyID, nil)
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
}

