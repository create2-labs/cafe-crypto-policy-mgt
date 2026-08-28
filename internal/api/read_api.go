package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/metrics"
)

var (
	ErrStoreNil = errors.New("api read store is nil")
)

// ReadStoreOptions configures catalogue loading for Crypto Policies and providers.
type ReadStoreOptions struct {
	CryptoPolicyPaths     []string
	ProviderManifestPaths []string
}

// ReadStore holds catalogue Crypto Policies and provider manifests for read/explore APIs.
type ReadStore struct {
	cryptoPolicies   []*policy.CryptoPolicy
	cryptoPolicyByID map[string]*policy.CryptoPolicy
	providers        *provider.Registry
}

func LoadReadStore(opts ReadStoreOptions) (*ReadStore, error) {
	if len(opts.CryptoPolicyPaths) == 0 {
		return nil, errors.New("at least one crypto policy path is required")
	}
	if len(opts.ProviderManifestPaths) == 0 {
		return nil, errors.New("at least one provider manifest path is required")
	}

	policies := make([]*policy.CryptoPolicy, 0, len(opts.CryptoPolicyPaths))
	byID := make(map[string]*policy.CryptoPolicy, len(opts.CryptoPolicyPaths))
	for _, path := range opts.CryptoPolicyPaths {
		cp, loadErr := policy.LoadCryptoPolicyFromFile(path)
		if loadErr != nil {
			return nil, fmt.Errorf("load crypto policy %q: %w", path, loadErr)
		}
		if _, exists := byID[cp.ID]; exists {
			return nil, fmt.Errorf("duplicate crypto policy id %q", cp.ID)
		}
		byID[cp.ID] = cp
		policies = append(policies, cp)
	}

	providers, err := provider.LoadRegistryFromFiles(opts.ProviderManifestPaths)
	if err != nil {
		return nil, fmt.Errorf("load provider manifests: %w", err)
	}

	store := &ReadStore{
		cryptoPolicies:   policies,
		cryptoPolicyByID: byID,
		providers:        providers,
	}
	emitCatalogueLoadSignals(store, log.Default())
	return store, nil
}

// emitCatalogueLoadSignals runs ADR §7.2.1 family-1 checks after a successful
// catalogue load (CPM-P11a). Failures are logged + metered; they do not abort load.
func emitCatalogueLoadSignals(store *ReadStore, logger interface {
	Printf(format string, v ...any)
}) {
	if store == nil {
		return
	}
	malformed := store.providers.ApplyManifestLoadSignals(logger)
	metrics.AddCatalogueMalformedManifests(malformed)
	orphans := policy.CheckPostureOrphanage(store.cryptoPolicies, store.providers, logger)
	metrics.AddCataloguePostureOrphans(orphans)
}

func RegisterReadRoutes(mux *http.ServeMux, store *ReadStore) error {
	if mux == nil {
		return errors.New("mux is nil")
	}
	if store == nil {
		return ErrStoreNil
	}
	registerCatalogRoutes(mux, store)
	registerExploreRoute(mux, store)
	return nil
}

func registerCatalogRoutes(mux *http.ServeMux, store *ReadStore) {
	mux.HandleFunc("GET "+cpmroutes.CryptoPolicies, func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.cryptoPolicies})
	})
	mux.HandleFunc("GET "+cpmroutes.CryptoPolicyByID, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("crypto_policy_id")
		cp, ok := store.cryptoPolicyByID[id]
		if !ok {
			respondJSON(w, http.StatusNotFound, map[string]any{"error": "crypto policy not found"})
			return
		}
		respondJSON(w, http.StatusOK, cp)
	})
	mux.HandleFunc("GET "+cpmroutes.Providers, func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.providers.List()})
	})
	mux.HandleFunc("GET "+cpmroutes.ProviderByID, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("provider_id")
		m, ok := store.providers.Get(id)
		if !ok {
			respondJSON(w, http.StatusNotFound, map[string]any{"error": "provider not found"})
			return
		}
		respondJSON(w, http.StatusOK, m)
	})
}

