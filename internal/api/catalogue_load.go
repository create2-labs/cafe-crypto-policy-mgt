package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

type catalogueLogger interface {
	Printf(format string, v ...any)
}

type catalogueKind string

const (
	catalogueKindCryptoPolicy     catalogueKind = "crypto_policy"
	catalogueKindProviderManifest catalogueKind = "provider_manifest"
	catalogueKindSkip             catalogueKind = "skip"
)

// peekCatalogueJSON classifies a catalogue JSON blob without full validation.
type peekCatalogueJSON struct {
	SchemaVersion      string          `json:"schema_version"`
	ProviderID         string          `json:"provider_id"`
	SolutionProfiles   json.RawMessage `json:"solution_profiles"`
	RequiredPosture    string          `json:"required_posture"`
	AllowedProviders   json.RawMessage `json:"allowed_providers"`
	SolutionProfileRef json.RawMessage `json:"solution_profile_ref"`
	TemplateID         string          `json:"template_id"`
	CatalogVersion     string          `json:"catalog_version"`
}

func classifyCatalogueJSON(raw []byte) catalogueKind {
	var peek peekCatalogueJSON
	if err := json.Unmarshal(raw, &peek); err != nil {
		return catalogueKindSkip
	}
	schema := strings.ToLower(strings.TrimSpace(peek.SchemaVersion))
	if strings.Contains(schema, "provider_manifest") {
		return catalogueKindProviderManifest
	}
	if strings.TrimSpace(peek.ProviderID) != "" && len(peek.SolutionProfiles) > 0 && string(peek.SolutionProfiles) != "null" {
		return catalogueKindProviderManifest
	}
	// Persisted / instance fixtures are not catalogue Crypto Policies.
	if strings.TrimSpace(peek.TemplateID) != "" ||
		strings.TrimSpace(peek.CatalogVersion) != "" ||
		(len(peek.SolutionProfileRef) > 0 && string(peek.SolutionProfileRef) != "null") {
		return catalogueKindSkip
	}
	if strings.TrimSpace(peek.RequiredPosture) != "" &&
		len(peek.AllowedProviders) > 0 &&
		string(peek.AllowedProviders) != "null" {
		return catalogueKindCryptoPolicy
	}
	return catalogueKindSkip
}

func listCatalogueJSONFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	sort.Strings(out)
	return out, nil
}

func loadReadStoreFromCatalogueDir(dir string, logger catalogueLogger) (*ReadStore, error) {
	if logger == nil {
		logger = log.Default()
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("catalogue dir is required")
	}
	files, err := listCatalogueJSONFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("read catalogue dir %q: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("catalogue dir %q contains no .json files", dir)
	}

	policies := make([]*policy.CryptoPolicy, 0)
	byID := make(map[string]*policy.CryptoPolicy)
	reg := provider.NewRegistry()

	for _, path := range files {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			logger.Printf("cpm: catalogue skip %q: read error: %v", path, readErr)
			continue
		}
		switch classifyCatalogueJSON(raw) {
		case catalogueKindProviderManifest:
			m, loadErr := provider.LoadProviderManifestFromBytes(raw)
			if loadErr != nil {
				logger.Printf("cpm: catalogue skip %q: incompatible provider manifest: %v", path, loadErr)
				continue
			}
			if addErr := reg.AddManifest(m); addErr != nil {
				logger.Printf("cpm: catalogue skip %q: provider registry reject: %v", path, addErr)
				continue
			}
			logger.Printf("cpm: catalogue loaded provider_manifest %q (provider_id=%s)", path, m.ProviderID)
		case catalogueKindCryptoPolicy:
			cp, loadErr := policy.LoadCryptoPolicyFromBytes(raw)
			if loadErr != nil {
				logger.Printf("cpm: catalogue skip %q: incompatible crypto policy: %v", path, loadErr)
				continue
			}
			if _, exists := byID[cp.ID]; exists {
				logger.Printf("cpm: catalogue skip %q: duplicate crypto policy id %q", path, cp.ID)
				continue
			}
			byID[cp.ID] = cp
			policies = append(policies, cp)
			logger.Printf("cpm: catalogue loaded crypto_policy %q (id=%s)", path, cp.ID)
		default:
			logger.Printf("cpm: catalogue skip %q: not a crypto policy or provider manifest", path)
		}
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("catalogue dir %q: no valid crypto policies loaded", dir)
	}
	if len(reg.List()) == 0 {
		return nil, fmt.Errorf("catalogue dir %q: no valid provider manifests loaded", dir)
	}

	store := &ReadStore{
		cryptoPolicies:   policies,
		cryptoPolicyByID: byID,
		providers:        reg,
	}
	emitCatalogueLoadSignals(store, logger)
	return store, nil
}
