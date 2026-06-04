package app

import (
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func registerOwnerWalletTargetContextRoute(mux *http.ServeMux, base string, store *persistence.OwnerScopedStore, obs *authObservability) {
	mux.HandleFunc("GET "+base+"/wallet-target-context", func(w http.ResponseWriter, r *http.Request) {
		handleOwnerGETWalletTargetContext(w, r, store, obs)
	})
}

func handleOwnerGETWalletTargetContext(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("target_address"))
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "TARGET_ADDRESS_REQUIRED",
			"message": "target_address query parameter is required",
		})
		return
	}
	norm, normErr := persistence.NormalizeWalletTargetAddress(raw)
	if normErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "TARGET_ADDRESS_INVALID",
			"message": normErr.Error(),
		})
		return
	}
	counts, err := store.CountActiveWalletCPMContext(principal, norm)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{jsonKeyError: errMsgInternalServerError})
		return
	}
	body := map[string]any{
		"exists":         counts.Exists,
		"policy_count":   counts.PolicyCount,
		"draft_count":    counts.DraftCount,
		"target_address": norm,
	}
	if counts.PlatformDraftID != "" {
		body["platform_draft_id"] = counts.PlatformDraftID
	}
	writeJSON(w, http.StatusOK, body)
}