func registerExploreRoute(mux *http.ServeMux, store *ReadStore) {
	// POST …/decisions/explore — HTTP explore wire v0.2 + couche A match (CPM-P9b).
	// Input: crypto_policy_id + policy_context only. Output: scan_compatible_providers.
	mux.HandleFunc("POST "+cpmroutes.PoliciesDecisionsExplore, func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeDecisionExploreRequest(r)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		cp, ok := store.cryptoPolicyByID[req.CryptoPolicyID]
		if !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown crypto_policy_id"})
			return
		}

		observation, err := observationFromDecisionExplore(req)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		evaluator := policy.ExploreCoucheAEvaluator{Providers: store.providers}
		decision, err := evaluator.EvaluateExploreCoucheA(observation, cp)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		recordExploreNoDeployableCandidate(r, req, decision)

		respondJSON(w, http.StatusOK, map[string]any{"decision": decision})
	})
}

type decisionExploreRequest struct {
	// Optional scan binding for AUTH-02 (scan authorization). Wire name is `scan_id` only.
	ScanID string `json:"scan_id,omitempty"`
	// CryptoPolicyID is the catalogue Crypto Policy id (required posture lives on the CP).
	CryptoPolicyID string `json:"crypto_policy_id"`
	// PolicyContext is required; scan context for couche A (full match in P9b).
	PolicyContext *walletPolicyContextWire `json:"policy_context"`
}

// Legacy / couche-B keys rejected on explore HTTP v0.2 (aligned with cafenatsv01 assessment).
var explorePayloadForbiddenKeys = map[string]struct{}{
	"selection_request":           {},
	"allow_new_wallet":            {},
	"address_continuity_required": {},
	"key_rotation_model":          {},
	"target_posture":              {},
	"target_chain_ids":            {},
	"require_multichain":          {},
	"recovery_required":           {},
	"minimum_maturity":            {},
	"allow_research":              {},
	"allowed_provider_modes":      {},
	"preferred_families":          {},
	"preferred_providers":         {},
	"require_bundler_available":   {},
	"require_paymaster_available": {},
	"approval_mode":               {},
}

func decodeDecisionExploreRequest(r *http.Request) (*decisionExploreRequest, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}
	defer func() {
		_ = r.Body.Close()
	}()
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	for key := range raw {
		if _, forbidden := explorePayloadForbiddenKeys[key]; forbidden {
			return nil, fmt.Errorf("legacy field rejected: %s", key)
		}
	}
	for key := range raw {
		switch key {
		case "scan_id", "crypto_policy_id", "policy_context":
		default:
			return nil, fmt.Errorf("unknown field %s", key)
		}
	}

	pcRaw, ok := raw["policy_context"]
	if !ok || len(bytesTrimSpaceJSON(pcRaw)) == 0 {
		return nil, errors.New("policy_context is required")
	}
	pc, err := parsePolicyContextFlexible(pcRaw)
	if err != nil {
		return nil, err
	}

	cpRaw, ok := raw["crypto_policy_id"]
	if !ok {
		return nil, errors.New("crypto_policy_id is required")
	}
	var cryptoPolicyID string
	if err := json.Unmarshal(cpRaw, &cryptoPolicyID); err != nil {
		return nil, fmt.Errorf("crypto_policy_id: %w", err)
	}
	cryptoPolicyID = strings.TrimSpace(cryptoPolicyID)
	if cryptoPolicyID == "" {
		return nil, errors.New("crypto_policy_id is required")
	}

	var scanID string
	if v, ok := raw["scan_id"]; ok {
		_ = json.Unmarshal(v, &scanID)
		scanID = strings.TrimSpace(scanID)
	}

	return &decisionExploreRequest{
		ScanID:         scanID,
		CryptoPolicyID: cryptoPolicyID,
		PolicyContext:  pc,
	}, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
