package payloadhash

import "fmt"

// Stable reject reasons for closed-set / type validation before JCS.
const (
	ReasonNotObject          = "hashed_payload_not_object"
	ReasonMissingField       = "hashed_payload_missing_field"
	ReasonUnknownField       = "hashed_payload_unknown_field"
	ReasonNullForbidden      = "hashed_payload_null_forbidden"
	ReasonNumberForbidden    = "hashed_payload_number_forbidden"
	ReasonUnsupportedType    = "hashed_payload_unsupported_type"
	ReasonFindingsNotArray   = "accepted_findings_not_array"
	ReasonFindingsItemType   = "accepted_findings_item_not_string"
	ReasonCanonicalizeFailed = "jcs_canonicalize_failed"
)

// Error is a validation failure before or during payload_sha256 computation.
type Error struct {
	Reason  string
	Path    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Path != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Reason, e.Message, e.Path)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Reason, e.Message)
	}
	return e.Reason
}

func reject(reason, path, message string) error {
	return &Error{Reason: reason, Path: path, Message: message}
}
