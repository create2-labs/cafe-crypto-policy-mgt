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
			_, _ = w.Write([]byte(`{"scan_id":"` + scanID + `","status":"completed","result":{"target_address":"` + wallet + `"}}`))
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
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, draftPersistValidPayloadJSON())
}

func createDraftPersistBoundDraftWithPayload(t *testing.T, h http.Handler, token, draftID, payloadJSON string) {
	t.Helper()
	body := `{"id":"` + draftID + `","scan_id":"` + draftPersistTestScanID + `","payload":` + payloadJSON + `}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create draft: %d %s", rec.Code, rec.Body.String())
	}
}

// draftPersistValidPayloadJSON is a cafe.crypto_policy.v0.2 draft body with pinned refs (CPM-P10).
func draftPersistValidPayloadJSON() string {
	return `{
		"schema_version":"cafe.crypto_policy.v0.2",
		"policy_kind":"wallet_migration_policy",
		"crypto_policy_id":"cpm_pq_account_validation_v1",
		"required_posture":"hybrid",
		"user_constraints":{
			"allow_new_wallet":true,
			"address_continuity_required":false,
			"key_rotation_model":"per_userop"
		},
		"solution_profile_ref":{
			"provider_id":"nicetry",
			"solution_profile_id":"nicetry.fors_c.erc4337.v0_1",
			"manifest_version":"2026-08",
			"verification_date":"2026-08-03"
		},
		"accepted_provider_snapshot":{
			"maturity":"research",
			"claim_status":"declared",
			"resulting_posture":"hybrid",
			"input_requirements":{"wallet_types":["EOA"],"requires_wallet_control_proof":true},
			"signature":{"scheme":"FORS+C","family":"hash_based","key_rotation_model":"per_userop"},
			"account_model":{"standard":"ERC-4337","execution_model":"erc4337_bundler","requires_bundler":true,"requires_entrypoint":true,"entrypoint_versions":["0.7"]},
			"constraints":{"requires_new_account":true,"address_continuity_supported":false,"requires_local_signer_state":true},
			"chain_support_used":{"chain_id":11155111,"status":"testnet_supported","capabilities":["deploy","sign_userop","rotate_signer"]},
			"references":[
				{"kind":"source_repo","url":"https://github.com/RivaLabs-Core/NiceTry","commit":"abc123deadbeef"},
				{"kind":"protocol_spec","url":"https://github.com/RivaLabs-Core/Ephemeral-Keys-Protocol","version":"v0.1.0-test"}
			],
			"accepted_findings":["requires_bundler","requires_local_signer_state"],
			"accepted_risk_notes":["test note"]
		},
		"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa","chain_ids":[11155111]}
	}`
}

func draftPersistUnpinnedPayloadJSON() string {
	return `{
		"schema_version":"cafe.crypto_policy.v0.2",
		"crypto_policy_id":"cpm_pq_account_validation_v1",
		"required_posture":"hybrid",
		"user_constraints":{
			"allow_new_wallet":true,
			"address_continuity_required":false,
			"key_rotation_model":"per_userop"
		},
		"solution_profile_ref":{
			"provider_id":"nicetry",
			"solution_profile_id":"nicetry.fors_c.erc4337.v0_1",
			"manifest_version":"2026-08"
		},
		"accepted_provider_snapshot":{
			"maturity":"research",
			"claim_status":"declared",
			"resulting_posture":"hybrid",
			"input_requirements":{"wallet_types":["EOA"],"requires_wallet_control_proof":true},
			"signature":{"scheme":"FORS+C","family":"hash_based","key_rotation_model":"per_userop"},
			"account_model":{"standard":"ERC-4337","execution_model":"erc4337_bundler","requires_bundler":true,"requires_entrypoint":true},
			"constraints":{"requires_new_account":true,"address_continuity_supported":false,"requires_local_signer_state":true},
			"chain_support_used":{"chain_id":11155111,"status":"testnet_supported","capabilities":["deploy","sign_userop","rotate_signer"]},
			"references":[
				{"kind":"source_repo","url":"https://github.com/RivaLabs-Core/NiceTry","commit":"unpinned_pending_fixture"}
			],
			"accepted_findings":["requires_bundler","requires_local_signer_state"]
		},
		"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa","chain_ids":[11155111]}
	}`
}

func draftPersistCoucheBKOPayloadJSON() string {
	return `{
		"schema_version":"cafe.crypto_policy.v0.2",
		"crypto_policy_id":"cpm_pq_account_validation_v1",
		"required_posture":"hybrid",
		"user_constraints":{
			"allow_new_wallet":true,
			"address_continuity_required":true,
			"key_rotation_model":"per_userop"
		},
		"solution_profile_ref":{
			"provider_id":"nicetry",
			"solution_profile_id":"nicetry.fors_c.erc4337.v0_1",
			"manifest_version":"2026-08"
		},
		"accepted_provider_snapshot":{
			"maturity":"research",
			"claim_status":"declared",
			"resulting_posture":"hybrid",
			"input_requirements":{"wallet_types":["EOA"],"requires_wallet_control_proof":true},
			"signature":{"scheme":"FORS+C","family":"hash_based","key_rotation_model":"per_userop"},
			"account_model":{"standard":"ERC-4337","execution_model":"erc4337_bundler","requires_bundler":true,"requires_entrypoint":true},
			"constraints":{"requires_new_account":true,"address_continuity_supported":false,"requires_local_signer_state":true},
			"chain_support_used":{"chain_id":11155111,"status":"testnet_supported","capabilities":["deploy","sign_userop","rotate_signer"]},
			"references":[
				{"kind":"source_repo","url":"https://github.com/RivaLabs-Core/NiceTry","commit":"abc123deadbeef"}
			],
			"accepted_findings":["requires_bundler","requires_local_signer_state"]
		},
		"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa","chain_ids":[11155111]}
	}`
}

func draftPersistLegacyTemplatePayloadJSON() string {
	return `{
		"schema_version":"cafe.crypto_policy.v0.2",
		"template_id":"tpl_pq_account_validation_v1",
		"required_posture":"hybrid",
		"user_constraints":{
			"allow_new_wallet":true,
			"address_continuity_required":false,
			"key_rotation_model":"per_userop"
		},
		"solution_profile_ref":{
			"provider_id":"nicetry",
			"solution_profile_id":"nicetry.fors_c.erc4337.v0_1",
			"manifest_version":"2026-08"
		},
		"accepted_provider_snapshot":{
			"maturity":"research",
			"claim_status":"declared",
			"resulting_posture":"hybrid",
			"input_requirements":{"wallet_types":["EOA"]},
			"signature":{"scheme":"FORS+C","family":"hash_based","key_rotation_model":"per_userop"},
			"account_model":{"requires_bundler":true},
			"constraints":{"requires_new_account":true,"requires_local_signer_state":true},
			"chain_support_used":{"chain_id":11155111,"status":"testnet_supported","capabilities":["deploy","sign_userop","rotate_signer"]},
			"references":[
				{"kind":"source_repo","url":"https://example.com/a","commit":"abc123"}
			],
			"accepted_findings":["requires_bundler","requires_local_signer_state"]
		},
		"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa"}
	}`
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

func TestDraftPersist_rejectsUnpinnedProviderRefs(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-unpinned")
	draftID := "draft-persist-unpinned"
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, draftPersistUnpinnedPayloadJSON())

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeProviderRefsUnpinned)
}

func TestDraftPersist_rejectsLegacyTemplateID(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-legacy-tpl")
	draftID := "draft-persist-legacy-tpl"
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, draftPersistLegacyTemplatePayloadJSON())

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeCryptoPolicyPayloadInvalid)
}

func TestDraftPersist_rejectsCoucheBKO(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-couche-b")
	draftID := "draft-persist-couche-b"
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, draftPersistCoucheBKOPayloadJSON())

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeProviderUserConstraintsIncompatible)
}

func TestDraftPersist_rejectsMissingSnapshot(t *testing.T) {
	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-no-snap")
	draftID := "draft-persist-no-snap"
	minimal := `{"policy_context":{"wallet_address":"` + draftPersistTestWallet + `","wallet_type":"eoa"}}`
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, minimal)

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeCryptoPolicyPayloadInvalid)
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
