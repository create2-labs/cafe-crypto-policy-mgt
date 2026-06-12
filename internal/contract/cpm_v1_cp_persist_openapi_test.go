package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v2"
)

// CP-PERSIST OpenAPI contract checks (openapi/cpm-v1.yaml — CP-PERSIST-T2 / PR2).
func TestCPMV1OpenAPI_CP_PersistRoutesDocumented(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths, ok := spec["paths"].(map[any]any)
	if !ok {
		t.Fatal("paths missing or invalid")
	}

	for _, route := range []string{
		"/wallet-challenges",
		"/drafts/{draft_id}/persist",
	} {
		if _, ok := paths[route]; !ok {
			t.Fatalf("openapi paths missing %q", route)
		}
	}

	if _, ok := paths["/wallet-challenges/verify"]; ok {
		t.Fatal("POST /wallet-challenges/verify must not be documented as V1 security requirement")
	}
}

func TestCPMV1OpenAPI_CP_PersistSchemasDocumented(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	components, ok := spec["components"].(map[any]any)
	if !ok {
		t.Fatal("components missing or invalid")
	}
	schemas, ok := components["schemas"].(map[any]any)
	if !ok {
		t.Fatal("components.schemas missing or invalid")
	}

	for _, name := range []string{
		"WalletChallengeRequest",
		"WalletChallengeResponse",
		"DraftPersistRequest",
		"DraftPersistResponse",
		"WalletAuthorizationErrorCode",
	} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("openapi schema missing %q", name)
		}
	}

	codes, ok := schemas["WalletAuthorizationErrorCode"].(map[any]any)
	if !ok {
		t.Fatal("WalletAuthorizationErrorCode invalid")
	}
	enumVals, ok := codes["enum"].([]any)
	if !ok {
		t.Fatal("WalletAuthorizationErrorCode.enum missing")
	}
	requiredCodes := []string{
		"WALLET_CONTROL_PROOF_REQUIRED",
		"INVALID_WALLET_SIGNATURE",
		"WALLET_SIGNATURE_ADDRESS_MISMATCH",
		"WALLET_AUTHORIZATION_EXPIRED",
		"WALLET_AUTHORIZATION_NOT_YET_VALID",
		"WALLET_AUTHORIZATION_VALIDITY_TOO_LONG",
		"DRAFT_ALREADY_PERSISTED",
		"UNSUPPORTED_WALLET_TYPE",
	}
	have := make(map[string]bool, len(enumVals))
	for _, v := range enumVals {
		have[v.(string)] = true
	}
	for _, code := range requiredCodes {
		if !have[code] {
			t.Fatalf("WalletAuthorizationErrorCode missing %q", code)
		}
	}
}

func TestCPMV1OpenAPI_PostPoliciesDocumentsEOABlock(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths := spec["paths"].(map[any]any)
	policies := paths["/policies"].(map[any]any)
	post := policies["post"].(map[any]any)
	desc, _ := post["description"].(string)
	lower := strings.ToLower(desc)
	for _, needle := range []string{
		"not the normative cp-persist endpoint",
		"wallet_control_proof_required",
		"post /drafts/{draft_id}/persist",
	} {
		if !strings.Contains(lower, needle) {
			t.Fatalf("POST /policies description missing %q", needle)
		}
	}
	responses, ok := post["responses"].(map[any]any)
	if !ok {
		t.Fatal("POST /policies responses missing")
	}
	if _, ok := responses["403"]; !ok {
		t.Fatal("POST /policies must document 403 WALLET_CONTROL_PROOF_REQUIRED for legacy EOA paths")
	}
}

func loadCPMV1OpenAPISpec(t *testing.T) map[any]any {
	t.Helper()
	path := filepath.Join("..", "..", "openapi", "cpm-v1.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}
	var spec map[any]any
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi: %v", err)
	}
	return spec
}
