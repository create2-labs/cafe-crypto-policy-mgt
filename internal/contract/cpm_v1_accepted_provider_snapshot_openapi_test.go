package contract

import (
	"testing"
)

func TestCPMV1OpenAPI_AcceptedProviderSnapshotAssistDocumented(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths, ok := spec["paths"].(map[any]any)
	if !ok {
		t.Fatal("paths missing or invalid")
	}
	route, ok := paths["/accepted-provider-snapshots"].(map[any]any)
	if !ok {
		t.Fatal("openapi paths missing /accepted-provider-snapshots")
	}
	post, ok := route["post"].(map[any]any)
	if !ok {
		t.Fatal("POST /accepted-provider-snapshots missing")
	}
	if post["operationId"] != "postAcceptedProviderSnapshots" {
		t.Fatalf("operationId: %#v", post["operationId"])
	}

	components := spec["components"].(map[any]any)
	schemas := components["schemas"].(map[any]any)
	for _, name := range []string{
		"AcceptedProviderSnapshotRequest",
		"AcceptedProviderSnapshotResponse",
	} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("openapi schema missing %q", name)
		}
	}

	req := schemas["AcceptedProviderSnapshotRequest"].(map[any]any)
	required := asStringSlice(t, req["required"])
	for _, field := range []string{"crypto_policy_id", "solution_profile_ref", "chain_id"} {
		if !contains(required, field) {
			t.Fatalf("AcceptedProviderSnapshotRequest.required missing %q", field)
		}
	}
}
