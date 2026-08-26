//go:build dev

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

const (
	walletChallengeTestScanID = "550e8400-e29b-41d4-a716-446655440000"
	walletChallengeTestWallet = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	walletChallengeTestDomain = "api.example.com"
)

func walletChallengePayloadGoldenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "contract", "testdata", "payload_sha256")
}

func readWalletChallengeGoldenPayload(t *testing.T, name string) (raw []byte, wantDigest string) {
	t.Helper()
	dir := walletChallengePayloadGoldenDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		t.Fatalf("read golden json: %v", err)
	}
	sha, err := os.ReadFile(filepath.Join(dir, name+".sha256"))
	if err != nil {
		t.Fatalf("read golden sha256: %v", err)
	}
	return raw, strings.TrimSpace(string(sha))
}

func newWalletChallengeTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return newWalletChallengeTestHandlerWithW2(t, walletChallengeTestWallet, walletChallengeTestScanID, true)
}

// newWalletChallengeTestHandlerWithW2 mocks Discovery latest=true for W2 engagement.
// When discoveryUp is false, Discovery returns 503 (fail-closed).
func newWalletChallengeTestHandlerWithW2(t *testing.T, wallet, latestScanID string, discoveryUp bool) http.Handler {
	t.Helper()
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	t.Cleanup(scanAuthz.Close)
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !discoveryUp {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"down"}`))
			return
		}
		if r.URL.Path == "/discovery/v1/wallets/scans" && r.URL.Query().Get("latest") == "true" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"scan_id":"` + latestScanID + `","status":"completed"}],"total":1,"limit":1,"offset":0}`))
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

func walletChallengeBody(t *testing.T, payloadJSON []byte, findingsOverride []string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if findingsOverride != nil {
		arr := make([]any, len(findingsOverride))
		for i, s := range findingsOverride {
			arr[i] = s
		}
		payload["accepted_findings"] = arr
	}
	envelope := map[string]any{
		"wallet_address": walletChallengeTestWallet,
		"chain_id":       1,
		"scan_id":        walletChallengeTestScanID,
		"action":         walletauth.ActionPersistCryptoPolicy,
		"payload":        payload,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return string(raw)
}

func TestWalletChallenge_returnsCanonicalMessageWithPayloadSHA256(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-ok")
	payloadJSON, wantDigest := readWalletChallengeGoldenPayload(t, "hashed_payload_minimal")

	req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(
		walletChallengeBody(t, payloadJSON, nil),
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
	if strings.Contains(resp.Message, "Draft ID:") {
		t.Fatalf("RD-P4 message must not include Draft ID: %q", resp.Message)
	}
	if !strings.Contains(resp.Message, "Payload SHA-256: "+wantDigest) {
		t.Fatalf("expected payload sha in message: %q", resp.Message)
	}
	if resp.PayloadSHA256 != wantDigest {
		t.Fatalf("payload_sha256 = %q want %q", resp.PayloadSHA256, wantDigest)
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
	if parsed.ScanID != walletChallengeTestScanID || parsed.PayloadSHA256 != wantDigest {
		t.Fatalf("parsed bindings mismatch: %#v", parsed)
	}
	if parsed.DraftID != "" {
		t.Fatalf("expected empty DraftID, got %q", parsed.DraftID)
	}
}

func TestWalletChallenge_findingsOrderYieldsSamePayloadSHA256(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-findings-order")
	payloadJSON, wantDigest := readWalletChallengeGoldenPayload(t, "hashed_payload_minimal")

	bodies := []string{
		walletChallengeBody(t, payloadJSON, []string{"requires_bundler", "requires_local_signer_state"}),
		walletChallengeBody(t, payloadJSON, []string{
			"requires_local_signer_state",
			"requires_bundler",
			"requires_bundler",
			"requires_local_signer_state",
		}),
	}

	var digests []string
	for i, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("case %d: expected 200, got %d body=%s", i, rec.Code, rec.Body.String())
		}
		var resp walletChallengeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("case %d decode: %v", i, err)
		}
		digests = append(digests, resp.PayloadSHA256)
		if !strings.Contains(resp.Message, "Payload SHA-256: "+resp.PayloadSHA256) {
			t.Fatalf("case %d: message missing payload sha line", i)
		}
	}
	if digests[0] != digests[1] {
		t.Fatalf("unordered/deduped findings must yield same payload_sha256: %q vs %q", digests[0], digests[1])
	}
	if digests[0] != wantDigest {
		t.Fatalf("payload_sha256 = %q want golden %q", digests[0], wantDigest)
	}
}

