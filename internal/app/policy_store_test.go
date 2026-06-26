package app

import (
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence/cphttp"
)

func TestNewPolicyStoreDefaultsPersistenceRequiresURL(t *testing.T) {
	t.Setenv("CPM_STORE", "")
	t.Setenv("CPM_PERSISTENCE_URL", "")
	t.Setenv("CAFE_PERSISTENCE_SERVICE_TOKEN", "token")
	_, err := newPolicyStore(config.LoadFromEnv())
	if err == nil || !strings.Contains(err.Error(), "CPM_PERSISTENCE_URL") {
		t.Fatalf("expected CPM_PERSISTENCE_URL error, got %v", err)
	}
}

func TestNewPolicyStorePersistenceRequiresURL(t *testing.T) {
	t.Setenv("CPM_STORE", "persistence")
	t.Setenv("CPM_PERSISTENCE_URL", "")
	t.Setenv("CAFE_PERSISTENCE_SERVICE_TOKEN", "token")
	_, err := newPolicyStore(config.LoadFromEnv())
	if err == nil || !strings.Contains(err.Error(), "CPM_PERSISTENCE_URL") {
		t.Fatalf("expected CPM_PERSISTENCE_URL error, got %v", err)
	}
}

func TestNewPolicyStorePersistenceRequiresToken(t *testing.T) {
	t.Setenv("CPM_STORE", "persistence")
	t.Setenv("CPM_PERSISTENCE_URL", "http://cafe-persistence:8082")
	t.Setenv("CAFE_PERSISTENCE_SERVICE_TOKEN", "")
	_, err := newPolicyStore(config.LoadFromEnv())
	if err == nil || !strings.Contains(err.Error(), "CAFE_PERSISTENCE_SERVICE_TOKEN") {
		t.Fatalf("expected token error, got %v", err)
	}
}

func TestNewPolicyStorePersistenceClient(t *testing.T) {
	t.Setenv("CPM_STORE", "persistence")
	t.Setenv("CPM_PERSISTENCE_URL", "http://cafe-persistence:8082")
	t.Setenv("CAFE_PERSISTENCE_SERVICE_TOKEN", "secret")
	t.Setenv("CPM_PERSISTENCE_TIMEOUT_SEC", "12")
	store, err := newPolicyStore(config.LoadFromEnv())
	if err != nil {
		t.Fatalf("newPolicyStore: %v", err)
	}
	if _, ok := store.(*cphttp.Client); !ok {
		t.Fatalf("expected *cphttp.Client, got %T", store)
	}
}

func TestNewPolicyStoreRejectsUnknownMode(t *testing.T) {
	t.Setenv("CPM_STORE", "redis")
	_, err := newPolicyStore(config.LoadFromEnv())
	if err == nil || !strings.Contains(err.Error(), "unsupported CPM_STORE") {
		t.Fatalf("expected unsupported store error, got %v", err)
	}
}
