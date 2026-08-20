package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

var (
	ErrStoreNil = errors.New("api read store is nil")
)

// ReadStoreOptions configures catalogue loading for Crypto Policies and providers.
// InstancePaths is optional and transitional for explore until CPM-P9 (not a public catalogue).
type ReadStoreOptions struct {
	CryptoPolicyPaths     []string
	ProviderManifestPaths []string
	InstancePaths         []string
}

// ReadStore holds catalogue Crypto Policies, provider manifests, and optional
// explore-only instances (removed from public catalogue in CPM-P8).
type ReadStore struct {
	cryptoPolicies   []*policy.CryptoPolicy
	cryptoPolicyByID map[string]*policy.CryptoPolicy
	instances        []*policy.CryptoPolicyInstance
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

	instances := make([]*policy.CryptoPolicyInstance, 0, len(opts.InstancePaths))
	instanceIDs := make(map[string]struct{}, len(opts.InstancePaths))
	for _, path := range opts.InstancePaths {
		inst, loadErr := policy.LoadCryptoPolicyInstanceFromFile(path)
		if loadErr != nil {
			return nil, fmt.Errorf("load instance %q: %w", path, loadErr)
		}
		if _, exists := instanceIDs[inst.ID]; exists {
			return nil, fmt.Errorf("duplicate instance id %q", inst.ID)
		}
		instanceIDs[inst.ID] = struct{}{}
		instances = append(instances, inst)
	}

	return &ReadStore{
		cryptoPolicies:   policies,
		cryptoPolicyByID: byID,
		instances:        instances,
		providers:        providers,
	}, nil
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
	// POST …/decisions/explore evaluates candidates in memory only (no persisted policy instances).
	// Candidate construction from catalogue instances is transitional until CPM-P9.
	mux.HandleFunc("POST "+cpmroutes.PoliciesDecisionsExplore, func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeDecisionExploreRequest(r)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		observation, err := observationFromDecisionExplore(req)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		candidates := make([]policy.PolicyDecisionCandidate, 0, len(store.instances))
		for _, inst := range store.instances {
			candidates = append(candidates, policy.PolicyDecisionCandidate{
				Instance:     inst,
				CryptoPolicy: store.cryptoPolicyByID[inst.TemplateID],
			})
		}

		decision, err := (policy.PolicyDecisionEvaluator{
			CompatibilityEvaluator: policy.PolicyCompatibilityEvaluator{Providers: store.providers},
		}).Evaluate(
			observation,
			&req.SelectionRequest,
			candidates,
		)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		recordExploreNoDeployableCandidate(r, req, decision, instanceScopeByID(store.instances))

		respondJSON(w, http.StatusOK, map[string]any{"decision": decision})
	})
}

type decisionExploreRequest struct {
	// Optional scan binding for AUTH-02 (scan authorization). Ignored by Evaluate; wire name is `scan_id` only.
	ScanID string `json:"scan_id,omitempty"`
	// PolicyContext is required; evaluator input is derived from it (no top-level observation).
	PolicyContext    *walletPolicyContextWire      `json:"policy_context"`
	SelectionRequest policy.PolicySelectionRequest `json:"selection_request"`
}

type decisionExploreBody struct {
	ScanID           string          `json:"scan_id,omitempty"`
	PolicyContext    json.RawMessage `json:"policy_context"`
	SelectionRequest json.RawMessage `json:"selection_request"`
}

func decodeDecisionExploreRequest(r *http.Request) (*decisionExploreRequest, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}
	defer func() {
		_ = r.Body.Close()
	}()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var body decisionExploreBody
	if err := dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	if dec.More() {
		return nil, errors.New("invalid json body: multiple values are not allowed")
	}
	if len(bytesTrimSpaceJSON(body.PolicyContext)) == 0 {
		return nil, errors.New("policy_context is required")
	}
	pc, err := parsePolicyContextFlexible(body.PolicyContext)
	if err != nil {
		return nil, err
	}
	decSel := json.NewDecoder(bytes.NewReader(body.SelectionRequest))
	decSel.DisallowUnknownFields()
	var sel policy.PolicySelectionRequest
	if err := decSel.Decode(&sel); err != nil {
		return nil, fmt.Errorf("selection_request: %w", err)
	}
	if decSel.More() {
		return nil, errors.New("selection_request: multiple values are not allowed")
	}
	return &decisionExploreRequest{
		ScanID:           body.ScanID,
		PolicyContext:    pc,
		SelectionRequest: sel,
	}, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
