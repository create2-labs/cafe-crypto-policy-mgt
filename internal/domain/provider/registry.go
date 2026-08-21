package provider

import (
	"fmt"
	"sort"
	"strings"
)

// ProfileRef identifies a solution profile inside a loaded ProviderManifest.
type ProfileRef struct {
	ProviderID        string
	SolutionProfileID string
	ManifestVersion   string // optional; when set, must match ProviderVersion
}

// ResolvedProfile is a solution profile bound to its parent manifest identity.
type ResolvedProfile struct {
	ProviderID      string
	ProviderVersion string
	Profile         SolutionProfile
}

// Registry indexes ProviderManifest files by (provider_id, solution_profile_id)
// and retains full manifests for catalogue listing.
type Registry struct {
	byKey     map[string]*ResolvedProfile
	manifests map[string]*ProviderManifest
}

func profileKey(providerID, solutionProfileID string) string {
	return strings.ToLower(strings.TrimSpace(providerID)) + "\x00" + strings.TrimSpace(solutionProfileID)
}

// LoadRegistryFromFiles loads and indexes one or more ProviderManifest JSON files.
// Duplicate (provider_id, solution_profile_id) pairs are rejected.
func LoadRegistryFromFiles(paths []string) (*Registry, error) {
	reg := &Registry{
		byKey:     make(map[string]*ResolvedProfile),
		manifests: make(map[string]*ProviderManifest),
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		m, err := LoadProviderManifestFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("provider manifest %q: %w", path, err)
		}
		if err := reg.addManifest(m); err != nil {
			return nil, fmt.Errorf("provider manifest %q: %w", path, err)
		}
	}
	return reg, nil
}

func (r *Registry) addManifest(m *ProviderManifest) error {
	if r == nil || m == nil {
		return fmt.Errorf("%w: nil registry or manifest", ErrInvalidManifest)
	}
	if r.byKey == nil {
		r.byKey = make(map[string]*ResolvedProfile)
	}
	if r.manifests == nil {
		r.manifests = make(map[string]*ProviderManifest)
	}
	providerID := strings.ToLower(strings.TrimSpace(m.ProviderID))
	if _, exists := r.manifests[providerID]; exists {
		return fmt.Errorf("%w: duplicate provider_id %s", ErrInvalidManifest, m.ProviderID)
	}
	r.manifests[providerID] = m
	for i := range m.SolutionProfiles {
		key := profileKey(m.ProviderID, m.SolutionProfiles[i].SolutionProfileID)
		if _, exists := r.byKey[key]; exists {
			return fmt.Errorf("%w: duplicate solution profile %s/%s", ErrInvalidManifest, m.ProviderID, m.SolutionProfiles[i].SolutionProfileID)
		}
		r.byKey[key] = &ResolvedProfile{
			ProviderID:      m.ProviderID,
			ProviderVersion: m.ProviderVersion,
			Profile:         m.SolutionProfiles[i],
		}
	}
	return nil
}

// Lookup resolves a solution profile. When ref.ManifestVersion is set, it must
// equal the manifest provider_version.
func (r *Registry) Lookup(ref ProfileRef) (*ResolvedProfile, bool) {
	if r == nil || r.byKey == nil {
		return nil, false
	}
	got, ok := r.byKey[profileKey(ref.ProviderID, ref.SolutionProfileID)]
	if !ok || got == nil {
		return nil, false
	}
	wantVer := strings.TrimSpace(ref.ManifestVersion)
	if wantVer != "" && wantVer != got.ProviderVersion {
		return nil, false
	}
	return got, true
}

// List returns loaded provider manifests sorted by provider_id.
func (r *Registry) List() []*ProviderManifest {
	if r == nil || len(r.manifests) == 0 {
		return nil
	}
	out := make([]*ProviderManifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ProviderID) < strings.ToLower(out[j].ProviderID)
	})
	return out
}

// Get returns a loaded provider manifest by provider_id.
func (r *Registry) Get(providerID string) (*ProviderManifest, bool) {
	if r == nil || r.manifests == nil {
		return nil, false
	}
	m, ok := r.manifests[strings.ToLower(strings.TrimSpace(providerID))]
	return m, ok
}

// ProfilesForProvider returns all resolved profiles for a provider_id,
// sorted by solution_profile_id for deterministic explore output.
func (r *Registry) ProfilesForProvider(providerID string) []*ResolvedProfile {
	m, ok := r.Get(providerID)
	if !ok || m == nil {
		return nil
	}
	out := make([]*ResolvedProfile, 0, len(m.SolutionProfiles))
	for i := range m.SolutionProfiles {
		out = append(out, &ResolvedProfile{
			ProviderID:      m.ProviderID,
			ProviderVersion: m.ProviderVersion,
			Profile:         m.SolutionProfiles[i],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Profile.SolutionProfileID < out[j].Profile.SolutionProfileID
	})
	return out
}
