package persistence

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

func TestPersistDraftOnce_transitionsDraftToPolicy(t *testing.T) {
	store := NewOwnerScopedStore()
	user := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	scanID := "550e8400-e29b-41d4-a716-446655440000"
	wallet := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	draftPayload := map[string]any{
		"policy_context": map[string]any{
			"wallet_address": wallet,
			"wallet_type":    "eoa",
		},
		"mode": "strict",
	}
	if _, err := store.SaveDraft(user, "draft-1", scanID, draftPayload); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	verifiedAt := time.Date(2026, 6, 10, 12, 1, 0, 0, time.UTC)
	result, err := store.PersistDraftOnce(user, "draft-1", PersistDraftInput{
		WalletAddress: wallet,
		ChainID:       1,
		VerifiedAt:    verifiedAt,
	})
	if err != nil {
		t.Fatalf("PersistDraftOnce: %v", err)
	}
	if result.PolicyID == "" || result.DraftID != "draft-1" || result.ScanID != scanID {
		t.Fatalf("unexpected result: %#v", result)
	}

	if _, err := store.GetDraft(user, "draft-1"); !errors.Is(err, ErrDraftNotFound) {
		t.Fatalf("expected draft removed after persist, got %v", err)
	}
	policy, err := store.GetPolicy(user, result.PolicyID)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if policy.Payload["ownership_status"] != "verified" {
		t.Fatalf("ownership_status = %#v", policy.Payload["ownership_status"])
	}
	if policy.Payload["wallet_control_method"] != "eoa_signature" {
		t.Fatalf("wallet_control_method = %#v", policy.Payload["wallet_control_method"])
	}
	if _, ok := policy.Payload["signature"]; ok {
		t.Fatal("raw signature must not be stored")
	}
	if _, ok := policy.Payload["signed_message"]; ok {
		t.Fatal("signed_message must not be stored")
	}
}

func TestPersistDraftOnce_replayReturnsAlreadyPersisted(t *testing.T) {
	store := NewOwnerScopedStore()
	user := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	scanID := "550e8400-e29b-41d4-a716-446655440000"
	wallet := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	if _, err := store.SaveDraft(user, "draft-replay", scanID, map[string]any{
		"policy_context": map[string]any{"wallet_address": wallet, "wallet_type": "eoa"},
	}); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	in := PersistDraftInput{WalletAddress: wallet, ChainID: 1, VerifiedAt: time.Now().UTC()}
	if _, err := store.PersistDraftOnce(user, "draft-replay", in); err != nil {
		t.Fatalf("first persist: %v", err)
	}
	if _, err := store.PersistDraftOnce(user, "draft-replay", in); !errors.Is(err, ErrDraftAlreadyPersisted) {
		t.Fatalf("replay: want ErrDraftAlreadyPersisted, got %v", err)
	}
}

func TestPersistDraftOnce_retryBeforeCompletionReusesPolicyID(t *testing.T) {
	store := NewOwnerScopedStore()
	user := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	scanID := "550e8400-e29b-41d4-a716-446655440000"
	wallet := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	if _, err := store.SaveDraft(user, "draft-retry", scanID, map[string]any{
		"policy_context": map[string]any{"wallet_address": wallet, "wallet_type": "eoa"},
	}); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	store.mu.Lock()
	store.draftPersisted["draft-retry"] = draftPersistState{PolicyID: "pending-policy-id"}
	store.mu.Unlock()

	in := PersistDraftInput{WalletAddress: wallet, ChainID: 1, VerifiedAt: time.Now().UTC()}
	result, err := store.PersistDraftOnce(user, "draft-retry", in)
	if err != nil {
		t.Fatalf("retry persist: %v", err)
	}
	if result.PolicyID != "pending-policy-id" {
		t.Fatalf("policy_id = %q want pending-policy-id", result.PolicyID)
	}
	if !strings.HasPrefix(result.PolicyID, "pending") {
		t.Fatalf("unexpected policy id %q", result.PolicyID)
	}
}
