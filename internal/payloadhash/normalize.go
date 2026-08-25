package payloadhash

import "sort"

// NormalizeAcceptedFindings returns a lexicographically sorted, deduplicated
// copy of findings (ADR §3.2.1). Empty input yields an empty non-nil slice.
// The same helper must be used at challenge (RD-P4) and persist (RD-P5).
func NormalizeAcceptedFindings(findings []string) []string {
	if len(findings) == 0 {
		return []string{}
	}
	out := append([]string(nil), findings...)
	sort.Strings(out)
	n := 0
	for i, s := range out {
		if i > 0 && s == out[i-1] {
			continue
		}
		out[n] = s
		n++
	}
	return out[:n]
}
