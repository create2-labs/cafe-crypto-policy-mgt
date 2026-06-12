package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

type walletChallengeRequest struct {
	WalletAddress string `json:"wallet_address"`
	ChainID       int64  `json:"chain_id"`
	ScanID        string `json:"scan_id"`
	DraftID       string `json:"draft_id"`
	Action        string `json:"action"`
}

type walletChallengeResponse struct {
	Message       string `json:"message"`
	WalletAddress string `json:"wallet_address"`
	ChainID       int64  `json:"chain_id"`
	ScanID        string `json:"scan_id"`
	DraftID       string `json:"draft_id"`
	Action        string `json:"action"`
	IssuedAt      string `json:"issued_at"`
	ExpiresAt     string `json:"expires_at"`
}

func registerWalletChallengeRoutes(mux *http.ServeMux, store *persistence.OwnerScopedStore, cfg authConfig, obs *authObservability) {
	mux.HandleFunc("POST "+cpmroutes.WalletChallenges, func(w http.ResponseWriter, r *http.Request) {
		handleWalletChallenge(w, r, store, cfg, obs)
	})
}

func handleWalletChallenge(w http.ResponseWriter, r *http.Request, store *persistence.OwnerScopedStore, cfg authConfig, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}

	req, err := decodeWalletChallengeRequest(r)
	if err != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, err.code, err.message)
		return
	}

	normWallet, normErr := persistence.NormalizeWalletTargetAddress(req.WalletAddress)
	if normErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, walletAuthCodeInvalidAddress, "wallet address is invalid")
		return
	}
	normScanID, scanNormErr := NormalizeDiscoveryScanID(req.ScanID)
	if scanNormErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, walletAuthCodeScanNotFound, errMsgScanNotFound)
		return
	}

	draft, draftErr := store.GetDraft(principal, req.DraftID)
	switch {
	case errors.Is(draftErr, persistence.ErrDraftNotFound), errors.Is(draftErr, persistence.ErrForbidden):
		writeWalletAuthorizationError(w, r, obs, http.StatusNotFound, walletAuthCodeDraftNotFound, "draft not found")
		return
	case draftErr != nil:
		writeWalletAuthorizationError(w, r, obs, http.StatusInternalServerError, errCodeInternalError, errMsgInternalServerError)
		return
	}

	if walletType := draftWalletType(draft.Payload); walletType != "" && !strings.EqualFold(walletType, "eoa") {
		writeWalletAuthorizationError(w, r, obs, http.StatusUnprocessableEntity, walletAuthCodeUnsupportedWallet, "only EOA wallets are supported for CP persistence in V1")
		return
	}

	draftWallet := walletAddressFromDraftPayload(draft.Payload)
	if draftWallet == "" {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftWalletMismatch, "draft wallet does not match requested wallet")
		return
	}
	if !walletAddressesEqual(draftWallet, normWallet) {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftWalletMismatch, "draft wallet does not match requested wallet")
		return
	}

	draftScan := strings.TrimSpace(draft.ScanID)
	if draftScan == "" {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftScanMismatch, "draft scan does not match requested scan")
		return
	}
	if !strings.EqualFold(draftScan, normScanID) {
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodeDraftScanMismatch, "draft scan does not match requested scan")
		return
	}

	if scanErr := ensureWalletScanExists(r, cfg, requestID, normScanID); scanErr != nil {
		writeWalletAuthorizationError(w, r, obs, scanErr.status, scanErr.code, scanErr.message)
		return
	}

	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(walletauth.MaxValidityWindow)
	domain := walletAuthDomain(r, cfg)
	message := walletauth.BuildMessage(walletauth.Fields{
		Domain:        domain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: normWallet,
		ChainID:       req.ChainID,
		ScanID:        normScanID,
		DraftID:       strings.TrimSpace(req.DraftID),
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
	})

	writeJSON(w, http.StatusOK, walletChallengeResponse{
		Message:       message,
		WalletAddress: normWallet,
		ChainID:       req.ChainID,
		ScanID:        normScanID,
		DraftID:       strings.TrimSpace(req.DraftID),
		Action:        walletauth.ActionPersistCryptoPolicy,
		IssuedAt:      issuedAt.Format(time.RFC3339),
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	})
}

