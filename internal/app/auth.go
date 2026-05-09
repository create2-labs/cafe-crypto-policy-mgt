package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

type authConfig struct {
	Required                      bool
	SessionValidationURL          string
	SessionValidationTimeoutSec   int
	SessionValidationServiceToken string
	ScanAuthorizationURL          string
	ScanAuthorizationTimeoutSec   int
	ScanAuthorizationServiceToken string
	ClockSkewSec                  int
	Observability                 *authObservability
}

type routeSpec struct {
	Method string
	Path   string
	Class  string
}

var routeInventory = []routeSpec{
	{Method: http.MethodGet, Path: "/healthz", Class: authz.RouteClassPublicHealth},
	{Method: http.MethodGet, Path: "/api/v1/policies/catalog", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodGet, Path: "/api/v1/policies/templates", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodGet, Path: "/api/v1/policies/instances", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodPost, Path: "/api/v1/policies/decisions/explore", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodPost, Path: "/api/v1/cpm/drafts", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodGet, Path: "/api/v1/cpm/drafts", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodPost, Path: "/api/v1/cpm/policies", Class: authz.RouteClassAuthenticated},
	{Method: http.MethodGet, Path: "/api/v1/cpm/policies", Class: authz.RouteClassAuthenticated},
}

type principalContextKey struct{}

func principalFromContext(ctx context.Context) (authz.Principal, bool) {
	value := ctx.Value(principalContextKey{})
	p, ok := value.(authz.Principal)
	return p, ok
}

func withAuthentication(next http.Handler, cfg authConfig) (http.Handler, error) {
	if err := validateRouteInventory(routeInventory); err != nil {
		return nil, err
	}
	if !cfg.Required {
		return next, nil
	}
	if strings.TrimSpace(cfg.SessionValidationURL) == "" {
		return nil, fmt.Errorf("auth required but session validation URL is not configured")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		obs := cfg.Observability
		if obs == nil {
			obs = newAuthObservability()
		}
		requestID := obs.ensureRequestID(w, r)
		class := classifyRoute(r.Method, r.URL.Path)
		if class == authz.RouteClassPublicHealth {
			next.ServeHTTP(w, r)
			return
		}
		if class == authz.RouteClassDeprecatedDisabled {
			obs.writeAuthError(w, r, http.StatusNotFound, "AUTH_ROUTE_NOT_ENABLED", "route is not enabled", map[string]any{})
			return
		}
		principal, authErr, status, reason := authenticateBearerToken(r, cfg, requestID)
		if authErr.Code != "" {
			outcome := "denied"
			if status == http.StatusServiceUnavailable {
				outcome = "unavailable"
			}
			obs.recordDecision(r, requestID, authCategoryAuthn, outcome, authErr.Code, reason, "", "")
			obs.writeAuthError(w, r, status, authErr.Code, authErr.Message, authErr.Details)
			return
		}
		routeScanIDs, scanErr, scanStatus := extractScanIDsForAuthorization(r)
		if scanErr.Code != "" {
			obs.recordDecision(r, requestID, authCategoryScanAuth, "malformed", scanErr.Code, "scan_id_malformed", principal.UserID, principal.TenantID)
			obs.writeAuthError(w, r, scanStatus, scanErr.Code, scanErr.Message, scanErr.Details)
			return
		}
		for _, routeScanID := range routeScanIDs {
			if scanAuthErr, scanAuthStatus, scanReason := authorizeScanAccess(r.Context(), principal, routeScanID, cfg, requestID); scanAuthErr.Code != "" {
				outcome := "denied"
				if scanAuthStatus == http.StatusServiceUnavailable {
					outcome = "unavailable"
				}
				obs.recordDecision(r, requestID, authCategoryScanAuth, outcome, scanAuthErr.Code, scanReason, principal.UserID, principal.TenantID)
				if scanAuthErr.Code == authCodeScanForbidden {
					obs.audit.RecordAuthEvent(authAuditEvent{
						Category:  authCategoryScanAuth,
						Outcome:   "denied",
						Code:      authCodeScanForbidden,
						RequestID: requestID,
						Route:     class,
						Method:    r.Method,
						UserID:    principal.UserID,
						TenantID:  principal.TenantID,
					})
				}
				obs.writeAuthError(w, r, scanAuthStatus, scanAuthErr.Code, scanAuthErr.Message, scanAuthErr.Details)
				return
			}
			obs.recordDecision(r, requestID, authCategoryScanAuth, "allowed", authCodeOK, "scan_access_allowed", principal.UserID, principal.TenantID)
		}
		obs.recordDecision(r, requestID, authCategoryAuthn, "allowed", authCodeOK, "session_validated", principal.UserID, principal.TenantID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal)))
	}), nil
}

