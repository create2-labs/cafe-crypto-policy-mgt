//go:build dev

package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func setPersistObservabilityForTest(obs persistObservability) func() {
	prev := persistObs
	persistObs = obs
	return func() { persistObs = prev }
}

type testPersistMetrics struct {
	increments int
}

func (m *testPersistMetrics) IncPersistUserConstraintsIncompatible() {
	m.increments++
}

type testPersistLogger struct {
	lines []string
}

func (l *testPersistLogger) Println(v ...any) {
	parts := make([]string, 0, len(v))
	for _, item := range v {
		parts = append(parts, fmt.Sprint(item))
	}
	l.lines = append(l.lines, strings.Join(parts, " "))
}

func TestPersistObservability_userConstraintsIncompatibleDistinctFromExplore(t *testing.T) {
	metrics := &testPersistMetrics{}
	logger := &testPersistLogger{}
	restore := setPersistObservabilityForTest(persistObservability{logger: logger, metrics: metrics})
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-Request-Id", "req-p11b")
	recordPersistUserConstraintsIncompatible(req, "cpm_pq_account_validation_v1", provider.FindingCodeContinuity)

	if metrics.increments != 1 {
		t.Fatalf("want 1 persist metric increment, got %d", metrics.increments)
	}
	if len(logger.lines) != 1 {
		t.Fatalf("want 1 log line, got %d (%v)", len(logger.lines), logger.lines)
	}
	line := logger.lines[0]
	if !strings.Contains(line, `event="cpm.persist.user_constraints_incompatible"`) {
		t.Fatalf("missing persist event: %s", line)
	}
	if !strings.Contains(line, `adr_signal="runtime.no_provider_after_user_constraints"`) {
		t.Fatalf("missing adr_signal: %s", line)
	}
	if strings.Contains(line, "cpm.explore.no_deployable_candidate") || strings.Contains(line, "runtime.no_scan_compatible") {
		t.Fatalf("persist signal must be distinct from explore: %s", line)
	}
	if !strings.Contains(line, `finding_code="`+provider.FindingCodeContinuity+`"`) {
		t.Fatalf("missing finding_code: %s", line)
	}
}

func TestPolicyPersist_coucheBKOEmitsRuntimeSignal(t *testing.T) {
	metrics := &testPersistMetrics{}
	logger := &testPersistLogger{}
	restore := setPersistObservabilityForTest(persistObservability{logger: logger, metrics: metrics})
	defer restore()

	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-p11b-signal")
	payloadJSON, _ := policyPersistHashedPayload(t)
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	uc := payload["user_constraints"].(map[string]any)
	uc["address_continuity_required"] = true
	uc["allow_new_wallet"] = false
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := payloadhash.DigestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, raw, digest, now, now.Add(walletauth.MaxValidityWindow))

	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "p11b-couche-b")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeProviderUserConstraintsIncompatible)

	if metrics.increments != 1 {
		t.Fatalf("want 1 persist runtime signal, got %d", metrics.increments)
	}
	if len(logger.lines) != 1 || !strings.Contains(logger.lines[0], `adr_signal="runtime.no_provider_after_user_constraints"`) {
		t.Fatalf("want couche B runtime log, got %v", logger.lines)
	}
}

func TestPolicyPersist_unpinnedDoesNotEmitCoucheBSignal(t *testing.T) {
	metrics := &testPersistMetrics{}
	restore := setPersistObservabilityForTest(persistObservability{metrics: metrics})
	defer restore()

	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "policy-persist-p11b-unpin")
	payloadJSON, _ := policyPersistHashedPayload(t)
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	snap := payload["accepted_provider_snapshot"].(map[string]any)
	snap["references"] = []any{
		map[string]any{"kind": "source_repo", "url": "https://example.com", "commit": "unpinned_pending_fixture"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("status = %d want 400 body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, persistCodeProviderRefsUnpinned)
	if metrics.increments != 0 {
		t.Fatalf("unpinned must not emit couche B signal, got %d", metrics.increments)
	}
}
