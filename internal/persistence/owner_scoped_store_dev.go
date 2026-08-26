//go:build dev

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

type draftPersistState struct {
	PolicyID    string
	OwnerUserID string
	TenantID    string
	Completed   bool
	PersistedAt time.Time
}

type OwnerScopedStore struct {
	mu             sync.RWMutex
	drafts         map[string]DraftRecord
	policies       map[string]PolicyRecord
	draftPersisted map[string]draftPersistState
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

	s.removeOtherPoliciesForScanLocked(
		principal.UserID,
		principal.TenantID,
		draft.ScanID,
		policyID,
	)

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

// CreatePolicy inserts a signed, gated CP (RD-P5). Enforces W1: one active policy per owner+wallet.
func (s *OwnerScopedStore) CreatePolicy(principal authz.Principal, in CreatePolicyInput) (CreatePolicyResult, error) {
	if err := principal.Validate(); err != nil {
		return CreatePolicyResult{}, ErrPrincipalRequired
	}
	scanID := strings.TrimSpace(in.ScanID)
	if scanID == "" {
		return CreatePolicyResult{}, errors.New("scan_id is required")
	}
	sha := strings.TrimSpace(in.PayloadSHA256)
	if sha == "" {
		return CreatePolicyResult{}, errors.New("payload_sha256 is required")
	}
	if in.Payload == nil {
		return CreatePolicyResult{}, errors.New("payload is required")
	}
	if in.ChainID < 1 {
		return CreatePolicyResult{}, errors.New("chain_id is required")
	}
	normWallet, err := NormalizeWalletTargetAddress(in.WalletAddress)
	if err != nil {
		return CreatePolicyResult{}, fmt.Errorf("wallet address is invalid: %w", err)
	}
	verifiedAt := in.WalletControlVerifiedAt.UTC()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range s.policies {
		if !sameOwner(rec.OwnerUserID, rec.TenantID, principal.UserID, principal.TenantID) {
			continue
		}
		if walletAddressesEqual(rec.WalletAddress, normWallet) ||
			walletAddressesEqual(walletFromPolicyPayload(rec.Payload), normWallet) {
			return CreatePolicyResult{}, ErrPolicyAlreadyExists
		}
	}

	policyID, err := newPolicyID()
	if err != nil {
		return CreatePolicyResult{}, err
	}
	payload := cloneMap(in.Payload)
	if payload == nil {
		payload = map[string]any{}
	}
	payload["scan_id"] = scanID
	payload["wallet_address"] = normWallet
	payload["chain_id"] = in.ChainID
	payload["ownership_status"] = "verified"
	payload["wallet_control_method"] = "eoa_signature"
	payload["wallet_control_verified_at"] = verifiedAt.Format(time.RFC3339)
	payload["persisted_at"] = verifiedAt.Format(time.RFC3339)
	delete(payload, "signed_message")
	delete(payload, "signature")
	delete(payload, "payload_sha256")

	record := PolicyRecord{
		ID:            policyID,
		OwnerUserID:   principal.UserID,
		TenantID:      principal.TenantID,
		ScanID:        scanID,
		Payload:       payload,
		PayloadSHA256: sha,
		WalletAddress: normWallet,
		ChainID:       in.ChainID,
		CreatedAt:     verifiedAt,
		UpdatedAt:     verifiedAt,
	}
	s.policies[policyID] = record
	return CreatePolicyResult{
		PolicyID:      policyID,
		ScanID:        scanID,
		WalletAddress: normWallet,
		ChainID:       in.ChainID,
		PayloadSHA256: sha,
		PersistedAt:   verifiedAt,
	}, nil
}

func walletFromPolicyPayload(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload["wallet_address"].(string); ok {
		if norm, err := NormalizeWalletTargetAddress(v); err == nil {
			return norm
		}
	}
	return ""
}

func walletAddressesEqual(a, b string) bool {
	na, errA := NormalizeWalletTargetAddress(a)
	nb, errB := NormalizeWalletTargetAddress(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return na == nb
}

// ListPersistedPoliciesForScan returns persisted policy instances for principal that
// reference scanID (trimmed exact match on PolicyRecord.ScanID). Drafts are excluded.
// Order is newest UpdatedAt first, then stable by id — at most one row is expected per
// scan after CP-PERSIST replacement (PersistDraftOnce supersedes prior policies).
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
	out := make([]PolicyRecord, 0, 1)
	for id, rec := range s.policies {
		if !sameOwner(rec.OwnerUserID, rec.TenantID, principal.UserID, principal.TenantID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(rec.ScanID), needle) {
			continue
		}
		rec.ID = id
		rec.Payload = cloneMap(rec.Payload)
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

// removeOtherPoliciesForScanLocked drops prior persisted CP rows for the same scan so
// replacement persist enforces at most one recommended policy per scan (CPM V1).
func (s *OwnerScopedStore) removeOtherPoliciesForScanLocked(userID, tenantID, scanID, keepPolicyID string) {
	needle := strings.TrimSpace(scanID)
	if needle == "" {
		return
	}
	keep := strings.TrimSpace(keepPolicyID)
	for id, rec := range s.policies {
		if keep != "" && id == keep {
			continue
		}
		if !sameOwner(rec.OwnerUserID, rec.TenantID, userID, tenantID) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(rec.ScanID), needle) {
			delete(s.policies, id)
		}
	}
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