func validateRouteInventory(routes []routeSpec) error {
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if route.Method == "" || route.Path == "" {
			return fmt.Errorf("route inventory has empty method/path: %+v", route)
		}
		if !authz.IsValidRouteClass(route.Class) {
			return fmt.Errorf("route inventory has invalid class %q for %s %s", route.Class, route.Method, route.Path)
		}
		key := route.Method + " " + route.Path
		if _, exists := seen[key]; exists {
			return fmt.Errorf("route inventory has duplicate route: %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func classifyRoute(method string, path string) string {
	for _, route := range routeInventory {
		if route.Method == method && route.Path == path {
			return route.Class
		}
	}
	return authz.RouteClassDeprecatedDisabled
}

func authenticateBearerToken(r *http.Request, cfg authConfig, requestID string) (authz.Principal, authz.APIError, int, string) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "missing_or_malformed_authorization_header"},
		}, http.StatusUnauthorized, "missing_or_malformed_authorization_header"
	}
	rawToken := strings.TrimSpace(header[len("Bearer "):])
	if rawToken == "" {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "empty_bearer_token"},
		}, http.StatusUnauthorized, "empty_bearer_token"
	}
	claims, err := parseAndValidateDiscoveryToken(rawToken, cfg.ClockSkewSec)
	if err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "malformed_or_expired_session_token"},
		}, http.StatusUnauthorized, "malformed_or_expired_session_token"
	}
	validation, err := validateTokenWithDiscovery(r.Context(), rawToken, cfg, requestID)
	if err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeValidationUnavailable,
			Message: "authentication validation unavailable",
			Details: map[string]any{"reason": "session_validation_unavailable"},
		}, http.StatusServiceUnavailable, "session_validation_unavailable"
	}
	if !validation.Accepted {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "session_validation_rejected"},
		}, http.StatusUnauthorized, "session_validation_rejected"
	}
	if validation.Claims != nil {
		claims = validation.Claims
	}
	userID, _ := claims["user_id"].(string)
	subject := userID
	tenantID, _ := claims["tenant_id"].(string)
	if strings.TrimSpace(userID) == "" {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "missing_user_id_claim"},
		}, http.StatusUnauthorized, "missing_user_id_claim"
	}
	principal := authz.Principal{
		UserID:   userID,
		Subject:  subject,
		TenantID: tenantID,
		Claims:   claims,
	}
	if err := principal.Validate(); err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    authCodeUnauthenticated,
			Message: "authentication required",
			Details: map[string]any{"reason": "invalid_principal"},
		}, http.StatusUnauthorized, "invalid_principal"
	}
	return principal, authz.APIError{}, http.StatusOK, "session_validated"
}

