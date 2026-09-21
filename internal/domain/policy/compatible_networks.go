package policy

import (
	"sort"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

// CompatibleNetwork is a deployable catalogue network fact derived from provider
// manifests (union of chain_support with status ≠ planned for allowed_providers).
type CompatibleNetwork struct {
	ChainID      int64                       `json:"chain_id"`
	Network      string                      `json:"network"`
	Status       provider.ChainSupportStatus `json:"status"`
	Capabilities []string                    `json:"capabilities,omitempty"`
}

// CryptoPolicyCatalogItem is the GET /crypto-policies* response shape: catalogue
// Crypto Policy fields plus derived compatible_networks (ADR CFB-P1) and
// allowed_provider_summaries (ADR CFB-P10).
type CryptoPolicyCatalogItem struct {
	ID                       string                      `json:"id"`
	Name                     string                      `json:"name"`
	Version                  string                      `json:"version"`
	Description              string                      `json:"description,omitempty"`
	RequiredPosture          vocabulary.CurrentPQPosture `json:"required_posture"`
	AllowedProviders         []string                    `json:"allowed_providers"`
	CompatibleNetworks       []CompatibleNetwork         `json:"compatible_networks"`
	AllowedProviderSummaries []AllowedProviderSummary    `json:"allowed_provider_summaries"`
}

// CatalogItemFromCryptoPolicy builds a catalogue response item with derived facts.
func CatalogItemFromCryptoPolicy(cp *CryptoPolicy, reg *provider.Registry) CryptoPolicyCatalogItem {
	if cp == nil {
		return CryptoPolicyCatalogItem{
			CompatibleNetworks:       []CompatibleNetwork{},
			AllowedProviderSummaries: []AllowedProviderSummary{},
		}
	}
	item := CryptoPolicyCatalogItem{
		ID:                       cp.ID,
		Name:                     cp.Name,
		Version:                  cp.Version,
		Description:              cp.Description,
		RequiredPosture:          cp.RequiredPosture,
		AllowedProviders:         append([]string(nil), cp.AllowedProviders...),
		CompatibleNetworks:       DeriveCompatibleNetworks(cp, reg),
		AllowedProviderSummaries: DeriveAllowedProviderSummaries(cp, reg),
	}
	return item
}

// DeriveCompatibleNetworks returns the sorted union of deployable chain_support
// entries across all solution profiles of cp.AllowedProviders in the registry.
// Entries with status planned are excluded. Unknown providers contribute nothing.
func DeriveCompatibleNetworks(cp *CryptoPolicy, reg *provider.Registry) []CompatibleNetwork {
	out := make([]CompatibleNetwork, 0)
	if cp == nil || reg == nil {
		return out
	}

	seen := make(map[int64]struct{})
	for _, providerID := range cp.AllowedProviders {
		for _, resolved := range reg.ProfilesForProvider(providerID) {
			if resolved == nil {
				continue
			}
			for _, cs := range resolved.Profile.ChainSupport {
				if cs.Status == provider.ChainStatusPlanned || cs.ChainID <= 0 {
					continue
				}
				if _, ok := seen[cs.ChainID]; ok {
					continue
				}
				seen[cs.ChainID] = struct{}{}
				out = append(out, CompatibleNetwork{
					ChainID:      cs.ChainID,
					Network:      cs.Network,
					Status:       cs.Status,
					Capabilities: append([]string(nil), cs.Capabilities...),
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ChainID < out[j].ChainID
	})
	return out
}
