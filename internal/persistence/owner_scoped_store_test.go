//go:build dev

package persistence

import (
	"errors"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
)

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

	if _, err := store.SavePolicy(invalid, "policy-x", "", nil); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("expected principal required for SavePolicy, got %v", err)
	}
	if _, err := store.ListPersistedPoliciesForScan(invalid, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("expected principal required for ListPersistedPoliciesForScan, got %v", err)
	}
}

func TestOwnerScopedStoreCountPoliciesForScan(t *testing.T) {
	store := NewOwnerScopedStore()
	userA := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	scan1 := "11111111-1111-1111-1111-111111111111"
	scan2 := "22222222-2222-2222-2222-222222222222"

	if _, err := store.ListPersistedPoliciesForScan(userA, "   "); err == nil {
		t.Fatal("expected error for empty scan_id")
	}

	assertPolicyCountForScan(t, store, userA, scan1, 0)
	mustSavePolicy(t, store, userA, "p1", scan1, map[string]any{"k": 1})
	mustSavePolicy(t, store, userA, "p2", scan1, map[string]any{"k": 2})
	mustSavePolicy(t, store, userA, "p3", scan2, map[string]any{"k": 3})

	assertPolicyCountForScan(t, store, userA, scan1, 2)
	assertPolicyCountForScan(t, store, userA, scan2, 1)

	userB := authz.Principal{UserID: "user-b", Subject: "user-b", TenantID: "t1"}
	assertPolicyCountForScan(t, store, userB, scan1, 0)

	userA2 := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t2"}
	mustSavePolicy(t, store, userA2, "p-other-tenant", scan1, nil)
	assertPolicyCountForScan(t, store, userA, scan1, 2)

	list := mustListPoliciesForScan(t, store, userA, scan1)
	if len(list) != 2 {
		t.Fatalf("ListPersistedPoliciesForScan: want len 2, got %d", len(list))
	}
	assertPolicyCountForScan(t, store, userA, scan1, len(list))
}

func mustSavePolicy(t *testing.T, store *OwnerScopedStore, p authz.Principal, id, scanID string, payload map[string]any) {
	t.Helper()
	if _, err := store.SavePolicy(p, id, scanID, payload); err != nil {
		t.Fatalf("SavePolicy %q: %v", id, err)
	}
}

func assertPolicyCountForScan(t *testing.T, store *OwnerScopedStore, p authz.Principal, scanID string, want int) {
	t.Helper()
	list, err := store.ListPersistedPoliciesForScan(p, scanID)
	if err != nil {
		t.Fatalf("ListPersistedPoliciesForScan(%q): %v", scanID, err)
	}
	if len(list) != want {
		t.Fatalf("ListPersistedPoliciesForScan(%q): want %d policies, got %d", scanID, want, len(list))
	}
}

func mustListPoliciesForScan(t *testing.T, store *OwnerScopedStore, p authz.Principal, scanID string) []PolicyRecord {
	t.Helper()
	list, err := store.ListPersistedPoliciesForScan(p, scanID)
	if err != nil {
		t.Fatalf("ListPersistedPoliciesForScan(%q): %v", scanID, err)
	}
	return list
}

func TestOwnerScopedStoreDeletePolicy(t *testing.T) {
	store := NewOwnerScopedStore()
	userA := authz.Principal{UserID: "user-a", Subject: "user-a", TenantID: "t1"}
	if _, err := store.SavePolicy(userA, "p-del", "33333333-3333-3333-3333-333333333333", map[string]any{"k": 1}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if err := store.DeletePolicy(userA, "p-del"); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	if err := store.DeletePolicy(userA, "p-del"); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("second delete: want ErrPolicyNotFound, got %v", err)
	}
	userB := authz.Principal{UserID: "user-b", Subject: "user-b", TenantID: "t1"}
	if err := store.DeletePolicy(userB, "missing"); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("delete missing: want ErrPolicyNotFound, got %v", err)
	}
	if _, err := store.SavePolicy(userA, "p-x", "44444444-4444-4444-4444-444444444444", nil); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if err := store.DeletePolicy(userB, "p-x"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("delete other owner: want ErrForbidden, got %v", err)
	}
}
