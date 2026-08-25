package payloadhash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func goldenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "contract", "testdata", "payload_sha256")
}

func readGolden(t *testing.T, name string) (raw []byte, wantDigest string) {
	t.Helper()
	dir := goldenDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		t.Fatalf("read golden json: %v", err)
	}
	sha, err := os.ReadFile(filepath.Join(dir, name+".sha256"))
	if err != nil {
		t.Fatalf("read golden sha256: %v", err)
	}
	return raw, strings.TrimSpace(string(sha))
}

func TestDigestJSON_MatchesRD_P1Goldens(t *testing.T) {
	for _, name := range []string{
		"hashed_payload_minimal",
		"hashed_payload_realistic_nested",
	} {
		t.Run(name, func(t *testing.T) {
			raw, want := readGolden(t, name)
			got, err := DigestJSON(raw)
			if err != nil {
				t.Fatalf("DigestJSON: %v", err)
			}
			if got != want {
				t.Fatalf("digest mismatch:\n got  %s\n want %s", got, want)
			}
		})
	}
}

func TestDigest_FindingsOrderAndDedupeInvariant(t *testing.T) {
	raw, want := readGolden(t, "hashed_payload_minimal")
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}

	// Scramble + duplicate findings; digest must match golden (already sorted).
	obj["accepted_findings"] = []any{
		"requires_local_signer_state",
		"requires_bundler",
		"requires_bundler",
		"requires_local_signer_state",
	}
	got, err := Digest(obj)
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	if got != want {
		t.Fatalf("normalized findings must match golden digest:\n got  %s\n want %s", got, want)
	}
}

func TestDigest_RealisticNestedStable(t *testing.T) {
	raw, want := readGolden(t, "hashed_payload_realistic_nested")
	got, err := DigestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("realistic nested digest mismatch:\n got  %s\n want %s", got, want)
	}
	// Second pass must be byte-stable.
	got2, err := DigestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != got {
		t.Fatalf("digest not stable across calls: %s vs %s", got, got2)
	}
}

func TestDigest_RejectCases(t *testing.T) {
	baseRaw, _ := readGolden(t, "hashed_payload_minimal")
	var base map[string]any
	if err := json.Unmarshal(baseRaw, &base); err != nil {
		t.Fatal(err)
	}

	clone := func() map[string]any {
		b, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	cases := []struct {
		name   string
		mutate func(map[string]any)
		reason string
	}{
		{
			name: "null_in_subtree",
			mutate: func(m map[string]any) {
				m["user_constraints"].(map[string]any)["allow_new_wallet"] = nil
			},
			reason: ReasonNullForbidden,
		},
		{
			name: "number_in_subtree",
			mutate: func(m map[string]any) {
				snap := m["accepted_provider_snapshot"].(map[string]any)
				chain := snap["chain_support_used"].(map[string]any)
				chain["chain_id"] = float64(11155111)
			},
			reason: ReasonNumberForbidden,
		},
		{
			name: "unknown_top_level_field",
			mutate: func(m map[string]any) {
				m["payload_sha256"] = "deadbeef"
			},
			reason: ReasonUnknownField,
		},
		{
			name: "missing_required_field",
			mutate: func(m map[string]any) {
				delete(m, "solution_profile_ref")
			},
			reason: ReasonMissingField,
		},
		{
			name: "accepted_findings_not_array",
			mutate: func(m map[string]any) {
				m["accepted_findings"] = "requires_bundler"
			},
			reason: ReasonFindingsNotArray,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := clone()
			tc.mutate(obj)
			_, err := Digest(obj)
			if err == nil {
				t.Fatal("expected error")
			}
			pe, ok := err.(*Error)
			if !ok {
				t.Fatalf("want *Error, got %T (%v)", err, err)
			}
			if pe.Reason != tc.reason {
				t.Fatalf("reason: got %q want %q (%v)", pe.Reason, tc.reason, err)
			}
		})
	}
}

func TestDigest_RejectRootNotObject(t *testing.T) {
	_, err := DigestJSON([]byte(`["not","object"]`))
	if err == nil {
		t.Fatal("expected error")
	}
	pe, ok := err.(*Error)
	if !ok || pe.Reason != ReasonNotObject {
		t.Fatalf("want %s, got %v", ReasonNotObject, err)
	}
}

func TestNormalizeAcceptedFindings(t *testing.T) {
	got := NormalizeAcceptedFindings([]string{
		"requires_local_signer_state",
		"requires_bundler",
		"requires_bundler",
	})
	want := []string{"requires_bundler", "requires_local_signer_state"}
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if NormalizeAcceptedFindings(nil) == nil {
		t.Fatal("empty normalize must return non-nil empty slice")
	}
}
