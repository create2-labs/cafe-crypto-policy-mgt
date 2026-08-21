package app

const (
	jsonKeyError      = "error"
	jsonKeyMessage    = "message"
	jsonKeyReason     = "reason"
	jsonKeyDetails    = "details"
	jsonKeyCode       = "code"
	jsonKeyRequestID  = "request_id"

	errMsgAuthenticationRequired       = "authentication required"
	errMsgIDRequired                   = "id is required"
	errMsgNotFound                     = "not found"
	errMsgInternalServerError          = "internal server error"
	errMsgWalletControlProofRequired   = "Persisting a Crypto Policy for a wallet requires a valid signed wallet authorization."
	errMsgScanNotFound                 = "scan not found"
	errMsgScanAuthorizationUnavailable = "scan authorization unavailable"

	errCodeNotFound                     = "not_found"
	errCodeDiscoveryUnavailable         = "discovery_unavailable"
	errCodeInternalError                = "internal_error"
	errCodeDiscoveryUpstreamUnavailable = "DISCOVERY_UPSTREAM_UNAVAILABLE"
	errCodePersistenceUnavailable       = "PERSISTENCE_UNAVAILABLE"

	authOutcomeDenied        = "denied"
	authOutcomeUnavailable   = "unavailable"

	authReasonPrincipalMissing              = "principal_missing"
	authReasonScanIDMalformed               = "scan_id_malformed"
	authReasonScanAuthBuildFailed           = "scan_authorization_request_build_failed"
	authReasonScanAuthFailed                = "scan_authorization_request_failed"
	authReasonScanAuthInvalidResponse       = "scan_authorization_invalid_response"
	authReasonScanAuthDenied                = "scan_authorization_denied"
	authReasonScanAuthURLNotConfigured      = "scan_authorization_url_not_configured"
	authReasonScanAuthUpstreamUnauthorized  = "scan_authorization_upstream_unauthorized"
	authReasonScanAuthUpstream5xx           = "scan_authorization_upstream_5xx"
	authReasonScanAuthUnexpectedStatus      = "scan_authorization_unexpected_status"

	errMsgScanIDMalformed = "scan_id is malformed"

	jsonFieldScanID           = "scan_id"
	jsonFieldCryptoPolicyID   = "crypto_policy_id"
	jsonFieldSelectionRequest = "selection_request"

	jwtAlgEdDSA   = "EdDSA"
	jwtAlgMLDSA65 = "ML-DSA-65"
)

func reasonDetails(reason string) map[string]any {
	return map[string]any{jsonKeyReason: reason}
}

func apiErrorJSON(code, message string) map[string]any {
	return map[string]any{jsonKeyError: code, jsonKeyMessage: message}
}

func apiErrorWithDetails(code, message string, details map[string]any) map[string]any {
	return map[string]any{jsonKeyError: code, jsonKeyMessage: message, jsonKeyDetails: details}
}
