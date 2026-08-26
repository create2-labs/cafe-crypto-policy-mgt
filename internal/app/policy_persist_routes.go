package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
	"github.com/ethereum/go-ethereum/accounts"
)

const walletAuthCodePolicyAlreadyExists = "POLICY_ALREADY_EXISTS"

type policyPersistRequest struct {
	WalletAddress string          `json:"wallet_address"`
	ChainID       int64           `json:"chain_id"`
	ScanID        string          `json:"scan_id"`
	SignedMessage string          `json:"signed_message"`
	Signature     string          `json:"signature"`
	Payload       json.RawMessage `json:"payload"`
	PayloadSHA256 string          `json:"payload_sha256"` // ignored — server authority
}

type policyPersistResponse struct {
	PolicyID            string `json:"policy_id"`
	ScanID              string `json:"scan_id"`
	WalletAddress       string `json:"wallet_address"`
	ChainID             int64  `json:"chain_id"`
	Status              string `json:"status"`
	OwnershipStatus     string `json:"ownership_status"`
	WalletControlMethod string `json:"wallet_control_method"`
	PayloadSHA256       string `json:"payload_sha256"`
	PersistedAt         string `json:"persisted_at"`
}

func registerPolicyPersistRoute(mux *http.ServeMux, store persistence.PolicyStore, cfg authConfig, obs *authObservability) {
	mux.HandleFunc("POST "+cpmroutes.Policies, func(w http.ResponseWriter, r *http.Request) {
		handlePolicyPersist(w, r, store, cfg, obs)
	})
}

func handlePolicyPersist(w http.ResponseWriter, r *http.Request, store persistence.PolicyStore, cfg authConfig, obs *authObservability) {
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}

	req, decErr := decodePolicyPersistRequest(r)
	if decErr != nil {
		writeWalletAuthorizationError(w, r, obs, decErr.status, decErr.code, decErr.message)
		return
	}

	if strings.TrimSpace(req.SignedMessage) == "" || strings.TrimSpace(req.Signature) == "" {
		writeWalletAuthorizationError(w, r, obs, http.StatusForbidden, walletAuthCodeControlProofRequired, errMsgWalletControlProofRequired)
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

	if w2Err := ensureEngagementW2(r.Context(), r, cfg, requestID, normWallet, normScanID); w2Err != nil {
		writeWalletAuthorizationError(w, r, obs, w2Err.status, w2Err.code, w2Err.message)
		return
	}

	digest, canonical, digestErr := payloadhash.DigestCanonicalJSON(req.Payload)
	if digestErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, cryptoPolicyPayloadInvalidMessage(digestErr))
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
		PayloadSHA256: digest,
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

	// Signature OK — still enforce business rejeu A+B (signature ≠ bypass).
	if gateErr := policy.ValidateDraftPayloadForPersist(canonical); gateErr != nil {
		if errors.Is(gateErr, policy.ErrProviderUserConstraintsIncompatible) {
			cpID := cryptoPolicyIDFromDraftPayload(canonical)
			recordPersistUserConstraintsIncompatible(r, "", cpID, policy.UserConstraintsIncompatibleFindingCode(gateErr))
		}
		status, code, message := mapPersistProviderGateError(gateErr)
		writeWalletAuthorizationError(w, r, obs, status, code, message)
		return
	}

	parsed, parseErr := walletauth.ParseMessage(req.SignedMessage)
	if parseErr != nil {
		writeWalletAuthorizationError(w, r, obs, http.StatusBadRequest, walletAuthCodeControlProofRequired, errMsgWalletControlProofRequired)
		return
	}
	issuedAt := parsed.IssuedAt.UTC()
	expiresAt := parsed.ExpiresAt.UTC()
	now := time.Now().UTC()
	msgHash := hex.EncodeToString(accounts.TextHash([]byte(req.SignedMessage)))

	result, persistErr := store.CreatePolicy(principal, persistence.CreatePolicyInput{
		ScanID:                  normScanID,
		WalletAddress:           normWallet,
		ChainID:                 req.ChainID,
		Payload:                 canonical,
		PayloadSHA256:           digest,
		SignedMessageHash:       msgHash,
		WalletControlMethod:     "eoa_signature",
		WalletControlVerifiedAt: now,
		ChallengeIssuedAt:       &issuedAt,
		ChallengeExpiresAt:      &expiresAt,
	})
	switch {
	case errors.Is(persistErr, persistence.ErrPolicyAlreadyExists):
		writeWalletAuthorizationError(w, r, obs, http.StatusConflict, walletAuthCodePolicyAlreadyExists, "an active crypto policy already exists for this wallet")
		return
	case errors.Is(persistErr, persistence.ErrPersistenceUnavailable):
		writeWalletAuthorizationError(w, r, obs, http.StatusServiceUnavailable, errCodePersistenceUnavailable, "persistence is temporarily unavailable")
		return
	case persistErr != nil:
		writeWalletAuthorizationError(w, r, obs, http.StatusInternalServerError, errCodeInternalError, errMsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, policyPersistResponse{
		PolicyID:            result.PolicyID,
		ScanID:              result.ScanID,
		WalletAddress:       result.WalletAddress,
		ChainID:             result.ChainID,
		Status:              "persisted",
		OwnershipStatus:     "verified",
		WalletControlMethod: "eoa_signature",
		PayloadSHA256:       result.PayloadSHA256,
		PersistedAt:         result.PersistedAt.UTC().Format(time.RFC3339),
	})
}

type policyPersistDecodeError struct {
	code    string
	message string
	status  int
}

func decodePolicyPersistRequest(r *http.Request) (policyPersistRequest, *policyPersistDecodeError) {
	var req policyPersistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return policyPersistRequest{}, &policyPersistDecodeError{
			code:    walletAuthCodeControlProofRequired,
			message: "request body must be a JSON object",
			status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(req.WalletAddress) == "" {
		return policyPersistRequest{}, &policyPersistDecodeError{
			code:    walletAuthCodeAddressRequired,
			message: "wallet_address is required",
			status:  http.StatusBadRequest,
		}
	}
	if req.ChainID < 1 {
		return policyPersistRequest{}, &policyPersistDecodeError{
			code:    walletAuthCodeChainIDRequired,
			message: "chain_id is required",
			status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(req.ScanID) == "" {
		return policyPersistRequest{}, &policyPersistDecodeError{
			code:    walletAuthCodeScanNotFound,
			message: "scan_id is required",
			status:  http.StatusBadRequest,
		}
	}
	if len(req.Payload) == 0 || string(req.Payload) == "null" {
		return policyPersistRequest{}, &policyPersistDecodeError{
			code:    persistCodeCryptoPolicyPayloadInvalid,
			message: "payload is required",
			status:  http.StatusBadRequest,
		}
	}
	return req, nil
}

// logDeleteReasonBestEffort records an optional DELETE reason without affecting response.
func logDeleteReasonBestEffort(reason, policyID, userID string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	log.Printf("cpm: policy delete reason policy_id=%s user_id=%s reason=%q", strings.TrimSpace(policyID), strings.TrimSpace(userID), reason)
}
