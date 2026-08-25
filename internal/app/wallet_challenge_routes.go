package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

type walletChallengeRequest struct {
	WalletAddress string          `json:"wallet_address"`
	ChainID       int64           `json:"chain_id"`
	ScanID        string          `json:"scan_id"`
	Action        string          `json:"action"`
	Payload       json.RawMessage `json:"payload"`
}

type walletChallengeResponse struct {
	Message       string `json:"message"`
	WalletAddress string `json:"wallet_address"`
	ChainID       int64  `json:"chain_id"`
	ScanID        string `json:"scan_id"`
	Action        string `json:"action"`
	PayloadSHA256 string `json:"payload_sha256"`
	IssuedAt      string `json:"issued_at"`
	ExpiresAt     string `json:"expires_at"`
}

// registerWalletChallengeRoutes registers POST /wallet-challenges.
// RD-P4: strictly stateless — hash payload via payloadhash, build canonical
// message, store nothing. No draft/policy store reads and no Discovery W2
// (W2 engagement gates land in RD-P5). Legacy /drafts* handlers may still
// exist elsewhere until RD-P5/P7.
func registerWalletChallengeRoutes(mux *http.ServeMux, cfg authConfig, obs *authObservability) {
	mux.HandleFunc("POST "+cpmroutes.WalletChallenges, func(w http.ResponseWriter, r *http.Request) {
		handleWalletChallenge(w, r, cfg, obs)
	})
}

func handleWalletChallenge(w http.ResponseWriter, r *http.Request, cfg authConfig, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	_, ok := principalFromContext(r.Context())
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

	digest, digestErr := payloadhash.DigestJSON(req.Payload)
	if digestErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, cryptoPolicyPayloadInvalidMessage(digestErr))
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
		PayloadSHA256: digest,
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
	})

	writeJSON(w, http.StatusOK, walletChallengeResponse{
		Message:       message,
		WalletAddress: normWallet,
		ChainID:       req.ChainID,
		ScanID:        normScanID,
		Action:        walletauth.ActionPersistCryptoPolicy,
		PayloadSHA256: digest,
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
	if strings.TrimSpace(req.ScanID) == "" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    walletAuthCodeScanNotFound,
			message: "scan_id is required",
		}
	}
	if len(req.Payload) == 0 || string(req.Payload) == "null" {
		return walletChallengeRequest{}, &walletChallengeDecodeError{
			code:    persistCodeCryptoPolicyPayloadInvalid,
			message: "payload is required",
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

func cryptoPolicyPayloadInvalidMessage(err error) string {
	var pe *payloadhash.Error
	if errors.As(err, &pe) && pe != nil && pe.Message != "" {
		return pe.Message
	}
	if err != nil && err.Error() != "" {
		return err.Error()
	}
	return "crypto policy payload is invalid"
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
