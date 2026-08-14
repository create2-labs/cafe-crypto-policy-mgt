package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	cpmmats "github.com/create2-labs/cafe-crypto-policy-mgt/internal/integration/nats"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/metrics"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Run starts a minimal HTTP server used as bootstrap for CPM.
func Run(cfg config.Config) error {
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		TemplatePaths:         cfg.PolicyTemplatePaths,
		InstancePaths:         cfg.PolicyInstancePaths,
		ProviderManifestPaths: cfg.ProviderManifestPaths,
	})
	if err != nil {
		return fmt.Errorf("load read store: %w", err)
	}

	ownerStore, err := newPolicyStore(cfg)
	if err != nil {
		return fmt.Errorf("policy store: %w", err)
	}
	log.Printf("cpm: policy store=persistence url=%s", cfg.PersistenceURL)

	var assessmentPublish func(context.Context, string, []byte) error
	if u := strings.TrimSpace(cfg.NATSURL); u != "" {
		closeNATS, pub, err := cpmmats.ConnectPublisher(u)
		if err != nil {
			return err
		}
		defer closeNATS()
		assessmentPublish = pub.Publish
	}

	h, err := handler(cfg, store, ownerStore, authConfig{
		Required:                      cfg.AuthRequired,
		SessionValidationURL:          cfg.SessionValidationURL,
		SessionValidationTimeoutSec:   cfg.SessionValidationTimeoutSec,
		SessionValidationServiceToken: cfg.SessionValidationServiceToken,
		ScanAuthorizationURL:          cfg.ScanAuthorizationURL,
		ScanAuthorizationTimeoutSec:   cfg.ScanAuthorizationTimeoutSec,
		ScanAuthorizationServiceToken: cfg.ScanAuthorizationServiceToken,
		ClockSkewSec:                  cfg.AuthClockSkewSec,
		DiscoveryHTTPBaseURL:          cfg.DiscoveryHTTPBaseURL,
		DiscoveryHTTPTimeoutSec:       cfg.DiscoveryHTTPTimeoutSec,
		AssessmentNATSPublish:         assessmentPublish,
		WalletAuthDomain:              cfg.WalletAuthDomain,
	})
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

func handler(cfg config.Config, store *api.ReadStore, ownerStore persistence.PolicyStore, authCfg authConfig) (http.Handler, error) {
	return handlerWithOwnerStore(cfg.ServiceName, store, ownerStore, authCfg)
}

func handlerWithOwnerStore(serviceName string, store *api.ReadStore, ownerStore persistence.PolicyStore, authCfg authConfig) (http.Handler, error) {
	mux := http.NewServeMux()
	obs := authCfg.Observability
	if obs == nil {
		obs = newAuthObservability()
	}
	authCfg.Observability = obs
	mux.HandleFunc(cpmroutes.Healthz, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(serviceName + " ok"))
	})
	mux.HandleFunc(cpmroutes.Version, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(version.Payload())
	})
	mux.Handle(cpmroutes.Metrics, promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{}))
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		return nil, fmt.Errorf("register read routes: %w", err)
	}
	registerPoliciesAssessmentRequestRoute(mux, authCfg)
	registerOwnerScopedRoutes(mux, ownerStore, obs)
	registerWalletChallengeRoutes(mux, ownerStore, authCfg, obs)
	registerDraftPersistRoutes(mux, ownerStore, authCfg, obs)
	protected, err := withAuthentication(mux, authCfg)
	if err != nil {
		return nil, fmt.Errorf("wire auth middleware: %w", err)
	}
	return protected, nil
}
