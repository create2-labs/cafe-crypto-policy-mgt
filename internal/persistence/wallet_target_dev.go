//go:build dev

package persistence

import (
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

// CountActiveWalletCPMContext returns how many persisted policies and platform drafts for principal
// reference the given normalized wallet target_address (WORKPLAN §2.2 W1, IMM-9b).
func (s *OwnerScopedStore) CountActiveWalletCPMContext(principal authz.Principal, normalizedTargetAddress string) (WalletTargetContextCounts, error) {
	return s.lookupActiveWalletCPMContext(principal, normalizedTargetAddress)
}

func (s *OwnerScopedStore) lookupActiveWalletCPMContext(principal authz.Principal, normalizedTargetAddress string) (WalletTargetContextCounts, error) {
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
	var soleDraftID string
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
			if draftCount == 1 {
				soleDraftID = strings.TrimSpace(rec.ID)
			} else {
				soleDraftID = ""
			}
		}
	}
	total := policyCount + draftCount
	out := WalletTargetContextCounts{
		Exists:      total > 0,
		PolicyCount: policyCount,
		DraftCount:  draftCount,
	}
	if draftCount == 1 {
		out.PlatformDraftID = soleDraftID
	}
	return out, nil
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
