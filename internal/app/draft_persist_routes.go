package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

type draftPersistRequest struct {
	WalletAddress string `json:"wallet_address"`
	ChainID       int64  `json:"chain_id"`
	ScanID        string `json:"scan_id"`
	SignedMessage string `json:"signed_message"`
	Signature     string `json:"signature"`
}

type draftPersistResponse struct {
	PolicyID            string `json:"policy_id"`
	DraftID             string `json:"draft_id"`
	ScanID              string `json:"scan_id"`
	WalletAddress       string `json:"wallet_address"`
	ChainID             int64  `json:"chain_id"`
	Status              string `json:"status"`
	OwnershipStatus     string `json:"ownership_status"`
	WalletControlMethod string `json:"wallet_control_method"`
	PersistedAt         string `json:"persisted_at"`
}

func registerDraftPersistRoutes(mux *http.ServeMux, store persistence.PolicyStore, cfg authConfig, obs *authObservability) {
	if !cpmroutes.PathMatches(cpmroutes.DraftPersist, cpmroutes.DraftPersistPath("_")) {
		panic("cpmroutes: draft persist path pattern mismatch")
	}
	mux.HandleFunc("POST "+cpmroutes.DraftPersist, func(w http.ResponseWriter, r *http.Request) {
		handleDraftPersist(w, r, store, cfg, obs)
	})
}

func handleDraftPersist(w http.ResponseWriter, r *http.Request, store persistence.PolicyStore, cfg authConfig, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}

	draftID := strings.TrimSpace(r.PathValue("draft_id"))
	if draftID == "" {
		writeWalletAuthorizationError(w, r, obs, http.StatusNotFound, walletAuthCodeDraftNotFound, "draft not found")
		return
	}

	req, decErr := decodeDraftPersistRequest(r)
	if decErr != nil {
		writeWalletAuthorizationError(w, r, obs, decErr.status, decErr.code, decErr.message)
		return
	}

	if strings.TrimSpace(req.SignedMessage) == "" || strings.TrimSpace(req.Signature) == "" {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, walletAuthCodeControlProofRequired, errMsgWalletControlProofRequired)
		return
	}

	normWallet, normErr := persistence.NormalizeWalletTargetAddress(req.WalletAddress)
	if normErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, walletAuthCodeInvalidAddress, "wallet address is invalid")
		return
	}
	normScanID, scanNormErr := NormalizeDiscoveryScanID(req.ScanID)
	if scanNormErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusNotFound, walletAuthCodeScanNotFound, errMsgScanNotFound)
		return
	}
	if statusErr := store.DraftPersistStatus(principal, draftID); errors.Is(statusErr, persistence.ErrDraftAlreadyPersisted) {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftAlreadyPersisted, "draft has already been persisted")
		return
	}

	draft, draftErr := store.GetDraft(principal, draftID)
	switch {
	case errors.Is(draftErr, persistence.ErrDraftNotFound), errors.Is(draftErr, persistence.ErrForbidden):
		writeWalletAuthorizationError(w, r, obs, http.StatusNotFound, walletAuthCodeDraftNotFound, "draft not found")
		return
	case draftErr != nil:
		if errors.Is(draftErr, persistence.ErrPersistenceUnavailable) {
			writeWalletAuthorizationError(w, r, obs, http.StatusServiceUnavailable, errCodePersistenceUnavailable, "persistence is temporarily unavailable")
			return
		}
		writeWalletAuthorizationError(w, r, obs, http.StatusInternalServerError, errCodeInternalError, errMsgInternalServerError)
		return
	}

	if walletType := draftWalletType(draft.Payload); walletType != "" && !strings.EqualFold(walletType, "eoa") {
		writeWalletAuthorizationError(w, r, obs, http.StatusUnprocessableEntity, walletAuthCodeUnsupportedWallet, "only EOA wallets are supported for CP persistence in V1")
		return
	}

	draftWallet := walletAddressFromDraftPayload(draft.Payload)
	if draftWallet == "" || !walletAddressesEqual(draftWallet, normWallet) {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftWalletMismatch, "draft wallet does not match requested wallet")
		return
	}
	draftScan := strings.TrimSpace(draft.ScanID)
	if draftScan == "" || !strings.EqualFold(draftScan, normScanID) {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftScanMismatch, "draft scan does not match requested scan")
		return
	}

	// CPM-P6: ADR §9 persist gate (snapshot + pinned refs + soft findings). Store stays opaque.
	if gateErr := policy.ValidateDraftPayloadForPersist(draft.Payload); gateErr != nil {
		status, code, message := mapPersistProviderGateError(gateErr)
		writeWalletAuthorizationError(w, r, obs, status, code, message)
		return
	}

	if scanErr := ensureWalletScanExists(r, cfg, requestID, normScanID); scanErr != nil {
		writeWalletAuthorizationError(w, r, obs, scanErr.status, scanErr.code, scanErr.message)
		return
	}

	skew := time.Duration(cfg.ClockSkewSec) * time.Second
	if skew <= 0 {
		skew = walletauth.DefaultClockSkew
	}
	verifyErr := walletauth.VerifyAuthorization(walletauth.VerifyInput{
		Domain:        walletAuthDomain(r, cfg),
		WalletAddress: normWallet,
		ChainID:       req.ChainID,
		ScanID:        normScanID,
		DraftID:       draftID,
		SignedMessage: req.SignedMessage,
		Signature:     req.Signature,
		Now:           time.Now().UTC(),
		ClockSkew:     skew,
	})
	if verifyErr != nil {
		status, code, message := mapWalletAuthorizationVerificationError(verifyErr)
		writeWalletAuthorizationError(w, r, obs, status, code, message)
		return
	}

	now := time.Now().UTC()
	result, persistErr := store.PersistDraftOnce(principal, draftID, persistence.PersistDraftInput{
		WalletAddress: normWallet,
		ChainID:       req.ChainID,
		VerifiedAt:    now,
	})
	switch {
	case errors.Is(persistErr, persistence.ErrDraftAlreadyPersisted):
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftAlreadyPersisted, "draft has already been persisted")
		return
	case errors.Is(persistErr, persistence.ErrDraftNotFound), errors.Is(persistErr, persistence.ErrForbidden):
		writeWalletAuthorizationError(w, r, obs, http.StatusNotFound, walletAuthCodeDraftNotFound, "draft not found")
		return
	case errors.Is(persistErr, persistence.ErrPersistenceUnavailable):
		writeWalletAuthorizationError(w, r, obs, http.StatusServiceUnavailable, errCodePersistenceUnavailable, "persistence is temporarily unavailable")
		return
	case persistErr != nil:
		writeWalletAuthorizationError(w, r, obs, http.StatusInternalServerError, errCodeInternalError, errMsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, draftPersistResponse{
		PolicyID:            result.PolicyID,
		DraftID:             result.DraftID,
		ScanID:              result.ScanID,
		WalletAddress:       result.WalletAddress,
		ChainID:             result.ChainID,
		Status:              "persisted",
		OwnershipStatus:     "verified",
		WalletControlMethod: "eoa_signature",
		PersistedAt:         result.PersistedAt.UTC().Format(time.RFC3339),
	})
}

