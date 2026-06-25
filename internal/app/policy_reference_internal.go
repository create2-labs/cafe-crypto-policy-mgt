package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

type policyScanReferenceRequest struct {
	ScanID   string `json:"scan_id"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// registerPolicyReferenceInternalRoute registers POST /internal/policies/references/scan
// (WORKPLAN_API_PR.md PR5). Caller must gate the mux with service-token auth for this path
// when CPM_AUTH_REQUIRED is true.
func registerPolicyReferenceInternalRoute(mux *http.ServeMux, store persistence.PolicyStore) {
	mux.HandleFunc("POST "+cpmroutes.InternalPolicyReferenceScan, func(w http.ResponseWriter, r *http.Request) {
		const maxBody = 1 << 16
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "could not read request body"})
			return
		}
		if len(body) > maxBody {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "request body too large"})
			return
		}
		var req policyScanReferenceRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "invalid request body"})
			return
		}
		scanID, err := NormalizeDiscoveryScanID(req.ScanID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "scan_id must be a valid UUID"})
			return
		}
		userID := strings.TrimSpace(req.UserID)
		if userID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "user_id is required"})
			return
		}
		principal := authz.Principal{
			UserID:   userID,
			Subject:  userID,
			TenantID: strings.TrimSpace(req.TenantID),
		}
		if err := principal.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "invalid principal"})
			return
		}
		list, err := store.ListPersistedPoliciesForScan(principal, scanID)
		if err != nil {
			if errors.Is(err, persistence.ErrPrincipalRequired) {
				writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: err.Error()})
				return
			}
			if errors.Is(err, persistence.ErrPersistenceUnavailable) {
				writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodePersistenceUnavailable, "persistence is temporarily unavailable"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{jsonKeyError: errMsgInternalServerError})
			return
		}
		count := len(list)
		writeJSON(w, http.StatusOK, map[string]any{
			"referenced": count > 0,
			"count":      count,
		})
	})
}
