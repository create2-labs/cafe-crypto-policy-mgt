//go:build dev

package app

import (
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func newDevPolicyStore(mode string, _ config.Config) (persistence.PolicyStore, bool, error) {
	if mode != storeMemory {
		return nil, false, nil
	}
	return persistence.NewOwnerScopedStore(), true, nil
}
