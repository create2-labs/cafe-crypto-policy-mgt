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

type OwnerScopedStore struct {
	mu       sync.RWMutex
	policies map[string]PolicyRecord
}

func NewOwnerScopedStore() *OwnerScopedStore {
	return &OwnerScopedStore{
		policies: make(map[string]PolicyRecord),
	}
}

func newPolicyID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
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
// reference scanID (trimmed exact match on PolicyRecord.ScanID).
// Order is newest UpdatedAt first, then stable by id.
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
