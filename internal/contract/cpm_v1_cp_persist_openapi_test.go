package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v2"
)

// CP-PERSIST OpenAPI contract checks (openapi/cpm-v1.yaml — RD-P1 remove drafts).
func TestCPMV1OpenAPI_CP_PersistRoutesDocumented(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths, ok := spec["paths"].(map[any]any)
	if !ok {
		t.Fatal("paths missing or invalid")
	}

	for _, route := range []string{
		"/wallet-challenges",
		"/policies",
	} {
		if _, ok := paths[route]; !ok {
			t.Fatalf("openapi paths missing %q", route)
		}
	}

	for _, removed := range []string{
		"/drafts",
		"/drafts/{draft_id}/persist",
		"/wallet-challenges/verify",
	} {
		if _, ok := paths[removed]; ok {
			t.Fatalf("openapi must not document removed path %q", removed)
		}
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
		"PolicyPersistRequest",
		"PolicyPersistResponse",
		"CryptoPolicyPersistPayload",
		"WalletAuthorizationErrorCode",
	} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("openapi schema missing %q", name)
		}
	}

	for _, removed := range []string{
		"DraftPersistRequest",
		"DraftPersistResponse",
		"DraftUpsertRequest",
		"DraftRecord",
		"PolicyRecordWrite",
	} {
		if _, ok := schemas[removed]; ok {
			t.Fatalf("openapi must not keep removed schema %q", removed)
		}
	}

	challengeReq := schemas["WalletChallengeRequest"].(map[any]any)
	required := asStringSlice(t, challengeReq["required"])
	if contains(required, "draft_id") {
		t.Fatal("WalletChallengeRequest must not require draft_id")
	}
	if !contains(required, "payload") {
		t.Fatal("WalletChallengeRequest must require payload")
	}

	challengeResp := schemas["WalletChallengeResponse"].(map[any]any)
	respRequired := asStringSlice(t, challengeResp["required"])
	if contains(respRequired, "draft_id") {
		t.Fatal("WalletChallengeResponse must not require draft_id")
	}
	if !contains(respRequired, "payload_sha256") {
		t.Fatal("WalletChallengeResponse must require payload_sha256")
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
		"POLICY_ALREADY_EXISTS",
		"SCAN_NOT_LATEST",
		"DISCOVERY_UNAVAILABLE",
		"PAYLOAD_SHA256_MISMATCH",
		"UNSUPPORTED_WALLET_TYPE",
	}
	forbiddenCodes := []string{
		"DRAFT_ALREADY_PERSISTED",
		"DRAFT_NOT_FOUND",
		"DRAFT_SCAN_MISMATCH",
		"DRAFT_WALLET_MISMATCH",
		"WALLET_AUTHORIZATION_DRAFT_MISMATCH",
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
	for _, code := range forbiddenCodes {
		if have[code] {
			t.Fatalf("WalletAuthorizationErrorCode must not include draft code %q", code)
		}
	}

	snapshot := schemas["AcceptedProviderSnapshot"].(map[any]any)
	props := snapshot["properties"].(map[any]any)
	chain := props["chain_support_used"].(map[any]any)
	chainProps := chain["properties"].(map[any]any)
	chainID := chainProps["chain_id"].(map[any]any)
	if chainID["type"] != "string" {
		t.Fatalf("AcceptedProviderSnapshot.chain_support_used.chain_id must be string, got %#v", chainID["type"])
	}
}

func TestCPMV1OpenAPI_PostPoliciesIsNormativeSignedPersist(t *testing.T) {
	spec := loadCPMV1OpenAPISpec(t)
	paths := spec["paths"].(map[any]any)
	policies := paths["/policies"].(map[any]any)
	post := policies["post"].(map[any]any)
	desc, _ := post["description"].(string)
	lower := strings.ToLower(desc)
	for _, needle := range []string{
		"normative eoa cp persistence",
		"wallet_control_proof_required",
		"payload_sha256",
		"scan_not_latest",
		"no shim",
	} {
		if !strings.Contains(lower, needle) {
			t.Fatalf("POST /policies description missing %q", needle)
		}
	}
	responses, ok := post["responses"].(map[any]any)
	if !ok {
		t.Fatal("POST /policies responses missing")
	}
	for _, code := range []string{"403", "409", "422", "503"} {
		if _, ok := responses[code]; !ok {
			t.Fatalf("POST /policies must document %s", code)
		}
	}
	body := post["requestBody"].(map[any]any)
	content := body["content"].(map[any]any)
	appJSON := content["application/json"].(map[any]any)
	schema := appJSON["schema"].(map[any]any)
	ref, _ := schema["$ref"].(string)
	if !strings.HasSuffix(ref, "/PolicyPersistRequest") {
		t.Fatalf("POST /policies body must ref PolicyPersistRequest, got %q", ref)
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

func asStringSlice(t *testing.T, v any) []string {
	t.Helper()
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", v)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("expected string in slice, got %T", item)
		}
		out = append(out, s)
	}
	return out
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
