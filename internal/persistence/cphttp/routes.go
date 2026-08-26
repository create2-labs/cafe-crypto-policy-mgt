// Package cphttp implements the CPM HTTP client for cafe-persistence internal/cp/v1 (PERS-D5a).
package cphttp

// Route constants mirror cafe-persistence/internal/cproutes (openapi/internal/cp/v1.yaml).
const (
	V1Base          = "/internal/cp/v1"
	PolicyByID      = "/policies/{policy_id}"
	Policies        = "/policies"
	ReferenceWallet = "/references/wallet"
	ReferenceScan   = "/references/scan"
)

const (
	headerAuthorization = "Authorization"
	headerUserID        = "X-User-Id"
	headerTenantID      = "X-Tenant-Id"
	headerRequestID     = "X-Request-Id"
)
