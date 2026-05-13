package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
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
// scanUUIDPattern matches canonical lowercase UUID strings (Discovery scan_id).
var scanUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func registerPolicyReferenceInternalRoute(mux *http.ServeMux, store *persistence.OwnerScopedStore) {
	mux.HandleFunc("POST /internal/policies/references/scan", func(w http.ResponseWriter, r *http.Request) {
		const maxBody = 1 << 16
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "could not read request body"})
			return
		}
		if len(body) > maxBody {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request body too large"})
			return
		}
		var req policyScanReferenceRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		scanID := strings.TrimSpace(req.ScanID)
		if !scanUUIDPattern.MatchString(scanID) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scan_id must be a valid UUID"})
			return
		}
		scanID = strings.ToLower(scanID)
		userID := strings.TrimSpace(req.UserID)
		if userID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "user_id is required"})
			return
		}
		principal := authz.Principal{
			UserID:   userID,
			Subject:  userID,
			TenantID: strings.TrimSpace(req.TenantID),
		}
		if err := principal.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid principal"})
			return
		}
		count, err := store.CountPersistedPoliciesForScan(principal, scanID)
		if err != nil {
			if errors.Is(err, persistence.ErrPrincipalRequired) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"referenced": count > 0,
			"count":      count,
		})
	})
}
