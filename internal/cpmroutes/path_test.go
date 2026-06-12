package cpmroutes_test

import (
	"net/http"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestPathMatches_draftPersist(t *testing.T) {
	t.Parallel()
	pattern := cpmroutes.DraftPersist
	actual := cpmroutes.DraftPersistPath("draft-123")
	if !cpmroutes.PathMatches(pattern, actual) {
		t.Fatalf("expected %q to match %q", actual, pattern)
	}
	if cpmroutes.PathMatches(pattern, cpmroutes.V1Base+"/drafts/draft-123/persist/extra") {
		t.Fatal("unexpected match for extra path segment")
	}
}

func TestDraftPersistPath_underV1Base(t *testing.T) {
	t.Parallel()
	got := cpmroutes.DraftPersistPath("draft-xyz")
	if got != cpmroutes.V1Base+"/drafts/draft-xyz/persist" {
		t.Fatalf("got %q", got)
	}
}

func TestPathMatches_authenticatedDraftPersistRoute(t *testing.T) {
	t.Parallel()
	found := false
	for _, route := range cpmroutes.AuthenticatedRoutes() {
		if route.Path == cpmroutes.DraftPersist {
			if route.Method != http.MethodPost {
				t.Fatalf("draft persist method = %s, want POST", route.Method)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("DraftPersist missing from AuthenticatedRoutes")
	}
}
