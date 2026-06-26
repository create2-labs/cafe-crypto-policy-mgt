//go:build !dev

package app

import "github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"

func devDefaultPolicyStore() persistence.PolicyStore {
	return nil
}
