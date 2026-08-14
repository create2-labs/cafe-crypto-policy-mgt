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

type ReadStoreOptions struct {
	TemplatePaths         []string
	InstancePaths         []string
	ProviderManifestPaths []string
}

type ReadStore struct {
	templates    []*policy.CryptoPolicyTemplate
	instances    []*policy.CryptoPolicyInstance
	templateByID map[string]*policy.CryptoPolicyTemplate
	providers    *provider.Registry
}

func LoadReadStore(opts ReadStoreOptions) (*ReadStore, error) {
	if len(opts.TemplatePaths) == 0 {
		return nil, errors.New("at least one template path is required")
	}
	if len(opts.InstancePaths) == 0 {
		return nil, errors.New("at least one instance path is required")
	}

	templates := make([]*policy.CryptoPolicyTemplate, 0, len(opts.TemplatePaths))
	templateByID := make(map[string]*policy.CryptoPolicyTemplate, len(opts.TemplatePaths))
	for _, path := range opts.TemplatePaths {
		tpl, loadErr := policy.LoadCryptoPolicyTemplateFromFile(path)
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

	var providers *provider.Registry
	if len(opts.ProviderManifestPaths) > 0 {
		var loadErr error
		providers, loadErr = provider.LoadRegistryFromFiles(opts.ProviderManifestPaths)
		if loadErr != nil {
			return nil, fmt.Errorf("load provider manifests: %w", loadErr)
		}
	}

	return &ReadStore{
		templates:    templates,
		instances:    instances,
		templateByID: templateByID,
		providers:    providers,
	}, nil
}

func RegisterReadRoutes(mux *http.ServeMux, store *ReadStore) error {
	if mux == nil {
		return errors.New("mux is nil")
	}
	if store == nil {
		return ErrStoreNil
	}
	registerReadRoutesForPrefix(mux, store, cpmroutes.PoliciesPrefix)
	return nil
}

func registerReadRoutesForPrefix(mux *http.ServeMux, store *ReadStore, prefix string) {
	mux.HandleFunc("GET "+prefix+"/catalog", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"templates": store.templates,
			"instances": store.instances,
		})
	})
	mux.HandleFunc("GET "+prefix+"/templates", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.templates})
	})
	mux.HandleFunc("GET "+prefix+"/instances", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": store.instances})
	})
	// POST …/decisions/explore evaluates candidates in memory only (no persisted policy instances).
	mux.HandleFunc("POST "+prefix+"/decisions/explore", func(w http.ResponseWriter, r *http.Request) {
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
				Instance: inst,
				Template: store.templateByID[inst.TemplateID],
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
