package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

const testAssessmentScanID = "705c9700-0000-4000-8000-000000000001"

func TestPoliciesAssessmentRequest_policyContextForbidden(t *testing.T) {
	h := newAssessmentTestHandler(t, assessmentHarness{})
	token := mustAssessmentToken(t)
	body := map[string]any{
		"scan_id":        testAssessmentScanID,
		"policy_context": map[string]any{"wallet_address": "0x1"},
		"selection_request": validAssessmentSelectionRequest(),
	}
	res := postAssessment(t, h, token, body)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", res.Code, res.Body.String())
	}
}

func TestPoliciesAssessmentRequest_walletScanAccepted(t *testing.T) {
	var published sync.Map
	h := newAssessmentTestHandler(t, assessmentHarness{
		scanAuthzAllowed: true,
		walletDetail:     validWalletScanDetailJSON(),
		natsPublish: func(_ context.Context, subject string, payload []byte) error {
			published.Store("subject", subject)
			published.Store("payload", append([]byte(nil), payload...))
			return nil
		},
	})
	token := mustAssessmentToken(t)
	res := postAssessment(t, h, token, validAssessmentBody())
	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 body=%s", res.Code, res.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["event_id"] == "" || out["correlation_id"] != testAssessmentScanID {
		t.Fatalf("unexpected body: %#v", out)
	}
	subj, _ := published.Load("subject")
	if subj != cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01 {
		t.Fatalf("subject = %v", subj)
	}
	raw, _ := published.Load("payload")
	var cmd cafenatsv01.PolicyAssessmentRequested
	if err := json.Unmarshal(raw.([]byte), &cmd); err != nil {
		t.Fatalf("event decode: %v", err)
	}
	if err := cmd.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(cmd.Payload.Observation.Payload.ChainIDs) == 0 {
		t.Fatalf("expected observation from Discovery detail")
	}
}

func TestPoliciesAssessmentRequest_unknownScanNotFound(t *testing.T) {
	h := newAssessmentTestHandler(t, assessmentHarness{
		scanAuthzStatus: http.StatusNotFound,
		natsPublish:     func(context.Context, string, []byte) error { return nil },
	})
	res := postAssessment(t, h, mustAssessmentToken(t), validAssessmentBody())
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 body=%s", res.Code, res.Body.String())
	}
}

func TestPoliciesAssessmentRequest_authzDenyNotFound(t *testing.T) {
	h := newAssessmentTestHandler(t, assessmentHarness{
		scanAuthzStatus: http.StatusForbidden,
		natsPublish:     func(context.Context, string, []byte) error { return nil },
	})
	res := postAssessment(t, h, mustAssessmentToken(t), validAssessmentBody())
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 body=%s", res.Code, res.Body.String())
	}
}

func TestPoliciesAssessmentRequest_tlsScanNotFound(t *testing.T) {
	detail := validWalletScanDetailJSON()
	var m map[string]any
	_ = json.Unmarshal(detail, &m)
	m["scan_family"] = "tls"
	raw, _ := json.Marshal(m)
	h := newAssessmentTestHandler(t, assessmentHarness{
		scanAuthzAllowed: true,
		walletDetail:     raw,
		natsPublish:      func(context.Context, string, []byte) error { return nil },
	})
	res := postAssessment(t, h, mustAssessmentToken(t), validAssessmentBody())
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 body=%s", res.Code, res.Body.String())
	}
}

func TestPoliciesAssessmentRequest_discoveryUnavailable(t *testing.T) {
	h := newAssessmentTestHandler(t, assessmentHarness{
		scanAuthzAllowed:   true,
		walletDetailStatus: http.StatusInternalServerError,
		natsPublish:        func(context.Context, string, []byte) error { return nil },
	})
	res := postAssessment(t, h, mustAssessmentToken(t), validAssessmentBody())
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 body=%s", res.Code, res.Body.String())
	}
}

func TestObservationPayloadFromDiscoveryWalletScanDetail(t *testing.T) {
	pl, err := api.ObservationPayloadFromDiscoveryWalletScanDetail(validWalletScanDetailJSON())
	if err != nil {
		t.Fatalf("ObservationPayloadFromDiscoveryWalletScanDetail: %v", err)
	}
	if pl.CurrentPQPosture != "hybrid" {
		t.Fatalf("posture = %q", pl.CurrentPQPosture)
	}
}

type assessmentHarness struct {
	scanAuthzAllowed   bool
	scanAuthzStatus    int
	walletDetail       []byte
	walletDetailStatus int
	natsPublish        func(context.Context, string, []byte) error
}

func newAssessmentTestHandler(t *testing.T, harness assessmentHarness) http.Handler {
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

	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	t.Cleanup(introspect.Close)

	scanAuthz := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("scan authz method = %s", r.Method)
		}
		if harness.scanAuthzStatus != 0 {
			w.WriteHeader(harness.scanAuthzStatus)
			return
		}
		if harness.scanAuthzAllowed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"allowed":true}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(scanAuthz.Close)

	detailStatus := harness.walletDetailStatus
	if detailStatus == 0 {
		detailStatus = http.StatusOK
	}
	detailBody := harness.walletDetail
	if detailBody == nil {
		detailBody = validWalletScanDetailJSON()
	}
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("discovery method = %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/discovery/v1/wallets/scans/") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(detailStatus)
		if detailStatus == http.StatusOK {
			_, _ = w.Write(detailBody)
		}
	}))
	t.Cleanup(discovery.Close)

	pub := harness.natsPublish
	if pub == nil {
		pub = func(context.Context, string, []byte) error { return nil }
	}

	h, err := handler("cafe-cpm", store, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ScanAuthorizationURL:        scanAuthz.URL,
		ScanAuthorizationTimeoutSec: 2,
		ClockSkewSec:                30,
		DiscoveryHTTPBaseURL:        discovery.URL,
		DiscoveryHTTPTimeoutSec:     3,
		AssessmentNATSPublish:       pub,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return h
}

func postAssessment(t *testing.T, h http.Handler, token string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, cpmroutes.PoliciesAssessmentRequest, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func mustAssessmentToken(t *testing.T) string {
	t.Helper()
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": "u-assessment",
		"email":   "assessment@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return token
}

func validAssessmentBody() map[string]any {
	return map[string]any{
		"scan_id":           testAssessmentScanID,
		"selection_request": validAssessmentSelectionRequest(),
	}
}

func validAssessmentSelectionRequest() map[string]any {
	return map[string]any{
		"target_posture":              string(vocabulary.PQPostureHybrid),
		"target_chain_ids":            []int64{1},
		"require_multichain":          false,
		"allow_new_wallet":            false,
		"address_continuity_required": true,
		"key_rotation_required":       true,
		"recovery_required":           true,
		"minimum_maturity":            1,
		"approval_mode":               "manual",
	}
}

func validWalletScanDetailJSON() []byte {
	body := map[string]any{
		"scan_id": testAssessmentScanID,
		"status":  "completed",
		"result": map[string]any{
			"target_address":     "0x742d35cc6634c0532925a3b844bc454e4438f44e",
			"chain_ids":          []int64{1},
			"wallet_type":        "eoa",
			"current_pq_posture": "hybrid",
			"algorithm":          "ECDSA-secp256k1",
			"scanned_at":         "2026-04-17T09:59:58Z",
		},
	}
	raw, _ := json.Marshal(body)
	return raw
}
