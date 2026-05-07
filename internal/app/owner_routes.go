package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

type ownerScopedUpsertRequest struct {
	ID          string         `json:"id"`
	ScanID      string         `json:"scan_id"`
	Payload     map[string]any `json:"payload"`
	OwnerUserID string         `json:"owner_user_id,omitempty"`
	TenantID    string         `json:"tenant_id,omitempty"`
}

func registerOwnerScopedRoutes(mux *http.ServeMux, store *persistence.OwnerScopedStore) {
	mux.HandleFunc("POST /api/v1/cpm/drafts", func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing principal"})
			return
		}
		req, err := decodeOwnerScopedUpsertRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		record, saveErr := store.SaveDraft(principal, req.ID, req.ScanID, req.Payload)
		if saveErr != nil {
			mapPersistenceError(w, saveErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("GET /api/v1/cpm/drafts", func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing principal"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		record, err := store.GetDraft(principal, id)
		if err != nil {
			mapPersistenceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("POST /api/v1/cpm/policies", func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing principal"})
			return
		}
		req, err := decodeOwnerScopedUpsertRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		record, saveErr := store.SavePolicy(principal, req.ID, req.ScanID, req.Payload)
		if saveErr != nil {
			mapPersistenceError(w, saveErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": record})
	})
	mux.HandleFunc("GET /api/v1/cpm/policies", func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing principal"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		record, err := store.GetPolicy(principal, id)
		if err != nil {
			mapPersistenceError(w, err)
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

func mapPersistenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, persistence.ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
	case errors.Is(err, persistence.ErrDraftNotFound), errors.Is(err, persistence.ErrPolicyNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	case errors.Is(err, persistence.ErrPrincipalRequired):
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing principal"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
