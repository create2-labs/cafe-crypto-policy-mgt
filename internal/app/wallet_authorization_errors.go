package app

import "net/http"

const (
	walletAuthCodeAddressRequired     = "WALLET_ADDRESS_REQUIRED"
	walletAuthCodeInvalidAddress      = "INVALID_WALLET_ADDRESS"
	walletAuthCodeChainIDRequired     = "CHAIN_ID_REQUIRED"
	walletAuthCodeDraftNotFound       = "DRAFT_NOT_FOUND"
	walletAuthCodeScanNotFound        = "SCAN_NOT_FOUND"
	walletAuthCodeDraftScanMismatch   = "DRAFT_SCAN_MISMATCH"
	walletAuthCodeDraftWalletMismatch = "DRAFT_WALLET_MISMATCH"
	walletAuthCodeUnsupportedWallet   = "UNSUPPORTED_WALLET_TYPE"
	walletAuthCodeActionRequired        = "WALLET_AUTHORIZATION_ACTION_MISMATCH"
	walletAuthCodeControlProofRequired  = "WALLET_CONTROL_PROOF_REQUIRED"
	walletAuthCodeDraftAlreadyPersisted = "DRAFT_ALREADY_PERSISTED"
)

func writeWalletAuthorizationError(w http.ResponseWriter, r *http.Request, obs *authObservability, status int, code string, message string) {
	obs.writeAuthError(w, r, status, code, message, map[string]any{})
}
