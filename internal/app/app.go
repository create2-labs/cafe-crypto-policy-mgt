package app

import (
	"fmt"
	"net/http"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
)

// Run starts a minimal HTTP server used as bootstrap for CPM.
func Run(cfg config.Config) error {
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		CatalogPath:   cfg.PolicyCatalogPath,
		TemplatePaths: cfg.PolicyTemplatePaths,
		InstancePaths: cfg.PolicyInstancePaths,
	})
	if err != nil {
		return fmt.Errorf("load read store: %w", err)
	}

	h, err := handler(cfg.ServiceName, store)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: h,
	}

	// PR0 scope: startup wiring only. Graceful shutdown orchestration comes later.
	return server.ListenAndServe()
}

func handler(serviceName string, store *api.ReadStore) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(serviceName + " ok"))
	})
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		return nil, fmt.Errorf("register read routes: %w", err)
	}
	return mux, nil
}
