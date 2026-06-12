package walletauth

import "errors"

// Verification error codes align with CP_PERSIST.md error semantics (persist-time checks).
const (
	CodeInvalidWalletSignature          = "INVALID_WALLET_SIGNATURE"
	CodeWalletSignatureAddressMismatch  = "WALLET_SIGNATURE_ADDRESS_MISMATCH"
	CodeWalletAuthorizationExpired      = "WALLET_AUTHORIZATION_EXPIRED"
	CodeWalletAuthorizationNotYetValid  = "WALLET_AUTHORIZATION_NOT_YET_VALID"
	CodeWalletAuthorizationValidityLong = "WALLET_AUTHORIZATION_VALIDITY_TOO_LONG"
	CodeWalletAuthorizationDraftMismatch  = "WALLET_AUTHORIZATION_DRAFT_MISMATCH"
	CodeWalletAuthorizationScanMismatch   = "WALLET_AUTHORIZATION_SCAN_MISMATCH"
	CodeWalletAuthorizationWalletMismatch = "WALLET_AUTHORIZATION_WALLET_MISMATCH"
	CodeWalletAuthorizationChainMismatch  = "WALLET_AUTHORIZATION_CHAIN_MISMATCH"
	CodeWalletAuthorizationActionMismatch = "WALLET_AUTHORIZATION_ACTION_MISMATCH"
)

// VerificationError carries a stable CP-PERSIST error code for authorization failures.
type VerificationError struct {
	Code    string
	Message string
}

func (e *VerificationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func verificationError(code, message string) error {
	return &VerificationError{Code: code, Message: message}
}

// Code returns the CP-PERSIST error code when err is a VerificationError.
func Code(err error) string {
	var ve *VerificationError
	if errors.As(err, &ve) && ve != nil {
		return ve.Code
	}
	return ""
}
