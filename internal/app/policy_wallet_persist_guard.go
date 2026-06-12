package app

import "strings"

// policyPayloadRequiresEOAWalletProof reports whether POST /policies would persist an EOA wallet CP
// without the normative signed-authorization flow (CP-PERSIST V1).
func policyPayloadRequiresEOAWalletProof(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	if swc, ok := payload["selected_wallet_policy_context"].(map[string]any); ok && swc != nil {
		// Legacy CLI-like Discovery→explore→POST /policies flows always carry this block.
		return true
	}
	walletType := draftWalletType(payload)
	if walletType != "" {
		return strings.EqualFold(walletType, "eoa")
	}
	return walletAddressFromDraftPayload(payload) != ""
}
