package persistence

import (
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

func TestCountActiveWalletCPMContext_policyAndDraft(t *testing.T) {
	store := NewOwnerScopedStore()
	user := authz.Principal{UserID: "u1", Subject: "u1", TenantID: "t1"}
	addr := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	other := "0x0802b015613ef6701192811e595e085a9c560caf"

	if _, err := store.SavePolicy(user, "pol-1", "550e8400-e29b-41d4-a716-446655440000", map[string]any{
		"selected_wallet_policy_context": map[string]any{
			"target_address": addr,
		},
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := store.SaveDraft(user, "draft-1", "550e8400-e29b-41d4-a716-446655440001", map[string]any{
		"policy_context": map[string]any{
			"result": map[string]any{
				"target_address": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			},
		},
	}); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if _, err := store.SavePolicy(user, "pol-other", "660e8400-e29b-41d4-a716-446655440002", map[string]any{
		"policy_context": map[string]any{"target_address": other},
	}); err != nil {
		t.Fatalf("SavePolicy other: %v", err)
	}

	got, err := store.CountActiveWalletCPMContext(user, addr)
	if err != nil {
		t.Fatalf("CountActiveWalletCPMContext: %v", err)
	}
	if !got.Exists || got.PolicyCount != 1 || got.DraftCount != 1 {
		t.Fatalf("got %+v, want exists=true policy=1 draft=1", got)
	}

	empty, err := store.CountActiveWalletCPMContext(user, "0x0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("CountActiveWalletCPMContext empty: %v", err)
	}
	if empty.Exists {
		t.Fatalf("unexpected exists for unused address: %+v", empty)
	}
}
