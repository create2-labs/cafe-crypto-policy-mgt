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

func TestNormalizeWalletTargetAddress_canonical(t *testing.T) {
	const want = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	cases := []string{
		"0x742d35cc6634c0532925a3b844bc454e4438f44e",
		"0X742d35Cc6634C0532925a3b844Bc454e4438f44e",
		"742d35cc6634c0532925a3b844bc454e4438f44e",
	}
	for _, in := range cases {
		got, err := NormalizeWalletTargetAddress(in)
		if err != nil {
			t.Fatalf("NormalizeWalletTargetAddress(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestWalletSubjectIDFromAddress_andNormalizeWalletSubjectID(t *testing.T) {
	const addr = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	const want = WalletSubjectPrefix + addr

	subject, err := WalletSubjectIDFromAddress("0X742d35Cc6634C0532925a3b844Bc454e4438f44e")
	if err != nil {
		t.Fatalf("WalletSubjectIDFromAddress: %v", err)
	}
	if subject != want {
		t.Fatalf("WalletSubjectIDFromAddress = %q, want %q", subject, want)
	}

	cases := map[string]string{
		"wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e": want,
		"wallet:" + addr: want,
		addr: want,
		"tls:cluster-1": "tls:cluster-1",
		"wallet:not-hex": "wallet:not-hex",
	}
	for in, expected := range cases {
		if got := NormalizeWalletSubjectID(in); got != expected {
			t.Fatalf("NormalizeWalletSubjectID(%q) = %q, want %q", in, got, expected)
		}
	}
}
