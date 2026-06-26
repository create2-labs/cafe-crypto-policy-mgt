//go:build dev

package app

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	draftPersistTestPrivateKey = "c87509a1a067e1eb07e3bcb3d0d47c41102a221097c02183ccac2fdaba05632c"
	draftPersistTestWallet     = "0xee387b44819eb54d7fff026a18229421738a8a24"
	draftPersistTestDomain     = "api.example.com"
	draftPersistTestScanID     = "550e8400-e29b-41d4-a716-446655440000"
)

func newDraftPersistTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return newWalletChallengeTestHandlerWithWallet(t, draftPersistTestWallet, draftPersistTestScanID, draftPersistTestDomain)
}

func newWalletChallengeTestHandlerWithWallet(t *testing.T, wallet, scanID, domain string) http.Handler {
	t.Helper()
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	t.Cleanup(scanAuthz.Close)
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, scanID) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"`+scanID+`","status":"completed","result":{"target_address":"`+wallet+`"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(discovery.Close)

	store, readStore := testReadStore(t)
	handler, err := testHandler(readStore, store, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ScanAuthorizationURL:        scanAuthz.URL,
		ScanAuthorizationTimeoutSec: 2,
		ClockSkewSec:                30,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     2,
		WalletAuthDomain:            domain,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return handler
}

func signDraftPersistMessage(t *testing.T, message string) string {
	t.Helper()
	privKey, err := crypto.HexToECDSA(draftPersistTestPrivateKey)
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	hash := accounts.TextHash([]byte(message))
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig)
}

func buildDraftPersistSignedRequest(t *testing.T, draftID string, issued, expires time.Time) (message, signature string) {
	t.Helper()
	message = walletauth.BuildMessage(walletauth.Fields{
		Domain:        draftPersistTestDomain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: draftPersistTestWallet,
		ChainID:       1,
		ScanID:        draftPersistTestScanID,
		DraftID:       draftID,
		IssuedAt:      issued,
		ExpiresAt:     expires,
	})
	return message, signDraftPersistMessage(t, message)
}

func createDraftPersistBoundDraft(t *testing.T, h http.Handler, token, draftID string) {
	t.Helper()
	body := `{"id":"` + draftID + `","scan_id":"` + draftPersistTestScanID + `","payload":{"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa"}}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create draft: %d %s", rec.Code, rec.Body.String())
	}
}

func TestDraftPersist_happyPath(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-ok")
	draftID := "draft-persist-ok"
	createDraftPersistBoundDraft(t, h, token, draftID)

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp draftPersistResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "persisted" || resp.OwnershipStatus != "verified" || resp.WalletControlMethod != "eoa_signature" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.PolicyID == "" || resp.DraftID != draftID {
		t.Fatalf("policy/draft ids: %#v", resp)
	}
}

func TestDraftPersist_missingSignatureReturnsProofRequired(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-missing-sig")
	draftID := "draft-persist-missing-sig"
	createDraftPersistBoundDraft(t, h, token, draftID)

	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `"}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeControlProofRequired)
}

func TestDraftPersist_replayReturnsAlreadyPersisted(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-replay")
	draftID := "draft-persist-replay"
	createDraftPersistBoundDraft(t, h, token, draftID)

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	first := post()
	if first.Code != http.StatusOK {
		t.Fatalf("first persist: %d %s", first.Code, first.Body.String())
	}
	second := post()
	if second.Code != http.StatusConflict {
		t.Fatalf("replay status = %d want 409 body=%s", second.Code, second.Body.String())
	}
	assertAuthErrorPayload(t, second, walletAuthCodeDraftAlreadyPersisted)
}

func TestPersistPolicyRequiresWalletSignatureForEOA(t *testing.T) {
	h := newAuthedTestHandlerWithScanAuthz(t)
	token := mustToken(t, "legacy-eoa-persist")
	scanID := optionAContractScanID
	body := `{"id":"legacy-eoa","scan_id":"` + scanID + `","payload":{"selected_wallet_policy_context":{"wallet_address":"0xabc","target_address":"0xabc","scan_id":"` + scanID + `"}}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d want 403 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeControlProofRequired)
}

func TestPOSTPolicies_fixtureBindingWithoutWalletStillAllowed(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "fixture-policy-ok")
	body := `{"id":"fixture-policy-ok","binding":"fixture","payload":{"mode":"strict"}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
