package walletauth

import "time"

const (
	// ActionPersistCryptoPolicy is the only supported CP-PERSIST V1 authorization action.
	ActionPersistCryptoPolicy = "persist_crypto_policy"

	messageTitle = "CAFE Crypto Policy Persistence"
	messageFooter = "By signing this message, I prove control of the wallet and authorize CAFE to persist the selected Crypto Policy draft for this wallet."

	// MaxValidityWindow is the maximum allowed signed message lifetime (CP_PERSIST.md §12).
	MaxValidityWindow = 10 * time.Minute

	// DefaultClockSkew is the recommended allowed future skew for issued_at (CP_PERSIST.md §12).
	DefaultClockSkew = 30 * time.Second
)
