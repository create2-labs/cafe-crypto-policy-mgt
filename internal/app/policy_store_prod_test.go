//go:build !dev

package app

import (
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
)

func TestNewPolicyStoreRejectsMemoryInProductionBuild(t *testing.T) {
	t.Setenv("CPM_STORE", "memory")
	_, err := newPolicyStore(config.LoadFromEnv())
	if err == nil || !strings.Contains(err.Error(), "memory") {
		t.Fatalf("expected memory rejection error, got %v", err)
	}
}