func parseAndValidateDiscoveryToken(rawToken string, clockSkewSec int) (map[string]any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil {
		return nil, fmt.Errorf("token decode: %w", err)
	}

	var envelope struct {
		Payload    string `json:"payload"`
		Signatures []struct {
			Protected string `json:"protected"`
			Signature string `json:"signature"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("token json: %w", err)
	}
	if envelope.Payload == "" || len(envelope.Signatures) < 2 {
		return nil, fmt.Errorf("missing payload or signatures")
	}

	algorithms := map[string]bool{"EdDSA": false, "ML-DSA-65": false}
	for _, signature := range envelope.Signatures {
		if signature.Protected == "" || signature.Signature == "" {
			return nil, fmt.Errorf("signature entry missing protected/signature")
		}
		protected, err := base64.RawURLEncoding.DecodeString(signature.Protected)
		if err != nil {
			return nil, fmt.Errorf("protected header decode: %w", err)
		}
		var header map[string]any
		if err := json.Unmarshal(protected, &header); err != nil {
			return nil, fmt.Errorf("protected header parse: %w", err)
		}
		if alg, _ := header["alg"].(string); alg != "" {
			if _, ok := algorithms[alg]; ok {
				algorithms[alg] = true
			}
		}
	}
	if !algorithms["EdDSA"] || !algorithms["ML-DSA-65"] {
		return nil, fmt.Errorf("missing required hybrid algorithms")
	}
	claimBytes, err := base64.RawURLEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	if err := validateDiscoveryClaims(claims, clockSkewSec); err != nil {
		return nil, err
	}
	if _, ok := claims["email"].(string); !ok {
		return nil, fmt.Errorf("missing email claim")
	}
	return claims, nil
}

func validateDiscoveryClaims(claims map[string]any, clockSkewSec int) error {
	now := time.Now().Unix()
	leeway := int64(clockSkewSec)
	exp, err := claimAsUnix(claims["exp"])
	if err != nil {
		return fmt.Errorf("invalid exp: %w", err)
	}
	if now > exp+leeway {
		return fmt.Errorf("token expired")
	}
	return nil
}

func claimAsUnix(value any) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported claim type %T", value)
	}
}

type discoveryValidationResponse struct {
	Accepted     bool           `json:"accepted"`
	ErrorMessage string         `json:"error,omitempty"`
	Claims       map[string]any `json:"claims,omitempty"`
}

type scanAuthorizationResponse struct {
	Allowed bool `json:"allowed"`
}

func validateTokenWithDiscovery(
	ctx context.Context,
	token string,
	cfg authConfig,
	requestID string,
) (discoveryValidationResponse, error) {
	timeout := time.Duration(cfg.SessionValidationTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.SessionValidationURL, strings.NewReader(string(body)))
	if err != nil {
		return discoveryValidationResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if cfg.SessionValidationServiceToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.SessionValidationServiceToken)
	}
	// TODO(auth-02): replace static service token header with first-class service identity mechanism (mTLS or signed service JWT).

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return discoveryValidationResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return discoveryValidationResponse{Accepted: false, ErrorMessage: fmt.Sprintf("discovery rejected token (%d)", resp.StatusCode)}, nil
	}
	if resp.StatusCode >= 500 {
		raw, _ := io.ReadAll(resp.Body)
		return discoveryValidationResponse{}, fmt.Errorf("session validation failed with status %d: %s", resp.StatusCode, string(raw))
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return discoveryValidationResponse{}, fmt.Errorf("unexpected session validation status %d: %s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) == 0 {
		return discoveryValidationResponse{Accepted: true}, nil
	}
	var parsed discoveryValidationResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return discoveryValidationResponse{Accepted: true}, nil
	}
	// Backward compatibility: if endpoint replies with claims but no explicit accepted flag, treat 200 as accepted.
	if !parsed.Accepted && parsed.ErrorMessage == "" && parsed.Claims != nil {
		parsed.Accepted = true
	}
	if !parsed.Accepted && parsed.ErrorMessage == "" {
		parsed.Accepted = true
	}
	return parsed, nil
}

func respondAuthError(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func extractScanIDsForAuthorization(r *http.Request) ([]string, authz.APIError, int) {
	if r == nil || r.Body == nil {
		return nil, authz.APIError{}, http.StatusOK
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return nil, authz.APIError{}, http.StatusOK
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, authz.APIError{
			Code:    "AUTHZ_SCAN_PAYLOAD_READ_FAILED",
			Message: "Could not read request payload for scan authorization.",
		}, http.StatusBadRequest
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(body) == 0 {
		return nil, authz.APIError{}, http.StatusOK
	}
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, authz.APIError{}, http.StatusOK
	}
	scanIDs, malformed := collectScanIDs(payload)
	if malformed {
		return nil, authz.APIError{
			Code:    authCodeScanIDMalformed,
			Message: "scan_id is malformed",
			Details: map[string]any{"reason": "scan_id_malformed"},
		}, http.StatusBadRequest
	}
	if len(scanIDs) == 0 {
		return nil, authz.APIError{}, http.StatusOK
	}
	base := scanIDs[0]
	for _, scanID := range scanIDs[1:] {
		if scanID != base {
			return nil, authz.APIError{
				Code:    authCodeScanIDConflict,
				Message: "scan_id values conflict",
				Details: map[string]any{"reason": "scan_id_conflict"},
			}, http.StatusBadRequest
		}
	}
	return []string{base}, authz.APIError{}, http.StatusOK
}

func collectScanIDs(payload map[string]any) ([]string, bool) {
	out := make([]string, 0, 2)
	add := func(raw any) bool {
		if raw == nil {
			return true
		}
		value, ok := raw.(string)
		if !ok {
			return false
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return false
		}
		out = append(out, value)
		return true
	}
	// Canonical JSON field for scan binding: `scan_id` (top-level and under `draft`).
	if !add(payload["scan_id"]) {
		return nil, true
	}
	if draft, ok := payload["draft"].(map[string]any); ok {
		if !add(draft["scan_id"]) {
			return nil, true
		}
	}
	return out, false
}

func authorizeScanAccess(
	ctx context.Context,
	principal authz.Principal,
	scanID string,
	cfg authConfig,
	requestID string,
) (authz.APIError, int, string) {
	if strings.TrimSpace(scanID) == "" {
		return authz.APIError{
			Code:    authCodeScanIDMalformed,
			Message: "scan_id is malformed",
			Details: map[string]any{"reason": "scan_id_malformed"},
		}, http.StatusBadRequest, "scan_id_malformed"
	}
	if strings.TrimSpace(cfg.ScanAuthorizationURL) == "" {
		return authz.APIError{
			Code:    authCodeScanUnavailable,
			Message: "scan authorization unavailable",
			Details: map[string]any{"reason": "scan_authorization_url_not_configured"},
		}, http.StatusServiceUnavailable, "scan_authorization_url_not_configured"
	}
	timeoutSec := cfg.ScanAuthorizationTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 3
	}
	endpoint := strings.TrimSuffix(cfg.ScanAuthorizationURL, "/") + "/" + url.PathEscape(scanID) + "/can-read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return authz.APIError{
			Code:    authCodeScanUnavailable,
			Message: "scan authorization unavailable",
			Details: map[string]any{"reason": "scan_authorization_request_build_failed"},
		}, http.StatusServiceUnavailable, "scan_authorization_request_build_failed"
	}
	req.Header.Set("X-User-Id", principal.UserID)
	if principal.TenantID != "" {
		req.Header.Set("X-Tenant-Id", principal.TenantID)
	}
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if cfg.ScanAuthorizationServiceToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.ScanAuthorizationServiceToken)
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return authz.APIError{
			Code:    authCodeScanUnavailable,
			Message: "scan authorization unavailable",
			Details: map[string]any{"reason": "scan_authorization_request_failed"},
		}, http.StatusServiceUnavailable, "scan_authorization_request_failed"
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		var parsed scanAuthorizationResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return authz.APIError{
				Code:    authCodeScanUnavailable,
				Message: "scan authorization unavailable",
				Details: map[string]any{"reason": "scan_authorization_invalid_response"},
			}, http.StatusServiceUnavailable, "scan_authorization_invalid_response"
		}
		if !parsed.Allowed {
			return authz.APIError{
				Code:    authCodeScanForbidden,
				Message: "scan access denied",
				Details: map[string]any{"reason": "scan_authorization_denied"},
			}, http.StatusForbidden, "scan_authorization_denied"
		}
		return authz.APIError{}, http.StatusOK, "scan_access_allowed"
	case http.StatusForbidden, http.StatusNotFound:
		return authz.APIError{
			Code:    authCodeScanForbidden,
			Message: "scan access denied",
			Details: map[string]any{"reason": "scan_authorization_denied"},
		}, http.StatusForbidden, "scan_authorization_denied"
	case http.StatusUnauthorized:
		return authz.APIError{
			Code:    authCodeScanUnavailable,
			Message: "scan authorization unavailable",
			Details: map[string]any{"reason": "scan_authorization_upstream_unauthorized"},
		}, http.StatusServiceUnavailable, "scan_authorization_upstream_unauthorized"
	default:
		if resp.StatusCode >= 500 {
			return authz.APIError{
				Code:    authCodeScanUnavailable,
				Message: "scan authorization unavailable",
				Details: map[string]any{"reason": "scan_authorization_upstream_5xx"},
			}, http.StatusServiceUnavailable, "scan_authorization_upstream_5xx"
		}
		return authz.APIError{
			Code:    authCodeScanUnavailable,
			Message: "scan authorization unavailable",
			Details: map[string]any{"reason": "scan_authorization_unexpected_status"},
		}, http.StatusServiceUnavailable, "scan_authorization_unexpected_status"
	}
}
