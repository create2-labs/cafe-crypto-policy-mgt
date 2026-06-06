package app

import (
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

// policyRecordResponse is the public GET/POST /policies row (OpenAPI PolicyRecord).
// Owner fields are omitted; shape mirrors draftRecordResponse.
type policyRecordResponse struct {
	ID        string         `json:"id"`
	ScanID    string         `json:"scan_id,omitempty"`
	Payload   map[string]any `json:"payload"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

func policyRecordResponseFromStore(record persistence.PolicyRecord) policyRecordResponse {
	resp := policyRecordResponse{
		ID:        record.ID,
		Payload:   record.Payload,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if scan := strings.TrimSpace(record.ScanID); scan != "" {
		resp.ScanID = scan
	}
	return resp
}

func policyRecordResponsesFromStore(list []persistence.PolicyRecord) []policyRecordResponse {
	out := make([]policyRecordResponse, 0, len(list))
	for _, record := range list {
		out = append(out, policyRecordResponseFromStore(record))
	}
	return out
}
