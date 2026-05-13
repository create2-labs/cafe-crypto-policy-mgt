package persistence

import (
	"errors"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

func TestOwnerScopedStoreDrafts(t *testing.T) {
	store := NewOwnerScopedStore()
	userA := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "tenant-1"}
	userB := authz.Principal{UserID: "user-b", Subject: "user-b", TenantID: "tenant-1"}

	saved, err := store.SaveDraft(userA, "draft-1", "scan-1", map[string]any{
		"name":          "my draft",
		"owner_user_id": "spoofed-user",
	})
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if saved.OwnerUserID != "user-a" || saved.TenantID != "tenant-1" {
		t.Fatalf("expected owner from principal, got owner=%q tenant=%q", saved.OwnerUserID, saved.TenantID)
	}
	if payloadOwner, _ := saved.Payload["owner_user_id"].(string); payloadOwner != "spoofed-user" {
		t.Fatalf("expected payload to remain opaque content, got %#v", saved.Payload)
	}

	readByOwner, err := store.GetDraft(userA, "draft-1")
	if err != nil {
		t.Fatalf("GetDraft by owner: %v", err)
	}
	if readByOwner.ID != "draft-1" {
		t.Fatalf("expected draft-1, got %q", readByOwner.ID)
	}

	if _, err := store.GetDraft(userB, "draft-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for cross-user access, got %v", err)
	}
	if _, err := store.SaveDraft(userB, "draft-1", "scan-1", map[string]any{"name": "override"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for cross-user overwrite, got %v", err)
	}
}

func TestOwnerScopedStorePolicies(t *testing.T) {
	store := NewOwnerScopedStore()
	userA := authz.Principal{UserID: "user-a", Subject: "user-a"}
	userB := authz.Principal{UserID: "user-b", Subject: "user-b"}

	if _, err := store.SavePolicy(userA, "policy-1", "scan-9", map[string]any{"mode": "strict"}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := store.GetPolicy(userB, "policy-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for policy read cross-user, got %v", err)
	}
}

func TestOwnerScopedStoreRequiresPrincipal(t *testing.T) {
	store := NewOwnerScopedStore()
	invalid := authz.Principal{}

	if _, err := store.SaveDraft(invalid, "draft-x", "", nil); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("expected principal required for SaveDraft, got %v", err)
	}
	if _, err := store.SavePolicy(invalid, "policy-x", "", nil); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("expected principal required for SavePolicy, got %v", err)
	}
	if _, err := store.CountPersistedPoliciesForScan(invalid, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("expected principal required for CountPersistedPoliciesForScan, got %v", err)
	}
}

func TestOwnerScopedStoreCountPoliciesForScan(t *testing.T) {
	store := NewOwnerScopedStore()
	userA := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	scan1 := "11111111-1111-1111-1111-111111111111"
	scan2 := "22222222-2222-2222-2222-222222222222"

	if _, err := store.CountPersistedPoliciesForScan(userA, "   "); err == nil {
		t.Fatal("expected error for empty scan_id")
	}

	if n, err := store.CountPersistedPoliciesForScan(userA, scan1); err != nil || n != 0 {
		t.Fatalf("expected count 0, got n=%d err=%v", n, err)
	}
	if _, err := store.SavePolicy(userA, "p1", scan1, map[string]any{"k": 1}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := store.SavePolicy(userA, "p2", scan1, map[string]any{"k": 2}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if _, err := store.SavePolicy(userA, "p3", scan2, map[string]any{"k": 3}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	n, err := store.CountPersistedPoliciesForScan(userA, scan1)
	if err != nil || n != 2 {
		t.Fatalf("expected count 2 for scan1, got n=%d err=%v", n, err)
	}
	n, err = store.CountPersistedPoliciesForScan(userA, scan2)
	if err != nil || n != 1 {
		t.Fatalf("expected count 1 for scan2, got n=%d err=%v", n, err)
	}

	userB := authz.Principal{UserID: "user-b", Subject: "user-b", TenantID: "t1"}
	if n, err := store.CountPersistedPoliciesForScan(userB, scan1); err != nil || n != 0 {
		t.Fatalf("expected other owner count 0, got n=%d err=%v", n, err)
	}

	userA2 := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t2"}
	if _, err := store.SavePolicy(userA2, "p-other-tenant", scan1, nil); err != nil {
		t.Fatalf("SavePolicy other tenant: %v", err)
	}
	if n, err := store.CountPersistedPoliciesForScan(userA, scan1); err != nil || n != 2 {
		t.Fatalf("expected tenant isolation: count 2 for userA t1, got %d", n)
	}
}
