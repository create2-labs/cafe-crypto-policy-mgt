package provider

import (
	"fmt"
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

// Registry indexes ProviderManifest files by (provider_id, solution_profile_id).
type Registry struct {
	byKey map[string]*ResolvedProfile
}

func profileKey(providerID, solutionProfileID string) string {
	return strings.ToLower(strings.TrimSpace(providerID)) + "\x00" + strings.TrimSpace(solutionProfileID)
}

// LoadRegistryFromFiles loads and indexes one or more ProviderManifest JSON files.
// Duplicate (provider_id, solution_profile_id) pairs are rejected.
func LoadRegistryFromFiles(paths []string) (*Registry, error) {
	reg := &Registry{byKey: make(map[string]*ResolvedProfile)}
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
