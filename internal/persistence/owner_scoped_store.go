package persistence

import (
	"errors"
	"sort"
	"strings"
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

// CountPersistedPoliciesForScan returns how many persisted policy instances belong to
// principal and reference the given scan_id (trimmed exact match on PolicyRecord.ScanID).
// Drafts are not counted — only persisted policies in the owner-scoped policy map.
// Semantics match len(ListPersistedPoliciesForScan(...)) for the same principal and scan_id.
func (s *OwnerScopedStore) CountPersistedPoliciesForScan(principal authz.Principal, scanID string) (int, error) {
	list, err := s.ListPersistedPoliciesForScan(principal, scanID)
	if err != nil {
		return 0, err
	}
	return len(list), nil
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
