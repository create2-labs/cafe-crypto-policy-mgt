//go:build !dev

package app

import (
	"fmt"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func newDevPolicyStore(mode string, _ config.Config) (persistence.PolicyStore, bool, error) {
	if mode != storeMemory {
		return nil, false, nil
	}
	return nil, true, fmt.Errorf("CPM_STORE=memory is not available in production builds (rebuild with -tags dev for local)")
}
