package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type captureLogger struct {
	lines []string
}

func (l *captureLogger) Printf(format string, v ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, v...))
}

func TestLoadReadStore_CatalogueDir_SkipsIncompatible(t *testing.T) {
	dir := t.TempDir()

	mustWrite := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	cpRaw, err := os.ReadFile(filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("read cp fixture: %v", err)
	}
	mustWrite("crypto_policy_pq_account_validation_v1.json", string(cpRaw))

	nicetryRaw, err := os.ReadFile(filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"))
	if err != nil {
		t.Fatalf("read nicetry fixture: %v", err)
	}
	mustWrite("provider_manifest_nicetry_v0_1.json", string(nicetryRaw))

	nicetry2Raw, err := os.ReadFile(filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry2_v0_1.json"))
	if err != nil {
		t.Fatalf("read nicetry2 fixture: %v", err)
	}
	mustWrite("provider_manifest_nicetry2_v0_1.json", string(nicetry2Raw))

	instRaw, err := os.ReadFile(filepath.Join("..", "domain", "policy", "testdata", "test_crypto_policy_instance_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("read instance fixture: %v", err)
	}
	mustWrite("test_crypto_policy_instance_pq_account_validation_v1.json", string(instRaw))

	mustWrite("notes.txt", "not json")
	mustWrite("garbage.json", `{"hello":"world"}`)
	mustWrite("broken_provider.json", `{
  "schema_version": "cafe.provider_manifest.v0.1",
  "provider_id": "broken"
}`)

	logger := &captureLogger{}
	store, err := loadReadStoreFromCatalogueDir(dir, logger)
	if err != nil {
		t.Fatalf("loadReadStoreFromCatalogueDir: %v", err)
	}
	if len(store.cryptoPolicies) != 1 {
		t.Fatalf("crypto policies: got %d", len(store.cryptoPolicies))
	}
	if len(store.providers.List()) != 2 {
		t.Fatalf("providers: got %d", len(store.providers.List()))
	}

	joined := strings.Join(logger.lines, "\n")
	for _, want := range []string{
		"catalogue skip",
		"not a crypto policy or provider manifest",
		"incompatible provider manifest",
		"catalogue loaded crypto_policy",
		"catalogue loaded provider_manifest",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected log to contain %q; logs=\n%s", want, joined)
		}
	}
}

func TestLoadReadStore_MissingOrEmptyCatalogueDirFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	if _, err := LoadReadStore(ReadStoreOptions{CatalogueDir: missing}); err == nil {
		t.Fatal("expected catalogue load to fail when the directory is missing")
	}

	empty := t.TempDir()
	_, err := LoadReadStore(ReadStoreOptions{CatalogueDir: empty})
	if err == nil {
		t.Fatal("expected catalogue load to fail when the directory has no json")
	}
	if !strings.Contains(err.Error(), "no .json") {
		t.Fatalf("unexpected empty-dir error: %v", err)
	}
}

func TestClassifyCatalogueJSON(t *testing.T) {
	if got := classifyCatalogueJSON([]byte(`{"schema_version":"cafe.provider_manifest.v0.1","provider_id":"x","solution_profiles":[]}`)); got != catalogueKindProviderManifest {
		t.Fatalf("provider: got %s", got)
	}
	if got := classifyCatalogueJSON([]byte(`{"id":"cp","required_posture":"hybrid","allowed_providers":["p"]}`)); got != catalogueKindCryptoPolicy {
		t.Fatalf("crypto policy: got %s", got)
	}
	if got := classifyCatalogueJSON([]byte(`{"id":"inst","template_id":"cp","solution_profile_ref":{"provider_id":"p"}}`)); got != catalogueKindSkip {
		t.Fatalf("instance: got %s", got)
	}
	if got := classifyCatalogueJSON([]byte(`{`)); got != catalogueKindSkip {
		t.Fatalf("broken json: got %s", got)
	}
}
