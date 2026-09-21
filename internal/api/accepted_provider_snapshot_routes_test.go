package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

func TestAcceptedProviderSnapshotAssist_buildsCanonicalNicetryMultichainSnapshot(t *testing.T) {
	mux := http.NewServeMux()
	store := testReadStore(t)
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	body := `{
		"crypto_policy_id": "cpm_pq_account_validation_v1",
		"solution_profile_ref": {
			"provider_id": "nicetry",
			"solution_profile_id": "nicetry.fors_c.erc4337.v0_1",
			"manifest_version": "2026-08"
		},
		"accepted_findings": ["requires_local_signer_state"]
	}`
	req := httptest.NewRequest(http.MethodPost, cpmroutes.AcceptedProviderSnapshots, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		CryptoPolicyID           string                          `json:"crypto_policy_id"`
		RequiredPosture          string                          `json:"required_posture"`
		SolutionProfileRef       policy.SolutionProfileRef       `json:"solution_profile_ref"`
		AcceptedProviderSnapshot policy.AcceptedProviderSnapshot `json:"accepted_provider_snapshot"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CryptoPolicyID != "cpm_pq_account_validation_v1" || resp.RequiredPosture != "hybrid" {
		t.Fatalf("cp fields: %+v", resp)
	}
	if len(resp.AcceptedProviderSnapshot.ChainSupportUsed) < 2 {
		t.Fatalf("want multi-chain, got %+v", resp.AcceptedProviderSnapshot.ChainSupportUsed)
	}
	// Wire must emit chain_support_used as array with string chain_ids.
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("raw: %v", err)
	}
	snap := raw["accepted_provider_snapshot"].(map[string]any)
	chains, ok := snap["chain_support_used"].([]any)
	if !ok || len(chains) < 2 {
		t.Fatalf("chain_support_used must be multi-entry array, got %#v", snap["chain_support_used"])
	}
	first := chains[0].(map[string]any)
	if _, ok := first["chain_id"].(string); !ok {
		t.Fatalf("chain_id must be string, got %#v", first["chain_id"])
	}
	findings := snap["accepted_findings"].([]any)
	if len(findings) != 2 {
		t.Fatalf("auto-merged findings: %#v", findings)
	}

	payload := map[string]any{
		"schema_version":             policy.CryptoPolicySchemaVersionV02,
		"crypto_policy_id":           resp.CryptoPolicyID,
		"required_posture":           resp.RequiredPosture,
		"user_constraints":           map[string]any{"allow_new_wallet": true, "address_continuity_required": false, "key_rotation_model": "per_userop"},
		"solution_profile_ref":       resp.SolutionProfileRef,
		"accepted_provider_snapshot": resp.AcceptedProviderSnapshot,
		"accepted_findings":          resp.AcceptedProviderSnapshot.AcceptedFindings,
	}
	if err := policy.ValidatePayloadForPersist(payload); err != nil {
		t.Fatalf("assist snapshot must be persist-gate clean (greenfield path): %v", err)
	}
}

func TestAcceptedProviderSnapshotAssist_validationMatrix(t *testing.T) {
	mux := http.NewServeMux()
	store := testReadStore(t)
	if err := RegisterReadRoutes(mux, store); err != nil {
		t.Fatalf("RegisterReadRoutes: %v", err)
	}

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantSubstr string
	}{
		{
			name:       "unknown_cp",
			body:       `{"crypto_policy_id":"missing","solution_profile_ref":{"provider_id":"nicetry","solution_profile_id":"nicetry.fors_c.erc4337.v0_1","manifest_version":"2026-08"}}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "unknown crypto_policy_id",
		},
		{
			name:       "legacy_chain_id_rejected",
			body:       `{"crypto_policy_id":"cpm_pq_account_validation_v1","solution_profile_ref":{"provider_id":"nicetry","solution_profile_id":"nicetry.fors_c.erc4337.v0_1","manifest_version":"2026-08"},"chain_id":"11155111"}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "unknown field chain_id",
		},
		{
			name:       "missing_ref",
			body:       `{"crypto_policy_id":"cpm_pq_account_validation_v1"}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "solution_profile_ref",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, cpmroutes.AcceptedProviderSnapshots, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.wantSubstr) {
				t.Fatalf("body=%s want substr %q", rec.Body.String(), tc.wantSubstr)
			}
		})
	}
}

func TestDecodeAcceptedProviderSnapshotRequest_noChainID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(bytes.NewBufferString(
		`{"crypto_policy_id":"cp","solution_profile_ref":{"provider_id":"nicetry","solution_profile_id":"p","manifest_version":"v"},"accepted_findings":["requires_bundler"]}`,
	)))
	got, err := decodeAcceptedProviderSnapshotRequest(req)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.CryptoPolicyID != "cp" {
		t.Fatalf("cp=%q", got.CryptoPolicyID)
	}
	if len(got.AcceptedFindings) != 1 || got.AcceptedFindings[0] != provider.FindingCodeRequiresBundler {
		t.Fatalf("findings=%v", got.AcceptedFindings)
	}
}

func TestMapSnapshotAssistError(t *testing.T) {
	status, msg := mapSnapshotAssistError(policy.ErrSnapshotAssistChainNotSupported)
	if status != http.StatusUnprocessableEntity || msg == "" {
		t.Fatalf("got %d %q", status, msg)
	}
	status, msg = mapSnapshotAssistError(errors.New("other"))
	if status != http.StatusBadRequest {
		t.Fatalf("got %d %q", status, msg)
	}
}
