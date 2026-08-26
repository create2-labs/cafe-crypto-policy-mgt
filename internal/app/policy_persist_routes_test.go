//go:build dev

package app

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	policyPersistTestPrivateKey = "c87509a1a067e1eb07e3bcb3d0d47c41102a221097c02183ccac2fdaba05632c"
	policyPersistTestWallet     = "0xee387b44819eb54d7fff026a18229421738a8a24"
	policyPersistTestDomain     = "api.example.com"
	policyPersistTestScanID     = "550e8400-e29b-41d4-a716-446655440000"
)

func newPolicyPersistTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return newPolicyPersistTestHandlerWithDiscovery(t, policyPersistTestScanID, true)
}

func newPolicyPersistTestHandlerWithDiscovery(t *testing.T, latestScanID string, discoveryUp bool) http.Handler {
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
		WalletAuthDomain:            policyPersistTestDomain,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return handler
}

func signPolicyPersistMessage(t *testing.T, message string) string {
	t.Helper()
	privKey, err := crypto.HexToECDSA(policyPersistTestPrivateKey)
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

func policyPersistHashedPayload(t *testing.T) ([]byte, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "contract", "testdata", "payload_sha256", "hashed_payload_realistic_nested.json"))
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	digest, err := payloadhash.DigestJSON(raw)
	if err != nil {
		t.Fatalf("DigestJSON: %v", err)
	}
	return raw, digest
}

func buildSignedPolicyPersistBody(t *testing.T, payloadJSON []byte, digest string, issued, expires time.Time) string {
	t.Helper()
	message := walletauth.BuildMessage(walletauth.Fields{
		Domain:        policyPersistTestDomain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: policyPersistTestWallet,
		ChainID:       1,
		ScanID:        policyPersistTestScanID,
		PayloadSHA256: digest,
		IssuedAt:      issued,
		ExpiresAt:     expires,
	})
	sig := signPolicyPersistMessage(t, message)
	var payload any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"wallet_address":  policyPersistTestWallet,
		"chain_id":        1,
		"scan_id":         policyPersistTestScanID,
		"payload":         payload,
		"payload_sha256":  "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", // ignored
		"signed_message":  message,
		"signature":       sig,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestPolicyPersist_happyPathAndClientHashIgnored(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-ok")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyPersistResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.PolicyID == "" || resp.PayloadSHA256 != digest || resp.Status != "persisted" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	getReq := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id="+resp.PolicyID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET want 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var got policyRecordResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.PayloadSHA256 != digest {
		t.Fatalf("GET payload_sha256 = %q want %q", got.PayloadSHA256, digest)
	}
	findings, ok := got.Payload["accepted_findings"].([]any)
	if !ok || len(findings) != 2 {
		t.Fatalf("persisted findings = %#v", got.Payload["accepted_findings"])
	}
}

func TestPolicyPersist_retrySamePayload409(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-409")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first persist: %d %s", rec.Code, rec.Body.String())
	}

	// Fresh signature window for retry (same payload / wallet → W1 conflict).
	now2 := time.Now().UTC().Truncate(time.Second)
	body2 := buildSignedPolicyPersistBody(t, payloadJSON, digest, now2, now2.Add(walletauth.MaxValidityWindow))
	req2 := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	assertAuthErrorPayload(t, rec2, walletAuthCodePolicyAlreadyExists)
}

func TestPolicyPersist_signAPersistBFails(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-replay")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	message := walletauth.BuildMessage(walletauth.Fields{
		Domain:        policyPersistTestDomain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: policyPersistTestWallet,
		ChainID:       1,
		ScanID:        policyPersistTestScanID,
		PayloadSHA256: digest,
		IssuedAt:      now,
		ExpiresAt:     now.Add(walletauth.MaxValidityWindow),
	})
	sig := signPolicyPersistMessage(t, message)

	// Mutate payload after signing (B) while keeping signature for A.
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	payload["crypto_policy_id"] = "cpm_other_policy_v1"
	body, _ := json.Marshal(map[string]any{
		"wallet_address": policyPersistTestWallet,
		"chain_id":       1,
		"scan_id":        policyPersistTestScanID,
		"payload":        payload,
		"signed_message": message,
		"signature":      sig,
	})
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletauth.CodePayloadSHA256Mismatch)
}

func TestPolicyPersist_unorderedFindingsSameHash(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-findings")
	payloadJSON, digest := policyPersistHashedPayload(t)
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	payload["accepted_findings"] = []any{
		"requires_local_signer_state",
		"requires_bundler",
		"requires_bundler",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, raw, digest, now, now.Add(walletauth.MaxValidityWindow))
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp policyPersistResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.PayloadSHA256 != digest {
		t.Fatalf("payload_sha256 = %q want %q", resp.PayloadSHA256, digest)
	}
}

func TestPolicyPersist_divergentSnapshotFindings400(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-divergent")
	payloadJSON, _ := policyPersistHashedPayload(t)
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	snap := payload["accepted_provider_snapshot"].(map[string]any)
	snap["accepted_findings"] = []any{"requires_bundler"}
	raw, _ := json.Marshal(payload)
	digest, err := payloadhash.DigestJSON(raw)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, raw, digest, now, now.Add(walletauth.MaxValidityWindow))
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeCryptoPolicyPayloadInvalid)
}

func TestPolicyPersist_scanNotLatest422(t *testing.T) {
	h := newPolicyPersistTestHandlerWithDiscovery(t, "11111111-1111-4111-8111-111111111111", true)
	token := mustToken(t, "policy-persist-w2")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeScanNotLatest)
}

func TestPolicyPersist_discoveryDown503(t *testing.T) {
	h := newPolicyPersistTestHandlerWithDiscovery(t, policyPersistTestScanID, false)
	token := mustToken(t, "policy-persist-disco")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, walletAuthCodeDiscoveryUnavailable)
}

func TestPolicyPersist_gateAfterSignatureNoInsert(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-gate")
	payloadJSON, _ := policyPersistHashedPayload(t)
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	snap := payload["accepted_provider_snapshot"].(map[string]any)
	snap["references"] = []any{
		map[string]any{"kind": "source_repo", "url": "https://example.com", "commit": "unpinned_pending_fixture"},
	}
	raw, _ := json.Marshal(payload)
	digest, err := payloadhash.DigestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, raw, digest, now, now.Add(walletauth.MaxValidityWindow))
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeProviderRefsUnpinned)
}
