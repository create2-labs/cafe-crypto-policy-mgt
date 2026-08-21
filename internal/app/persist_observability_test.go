//go:build dev

package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
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
	recordPersistUserConstraintsIncompatible(req, "draft-1", "cpm_pq_account_validation_v1", provider.FindingCodeContinuity)

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

func TestDraftPersist_coucheBKOEmitsRuntimeSignal(t *testing.T) {
	metrics := &testPersistMetrics{}
	logger := &testPersistLogger{}
	restore := setPersistObservabilityForTest(persistObservability{logger: logger, metrics: metrics})
	defer restore()

	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-p11b-signal")
	draftID := "draft-persist-p11b-signal"
	createDraftPersistBoundDraftWithPayload(t, h, token, draftID, draftPersistCoucheBKOPayloadJSON())

	issued := time.Now().UTC().Truncate(time.Second)
	expires := issued.Add(10 * time.Minute)
	message, signature := buildDraftPersistSignedRequest(t, draftID, issued, expires)
	body := `{"wallet_address":"` + draftPersistTestWallet + `","chain_id":1,"scan_id":"` + draftPersistTestScanID + `","signed_message":` + jsonString(message) + `,"signature":"` + signature + `"}`

	req := httptest.NewRequest(http.MethodPost, cpmroutes.DraftPersistPath(draftID), strings.NewReader(body))
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
	if !strings.Contains(logger.lines[0], `finding_code="`+provider.FindingCodeContinuity+`"`) {
		t.Fatalf("want continuity finding_code, got %s", logger.lines[0])
	}
}

func TestDraftPersist_unpinnedDoesNotEmitCoucheBSignal(t *testing.T) {
	metrics := &testPersistMetrics{}
	restore := setPersistObservabilityForTest(persistObservability{metrics: metrics})
	defer restore()

	h := newDraftPersistTestHandler(t)
	token := mustToken(t, "draft-persist-p11b-unpin")
	draftID := "draft-persist-p11b-unpin"
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
	if metrics.increments != 0 {
		t.Fatalf("unpinned must not emit couche B signal, got %d", metrics.increments)
	}
}
