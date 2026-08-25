package payloadhash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// Closed top-level fields for the V1 hashed payload (ADR §3.2.1). Order is
// documentation-only; JCS sorts object keys.
var closedFields = []string{
	"schema_version",
	"crypto_policy_id",
	"required_posture",
	"user_constraints",
	"solution_profile_ref",
	"accepted_provider_snapshot",
	"accepted_findings",
}

var closedFieldSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(closedFields))
	for _, f := range closedFields {
		m[f] = struct{}{}
	}
	return m
}()

// DigestJSON unmarshals raw JSON, validates the closed hashed set, normalizes
// accepted_findings, and returns lowercase hex SHA-256 of RFC 8785 JCS bytes.
func DigestJSON(raw []byte) (string, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", reject(ReasonNotObject, "$", "invalid JSON: "+err.Error())
	}
	return Digest(root)
}

// Digest validates a decoded JSON value as the closed hashed payload and
// returns lowercase hex SHA-256 of RFC 8785 JCS bytes.
func Digest(root any) (string, error) {
	obj, ok := root.(map[string]any)
	if !ok || obj == nil {
		return "", reject(ReasonNotObject, "$", "hashed payload must be a JSON object")
	}

	for key := range obj {
		if _, known := closedFieldSet[key]; !known {
			return "", reject(ReasonUnknownField, "$."+key, "unknown top-level hashed field")
		}
	}
	for _, field := range closedFields {
		if _, present := obj[field]; !present {
			return "", reject(ReasonMissingField, "$."+field, "required closed field missing")
		}
	}

	if err := assertAllowedSubtree(obj, "$"); err != nil {
		return "", err
	}

	findings, err := findingsAsStrings(obj["accepted_findings"], "$.accepted_findings")
	if err != nil {
		return "", err
	}
	normalized := NormalizeAcceptedFindings(findings)

	closed := make(map[string]any, len(closedFields))
	for _, field := range closedFields {
		if field == "accepted_findings" {
			// Prefer []any for json.Marshal symmetry with golden fixtures.
			arr := make([]any, len(normalized))
			for i, s := range normalized {
				arr[i] = s
			}
			closed[field] = arr
			continue
		}
		closed[field] = obj[field]
	}

	marshaled, err := json.Marshal(closed)
	if err != nil {
		return "", fmt.Errorf("marshal closed hashed payload: %w", err)
	}
	canon, err := jcs.Transform(marshaled)
	if err != nil {
		return "", reject(ReasonCanonicalizeFailed, "$", err.Error())
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}

func findingsAsStrings(v any, path string) ([]string, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, reject(ReasonFindingsNotArray, path, "accepted_findings must be a JSON array")
	}
	out := make([]string, 0, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, reject(ReasonFindingsItemType, fmt.Sprintf("%s[%d]", path, i), "accepted_findings items must be strings")
		}
		out = append(out, s)
	}
	return out, nil
}

func assertAllowedSubtree(v any, path string) error {
	switch t := v.(type) {
	case nil:
		return reject(ReasonNullForbidden, path, "null forbidden in hashed subtree")
	case float64:
		return reject(ReasonNumberForbidden, path, fmt.Sprintf("number forbidden in hashed subtree (got %v)", t))
	case json.Number:
		return reject(ReasonNumberForbidden, path, fmt.Sprintf("number forbidden in hashed subtree (got %v)", t))
	case bool, string:
		return nil
	case map[string]any:
		for k, child := range t {
			if err := assertAllowedSubtree(child, path+"."+k); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for i, child := range t {
			if err := assertAllowedSubtree(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return reject(ReasonUnsupportedType, path, fmt.Sprintf("unsupported JSON type %T", v))
	}
}
