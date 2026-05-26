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

type policyWalletTargetReferenceRequest struct {
	TargetAddress string `json:"target_address"`
	UserID        string `json:"user_id"`
	TenantID      string `json:"tenant_id"`
}

// registerPolicyWalletTargetReferenceInternalRoute registers POST /internal/policies/references/wallet-target
// (IMM-9b) for Discovery POST /scan W1 guard.
func registerPolicyWalletTargetReferenceInternalRoute(mux *http.ServeMux, store *persistence.OwnerScopedStore) {
	mux.HandleFunc("POST "+cpmroutes.InternalPolicyReferenceWalletTarget, func(w http.ResponseWriter, r *http.Request) {
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
		var req policyWalletTargetReferenceRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		userID := strings.TrimSpace(req.UserID)
		if userID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "user_id is required"})
			return
		}
		norm, err := persistence.NormalizeWalletTargetAddress(req.TargetAddress)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
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
		counts, err := store.CountActiveWalletCPMContext(principal, norm)
		if err != nil {
			if errors.Is(err, persistence.ErrPrincipalRequired) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"exists":        counts.Exists,
			"policy_count":  counts.PolicyCount,
			"draft_count":   counts.DraftCount,
		})
	})
}
