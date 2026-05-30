// Package cpmroutes defines canonical HTTP path constants for the CPM API (WORKPLAN_API_PR PR11c).
// Public contract is fixed at /api/cpm/v1 after PR11b; mux, auth inventory, tests, and scripts import these literals.
package cpmroutes

import "net/http"

const (
	// V1Base is the public API prefix registered on the CPM mux (edge: /api/cpm/v1).
	V1Base = "/api/cpm/v1"

	PoliciesPrefix           = V1Base + "/policies"
	PoliciesCatalog          = PoliciesPrefix + "/catalog"
	PoliciesTemplates        = PoliciesPrefix + "/templates"
	PoliciesInstances        = PoliciesPrefix + "/instances"
	PoliciesDecisionsExplore = PoliciesPrefix + "/decisions/explore"
	// PoliciesAssessmentRequest is the async wallet-scan-only policy assessment trigger (WORKPLAN_API_PR PR13g).
	PoliciesAssessmentRequest = PoliciesPrefix + "/assessment/request"

	Drafts = V1Base + "/drafts"

	Healthz = "/healthz"

	// InternalPolicyReferenceScan is the service-token-gated lookup used by Discovery DELETE (PR5).
	InternalPolicyReferenceScan = "/internal/policies/references/scan"
	// InternalPolicyReferenceWalletTarget is the service-token-gated W1 lookup by normalized target_address (IMM-9b).
	InternalPolicyReferenceWalletTarget = "/internal/policies/references/wallet-target"
)

// Policies is the owner-scoped collection path (GET/POST/DELETE ?id= / ?scan_id=).
const Policies = PoliciesPrefix

// AuthenticatedRoute is a method + path pair listed in the auth route inventory.
type AuthenticatedRoute struct {
	Method string
	Path   string
}

// AuthenticatedRoutes returns every authenticated CPM v1 route (excludes /healthz and internal).
func AuthenticatedRoutes() []AuthenticatedRoute {
	return []AuthenticatedRoute{
		{Method: http.MethodGet, Path: PoliciesCatalog},
		{Method: http.MethodGet, Path: PoliciesTemplates},
		{Method: http.MethodGet, Path: PoliciesInstances},
		{Method: http.MethodPost, Path: PoliciesDecisionsExplore},
		{Method: http.MethodPost, Path: PoliciesAssessmentRequest},
		{Method: http.MethodPost, Path: Drafts},
		{Method: http.MethodGet, Path: Drafts},
		{Method: http.MethodDelete, Path: Drafts},
		{Method: http.MethodPost, Path: Policies},
		{Method: http.MethodGet, Path: Policies},
		{Method: http.MethodDelete, Path: Policies},
	}
}