type walletChallengeDecodeError struct {
	code    string
	message string
}

func decodeWalletChallengeRequest(r *http.Request) (walletChallengeRequest, *walletChallengeDecodeError) {
	var req walletChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeAddressRequired,
			message: "request body must be a JSON object",
		}
	}
	if strings.TrimSpace(req.WalletAddress) == "" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeAddressRequired,
			message: "wallet_address is required",
		}
	}
	if req.ChainID < 1 {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeChainIDRequired,
			message: "chain_id is required",
		}
	}
	if strings.TrimSpace(req.DraftID) == "" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeDraftNotFound,
			message: "draft_id is required",
		}
	}
	if strings.TrimSpace(req.ScanID) == "" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeScanNotFound,
			message: "scan_id is required",
		}
	}
	if action := strings.TrimSpace(req.Action); action == "" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeActionRequired,
			message: "action is required",
		}
	} else if action != walletauth.ActionPersistCryptoPolicy {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeActionRequired,
			message: "unsupported authorization action",
		}
	}
	return req, nil
}

type walletScanLookupError struct {
	code    string
	message string
	status  int
}

func ensureWalletScanExists(r *http.Request, cfg authConfig, requestID, scanID string) *walletScanLookupError {
	if strings.TrimSpace(cfg.DiscoveryHTTPBaseURL) == "" {
		return nil
	}
	authzHeader := ""
	if r != nil {
		authzHeader = r.Header.Get("Authorization")
	}
	_, st, err := fetchDiscoveryWalletScanDetail(r.Context(), cfg, authzHeader, requestID, scanID)
	if err != nil {
		return &walletScanLookupError{
			code:    errCodeDiscoveryUpstreamUnavailable,
			message: err.Error(),
			status:  http.StatusServiceUnavailable,
		}
	}
	switch st {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &walletScanLookupError{
			code:    walletAuthCodeScanNotFound,
			message: errMsgScanNotFound,
			status:  http.StatusNotFound,
		}
	default:
		return &walletScanLookupError{
			code:    errCodeDiscoveryUpstreamUnavailable,
			message: "discovery wallet scan lookup failed",
			status:  http.StatusServiceUnavailable,
		}
	}
}

func walletAuthDomain(r *http.Request, cfg authConfig) string {
	if domain := strings.TrimSpace(cfg.WalletAuthDomain); domain != "" {
		return domain
	}
	if r != nil {
		if host := strings.TrimSpace(r.Host); host != "" {
			if i := strings.Index(host, ":"); i > 0 {
				return host[:i]
			}
			return host
		}
	}
	return "localhost"
}

func walletAddressFromDraftPayload(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if pc, ok := payload["policy_context"].(map[string]any); ok {
		if addr := firstWalletAddressField(pc); addr != "" {
			return addr
		}
	}
	if swc, ok := payload["selected_wallet_policy_context"].(map[string]any); ok {
		if addr := firstWalletAddressField(swc); addr != "" {
			return addr
		}
	}
	return firstWalletAddressField(payload)
}

func firstWalletAddressField(m map[string]any) string {
	for _, key := range []string{"target_address", "wallet_address", "walletAddress"} {
		if v, ok := m[key].(string); ok {
			if norm, err := persistence.NormalizeWalletTargetAddress(v); err == nil {
				return norm
			}
		}
	}
	if res, ok := m["result"].(map[string]any); ok {
		if v, ok := res["target_address"].(string); ok {
			if norm, err := persistence.NormalizeWalletTargetAddress(v); err == nil {
				return norm
			}
		}
	}
	return ""
}

func draftWalletType(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if pc, ok := payload["policy_context"].(map[string]any); ok {
		if wt, ok := pc["wallet_type"].(string); ok {
			return strings.TrimSpace(wt)
		}
	}
	return ""
}

func walletAddressesEqual(a, b string) bool {
	na, errA := persistence.NormalizeWalletTargetAddress(a)
	nb, errB := persistence.NormalizeWalletTargetAddress(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return na == nb
}
