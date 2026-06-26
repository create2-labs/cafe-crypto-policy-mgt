package persistence

import "github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"

// PolicyStore is the owner-scoped durable CP storage surface used by CPM HTTP handlers.
// Deployed CPM uses cphttp.Client (CPM_STORE=persistence). OwnerScopedStore exists only
// under //go:build dev for unit/handler tests — not linked into production binaries (PERS-D5c).
type PolicyStore interface {
	SaveDraft(principal authz.Principal, id string, scanID string, payload map[string]any) (DraftRecord, error)
	GetDraft(principal authz.Principal, id string) (DraftRecord, error)
	DeleteDraft(principal authz.Principal, id string) error
	PersistDraftOnce(principal authz.Principal, draftID string, in PersistDraftInput) (PersistDraftResult, error)
	DraftPersistStatus(principal authz.Principal, draftID string) error
	SavePolicy(principal authz.Principal, id string, scanID string, payload map[string]any) (PolicyRecord, error)
	ListPersistedPoliciesForScan(principal authz.Principal, scanID string) ([]PolicyRecord, error)
	DeletePolicy(principal authz.Principal, id string) error
	GetPolicy(principal authz.Principal, id string) (PolicyRecord, error)
	CountActiveWalletCPMContext(principal authz.Principal, normalizedTargetAddress string) (WalletTargetContextCounts, error)
}
