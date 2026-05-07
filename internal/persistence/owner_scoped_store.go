package persistence

import (
	"errors"
	"sync"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

var (
	ErrPrincipalRequired = errors.New("principal is required")
	ErrDraftNotFound     = errors.New("draft not found")
	ErrPolicyNotFound    = errors.New("policy not found")
	ErrForbidden         = errors.New("forbidden")
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

type OwnerScopedStore struct {
	mu       sync.RWMutex
	drafts   map[string]DraftRecord
	policies map[string]PolicyRecord
}

func NewOwnerScopedStore() *OwnerScopedStore {
	return &OwnerScopedStore{
		drafts:   make(map[string]DraftRecord),
		policies: make(map[string]PolicyRecord),
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
