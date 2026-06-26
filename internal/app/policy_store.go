package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence/cphttp"
)

const (
	storeMemory      = "memory"
	storePersistence = "persistence"
)

func newPolicyStore(cfg config.Config) (persistence.PolicyStore, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Store))
	if mode == "" {
		mode = storePersistence
	}
	if store, handled, err := newDevPolicyStore(mode, cfg); handled {
		return store, err
	}
	switch mode {
	case storePersistence:
		return newPersistencePolicyStore(cfg)
	default:
		return nil, fmt.Errorf("unsupported CPM_STORE %q (want persistence)", cfg.Store)
	}
}

func newPersistencePolicyStore(cfg config.Config) (persistence.PolicyStore, error) {
	if strings.TrimSpace(cfg.PersistenceURL) == "" {
		return nil, fmt.Errorf("CPM_PERSISTENCE_URL is required when CPM_STORE=persistence")
	}
	if strings.TrimSpace(cfg.PersistenceServiceToken) == "" {
		return nil, fmt.Errorf("CAFE_PERSISTENCE_SERVICE_TOKEN is required when CPM_STORE=persistence")
	}
	timeout := time.Duration(cfg.PersistenceTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return cphttp.NewClient(cphttp.Config{
		BaseURL:    cfg.PersistenceURL,
		Token:      cfg.PersistenceServiceToken,
		HTTPClient: &http.Client{Timeout: timeout},
	}), nil
}
