package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

const draftStatusServerDraft = "server_draft"

const (
	draftCodeIDRequired           = "DRAFT_ID_REQUIRED"
	draftCodePayloadRequired      = "DRAFT_PAYLOAD_REQUIRED"
	draftCodeScanIDInvalid        = "DRAFT_SCAN_ID_INVALID"
	draftCodeBindingForbidden     = "DRAFT_BINDING_FORBIDDEN"
	draftCodeOwnerFieldsForbidden = "DRAFT_OWNER_FIELDS_FORBIDDEN"
	draftCodeNotFound             = "DRAFT_NOT_FOUND"
	draftCodeInternalError        = "INTERNAL_ERROR"
)

// draftUpsertRequest is the normative POST /drafts body (CPM-DRAFT-1).
type draftUpsertRequest struct {
	ID      string         `json:"id"`
	ScanID  string         `json:"scan_id,omitempty"`
	Payload map[string]any `json:"payload"`
}

// draftUpsertResponse is the minimal POST /drafts acknowledgement.
type draftUpsertResponse struct {
	DraftID string `json:"draft_id"`
	SavedAt string `json:"saved_at"`
	Status  string `json:"status"`
}

// draftRecordResponse is the GET /drafts?id=… row (owner fields omitted).
type draftRecordResponse struct {
	ID        string         `json:"id"`
	ScanID    string         `json:"scan_id,omitempty"`
	Payload   map[string]any `json:"payload"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type draftDecodeError struct {
	code    string
	message string
}

func (e *draftDecodeError) Error() string {
	return e.message
}

func decodeDraftUpsertRequest(r *http.Request) (draftUpsertRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodePayloadRequired,
			message: "request body must be a JSON object with id and payload",
		}
	}
	if _, ok := raw["binding"]; ok {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodeBindingForbidden,
			message: "binding is not allowed on platform drafts",
		}
	}
	if _, ok := raw["owner_user_id"]; ok {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodeOwnerFieldsForbidden,
			message: "owner_user_id and tenant_id are server-managed",
		}
	}
	if _, ok := raw["tenant_id"]; ok {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodeOwnerFieldsForbidden,
			message: "owner_user_id and tenant_id are server-managed",
		}
	}
	if _, ok := raw["draft"]; ok {
		if _, hasID := raw["id"]; !hasID {
			return draftUpsertRequest{}, &draftDecodeError{
				code:    draftCodeIDRequired,
				message: "id is required",
			}
		}
	}
	idRaw, hasID := raw["id"]
	if !hasID {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodeIDRequired,
			message: "id is required",
		}
	}
	var id string
	if err := json.Unmarshal(idRaw, &id); err != nil || strings.TrimSpace(id) == "" {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodeIDRequired,
			message: "id is required",
		}
	}
	payloadRaw, hasPayload := raw["payload"]
	if !hasPayload {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodePayloadRequired,
			message: "payload is required and must be a JSON object",
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodePayloadRequired,
			message: "payload is required and must be a JSON object",
		}
	}
	if payload == nil {
		return draftUpsertRequest{}, &draftDecodeError{
			code:    draftCodePayloadRequired,
			message: "payload is required and must be a JSON object",
		}
	}
	scanID := ""
	if scanRaw, ok := raw["scan_id"]; ok {
		if err := json.Unmarshal(scanRaw, &scanID); err != nil {
			return draftUpsertRequest{}, &draftDecodeError{
				code:    draftCodeScanIDInvalid,
				message: "scan_id must be a valid UUID",
			}
		}
		scanID = strings.TrimSpace(scanID)
		if scanID != "" {
			norm, err := NormalizeDiscoveryScanID(scanID)
			if err != nil {
				return draftUpsertRequest{}, &draftDecodeError{
					code:    draftCodeScanIDInvalid,
					message: "scan_id must be a valid UUID",
				}
			}
			scanID = norm
		}
	}
	return draftUpsertRequest{
		ID:      strings.TrimSpace(id),
		ScanID:  scanID,
		Payload: payload,
	}, nil
}

func draftUpsertResponseFromRecord(record persistence.DraftRecord) draftUpsertResponse {
	return draftUpsertResponse{
		DraftID: record.ID,
		SavedAt: record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Status:  draftStatusServerDraft,
	}
}

func draftRecordResponseFromStore(record persistence.DraftRecord) draftRecordResponse {
	resp := draftRecordResponse{
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

func writeDraftStructuredError(w http.ResponseWriter, r *http.Request, obs *authObservability, status int, code string, message string, details map[string]any) {
	obs.writeAuthError(w, r, status, code, message, details)
}

func mapDraftPersistenceError(w http.ResponseWriter, r *http.Request, obs *authObservability, principal authz.Principal, err error) {
	requestID := obs.ensureRequestID(w, r)
	switch {
	case errors.Is(err, persistence.ErrForbidden):
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodeOwnerForbidden, "owner_forbidden", principal.UserID, principal.TenantID)
		obs.audit.RecordAuthEvent(authAuditEvent{
			Category:  authCategoryOwner,
			Outcome:   authOutcomeDenied,
			Code:      authCodeOwnerForbidden,
			RequestID: requestID,
			Route:     classifyRoute(r.Method, r.URL.Path),
			Method:    r.Method,
			UserID:    principal.UserID,
			TenantID:  principal.TenantID,
		})
		writeDraftStructuredError(w, r, obs, http.StatusForbidden, authCodeOwnerForbidden, "owner access denied", reasonDetails("owner_forbidden"))
	case errors.Is(err, persistence.ErrDraftNotFound):
		writeDraftStructuredError(w, r, obs, http.StatusNotFound, draftCodeNotFound, "draft not found", map[string]any{})
	case errors.Is(err, persistence.ErrPrincipalRequired):
		writeDraftStructuredError(w, r, obs, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
	default:
		writeDraftStructuredError(w, r, obs, http.StatusInternalServerError, draftCodeInternalError, errMsgInternalServerError, map[string]any{})
	}
}
