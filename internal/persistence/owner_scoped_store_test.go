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
}
