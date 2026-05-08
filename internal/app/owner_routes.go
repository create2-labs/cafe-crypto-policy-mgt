package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

type ownerScopedUpsertRequest struct {
	ID          string         `json:"id"`
	ScanID      string         `json:"scan_id"`
	Payload     map[string]any `json:"payload"`
	OwnerUserID string         `json:"owner_user_id,omitempty"`
	TenantID    string         `json:"tenant_id,omitempty"`
}

func registerOwnerScopedRoutes(mux *http.ServeMux, store *persistence.OwnerScopedStore, obs *authObservability) {
	if obs == nil {
		obs = newAuthObservability()
	}
	mux.HandleFunc("POST /api/v1/cpm/drafts", func(w http.ResponseWriter, r *http.Request) {
		requestID := obs.ensureRequestID(w, r)
		principal, ok := principalFromContext(r.Context())
		if !ok {
			obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
			obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
			return
		}
		req, err := decodeOwnerScopedUpsertRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		record, saveErr := store.SaveDraft(principal, req.ID, req.ScanID, req.Payload)
		if saveErr != nil {
			mapPersistenceError(w, r, obs, principal, saveErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("GET /api/v1/cpm/drafts", func(w http.ResponseWriter, r *http.Request) {
		requestID := obs.ensureRequestID(w, r)
		principal, ok := principalFromContext(r.Context())
		if !ok {
			obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
			obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		record, err := store.GetDraft(principal, id)
		if err != nil {
			mapPersistenceError(w, r, obs, principal, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("POST /api/v1/cpm/policies", func(w http.ResponseWriter, r *http.Request) {
		requestID := obs.ensureRequestID(w, r)
		principal, ok := principalFromContext(r.Context())
		if !ok {
			obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
			obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
			return
		}
		req, err := decodeOwnerScopedUpsertRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		record, saveErr := store.SavePolicy(principal, req.ID, req.ScanID, req.Payload)
		if saveErr != nil {
			mapPersistenceError(w, r, obs, principal, saveErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("GET /api/v1/cpm/policies", func(w http.ResponseWriter, r *http.Request) {
		requestID := obs.ensureRequestID(w, r)
		principal, ok := principalFromContext(r.Context())
		if !ok {
			obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
			obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		record, err := store.GetPolicy(principal, id)
		if err != nil {
			mapPersistenceError(w, r, obs, principal, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
}

func decodeOwnerScopedUpsertRequest(r *http.Request) (ownerScopedUpsertRequest, error) {
	var req ownerScopedUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return ownerScopedUpsertRequest{}, err
	}
	if strings.TrimSpace(req.ID) == "" {
		return ownerScopedUpsertRequest{}, errors.New("id is required")
	}
	if strings.TrimSpace(req.OwnerUserID) != "" || strings.TrimSpace(req.TenantID) != "" {
		return ownerScopedUpsertRequest{}, errors.New("owner_user_id and tenant_id are server-managed")
	}
	return req, nil
}

func mapPersistenceError(w http.ResponseWriter, r *http.Request, obs *authObservability, principal authz.Principal, err error) {
	requestID := obs.ensureRequestID(w, r)
	switch {
	case errors.Is(err, persistence.ErrForbidden):
		obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodeOwnerForbidden, "owner_forbidden", principal.UserID, principal.TenantID)
		obs.audit.RecordAuthEvent(authAuditEvent{
			Category:  authCategoryOwner,
			Outcome:   "denied",
			Code:      authCodeOwnerForbidden,
			RequestID: requestID,
			Route:     classifyRoute(r.Method, r.URL.Path),
			Method:    r.Method,
			UserID:    principal.UserID,
			TenantID:  principal.TenantID,
		})
		obs.writeAuthError(w, r, http.StatusForbidden, authCodeOwnerForbidden, "owner access denied", map[string]any{"reason": "owner_forbidden"})
	case errors.Is(err, persistence.ErrDraftNotFound), errors.Is(err, persistence.ErrPolicyNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	case errors.Is(err, persistence.ErrPrincipalRequired):
		obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
