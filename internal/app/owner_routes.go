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

const (
	ownerDraftsPath   = "/drafts"
	ownerPoliciesPath = "/policies"
)

type ownerScopedUpsertRequest struct {
	ID          string         `json:"id"`
	ScanID      string         `json:"scan_id"`
	Binding     string         `json:"binding,omitempty"`
	Payload     map[string]any `json:"payload"`
	OwnerUserID string         `json:"owner_user_id,omitempty"`
	TenantID    string         `json:"tenant_id,omitempty"`
}

func registerOwnerScopedRoutes(mux *http.ServeMux, store *persistence.OwnerScopedStore, obs *authObservability) {
	if obs == nil {
		obs = newAuthObservability()
	}
	registerOwnerScopedRoutesForPrefix(mux, cpmroutes.V1Base, store, obs)
}

func registerOwnerScopedRoutesForPrefix(mux *http.ServeMux, base string, store *persistence.OwnerScopedStore, obs *authObservability) {
	mux.HandleFunc("POST "+base+ownerDraftsPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerPOSTDrafts(w, r, store, obs)
	})
	mux.HandleFunc("GET "+base+ownerDraftsPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerGETDrafts(w, r, store, obs)
	})
	mux.HandleFunc("DELETE "+base+ownerDraftsPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerDELETEDrafts(w, r, store, obs)
	})
	registerOwnerWalletTargetContextRoute(mux, base, store, obs)
	mux.HandleFunc("POST "+base+ownerPoliciesPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerPOSTPolicies(w, r, store, obs)
	})
	mux.HandleFunc("GET "+base+ownerPoliciesPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerGETPolicies(w, r, store, obs)
	})
	mux.HandleFunc("DELETE "+base+ownerPoliciesPath, func(w http.ResponseWriter, r *http.Request) {
		handleOwnerDELETEPolicies(w, r, store, obs)
	})
}

func handleOwnerPOSTDrafts(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	req, err := decodeDraftUpsertRequest(r)
	if err != nil {
		var decErr *draftDecodeError
		if errors.As(err, &decErr) {
			writeDraftStructuredError(w, r, obs, http.StatusBadRequest, decErr.code, decErr.message, map[string]any{})
			return
		}
		writeDraftStructuredError(w, r, obs, http.StatusBadRequest, draftCodePayloadRequired, err.Error(), map[string]any{})
		return
	}
	record, saveErr := store.SaveDraft(principal, req.ID, req.ScanID, req.Payload)
	if saveErr != nil {
		mapDraftPersistenceError(w, r, obs, principal, saveErr)
		return
	}
	writeJSON(w, http.StatusOK, draftUpsertResponseFromRecord(record))
}

func handleOwnerGETDrafts(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeDraftStructuredError(w, r, obs, http.StatusBadRequest, draftCodeIDRequired, "id is required", map[string]any{})
		return
	}
	record, err := store.GetDraft(principal, id)
	if err != nil {
		mapDraftPersistenceError(w, r, obs, principal, err)
		return
	}
	writeJSON(w, http.StatusOK, draftRecordResponseFromStore(record))
}

func handleOwnerDELETEDrafts(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeDraftStructuredError(w, r, obs, http.StatusBadRequest, draftCodeIDRequired, "id is required", map[string]any{})
		return
	}
	err := store.DeleteDraft(principal, id)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, persistence.ErrDraftNotFound), errors.Is(err, persistence.ErrForbidden):
		writeDraftStructuredError(w, r, obs, http.StatusNotFound, draftCodeNotFound, "draft not found", map[string]any{})
	case errors.Is(err, persistence.ErrPrincipalRequired):
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
	default:
		writeDraftStructuredError(w, r, obs, http.StatusInternalServerError, draftCodeInternalError, errMsgInternalServerError, map[string]any{})
	}
}

func handleOwnerPOSTPolicies(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	req, err := decodeOwnerScopedUpsertRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: err.Error()})
		return
	}
	if policyPayloadRequiresEOAWalletProof(req.Payload) {
		writeWalletAuthorizationError(w, r, obs, http.StatusForbidden, walletAuthCodeControlProofRequired, errMsgWalletControlProofRequired)
		return
	}
	scanIDToStore, err := validatePolicyPersistBinding(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{jsonKeyError: err.Error()})
		return
	}
	record, saveErr := store.SavePolicy(principal, req.ID, scanIDToStore, req.Payload)
	if saveErr != nil {
		mapPersistenceError(w, r, obs, principal, saveErr)
		return
	}
	writeJSON(w, http.StatusOK, policyRecordResponseFromStore(record))
}

func validatePolicyPersistBinding(req ownerScopedUpsertRequest) (scanIDToStore string, err error) {
	binding := strings.ToLower(strings.TrimSpace(req.Binding))
	if binding == "" {
		binding = "discovery"
	}
	switch binding {
	case "discovery":
		norm, err := NormalizeDiscoveryScanID(req.ScanID)
		if err != nil {
			return "", errors.New("scan_id is required and must be a valid UUID for binding=discovery (Discovery→CPM flows)")
		}
		return norm, nil
	case "fixture", "catalog", "none":
		raw := strings.TrimSpace(req.ScanID)
		if raw == "" {
			return "", nil
		}
		norm, err := NormalizeDiscoveryScanID(raw)
		if err != nil {
			return "", errors.New("scan_id must be a valid UUID when provided")
		}
		return norm, nil
	default:
		return "", errors.New("binding must be one of: discovery, fixture, catalog, none")
	}
}

func handleOwnerGETPolicies(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
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

func handleOwnerDELETEPolicies(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, obs *authObservability) {
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

func decodeOwnerScopedUpsertRequest(r *http.Request) (ownerScopedUpsertRequest, error) {
	var req ownerScopedUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return ownerScopedUpsertRequest{}, err
	}
	if strings.TrimSpace(req.ID) == "" {
		return ownerScopedUpsertRequest{}, errors.New(errMsgIDRequired)
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
