package app

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	authCodeUnauthenticated       = "AUTH_UNAUTHENTICATED"
	authCodeValidationUnavailable = "AUTH_VALIDATION_UNAVAILABLE"
	authCodeScanIDMalformed       = "AUTHZ_SCAN_ID_MALFORMED"
	authCodeScanIDConflict        = "AUTHZ_SCAN_ID_CONFLICT"
	authCodeScanForbidden         = "AUTHZ_SCAN_FORBIDDEN"
	authCodeScanUnavailable       = "AUTHZ_SCAN_UNAVAILABLE"
	authCodeOwnerForbidden        = "AUTHZ_OWNER_FORBIDDEN"
	authCodePrincipalRequired     = "AUTHZ_PRINCIPAL_REQUIRED"
	authCodeInternalMisconfigured = "AUTH_INTERNAL_MISCONFIGURED"
	authCodeInternalForbidden     = "AUTH_INTERNAL_FORBIDDEN"
	authCodeOK                    = "OK"
)

const (
	authCategoryAuthn    = "authn"
	authCategoryScanAuth = "scan_authz"
	authCategoryOwner    = "owner_authz"
)

type authAuditEvent struct {
	Category  string
	Outcome   string
	Code      string
	RequestID string
	Route     string
	Method    string
	UserID    string
	TenantID  string
}

var requestIDAllowedPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type authAuditSink interface {
	RecordAuthEvent(event authAuditEvent)
}

type noopAuthAuditSink struct{}

func (noopAuthAuditSink) RecordAuthEvent(authAuditEvent) {}

type authMetrics interface {
	IncDecision(category string, outcome string, code string, route string)
}

type noopAuthMetrics struct{}

func (noopAuthMetrics) IncDecision(string, string, string, string) {}

type authDecisionCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newAuthDecisionCounter() *authDecisionCounter {
	return &authDecisionCounter{counts: map[string]int{}}
}

func (c *authDecisionCounter) IncDecision(category string, outcome string, code string, route string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := strings.Join([]string{category, outcome, code, route}, "|")
	c.counts[key]++
}

func (c *authDecisionCounter) Count(category string, outcome string, code string, route string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[strings.Join([]string{category, outcome, code, route}, "|")]
}

type authObservability struct {
	logger       *log.Logger
	metrics      authMetrics
	audit        authAuditSink
	nextRequest  atomic.Uint64
	requestIDGen func() string
}

func newAuthObservability() *authObservability {
	obs := &authObservability{
		logger:  log.Default(),
		metrics: noopAuthMetrics{},
		audit:   noopAuthAuditSink{},
	}
	obs.requestIDGen = func() string {
		seq := obs.nextRequest.Add(1)
		return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), seq)
	}
	return obs
}

func (o *authObservability) ensureRequestID(w http.ResponseWriter, r *http.Request) string {
	requestID, ok := sanitizeRequestID(r.Header.Get("X-Request-Id"))
	if !ok {
		requestID = o.requestIDGen()
	}
	r.Header.Set("X-Request-Id", requestID)
	w.Header().Set("X-Request-Id", requestID)
	return requestID
}

func sanitizeRequestID(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if !requestIDAllowedPattern.MatchString(value) {
		return "", false
	}
	return value, true
}

func (o *authObservability) writeAuthError(w http.ResponseWriter, r *http.Request, status int, code string, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	requestID := o.ensureRequestID(w, r)
	respondAuthError(w, status, authErrorPayload(code, message, details, requestID))
}

func authErrorPayload(code string, message string, details map[string]any, requestID string) map[string]any {
	if details == nil {
		details = map[string]any{}
	}
	return map[string]any{
		"code":       code,
		"message":    message,
		"details":    details,
		"request_id": requestID,
	}
}

func (o *authObservability) recordDecision(
	r *http.Request,
	requestID string,
	category string,
	outcome string,
	code string,
	reason string,
	userID string,
	tenantID string,
) {
	route := classifyRoute(r.Method, r.URL.Path)
	o.metrics.IncDecision(category, outcome, code, route)
	msg := []string{
		"event=auth_decision",
		"request_id=" + strconv.Quote(requestID),
		"method=" + strconv.Quote(r.Method),
		"route=" + strconv.Quote(route),
		"category=" + strconv.Quote(category),
		"outcome=" + strconv.Quote(outcome),
		"code=" + strconv.Quote(code),
		"reason=" + strconv.Quote(reason),
	}
	if strings.TrimSpace(userID) != "" {
		msg = append(msg, "user_id="+strconv.Quote(userID))
	}
	if strings.TrimSpace(tenantID) != "" {
		msg = append(msg, "tenant_id="+strconv.Quote(tenantID))
	}
	o.logger.Println(strings.Join(msg, " "))
}
