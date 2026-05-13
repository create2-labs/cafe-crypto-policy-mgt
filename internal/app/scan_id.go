package app

import (
	"errors"
	"regexp"
	"strings"
)

// scanUUIDPattern matches canonical UUID strings (Discovery scan_id).
var scanUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var errInvalidDiscoveryScanUUID = errors.New("scan_id must be a valid UUID")

// NormalizeDiscoveryScanID trims, validates UUID shape, and lowercases for stable lookups
// (shared with internal policy reference check and public GET …/policies?scan_id=).
func NormalizeDiscoveryScanID(s string) (string, error) {
	scan := strings.TrimSpace(s)
	if !scanUUIDPattern.MatchString(scan) {
		return "", errInvalidDiscoveryScanUUID
	}
	return strings.ToLower(scan), nil
}

func isOwnerPoliciesGETPath(path string) bool {
	switch path {
	case "/api/v1/cpm/policies", "/api/cpm/v1/policies":
		return true
	default:
		return false
	}
}
