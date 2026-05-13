package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

func TestValidateRouteInventory(t *testing.T) {
	if err := validateRouteInventory(routeInventory); err != nil {
		t.Fatalf("expected valid route inventory, got: %v", err)
	}
}

func TestClassifyRoute(t *testing.T) {
	if got := classifyRoute(http.MethodGet, "/healthz"); got != authz.RouteClassPublicHealth {
		t.Fatalf("expected healthz class %q, got %q", authz.RouteClassPublicHealth, got)
	}
	if got := classifyRoute(http.MethodGet, "/api/v1/policies/catalog"); got != authz.RouteClassAuthenticated {
		t.Fatalf("expected catalog class %q, got %q", authz.RouteClassAuthenticated, got)
	}
	if got := classifyRoute(http.MethodGet, "/does-not-exist"); got != authz.RouteClassDeprecatedDisabled {
		t.Fatalf("expected unknown route class %q, got %q", authz.RouteClassDeprecatedDisabled, got)
	}
	if got := classifyRoute(http.MethodPost, "/internal/policies/references/scan"); got != authz.RouteClassInternalService {
		t.Fatalf("expected internal service class %q, got %q", authz.RouteClassInternalService, got)
	}
}

func TestWithAuthenticationFailsWithoutValidationURLWhenRequired(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	_, err := withAuthentication(next, authConfig{Required: true})
	if err == nil {
		t.Fatal("expected middleware wiring to fail when validation URL is missing")
	}
}

func TestPrincipalInjectionOnlyOnSuccess(t *testing.T) {
	introspect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
	}))
	defer introspect.Close()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if p, ok := principalFromContext(r.Context()); !ok || p.UserID == "" {
			t.Fatalf("expected principal in context on successful auth, got ok=%v principal=%+v", ok, p)
		}
		w.WriteHeader(http.StatusOK)
	})
	handler, err := withAuthentication(next, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 1,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("withAuthentication: %v", err)
	}

	token, err := makeTokenEnvelope(map[string]any{
		"user_id": "u1",
		"email":   "u@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}

	// Failure path: malformed token should not inject principal and should not call next.
	called = false
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/policies/catalog", nil)
	req2.Header.Set("Authorization", "Bearer malformed")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on malformed token, got %d", rr2.Code)
	}
	if called {
		t.Fatal("did not expect next handler call on auth failure")
	}

	// Health route remains public even when auth is required.
	calledHealth := false
	nextHealth := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledHealth = true
		w.WriteHeader(http.StatusOK)
	})
	healthHandler, err := withAuthentication(nextHealth, authConfig{
		Required:             true,
		SessionValidationURL: introspect.URL,
	})
	if err != nil {
		t.Fatalf("withAuthentication (health): %v", err)
	}
	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	healthHandler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK || !calledHealth {
		t.Fatalf("expected public health route pass-through, code=%d called=%v", healthRec.Code, calledHealth)
	}
}

func TestContextLookupWithoutPrincipal(t *testing.T) {
	if _, ok := principalFromContext(context.Background()); ok {
		t.Fatal("expected no principal in empty context")
	}
}
