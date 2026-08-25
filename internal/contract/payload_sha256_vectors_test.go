package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// RD-P1: shared payload_sha256 vectors must be free of JSON number/null in the hashed subtree.
func TestPayloadSHA256Vectors_NoNumberOrNull(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "payload_sha256")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	shaRe := regexp.MustCompile(`^[0-9a-f]{64}$`)
	var jsonCount int
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		jsonCount++
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if err := assertNoNumberOrNull(v, "$"); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		shaPath := filepath.Join(dir, strings.TrimSuffix(name, ".json")+".sha256")
		shaRaw, err := os.ReadFile(shaPath)
		if err != nil {
			t.Fatalf("missing companion sha256 for %s: %v", name, err)
		}
		digest := strings.TrimSpace(string(shaRaw))
		if !shaRe.MatchString(digest) {
			t.Fatalf("%s: invalid sha256 %q", name, digest)
		}
	}
	if jsonCount < 2 {
		t.Fatalf("expected at least 2 golden JSON vectors, got %d", jsonCount)
	}
}

func TestPayloadSHA256Vectors_RealisticNestedPresent(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "payload_sha256")
	raw, err := os.ReadFile(filepath.Join(dir, "hashed_payload_realistic_nested.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	snap, ok := v["accepted_provider_snapshot"].(map[string]any)
	if !ok {
		t.Fatal("missing accepted_provider_snapshot")
	}
	chain, ok := snap["chain_support_used"].(map[string]any)
	if !ok {
		t.Fatal("missing chain_support_used")
	}
	id, ok := chain["chain_id"].(string)
	if !ok || id == "" {
		t.Fatalf("chain_id must be non-empty string, got %#v", chain["chain_id"])
	}
	refs, ok := snap["references"].([]any)
	if !ok || len(refs) < 2 {
		t.Fatalf("realistic fixture should have nested refs, got %#v", snap["references"])
	}
}

func assertNoNumberOrNull(v any, path string) error {
	switch t := v.(type) {
	case nil:
		return fmt.Errorf("%s: null forbidden in hashed subtree", path)
	case float64:
		return fmt.Errorf("%s: number forbidden in hashed subtree (got %v)", path, t)
	case json.Number:
		return fmt.Errorf("%s: number forbidden in hashed subtree (got %v)", path, t)
	case map[string]any:
		for k, child := range t {
			if err := assertNoNumberOrNull(child, path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range t {
			if err := assertNoNumberOrNull(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case string, bool:
		return nil
	default:
		return fmt.Errorf("%s: unsupported JSON type %T", path, v)
	}
	return nil
}
