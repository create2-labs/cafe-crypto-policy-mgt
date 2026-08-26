package cpmroutes_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestPathMatches_paramSegment(t *testing.T) {
	t.Parallel()
	pattern := cpmroutes.CryptoPolicyByID
	actual := cpmroutes.V1Base + "/crypto-policies/cp-123"
	if !cpmroutes.PathMatches(pattern, actual) {
		t.Fatalf("expected %q to match %q", actual, pattern)
	}
}

func TestAuthenticatedRoutes_noDraftsAfterRDP5(t *testing.T) {
	t.Parallel()
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if strings.Contains(route.Path, "/drafts") {
			t.Fatalf("draft route must not be authenticated after RD-P5: %s %s", route.Method, route.Path)
		}
	}
	foundPolicies := false
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if route.Path == cpmroutes.Policies && route.Method == http.MethodPost {
			foundPolicies = true
		}
	}
	if !foundPolicies {
		t.Fatal("POST /policies must remain authenticated")
	}
}
