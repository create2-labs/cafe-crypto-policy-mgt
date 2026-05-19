package cpmroutes_test

import (
	"net/http"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestAuthenticatedRoutes_UseV1Base(t *testing.T) {
	t.Parallel()
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if route.Path == "" || route.Method == "" {
			t.Fatalf("empty method or path: %+v", route)
		}
		if route.Path != cpmroutes.Healthz && route.Path[:len(cpmroutes.V1Base)] != cpmroutes.V1Base {
			t.Fatalf("path %q must be under %q", route.Path, cpmroutes.V1Base)
		}
	}
}

func TestPoliciesDecisionsExplore_MethodPost(t *testing.T) {
	t.Parallel()
	found := false
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if route.Path == cpmroutes.PoliciesDecisionsExplore {
			if route.Method != http.MethodPost {
				t.Fatalf("decisions/explore method = %s, want POST", route.Method)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("PoliciesDecisionsExplore missing from AuthenticatedRoutes")
	}
}

func TestPoliciesAssessmentRequest_MethodPost(t *testing.T) {
	t.Parallel()
	found := false
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if route.Path == cpmroutes.PoliciesAssessmentRequest {
			if route.Method != http.MethodPost {
				t.Fatalf("assessment/request method = %s, want POST", route.Method)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("PoliciesAssessmentRequest missing from AuthenticatedRoutes")
	}
}
