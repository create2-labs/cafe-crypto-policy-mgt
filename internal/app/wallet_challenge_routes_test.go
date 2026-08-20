//go:build dev

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

const (
	walletChallengeTestScanID = "550e8400-e29b-41d4-a716-446655440000"
	walletChallengeTestWallet = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	walletChallengeTestDomain = "api.example.com"
)

func newWalletChallengeTestHandler(t *testing.T) http.Handler {
	t.Helper()
	// Wallet auth domain + discovery stub for scan existence.
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	t.Cleanup(scanAuthz.Close)
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, walletChallengeTestScanID) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"scan_id":"` + walletChallengeTestScanID + `","status":"completed","result":{"target_address":"` + walletChallengeTestWallet + `"}}`))
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
		WalletAuthDomain:            walletChallengeTestDomain,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return handler
}

func testReadStore(t *testing.T) (*persistence.OwnerScopedStore, *api.ReadStore) {
	t.Helper()
	// reuse paths from newAuthedTestHandler
	store := persistence.NewOwnerScopedStore()
	readStore, err := api.LoadReadStore(api.ReadStoreOptions{
		CryptoPolicyPaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_pq_account_validation_v1.json"),
		},
		InstancePaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_instance_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		},

	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	return store, readStore
}

func createWalletBoundDraft(t *testing.T, h http.Handler, token, draftID string) {
	t.Helper()
	body := `{"id":"` + draftID + `","scan_id":"` + walletChallengeTestScanID + `","payload":{"policy_context":{"wallet_address":"` + walletChallengeTestWallet + `","wallet_type":"eoa"}}}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create draft: %d %s", rec.Code, rec.Body.String())
	}
}

func TestWalletChallenge_returnsCanonicalMessage(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-ok")
	draftID := "draft-wallet-challenge-ok"
	createWalletBoundDraft(t, h, token, draftID)

	req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(
		`{"wallet_address":"`+walletChallengeTestWallet+`","chain_id":1,"scan_id":"`+walletChallengeTestScanID+`","draft_id":"`+draftID+`","action":"persist_crypto_policy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp walletChallengeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("expected canonical message")
	}
	if !strings.Contains(resp.Message, "Domain: "+walletChallengeTestDomain) {
		t.Fatalf("expected configured domain in message: %q", resp.Message)
	}
	if resp.WalletAddress != strings.ToLower(walletChallengeTestWallet) {
		t.Fatalf("wallet_address = %q", resp.WalletAddress)
	}
	if resp.Action != walletauth.ActionPersistCryptoPolicy {
		t.Fatalf("action = %q", resp.Action)
	}
	parsed, err := walletauth.ParseMessage(resp.Message)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if parsed.ScanID != walletChallengeTestScanID || parsed.DraftID != draftID {
		t.Fatalf("parsed bindings mismatch: %#v", parsed)
	}
}

func TestWalletChallenge_validationMatrix(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-matrix")
	draftID := "draft-wallet-challenge-matrix"
	createWalletBoundDraft(t, h, token, draftID)

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "missing wallet", body: `{"chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","draft_id":"` + draftID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeAddressRequired},
		{name: "invalid wallet", body: `{"wallet_address":"not-an-address","chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","draft_id":"` + draftID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeInvalidAddress},
		{name: "missing chain", body: `{"wallet_address":"` + walletChallengeTestWallet + `","scan_id":"` + walletChallengeTestScanID + `","draft_id":"` + draftID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeChainIDRequired},
		{name: "unknown draft", body: `{"wallet_address":"` + walletChallengeTestWallet + `","chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","draft_id":"missing-draft","action":"persist_crypto_policy"}`, wantStatus: http.StatusNotFound, wantCode: walletAuthCodeDraftNotFound},
		{name: "draft scan mismatch", body: `{"wallet_address":"` + walletChallengeTestWallet + `","chain_id":1,"scan_id":"00000000-0000-0000-0000-000000000099","draft_id":"` + draftID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusConflict, wantCode: walletAuthCodeDraftScanMismatch},
		{name: "draft wallet mismatch", body: `{"wallet_address":"0x0000000000000000000000000000000000000001","chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","draft_id":"` + draftID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusConflict, wantCode: walletAuthCodeDraftWalletMismatch},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d want %d body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			assertAuthErrorPayload(t, rec, tc.wantCode)
		})
	}
}
