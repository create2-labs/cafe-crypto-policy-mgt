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
		class := classifyRoute(r.Method, r.URL.Path)
		if class == authz.RouteClassPublicHealth {
			next.ServeHTTP(w, r)
			return
		}
		if class == authz.RouteClassDeprecatedDisabled {
			respondAuthError(w, http.StatusNotFound, authz.APIError{
				Code:    "AUTH_ROUTE_NOT_ENABLED",
				Message: "Route is not enabled.",
			})
			return
		}
		principal, authErr, status := authenticateBearerToken(r, cfg)
		if authErr.Code != "" {
			respondAuthError(w, status, authErr)
			return
		}
		routeScanIDs, scanErr, scanStatus := extractScanIDsForAuthorization(r)
		if scanErr.Code != "" {
			respondAuthError(w, scanStatus, scanErr)
			return
		}
		for _, routeScanID := range routeScanIDs {
			if scanAuthErr, scanAuthStatus := authorizeScanAccess(r.Context(), principal, routeScanID, cfg, r.Header.Get("X-Request-Id")); scanAuthErr.Code != "" {
				respondAuthError(w, scanAuthStatus, scanAuthErr)
				return
			}
		}
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

func authenticateBearerToken(r *http.Request, cfg authConfig) (authz.Principal, authz.APIError, int) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_MISSING_BEARER",
			Message: "Missing or invalid Authorization bearer token.",
		}, http.StatusUnauthorized
	}
	rawToken := strings.TrimSpace(header[len("Bearer "):])
	if rawToken == "" {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_EMPTY_TOKEN",
			Message: "Bearer token is empty.",
		}, http.StatusUnauthorized
	}
	claims, err := parseAndValidateDiscoveryToken(rawToken, cfg.ClockSkewSec)
	if err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_INVALID_TOKEN",
			Message: "Bearer token validation failed.",
			Details: map[string]any{"error": errorString(err)},
		}, http.StatusUnauthorized
	}
	validation, err := validateTokenWithDiscovery(r.Context(), rawToken, cfg, r.Header.Get("X-Request-Id"))
	if err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_VALIDATION_UNAVAILABLE",
			Message: "Session validation service unavailable.",
			Details: map[string]any{"error": errorString(err)},
		}, http.StatusServiceUnavailable
	}
	if !validation.Accepted {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_INVALID_TOKEN",
			Message: "Bearer token validation failed.",
			Details: map[string]any{"error": validation.ErrorMessage},
		}, http.StatusUnauthorized
	}
	if validation.Claims != nil {
		claims = validation.Claims
	}
	userID, _ := claims["user_id"].(string)
	subject := userID
	tenantID, _ := claims["tenant_id"].(string)
	if strings.TrimSpace(userID) == "" {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_INVALID_CLAIMS",
			Message: "Token is missing required claims.",
		}, http.StatusUnauthorized
	}
	principal := authz.Principal{
		UserID:   userID,
		Subject:  subject,
		TenantID: tenantID,
		Claims:   claims,
	}
	if err := principal.Validate(); err != nil {
		return authz.Principal{}, authz.APIError{
			Code:    "AUTH_INVALID_PRINCIPAL",
			Message: "Derived principal is invalid.",
			Details: map[string]any{"error": err.Error()},
		}, http.StatusUnauthorized
	}
	return principal, authz.APIError{}, http.StatusOK
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

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func respondAuthError(w http.ResponseWriter, status int, payload authz.APIError) {
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
			Code:    "AUTHZ_SCAN_ID_MALFORMED",
			Message: "scanId is malformed.",
		}, http.StatusBadRequest
	}
	if len(scanIDs) == 0 {
		return nil, authz.APIError{}, http.StatusOK
	}
	base := scanIDs[0]
	for _, scanID := range scanIDs[1:] {
		if scanID != base {
			return nil, authz.APIError{
				Code:    "AUTHZ_SCAN_ID_CONFLICT",
				Message: "Request carries conflicting scanId values.",
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
	if !add(payload["scanId"]) {
		return nil, true
	}
	if !add(payload["scan_id"]) {
		return nil, true
	}
	if draft, ok := payload["draft"].(map[string]any); ok {
		if !add(draft["scanId"]) {
			return nil, true
		}
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
) (authz.APIError, int) {
	if strings.TrimSpace(scanID) == "" {
		return authz.APIError{
			Code:    "AUTHZ_SCAN_ID_MALFORMED",
			Message: "scanId is malformed.",
		}, http.StatusBadRequest
	}
	if strings.TrimSpace(cfg.ScanAuthorizationURL) == "" {
		return authz.APIError{
			Code:    "AUTHZ_UNAVAILABLE",
			Message: "Scan authorization service unavailable.",
			Details: map[string]any{"error": "scan authorization URL not configured"},
		}, http.StatusServiceUnavailable
	}
	timeoutSec := cfg.ScanAuthorizationTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 3
	}
	endpoint := strings.TrimSuffix(cfg.ScanAuthorizationURL, "/") + "/" + url.PathEscape(scanID) + "/can-read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return authz.APIError{
			Code:    "AUTHZ_UNAVAILABLE",
			Message: "Scan authorization service unavailable.",
			Details: map[string]any{"error": err.Error()},
		}, http.StatusServiceUnavailable
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
			Code:    "AUTHZ_UNAVAILABLE",
			Message: "Scan authorization service unavailable.",
			Details: map[string]any{"error": err.Error()},
		}, http.StatusServiceUnavailable
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		var parsed scanAuthorizationResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return authz.APIError{
				Code:    "AUTHZ_UNAVAILABLE",
				Message: "Scan authorization service unavailable.",
				Details: map[string]any{"error": "invalid authorization response"},
			}, http.StatusServiceUnavailable
		}
		if !parsed.Allowed {
			return authz.APIError{
				Code:    "AUTHZ_SCAN_FORBIDDEN",
				Message: "Scan access denied.",
			}, http.StatusForbidden
		}
		return authz.APIError{}, http.StatusOK
	case http.StatusForbidden, http.StatusNotFound:
		return authz.APIError{
			Code:    "AUTHZ_SCAN_FORBIDDEN",
			Message: "Scan access denied.",
		}, http.StatusForbidden
	case http.StatusUnauthorized:
		return authz.APIError{
			Code:    "AUTHZ_UNAVAILABLE",
			Message: "Scan authorization service unavailable.",
		}, http.StatusServiceUnavailable
	default:
		if resp.StatusCode >= 500 {
			return authz.APIError{
				Code:    "AUTHZ_UNAVAILABLE",
				Message: "Scan authorization service unavailable.",
			}, http.StatusServiceUnavailable
		}
		return authz.APIError{
			Code:    "AUTHZ_UNAVAILABLE",
			Message: "Scan authorization service unavailable.",
		}, http.StatusServiceUnavailable
	}
}
