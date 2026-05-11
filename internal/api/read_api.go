package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
)

var (
	ErrStoreNil = errors.New("api read store is nil")
)

type ReadStoreOptions struct {
	CatalogPath   string
	TemplatePaths []string
	InstancePaths []string
}

type ReadStore struct {
	catalog      *policy.PolicyGraphCatalog
	templates    []*policy.CryptoPolicyTemplate
	instances    []*policy.CryptoPolicyInstance
	templateByID map[string]*policy.CryptoPolicyTemplate
}

func LoadReadStore(opts ReadStoreOptions) (*ReadStore, error) {
	if opts.CatalogPath == "" {
		return nil, errors.New("catalog path is required")
	}
	if len(opts.TemplatePaths) == 0 {
		return nil, errors.New("at least one template path is required")
	}
	if len(opts.InstancePaths) == 0 {
		return nil, errors.New("at least one instance path is required")
	}

	catalog, err := policy.LoadPolicyGraphCatalogFromFile(opts.CatalogPath)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}

	templates := make([]*policy.CryptoPolicyTemplate, 0, len(opts.TemplatePaths))
	templateByID := make(map[string]*policy.CryptoPolicyTemplate, len(opts.TemplatePaths))
	for _, path := range opts.TemplatePaths {
		tpl, loadErr := policy.LoadCryptoPolicyTemplateFromFile(path, catalog)
		if loadErr != nil {
			return nil, fmt.Errorf("load template %q: %w", path, loadErr)
		}
		if _, exists := templateByID[tpl.ID]; exists {
			return nil, fmt.Errorf("duplicate template id %q", tpl.ID)
		}
		templateByID[tpl.ID] = tpl
		templates = append(templates, tpl)
	}

	instances := make([]*policy.CryptoPolicyInstance, 0, len(opts.InstancePaths))
	instanceIDs := make(map[string]struct{}, len(opts.InstancePaths))
	for _, path := range opts.InstancePaths {
		inst, loadErr := policy.LoadCryptoPolicyInstanceFromFile(path, catalog)
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
		catalog:      catalog,
		templates:    templates,
		instances:    instances,
		templateByID: templateByID,
	}, nil
}

func RegisterReadRoutes(mux *http.ServeMux, store *ReadStore) error {
	if mux == nil {
		return errors.New("mux is nil")
	}
	if store == nil {
		return ErrStoreNil
	}

	mux.HandleFunc("GET /api/v1/policies/catalog", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"catalog": store.catalog})
	})
	mux.HandleFunc("GET /api/v1/policies/templates", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.templates})
	})
	mux.HandleFunc("GET /api/v1/policies/instances", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.instances})
	})
	mux.HandleFunc("POST /api/v1/policies/decisions/explore", func(w http.ResponseWriter, r *http.Request) {
		var req decisionExploreRequest
		if err := decodeJSON(r, &req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		observation, err := observationFromDecisionExplore(&req)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		candidates := make([]policy.PolicyDecisionCandidate, 0, len(store.instances))
		for _, inst := range store.instances {
			candidates = append(candidates, policy.PolicyDecisionCandidate{
				Instance: inst,
				Template: store.templateByID[inst.TemplateID],
			})
		}

		decision, err := (policy.PolicyDecisionEvaluator{}).Evaluate(
			observation,
			&req.SelectionRequest,
			candidates,
			store.catalog,
		)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{"decision": decision})
	})
	return nil
}

type decisionExploreRequest struct {
	// Optional scan binding for AUTH-02 (scan authorization). Ignored by Evaluate; wire name is `scan_id` only.
	ScanID string `json:"scan_id,omitempty"`
	// PolicyContext is required; evaluator input is derived from it (no top-level observation).
	PolicyContext    *walletPolicyContextWire       `json:"policy_context"`
	SelectionRequest policy.PolicySelectionRequest `json:"selection_request"`
}

func decodeJSON(r *http.Request, out any) error {
	if r == nil {
		return errors.New("request is nil")
	}
	defer func() {
		_ = r.Body.Close()
	}()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
	}
	if dec.More() {
		return errors.New("invalid json body: multiple values are not allowed")
	}
	return nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