type draftPersistDecodeError struct {
	code    string
	message string
	status  int
}

func decodeDraftPersistRequest(r *http.Request) (draftPersistRequest, *draftPersistDecodeError) {
	var req draftPersistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return draftPersistRequest{}, &draftPersistDecodeError{
			code:    walletAuthCodeControlProofRequired,
			message: "request body must be a JSON object",
			status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(req.WalletAddress) == "" {
		return draftPersistRequest{}, &draftPersistDecodeError{
			code:    walletAuthCodeAddressRequired,
			message: "wallet_address is required",
			status:  http.StatusBadRequest,
		}
	}
	if req.ChainID < 1 {
		return draftPersistRequest{}, &draftPersistDecodeError{
			code:    walletAuthCodeChainIDRequired,
			message: "chain_id is required",
			status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(req.ScanID) == "" {
		return draftPersistRequest{}, &draftPersistDecodeError{
			code:    walletAuthCodeScanNotFound,
			message: "scan_id is required",
			status:  http.StatusBadRequest,
		}
	}
	return req, nil
}

func mapWalletAuthorizationVerificationError(err error) (status int, code string, message string) {
	code = walletauth.Code(err)
	if code == "" {
		return http.StatusBadRequest, walletAuthCodeControlProofRequired, err.Error()
	}
	message = err.Error()
	switch code {
	case walletauth.CodeWalletControlProofRequired, walletauth.CodeWalletAuthorizationNotYetValid, walletauth.CodeWalletAuthorizationValidityLong:
		return http.StatusBadRequest, code, message
	case walletauth.CodeInvalidWalletSignature, walletauth.CodeWalletSignatureAddressMismatch:
		return http.StatusUnauthorized, code, message
	case walletauth.CodeWalletAuthorizationExpired:
		return http.StatusGone, code, message
	case walletauth.CodeWalletAuthorizationDraftMismatch, walletauth.CodeWalletAuthorizationScanMismatch,
		walletauth.CodeWalletAuthorizationWalletMismatch, walletauth.CodeWalletAuthorizationChainMismatch,
		walletauth.CodeWalletAuthorizationActionMismatch:
		return http.StatusConflict, code, message
	default:
		return http.StatusBadRequest, code, message
	}
}

func mapPersistProviderGateError(err error) (status int, code string, message string) {
	message = err.Error()
	switch {
	case errors.Is(err, policy.ErrProviderRefsUnpinned):
		return http.StatusBadRequest, persistCodeProviderRefsUnpinned, message
	case errors.Is(err, policy.ErrProviderSoftFindingsRequired):
		return http.StatusBadRequest, persistCodeProviderSoftFindingsRequired, message
	case errors.Is(err, policy.ErrProviderChainPlanned):
		return http.StatusBadRequest, persistCodeProviderChainPlanned, message
	case errors.Is(err, policy.ErrCryptoPolicyPayloadInvalid):
		return http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, message
	default:
		return http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, message
	}
}
