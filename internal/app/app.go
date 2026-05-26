package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/config"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	cpmmats "github.com/create2-labs/cafe-crypto-policy-mgt/internal/integration/nats"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
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

	var assessmentPublish func(context.Context, string, []byte) error
	if u := strings.TrimSpace(cfg.NATSURL); u != "" {
		closeNATS, pub, err := cpmmats.ConnectPublisher(u)
		if err != nil {
			return err
		}
		defer closeNATS()
		assessmentPublish = pub.Publish
	}

	h, err := handler(cfg.ServiceName, store, authConfig{
		Required:                            cfg.AuthRequired,
		SessionValidationURL:                cfg.SessionValidationURL,
		SessionValidationTimeoutSec:         cfg.SessionValidationTimeoutSec,
		SessionValidationServiceToken:       cfg.SessionValidationServiceToken,
		ScanAuthorizationURL:                cfg.ScanAuthorizationURL,
		ScanAuthorizationTimeoutSec:         cfg.ScanAuthorizationTimeoutSec,
		ScanAuthorizationServiceToken:       cfg.ScanAuthorizationServiceToken,
		PolicyReferenceInternalServiceToken: cfg.PolicyReferenceInternalServiceToken,
		ClockSkewSec:                        cfg.AuthClockSkewSec,
		DiscoveryHTTPBaseURL:                cfg.DiscoveryHTTPBaseURL,
		DiscoveryHTTPTimeoutSec:             cfg.DiscoveryHTTPTimeoutSec,
		AssessmentNATSPublish:               assessmentPublish,
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

func handler(serviceName string, store *api.ReadStore, authCfg authConfig) (http.Handler, error) {
	return handlerWithOwnerStore(serviceName, store, persistence.NewOwnerScopedStore(), authCfg)
}

func handlerWithOwnerStore(serviceName string, store *api.ReadStore, ownerStore *persistence.OwnerScopedStore, authCfg authConfig) (http.Handler, error) {
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
	if err := api.RegisterReadRoutes(mux, store); err != nil {
		return nil, fmt.Errorf("register read routes: %w", err)
	}
	registerPoliciesAssessmentRequestRoute(mux, authCfg)
	registerOwnerScopedRoutes(mux, ownerStore, obs)
	registerPolicyReferenceInternalRoute(mux, ownerStore)
	registerPolicyWalletTargetReferenceInternalRoute(mux, ownerStore)
	protected, err := withAuthentication(mux, authCfg)
	if err != nil {
		return nil, fmt.Errorf("wire auth middleware: %w", err)
	}
	return protected, nil
}
