//go:build dev

package app

import (
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func TestNewPolicyStoreMemoryExplicitWithDevTag(t *testing.T) {
	t.Setenv("CPM_STORE", "memory")
	store, err := newPolicyStore(config.LoadFromEnv())
	if err != nil {
		t.Fatalf("newPolicyStore: %v", err)
	}
	if _, ok := store.(*persistence.OwnerScopedStore); !ok {
		t.Fatalf("expected *OwnerScopedStore, got %T", store)
	}
}
