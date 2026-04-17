package app

import (
	"net/http"

	"github.com/create2-labs/cafe-cpm/internal/config"
)

// Run starts a minimal HTTP server used as bootstrap for CPM.
func Run(cfg config.Config) error {
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler(cfg.ServiceName),
	}

	// PR0 scope: startup wiring only. Graceful shutdown orchestration comes later.
	return server.ListenAndServe()
}

func handler(serviceName string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(serviceName + " ok"))
	})
	return mux
}