func TestWalletChallenge_validationMatrix(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-matrix")
	payloadJSON, _ := readWalletChallengeGoldenPayload(t, "hashed_payload_minimal")
	validPayload := walletChallengeBody(t, payloadJSON, nil)

	var base map[string]any
	if err := json.Unmarshal(payloadJSON, &base); err != nil {
		t.Fatal(err)
	}

	unknownPayload := cloneMap(base)
	unknownPayload["extra_unknown"] = "nope"
	badUnknown, err := json.Marshal(map[string]any{
		"wallet_address": walletChallengeTestWallet,
		"chain_id":       1,
		"scan_id":        walletChallengeTestScanID,
		"action":         walletauth.ActionPersistCryptoPolicy,
		"payload":        unknownPayload,
	})
	if err != nil {
		t.Fatal(err)
	}

	numberPayload := cloneMap(base)
	snap := cloneMap(numberPayload["accepted_provider_snapshot"].(map[string]any))
	chain := cloneMap(snap["chain_support_used"].(map[string]any))
	chain["chain_id"] = float64(11155111) // forbidden number in hashed subtree
	snap["chain_support_used"] = chain
	numberPayload["accepted_provider_snapshot"] = snap
	badNumber, err := json.Marshal(map[string]any{
		"wallet_address": walletChallengeTestWallet,
		"chain_id":       1,
		"scan_id":        walletChallengeTestScanID,
		"action":         walletauth.ActionPersistCryptoPolicy,
		"payload":        numberPayload,
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "missing wallet", body: `{"chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","action":"persist_crypto_policy","payload":` + string(payloadJSON) + `}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeAddressRequired},
		{name: "invalid wallet", body: `{"wallet_address":"not-an-address","chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","action":"persist_crypto_policy","payload":` + string(payloadJSON) + `}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeInvalidAddress},
		{name: "missing chain", body: `{"wallet_address":"` + walletChallengeTestWallet + `","scan_id":"` + walletChallengeTestScanID + `","action":"persist_crypto_policy","payload":` + string(payloadJSON) + `}`, wantStatus: http.StatusBadRequest, wantCode: walletAuthCodeChainIDRequired},
		{name: "missing payload", body: `{"wallet_address":"` + walletChallengeTestWallet + `","chain_id":1,"scan_id":"` + walletChallengeTestScanID + `","action":"persist_crypto_policy"}`, wantStatus: http.StatusBadRequest, wantCode: persistCodeCryptoPolicyPayloadInvalid},
		{name: "unknown hashed field", body: string(badUnknown), wantStatus: http.StatusBadRequest, wantCode: persistCodeCryptoPolicyPayloadInvalid},
		{name: "number in hashed subtree", body: string(badNumber), wantStatus: http.StatusBadRequest, wantCode: persistCodeCryptoPolicyPayloadInvalid},
		{name: "valid baseline", body: validPayload, wantStatus: http.StatusOK, wantCode: ""},
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
			if tc.wantCode != "" {
				assertAuthErrorPayload(t, rec, tc.wantCode)
			}
		})
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func TestWalletChallenge_messageStableAcrossCallsExceptTimestamps(t *testing.T) {
	h := newWalletChallengeTestHandler(t)
	token := mustToken(t, "wallet-challenge-stable")
	payloadJSON, wantDigest := readWalletChallengeGoldenPayload(t, "hashed_payload_realistic_nested")
	body := walletChallengeBody(t, payloadJSON, nil)

	digestDirect, err := payloadhash.DigestJSON(payloadJSON)
	if err != nil {
		t.Fatalf("DigestJSON: %v", err)
	}
	if digestDirect != wantDigest {
		t.Fatalf("direct digest %q want %q", digestDirect, wantDigest)
	}

	req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp walletChallengeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.PayloadSHA256 != wantDigest {
		t.Fatalf("payload_sha256 = %q want %q", resp.PayloadSHA256, wantDigest)
	}
	parsed, err := walletauth.ParseMessage(resp.Message)
	if err != nil {
		t.Fatal(err)
	}
	rebuild := walletauth.BuildMessage(walletauth.Fields{
		Domain:        walletChallengeTestDomain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: strings.ToLower(walletChallengeTestWallet),
		ChainID:       1,
		ScanID:        walletChallengeTestScanID,
		PayloadSHA256: wantDigest,
		IssuedAt:      parsed.IssuedAt,
		ExpiresAt:     parsed.ExpiresAt,
	})
	if rebuild != resp.Message {
		t.Fatalf("message not stable for same bindings:\n got  %q\n want %q", resp.Message, rebuild)
	}
}

func TestWalletChallenge_W2RejectsNonLatestScan(t *testing.T) {
	historical := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	h := newWalletChallengeTestHandlerWithW2(t, walletChallengeTestWallet, walletChallengeTestScanID, true)
	token := mustToken(t, "wallet-challenge-w2-stale")
	payloadJSON, _ := readWalletChallengeGoldenPayload(t, "hashed_payload_minimal")

	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"wallet_address": walletChallengeTestWallet,
		"chain_id":       1,
		"scan_id":        historical,
		"action":         walletauth.ActionPersistCryptoPolicy,
		"payload":        payload,
	})
	req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeScanNotLatest)
}

func TestWalletChallenge_DiscoveryDownFailClosed503(t *testing.T) {
	h := newWalletChallengeTestHandlerWithW2(t, walletChallengeTestWallet, walletChallengeTestScanID, false)
	token := mustToken(t, "wallet-challenge-w2-down")
	payloadJSON, _ := readWalletChallengeGoldenPayload(t, "hashed_payload_minimal")
	req := httptest.NewRequest(http.MethodPost, cpmroutes.WalletChallenges, strings.NewReader(
		walletChallengeBody(t, payloadJSON, nil),
	))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeDiscoveryUnavailable)
}
