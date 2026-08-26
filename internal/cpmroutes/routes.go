// Package cpmroutes defines canonical HTTP path constants for the CPM API (WORKPLAN_API_PR PR11c).
// Public contract is fixed at /api/cpm/v1 after PR11b; mux, auth inventory, tests, and scripts import these literals.
package cpmroutes

import "net/http"

const (
	// V1Base is the public API prefix registered on the CPM mux (edge: /api/cpm/v1).
	V1Base = "/api/cpm/v1"

	// CryptoPolicies lists catalogued Crypto Policies (ADR §5.2 / CPM-P8).
	CryptoPolicies = V1Base + "/crypto-policies"
	// CryptoPolicyByID returns one Crypto Policy by stable crypto_policy_id.
	CryptoPolicyByID = CryptoPolicies + "/{crypto_policy_id}"
	// Providers lists Capability Provider manifests (ADR §5.1 / CPM-P8).
	Providers = V1Base + "/providers"
	// ProviderByID returns one provider manifest by provider_id.
	ProviderByID = Providers + "/{provider_id}"

	PoliciesPrefix = V1Base + "/policies"
	// PoliciesDecisionsExplore evaluates candidates in memory only (transitional until CPM-P9).
	PoliciesDecisionsExplore = PoliciesPrefix + "/decisions/explore"
	// PoliciesAssessmentRequest is the async wallet-scan-only policy assessment trigger (WORKPLAN_API_PR PR13g).
	PoliciesAssessmentRequest = PoliciesPrefix + "/assessment/request"

	// WalletChallenges is the mandatory stateless canonical message helper (CP-PERSIST / RD-P4+).
	WalletChallenges = V1Base + "/wallet-challenges"
	// WalletTargetContext is the owner-scoped IMM-9b lookup for proactive wallet scan UI (FE-IMM-2).
	WalletTargetContext = V1Base + "/wallet-target-context"

	Healthz = "/healthz"
	Metrics = "/metrics"
	// Version is the public ops endpoint for deployed service version (CPM-OPS-3).
	Version = "/version"
)

// Policies is the owner-scoped collection path (GET/POST/DELETE ?id= / ?scan_id=).
const Policies = PoliciesPrefix

// AuthenticatedRoute is a method + path pair listed in the auth route inventory.
type AuthenticatedRoute struct {
	Method string
	Path   string
}

// AuthenticatedRoutes returns every authenticated CPM v1 route (excludes /healthz, /metrics, /version, and internal).
func AuthenticatedRoutes() []AuthenticatedRoute {
	return []AuthenticatedRoute{
		{Method: http.MethodGet, Path: CryptoPolicies},
		{Method: http.MethodGet, Path: CryptoPolicyByID},
		{Method: http.MethodGet, Path: Providers},
		{Method: http.MethodGet, Path: ProviderByID},
		{Method: http.MethodPost, Path: PoliciesDecisionsExplore},
		{Method: http.MethodPost, Path: PoliciesAssessmentRequest},
		{Method: http.MethodPost, Path: WalletChallenges},
		{Method: http.MethodGet, Path: WalletTargetContext},
		{Method: http.MethodPost, Path: Policies},
		{Method: http.MethodGet, Path: Policies},
		{Method: http.MethodDelete, Path: Policies},
	}
}
