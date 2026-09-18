package app

import (
	"net/http"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func testHandler(readStore *api.ReadStore, owner persistence.PolicyStore, authCfg authConfig) (http.Handler, error) {
	if owner == nil {
		owner = devDefaultPolicyStore()
	}
	return handler(config.Config{ServiceName: "cafe-cpm"}, readStore, owner, authCfg)
}
