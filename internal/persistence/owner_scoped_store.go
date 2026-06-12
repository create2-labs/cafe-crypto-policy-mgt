package persistence

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

var (
	ErrPrincipalRequired      = errors.New("principal is required")
	ErrDraftNotFound          = errors.New("draft not found")
	ErrDraftAlreadyPersisted  = errors.New("draft already persisted")
	ErrPolicyNotFound         = errors.New("policy not found")
	ErrForbidden              = errors.New("forbidden")
)

type DraftRecord struct {
	ID          string
	OwnerUserID string
	TenantID    string
	ScanID      string
	Payload     map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyRecord struct {
	ID          string
	OwnerUserID string
	TenantID    string
	ScanID      string
	Payload     map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type draftPersistState struct {
	PolicyID    string
	OwnerUserID string
	TenantID    string
	Completed   bool
	PersistedAt time.Time
}

// PersistDraftInput carries wallet ownership metadata applied to the persisted policy payload.
type PersistDraftInput struct {
	WalletAddress string
	ChainID       int64
	VerifiedAt    time.Time
}

// PersistDraftResult is the durable outcome of a successful draft persist transition.
type PersistDraftResult struct {
	PolicyID      string
	DraftID       string
	ScanID        string
	WalletAddress string
	ChainID       int64
	PersistedAt   time.Time
}

type OwnerScopedStore struct {
	mu              sync.RWMutex
	drafts          map[string]DraftRecord
	policies        map[string]PolicyRecord
	draftPersisted  map[string]draftPersistState
}

func NewOwnerScopedStore() *OwnerScopedStore {
	return &OwnerScopedStore{
		drafts:         make(map[string]DraftRecord),
		policies:       make(map[string]PolicyRecord),
		draftPersisted: make(map[string]draftPersistState),
	}
}

func (s *OwnerScopedStore) SaveDraft(principal authz.Principal, id string, scanID string, payload map[string]any) (DraftRecord, error) {
	if err := principal.Validate(); err != nil {
		return DraftRecord{}, ErrPrincipalRequired
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, hasExisting := s.drafts[id]
	if hasExisting && !sameOwner(existing.OwnerUserID, existing.TenantID, principal.UserID, principal.TenantID) {
		return DraftRecord{}, ErrForbidden
	}
	createdAt := now
	if hasExisting {
		createdAt = existing.CreatedAt
	}
	record := DraftRecord{
		ID:          id,
		OwnerUserID: principal.UserID,
		TenantID:    principal.TenantID,
		ScanID:      scanID,
		Payload:     cloneMap(payload),
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	s.drafts[id] = record
	return record, nil
}

func (s *OwnerScopedStore) GetDraft(principal authz.Principal, id string) (DraftRecord, error) {
	if err := principal.Validate(); err != nil {
		return DraftRecord{}, ErrPrincipalRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.drafts[id]
	if !ok {
		return DraftRecord{}, ErrDraftNotFound
	}
	if !sameOwner(record.OwnerUserID, record.TenantID, principal.UserID, principal.TenantID) {
		return DraftRecord{}, ErrForbidden
	}
	record.Payload = cloneMap(record.Payload)
	return record, nil
}

// DeleteDraft removes a platform draft by id for principal. ErrDraftNotFound if missing;
// ErrForbidden if owned by another principal (callers may map to 404).
func (s *OwnerScopedStore) DeleteDraft(principal authz.Principal, id string) error {
	if err := principal.Validate(); err != nil {
		return ErrPrincipalRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.drafts[id]
	if !ok {
		return ErrDraftNotFound
	}
	if !sameOwner(record.OwnerUserID, record.TenantID, principal.UserID, principal.TenantID) {
		return ErrForbidden
	}
	delete(s.drafts, id)
	return nil
}

// PersistDraftOnce transitions an owner-scoped draft to a persisted policy exactly once.
// If policy creation fails before completion, the same draft may be retried with the same policy id.
func (s *OwnerScopedStore) PersistDraftOnce(principal authz.Principal, draftID string, in PersistDraftInput) (PersistDraftResult, error) {
	if err := principal.Validate(); err != nil {
		return PersistDraftResult{}, ErrPrincipalRequired
	}
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return PersistDraftResult{}, ErrDraftNotFound
	}
	verifiedAt := in.VerifiedAt.UTC()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	normWallet, err := NormalizeWalletTargetAddress(in.WalletAddress)
	if err != nil {
		return PersistDraftResult{}, fmt.Errorf("wallet address is invalid: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if state, ok := s.draftPersisted[draftID]; ok && state.Completed {
		if !sameOwner(state.OwnerUserID, state.TenantID, principal.UserID, principal.TenantID) {
			return PersistDraftResult{}, ErrDraftNotFound
		}
		return PersistDraftResult{}, ErrDraftAlreadyPersisted
	}

	draft, ok := s.drafts[draftID]
	if !ok {
		return PersistDraftResult{}, ErrDraftNotFound
	}
	if !sameOwner(draft.OwnerUserID, draft.TenantID, principal.UserID, principal.TenantID) {
		return PersistDraftResult{}, ErrForbidden
	}

	state := s.draftPersisted[draftID]
	state.OwnerUserID = principal.UserID
	state.TenantID = principal.TenantID
	policyID := state.PolicyID
	if policyID == "" {
		policyID, err = newPolicyID()
		if err != nil {
			return PersistDraftResult{}, err
		}
		state.PolicyID = policyID
	}

	persistedAt := verifiedAt
	payload := policyPayloadFromDraft(draft.Payload, draftID, draft.ScanID, normWallet, in.ChainID, persistedAt)
	record := PolicyRecord{
		ID:          policyID,
		OwnerUserID: principal.UserID,
		TenantID:    principal.TenantID,
		ScanID:      draft.ScanID,
		Payload:     payload,
		CreatedAt:   persistedAt,
		UpdatedAt:   persistedAt,
	}
	if existing, hasExisting := s.policies[policyID]; hasExisting {
		record.CreatedAt = existing.CreatedAt
	}
	s.policies[policyID] = record
	state.Completed = true
	state.PersistedAt = persistedAt
	s.draftPersisted[draftID] = state
	delete(s.drafts, draftID)


	return PersistDraftResult{
		PolicyID:      policyID,
		DraftID:       draftID,
		ScanID:        strings.TrimSpace(draft.ScanID),
		WalletAddress: normWallet,
		ChainID:       in.ChainID,
		PersistedAt:   persistedAt,
	}, nil
}

func policyPayloadFromDraft(draftPayload map[string]any, draftID, scanID, wallet string, chainID int64, verifiedAt time.Time) map[string]any {
	out := cloneMap(draftPayload)
	if out == nil {
		out = make(map[string]any)
	}
	out["draft_id"] = draftID
	out["scan_id"] = strings.TrimSpace(scanID)
	out["wallet_address"] = wallet
	out["chain_id"] = chainID
	out["ownership_status"] = "verified"
	out["wallet_control_method"] = "eoa_signature"
	out["wallet_control_verified_at"] = verifiedAt.Format(time.RFC3339)
	out["persisted_at"] = verifiedAt.Format(time.RFC3339)
	delete(out, "signed_message")
	delete(out, "signature")
	return out
}

func newPolicyID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// DraftPersistStatus reports whether a draft was already persisted for principal.
func (s *OwnerScopedStore) DraftPersistStatus(principal authz.Principal, draftID string) error {
	if err := principal.Validate(); err != nil {
		return ErrPrincipalRequired
	}
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return ErrDraftNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.draftPersisted[draftID]
	if !ok || !state.Completed {
		return nil
	}
	if !sameOwner(state.OwnerUserID, state.TenantID, principal.UserID, principal.TenantID) {
		return ErrDraftNotFound
	}
	return ErrDraftAlreadyPersisted
}

func (s *OwnerScopedStore) SavePolicy(principal authz.Principal, id string, scanID string, payload map[string]any) (PolicyRecord, error) {
	if err := principal.Validate(); err != nil {
		return PolicyRecord{}, ErrPrincipalRequired
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, hasExisting := s.policies[id]
	if hasExisting && !sameOwner(existing.OwnerUserID, existing.TenantID, principal.UserID, principal.TenantID) {
		return PolicyRecord{}, ErrForbidden
	}
	createdAt := now
	if hasExisting {
		createdAt = existing.CreatedAt
	}
	record := PolicyRecord{
		ID:          id,
		OwnerUserID: principal.UserID,
		TenantID:    principal.TenantID,
		ScanID:      scanID,
		Payload:     cloneMap(payload),
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	s.policies[id] = record
	return record, nil
}

// ListPersistedPoliciesForScan returns persisted policy instances for principal that
// reference scanID (trimmed exact match on PolicyRecord.ScanID). Order is stable by id.
// Drafts are excluded — only entries in the owner-scoped policy map.
func (s *OwnerScopedStore) ListPersistedPoliciesForScan(principal authz.Principal, scanID string) ([]PolicyRecord, error) {
	if err := principal.Validate(); err != nil {
		return nil, ErrPrincipalRequired
	}
	needle := strings.TrimSpace(scanID)
	if needle == "" {
		return nil, errors.New("scan_id is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []string
	for id, rec := range s.policies {
		if !sameOwner(rec.OwnerUserID, rec.TenantID, principal.UserID, principal.TenantID) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(rec.ScanID), needle) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]PolicyRecord, 0, len(ids))
	for _, id := range ids {
		rec := s.policies[id]
		rec.Payload = cloneMap(rec.Payload)
		out = append(out, rec)
	}
	return out, nil
}

// DeletePolicy removes a persisted policy instance by id for principal. ErrPolicyNotFound if
// missing; ErrForbidden if owned by another principal (callers may map to 404).
func (s *OwnerScopedStore) DeletePolicy(principal authz.Principal, id string) error {
	if err := principal.Validate(); err != nil {
		return ErrPrincipalRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.policies[id]
	if !ok {
		return ErrPolicyNotFound
	}
	if !sameOwner(record.OwnerUserID, record.TenantID, principal.UserID, principal.TenantID) {
		return ErrForbidden
	}
	delete(s.policies, id)
	return nil
}

func (s *OwnerScopedStore) GetPolicy(principal authz.Principal, id string) (PolicyRecord, error) {
	if err := principal.Validate(); err != nil {
		return PolicyRecord{}, ErrPrincipalRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.policies[id]
	if !ok {
		return PolicyRecord{}, ErrPolicyNotFound
	}
	if !sameOwner(record.OwnerUserID, record.TenantID, principal.UserID, principal.TenantID) {
		return PolicyRecord{}, ErrForbidden
	}
	record.Payload = cloneMap(record.Payload)
	return record, nil
}

func sameOwner(recordUserID string, recordTenantID string, userID string, tenantID string) bool {
	return recordUserID == userID && recordTenantID == tenantID
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
