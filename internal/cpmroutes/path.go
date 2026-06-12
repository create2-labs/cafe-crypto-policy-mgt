package cpmroutes

import "strings"

// DraftPersistPath returns the concrete persist path for a platform draft id.
func DraftPersistPath(draftID string) string {
	return V1Base + "/drafts/" + strings.TrimSpace(draftID) + "/persist"
}

// PathMatches reports whether actual matches pattern, including {param} segments.
func PathMatches(pattern, actual string) bool {
	if pattern == actual {
		return true
	}
	if !strings.Contains(pattern, "{") {
		return false
	}
	patParts := strings.Split(strings.Trim(pattern, "/"), "/")
	actParts := strings.Split(strings.Trim(actual, "/"), "/")
	if len(patParts) != len(actParts) {
		return false
	}
	for i, pat := range patParts {
		if strings.HasPrefix(pat, "{") && strings.HasSuffix(pat, "}") {
			if actParts[i] == "" {
				return false
			}
			continue
		}
		if pat != actParts[i] {
			return false
		}
	}
	return true
}
