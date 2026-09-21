package policy

import (
	"sort"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

// ProviderSignatureSummary is the catalogue signature fact for one allowed provider
// (scheme / family only — not the admin SignatureProfile).
type ProviderSignatureSummary struct {
	Scheme string `json:"scheme"`
	Family string `json:"family"`
}

// AllowedProviderSummary is one catalogue table row derived from the registry for an
// allowed_providers entry (ADR CFB-P10 / US-CAT-PROVIDERS).
type AllowedProviderSummary struct {
	ProviderID string                   `json:"provider_id"`
	Signature  ProviderSignatureSummary `json:"signature"`
	Networks   []CompatibleNetwork      `json:"networks"`
}

// DeriveAllowedProviderSummaries returns one summary per cp.AllowedProviders id that
// resolves in the registry. When a provider has several solution profiles, a
// posture-compatible profile (resulting_posture == required_posture) is preferred;
// ties follow ProfilesForProvider sort (solution_profile_id). Planned chains are
// excluded from networks. Unknown providers contribute nothing.
func DeriveAllowedProviderSummaries(cp *CryptoPolicy, reg *provider.Registry) []AllowedProviderSummary {
	out := make([]AllowedProviderSummary, 0)
	if cp == nil || reg == nil {
		return out
	}
	required := strings.TrimSpace(string(cp.RequiredPosture))

	for _, providerID := range cp.AllowedProviders {
		providerID = strings.TrimSpace(providerID)
		if providerID == "" {
			continue
		}
		profiles := reg.ProfilesForProvider(providerID)
		chosen := pickCatalogProfile(profiles, required)
		if chosen == nil {
			continue
		}
		out = append(out, AllowedProviderSummary{
			ProviderID: providerID,
			Signature: ProviderSignatureSummary{
				Scheme: chosen.Profile.Signature.Scheme,
				Family: chosen.Profile.Signature.Family,
			},
			Networks: deployableNetworksFromProfile(&chosen.Profile),
		})
	}
	return out
}

func pickCatalogProfile(profiles []*provider.ResolvedProfile, requiredPosture string) *provider.ResolvedProfile {
	var first *provider.ResolvedProfile
	for _, resolved := range profiles {
		if resolved == nil {
			continue
		}
		if first == nil {
			first = resolved
		}
		if requiredPosture != "" &&
			strings.TrimSpace(resolved.Profile.ResultingPosture) == requiredPosture {
			return resolved
		}
	}
	return first
}

func deployableNetworksFromProfile(profile *provider.SolutionProfile) []CompatibleNetwork {
	out := make([]CompatibleNetwork, 0)
	if profile == nil {
		return out
	}
	seen := make(map[int64]struct{})
	for _, cs := range profile.ChainSupport {
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
	sort.Slice(out, func(i, j int) bool {
		return out[i].ChainID < out[j].ChainID
	})
	return out
}
