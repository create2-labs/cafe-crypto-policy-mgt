package persistence

import "github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"

// PolicyStore is the owner-scoped durable CP storage surface used by CPM HTTP handlers.
// Memory (OwnerScopedStore) is the default; HTTP (cphttp.Client) backs CPM_STORE=persistence (PERS-D5a+).
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
