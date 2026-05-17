package app

import (
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

// TestRouteInventoryAlignedWithCpmRoutes ensures auth routeInventory paths match cpmroutes constants (PR11c).
func TestRouteInventoryAlignedWithCpmRoutes(t *testing.T) {
	t.Parallel()
	wantAuth := make(map[string]struct{}, len(cpmroutes.AuthenticatedRoutes()))
	for _, ar := range cpmroutes.AuthenticatedRoutes() {
		wantAuth[ar.Method+" "+ar.Path] = struct{}{}
	}
	gotAuth := make(map[string]struct{})
	for _, route := range routeInventory {
		if route.Class != authz.RouteClassAuthenticated {
			continue
		}
		gotAuth[route.Method+" "+route.Path] = struct{}{}
	}
	if len(gotAuth) != len(wantAuth) {
		t.Fatalf("authenticated route count: inventory=%d cpmroutes=%d", len(gotAuth), len(wantAuth))
	}
	for key := range wantAuth {
		if _, ok := gotAuth[key]; !ok {
			t.Fatalf("route inventory missing authenticated route %q", key)
		}
	}
}
