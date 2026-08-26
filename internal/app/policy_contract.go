package app

import (
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

// policyRecordResponse is the public GET /policies row (OpenAPI PolicyRecord).
type policyRecordResponse struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id,omitempty"`
	Payload       map[string]any `json:"payload"`
	PayloadSHA256 string         `json:"payload_sha256,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

func policyRecordResponseFromStore(record persistence.PolicyRecord) policyRecordResponse {
	resp := policyRecordResponse{
		ID:            record.ID,
		Payload:       record.Payload,
		PayloadSHA256: strings.TrimSpace(record.PayloadSHA256),
		CreatedAt:     record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     record.UpdatedAt.UTC().Format(time.RFC3339Nano),
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
