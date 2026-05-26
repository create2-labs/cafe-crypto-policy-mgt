package persistence

import (
	"errors"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

// WalletTargetContextCounts is the minimal IMM-9b lookup result for a normalized wallet target_address.
type WalletTargetContextCounts struct {
	Exists       bool
	PolicyCount  int
	DraftCount   int
}

// CountActiveWalletCPMContext returns how many persisted policies and platform drafts for principal
// reference the given normalized wallet target_address (WORKPLAN §2.2 W1, IMM-9b).
func (s *OwnerScopedStore) CountActiveWalletCPMContext(principal authz.Principal, normalizedTargetAddress string) (WalletTargetContextCounts, error) {
	if err := principal.Validate(); err != nil {
		return WalletTargetContextCounts{}, ErrPrincipalRequired
	}
	needle, err := NormalizeWalletTargetAddress(normalizedTargetAddress)
	if err != nil {
		return WalletTargetContextCounts{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var policyCount, draftCount int
	for _, rec := range s.policies {
		if !sameOwner(rec.OwnerUserID, rec.TenantID, principal.UserID, principal.TenantID) {
			continue
		}
		if walletTargetFromPayload(rec.Payload) == needle {
			policyCount++
		}
	}
	for _, rec := range s.drafts {
		if !sameOwner(rec.OwnerUserID, rec.TenantID, principal.UserID, principal.TenantID) {
			continue
		}
		if walletTargetFromPayload(rec.Payload) == needle {
			draftCount++
		}
	}
	total := policyCount + draftCount
	return WalletTargetContextCounts{
		Exists:      total > 0,
		PolicyCount: policyCount,
		DraftCount:  draftCount,
	}, nil
}

// NormalizeWalletTargetAddress applies the same normalization as Discovery wallet scans (0x + lowercase).
func NormalizeWalletTargetAddress(address string) (string, error) {
	a := strings.TrimSpace(address)
	if a == "" {
		return "", errors.New("target_address is required")
	}
	if strings.HasPrefix(a, "0X") {
		a = "0x" + a[2:]
	}
	if !strings.HasPrefix(a, "0x") {
		a = "0x" + a
	}
	a = strings.ToLower(a)
	if len(a) != 42 || !strings.HasPrefix(a, "0x") {
		return "", errors.New("target_address must be a normalized EVM address")
	}
	for _, c := range a[2:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", errors.New("target_address must be a normalized EVM address")
		}
	}
	return a, nil
}

func walletTargetFromPayload(payload map[string]any) string {
	if addr := extractWalletTargetAddress(payload); addr != "" {
		norm, err := NormalizeWalletTargetAddress(addr)
		if err == nil {
			return norm
		}
	}
	return ""
}

func extractWalletTargetAddress(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if pc, ok := payload["policy_context"].(map[string]any); ok {
		if addr := walletTargetFromPolicyContext(pc); addr != "" {
			return addr
		}
	}
	if swc, ok := payload["selected_wallet_policy_context"].(map[string]any); ok {
		if addr := walletTargetFromScanContext(swc); addr != "" {
			return addr
		}
	}
	if draft, ok := payload["draft"].(map[string]any); ok {
		if addr := extractWalletTargetAddress(draft); addr != "" {
			return addr
		}
	}
	return firstNonEmpty(
		stringField(payload, "target_address"),
		stringField(payload, "wallet_address"),
		stringField(payload, "walletAddress"),
	)
}

func walletTargetFromPolicyContext(pc map[string]any) string {
	if addr := firstNonEmpty(
		stringField(pc, "target_address"),
		stringField(pc, "wallet_address"),
		stringField(pc, "walletAddress"),
	); addr != "" {
		return addr
	}
	if res, ok := pc["result"].(map[string]any); ok {
		return stringField(res, "target_address")
	}
	return ""
}

func walletTargetFromScanContext(ctx map[string]any) string {
	if addr := firstNonEmpty(
		stringField(ctx, "target_address"),
		stringField(ctx, "wallet_address"),
	); addr != "" {
		return addr
	}
	if res, ok := ctx["result"].(map[string]any); ok {
		return stringField(res, "target_address")
	}
	return ""
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
