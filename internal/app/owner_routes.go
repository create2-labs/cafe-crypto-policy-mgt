package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

const ownerPoliciesPath = "/policies"

func registerOwnerScopedRoutes(mux *http.ServeMux, store persistence.PolicyStore, obs *authObservability) {
	if obs == nil {
		obs = newAuthObservability()
	}
	registerOwnerScopedRoutesForPrefix(mux, cpmroutes.V1Base, store, obs)
}

func registerOwnerScopedRoutesForPrefix(mux *http.ServeMux, base string, store persistence.PolicyStore, obs *authObservability) {
	// RD-P5: public /drafts* removed. POST /policies is registered via registerPolicyPersistRoute.
	registerOwnerWalletTargetContextRoute(mux, base, store, obs)
	mux.HandleFunc("GET "+base+ownerPoliciesPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerGETPolicies(w, r, store, obs)
	})
	mux.HandleFunc("DELETE "+base+ownerPoliciesPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerDELETEPolicies(w, r, store, obs)
	})
}

func handleOwnerGETPolicies(w http.ResponseWriter, r *http.Request, store persistence.PolicyStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	scanID := strings.TrimSpace(r.URL.Query().Get("scan_id"))
	if id != "" && scanID != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "id and scan_id are mutually exclusive"})
		return
	}
	if id == "" && scanID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "exactly one of id or scan_id is required"})
		return
	}
	if id != "" {
		record, err := store.GetPolicy(principal, id)
		if err != nil {
			mapPersistenceError(w, r, obs, principal, err)
			return
		}
		writeJSON(w, http.StatusOK, policyRecordResponseFromStore(record))
		return
	}
	norm, err := NormalizeDiscoveryScanID(scanID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: "scan_id must be a valid UUID"})
		return
	}
	list, err := store.ListPersistedPoliciesForScan(principal, norm)
	if err != nil {
		if errors.Is(err, persistence.ErrPrincipalRequired) {
			writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{jsonKeyError: errMsgInternalServerError})
		return
	}
	total := len(list)
	limit := total
	if limit < 1 {
		limit = 1
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  policyRecordResponsesFromStore(list),
		"total":  total,
		"limit":  limit,
		"offset": 0,
	})
}

func handleOwnerDELETEPolicies(w http.ResponseWriter, r *http.Request, store persistence.PolicyStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: errMsgIDRequired})
		return
	}
	// Optional reason is best-effort audit only (not stored in persistence).
	logDeleteReasonBestEffort(r.URL.Query().Get("reason"), id, principal.UserID)
	err := store.DeletePolicy(principal, id)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, persistence.ErrPolicyNotFound), errors.Is(err, persistence.ErrForbidden):
		writeJSON(w, http.StatusNotFound, map[string]any{jsonKeyError: errMsgNotFound})
	case errors.Is(err, persistence.ErrPrincipalRequired):
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{jsonKeyError: errMsgInternalServerError})
	}
}

func mapPersistenceError(w http.ResponseWriter, r *http.Request, obs *authObservability, principal authz.Principal, err error) {
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
		obs.writeAuthError(w, r, http.StatusForbidden, authCodeOwnerForbidden, "owner access denied", reasonDetails("owner_forbidden"))
	case errors.Is(err, persistence.ErrDraftNotFound), errors.Is(err, persistence.ErrPolicyNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{jsonKeyError: errMsgNotFound})
	case errors.Is(err, persistence.ErrPersistenceUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodePersistenceUnavailable, "persistence is temporarily unavailable"))
	case errors.Is(err, persistence.ErrUnsupportedStoreOperation):
		writeJSON(w, http.StatusUnprocessableEntity, apiErrorJSON(errCodeInternalError, "operation is not supported when CPM_STORE=persistence"))
	case errors.Is(err, persistence.ErrPrincipalRequired):
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{jsonKeyError: errMsgInternalServerError})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
