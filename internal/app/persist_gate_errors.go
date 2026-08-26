package app

import (
	"errors"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func mapWalletAuthorizationVerificationError(err error) (status int, code string, message string) {
	code = walletauth.Code(err)
	if code == "" {
		return http.StatusBadRequest, walletAuthCodeControlProofRequired, err.Error()
	}
	message = err.Error()
	switch code {
	case walletauth.CodeWalletControlProofRequired, walletauth.CodeWalletAuthorizationNotYetValid, walletauth.CodeWalletAuthorizationValidityLong,
		walletauth.CodePayloadSHA256Mismatch:
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
	case errors.Is(err, policy.ErrProviderScanCompatFailed):
		return http.StatusBadRequest, persistCodeProviderScanCompatFailed, message
	case errors.Is(err, policy.ErrProviderUserConstraintsIncompatible):
		return http.StatusBadRequest, persistCodeProviderUserConstraintsIncompatible, message
	case errors.Is(err, policy.ErrAcceptedFindingsDivergent):
		return http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, message
	case errors.Is(err, policy.ErrCryptoPolicyPayloadInvalid):
		return http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, message
	default:
		return http.StatusBadRequest, persistCodeCryptoPolicyPayloadInvalid, message
	}
}

func cryptoPolicyIDFromDraftPayload(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload["crypto_policy_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
