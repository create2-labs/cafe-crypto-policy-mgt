package contract

import "testing"

func TestCPMV1OpenAPI_CryptoPolicyCompatibleNetworksDocumented(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths, ok := spec["paths"].(map[any]any)
	if !ok {
		t.Fatal("paths missing or invalid")
	}
	components, ok := spec["components"].(map[any]any)
	if !ok {
		t.Fatal("components missing")
	}
	schemas, ok := components["schemas"].(map[any]any)
	if !ok {
		t.Fatal("components.schemas missing")
	}

	for _, name := range []string{
		"CompatibleNetwork",
		"CryptoPolicyCatalogItem",
		"CryptoPolicyListResponse",
	} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("openapi schema missing %q", name)
		}
	}

	item := schemas["CryptoPolicyCatalogItem"].(map[any]any)
	required := asStringSlice(t, item["required"])
	if !contains(required, "compatible_networks") {
		t.Fatal("CryptoPolicyCatalogItem must require compatible_networks")
	}
	props := item["properties"].(map[any]any)
	if _, ok := props["compatible_networks"]; !ok {
		t.Fatal("CryptoPolicyCatalogItem.compatible_networks missing")
	}

	listPath := paths["/crypto-policies"].(map[any]any)
	getOp := listPath["get"].(map[any]any)
	resp200 := getOp["responses"].(map[any]any)["200"].(map[any]any)
	content := resp200["content"].(map[any]any)["application/json"].(map[any]any)
	schema := content["schema"].(map[any]any)
	if ref, _ := schema["$ref"].(string); ref != "#/components/schemas/CryptoPolicyListResponse" {
		t.Fatalf("listCryptoPolicies 200 schema $ref: got %#v", schema["$ref"])
	}

	byID := paths["/crypto-policies/{crypto_policy_id}"].(map[any]any)
	getByID := byID["get"].(map[any]any)
	byID200 := getByID["responses"].(map[any]any)["200"].(map[any]any)
	byIDContent := byID200["content"].(map[any]any)["application/json"].(map[any]any)
	byIDSchema := byIDContent["schema"].(map[any]any)
	if ref, _ := byIDSchema["$ref"].(string); ref != "#/components/schemas/CryptoPolicyCatalogItem" {
		t.Fatalf("getCryptoPolicy 200 schema $ref: got %#v", byIDSchema["$ref"])
	}

	network := schemas["CompatibleNetwork"].(map[any]any)
	status := network["properties"].(map[any]any)["status"].(map[any]any)
	enumVals := asStringSlice(t, status["enum"])
	if contains(enumVals, "planned") {
		t.Fatal("CompatibleNetwork.status must not include planned")
	}
}
