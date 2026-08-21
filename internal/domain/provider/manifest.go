package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	SchemaVersionV01       = "cafe.provider_manifest.v0.1"
	UnpinnedPendingFixture = "unpinned_pending_fixture" // fixture-only; not persistable (ADR §6)
)

type Maturity string

const (
	MaturityResearch   Maturity = "research"
	MaturityTestnet    Maturity = "testnet"
	MaturityProduction Maturity = "production"
)

// ClaimStatus is the CAFE attestation level. "declared" is never audited/executed proof.
type ClaimStatus string

const (
	ClaimDeclared          ClaimStatus = "declared"
	ClaimCafeReviewed      ClaimStatus = "cafe_reviewed"
	ClaimExternallyAudited ClaimStatus = "externally_audited"
	ClaimExecutedObserved  ClaimStatus = "executed_observed"
)

type KeyRotationModel string

const (
	KeyRotationNone      KeyRotationModel = "none"
	KeyRotationPerUserOp KeyRotationModel = "per_userop"
)

type ChainSupportStatus string

const (
	ChainStatusTestnetSupported ChainSupportStatus = "testnet_supported"
	ChainStatusPlanned          ChainSupportStatus = "planned"
	ChainStatusProduction       ChainSupportStatus = "production_supported"
)

type ReferenceKind string

const (
	ReferenceKindSourceRepo   ReferenceKind = "source_repo"
	ReferenceKindProtocolSpec ReferenceKind = "protocol_spec"
)

var (
	ErrInvalidManifest    = errors.New("provider manifest is invalid")
	ErrClaimStatusInvalid = errors.New("solution_profile claim_status is invalid")
)

type ProviderManifest struct {
	SchemaVersion    string            `json:"schema_version"`
	ProviderID       string            `json:"provider_id"`
	ProviderName     string            `json:"provider_name"`
	ProviderVersion  string            `json:"provider_version"`
	ProviderMaturity Maturity          `json:"provider_maturity"`
	Website          string            `json:"website,omitempty"`
	SolutionProfiles []SolutionProfile `json:"solution_profiles"`
}

type SolutionProfile struct {
	SolutionProfileID        string                    `json:"solution_profile_id"`
	DisplayName              string                    `json:"display_name"`
	Maturity                 Maturity                  `json:"maturity"`
	ClaimStatus              ClaimStatus               `json:"claim_status"`
	ResultingPosture         string                    `json:"resulting_posture"` // CAFE: classical_only|hybrid|full_pq
	InputRequirements        InputRequirements         `json:"input_requirements"`
	Signature                SignatureProfile          `json:"signature"`
	AccountModel             AccountModel              `json:"account_model"`
	Constraints              ProfileConstraints        `json:"constraints"`
	SuggestedUserConstraints *SuggestedUserConstraints `json:"suggested_user_constraints,omitempty"`
	ChainSupport             []ChainSupport            `json:"chain_support"`
	References               []Reference               `json:"references,omitempty"`
	RiskNotes                []string                  `json:"risk_notes,omitempty"`
}

// SuggestedUserConstraints are indicative couche-B defaults for the UI (ADR §6.2 rule 9).
// They are not authoritative for explore scan-compatible membership.
type SuggestedUserConstraints struct {
	AllowNewWallet            bool             `json:"allow_new_wallet"`
	AddressContinuityRequired bool             `json:"address_continuity_required"`
	KeyRotationModel          KeyRotationModel `json:"key_rotation_model"`
}

type InputRequirements struct {
	WalletTypes                []string `json:"wallet_types"`
	RequiresWalletControlProof bool     `json:"requires_wallet_control_proof"`
}

type SignatureProfile struct {
	Scheme           string           `json:"scheme"`
	Family           string           `json:"family"`
	KeyRotationModel KeyRotationModel `json:"key_rotation_model"`
}

type AccountModel struct {
	Standard           string   `json:"standard"`
	ExecutionModel     string   `json:"execution_model"`
	RequiresBundler    bool     `json:"requires_bundler"`
	RequiresEntrypoint bool     `json:"requires_entrypoint"`
	EntrypointVersions []string `json:"entrypoint_versions,omitempty"`
}

type ProfileConstraints struct {
	RequiresNewAccount         bool `json:"requires_new_account"`
	AddressContinuitySupported bool `json:"address_continuity_supported"`
	RequiresLocalSignerState   bool `json:"requires_local_signer_state"`
}

type ChainSupport struct {
	ChainID      int64              `json:"chain_id"`
	Network      string             `json:"network"`
	Status       ChainSupportStatus `json:"status"`
	Capabilities []string           `json:"capabilities,omitempty"`
}

type Reference struct {
	Kind    ReferenceKind `json:"kind"`
	URL     string        `json:"url"`
	Commit  string        `json:"commit,omitempty"`
	Version string        `json:"version,omitempty"`
}

func LoadProviderManifestFromFile(path string) (*ProviderManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider manifest file: %w", err)
	}
	var m ProviderManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode provider manifest file: %w", err)
	}
	if err := m.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *ProviderManifest) Normalize() {
	if m == nil {
		return
	}
	m.SchemaVersion = strings.TrimSpace(m.SchemaVersion)
	m.ProviderID = strings.TrimSpace(m.ProviderID)
	m.ProviderName = strings.TrimSpace(m.ProviderName)
	m.ProviderVersion = strings.TrimSpace(m.ProviderVersion)
	m.ProviderMaturity = Maturity(strings.TrimSpace(string(m.ProviderMaturity)))
	m.Website = strings.TrimSpace(m.Website)
	for i := range m.SolutionProfiles {
		normalizeSolutionProfile(&m.SolutionProfiles[i])
	}
}

func normalizeSolutionProfile(p *SolutionProfile) {
	p.SolutionProfileID = strings.TrimSpace(p.SolutionProfileID)
	p.DisplayName = strings.TrimSpace(p.DisplayName)
	p.Maturity = Maturity(strings.TrimSpace(string(p.Maturity)))
	p.ClaimStatus = ClaimStatus(strings.TrimSpace(string(p.ClaimStatus)))
	p.ResultingPosture = strings.TrimSpace(p.ResultingPosture)
	p.InputRequirements.WalletTypes = dedupeStrings(p.InputRequirements.WalletTypes)
	p.Signature.Scheme = strings.TrimSpace(p.Signature.Scheme)
	p.Signature.Family = strings.TrimSpace(p.Signature.Family)
	p.Signature.KeyRotationModel = KeyRotationModel(strings.TrimSpace(string(p.Signature.KeyRotationModel)))
	p.AccountModel.Standard = strings.TrimSpace(p.AccountModel.Standard)
	p.AccountModel.ExecutionModel = strings.TrimSpace(p.AccountModel.ExecutionModel)
	p.AccountModel.EntrypointVersions = dedupeStrings(p.AccountModel.EntrypointVersions)
	if p.SuggestedUserConstraints != nil {
		p.SuggestedUserConstraints.KeyRotationModel = KeyRotationModel(
			strings.TrimSpace(string(p.SuggestedUserConstraints.KeyRotationModel)),
		)
	}
	for j := range p.ChainSupport {
		p.ChainSupport[j].Network = strings.TrimSpace(p.ChainSupport[j].Network)
		p.ChainSupport[j].Status = ChainSupportStatus(strings.TrimSpace(string(p.ChainSupport[j].Status)))
		p.ChainSupport[j].Capabilities = dedupeStrings(p.ChainSupport[j].Capabilities)
	}
	for j := range p.References {
		p.References[j].Kind = ReferenceKind(strings.TrimSpace(string(p.References[j].Kind)))
		p.References[j].URL = strings.TrimSpace(p.References[j].URL)
		p.References[j].Commit = strings.TrimSpace(p.References[j].Commit)
		p.References[j].Version = strings.TrimSpace(p.References[j].Version)
	}
	p.RiskNotes = dedupeStrings(p.RiskNotes)
}

func (m *ProviderManifest) Validate() error {
	if m == nil {
		return fmt.Errorf("%w: nil", ErrInvalidManifest)
	}
	if m.SchemaVersion != SchemaVersionV01 {
		return fmt.Errorf("%w: schema_version %q", ErrInvalidManifest, m.SchemaVersion)
	}
	if m.ProviderID == "" || m.ProviderName == "" || m.ProviderVersion == "" {
		return fmt.Errorf("%w: provider identity fields required", ErrInvalidManifest)
	}
	if !oneOf(string(m.ProviderMaturity), "research", "testnet", "production") {
		return fmt.Errorf("%w: provider_maturity %q", ErrInvalidManifest, m.ProviderMaturity)
	}
	if len(m.SolutionProfiles) == 0 {
		return fmt.Errorf("%w: solution_profiles required", ErrInvalidManifest)
	}
	seen := make(map[string]struct{}, len(m.SolutionProfiles))
	for i := range m.SolutionProfiles {
		if err := validateSolutionProfile(&m.SolutionProfiles[i]); err != nil {
			return fmt.Errorf("solution_profiles[%d]: %w", i, err)
		}
		id := m.SolutionProfiles[i].SolutionProfileID
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: duplicate solution_profile_id %q", ErrInvalidManifest, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (m *ProviderManifest) NormalizeAndValidate() error {
	m.Normalize()
	return m.Validate()
}

func (m *ProviderManifest) HasUnpinnedReferences() bool {
	if m == nil {
		return false
	}
	for i := range m.SolutionProfiles {
		for _, ref := range m.SolutionProfiles[i].References {
			if ref.IsUnpinned() {
				return true
			}
		}
	}
	return false
}

func (r Reference) IsUnpinned() bool {
	if r.Commit == UnpinnedPendingFixture || r.Version == UnpinnedPendingFixture {
		return true
	}
	return r.Commit == "" && r.Version == ""
}

func validateSolutionProfile(p *SolutionProfile) error {
	if p.SolutionProfileID == "" || p.DisplayName == "" {
		return fmt.Errorf("%w: solution_profile_id and display_name required", ErrInvalidManifest)
	}
	if !oneOf(string(p.Maturity), "research", "testnet", "production") {
		return fmt.Errorf("%w: maturity %q", ErrInvalidManifest, p.Maturity)
	}
	if p.ClaimStatus == "" {
		return fmt.Errorf("%w: claim_status required", ErrInvalidManifest)
	}
	if !oneOf(string(p.ClaimStatus), "declared", "cafe_reviewed", "externally_audited", "executed_observed") {
		return fmt.Errorf("%w: %q", ErrClaimStatusInvalid, p.ClaimStatus)
	}
	if !oneOf(p.ResultingPosture, "classical_only", "hybrid", "full_pq") {
		return fmt.Errorf("%w: resulting_posture %q", ErrInvalidManifest, p.ResultingPosture)
	}
	if len(p.InputRequirements.WalletTypes) == 0 {
		return fmt.Errorf("%w: wallet_types required", ErrInvalidManifest)
	}
	sig := p.Signature
	if sig.Scheme == "" || sig.Family == "" {
		return fmt.Errorf("%w: signature scheme and family required", ErrInvalidManifest)
	}
	if !oneOf(string(sig.KeyRotationModel), "none", "per_userop") {
		return fmt.Errorf("%w: key_rotation_model %q", ErrInvalidManifest, sig.KeyRotationModel)
	}
	if p.AccountModel.Standard == "" || p.AccountModel.ExecutionModel == "" {
		return fmt.Errorf("%w: account_model fields required", ErrInvalidManifest)
	}
	if len(p.ChainSupport) == 0 {
		return fmt.Errorf("%w: chain_support required", ErrInvalidManifest)
	}
	for i, c := range p.ChainSupport {
		if c.ChainID <= 0 || c.Network == "" || !oneOf(string(c.Status), "testnet_supported", "planned", "production_supported") {
			return fmt.Errorf("%w: chain_support[%d]", ErrInvalidManifest, i)
		}
	}
	for i, r := range p.References {
		if !oneOf(string(r.Kind), "source_repo", "protocol_spec") || r.URL == "" {
			return fmt.Errorf("%w: references[%d]", ErrInvalidManifest, i)
		}
	}
	if p.SuggestedUserConstraints != nil {
		if !oneOf(string(p.SuggestedUserConstraints.KeyRotationModel), "none", "per_userop") {
			return fmt.Errorf("%w: suggested_user_constraints.key_rotation_model %q", ErrInvalidManifest, p.SuggestedUserConstraints.KeyRotationModel)
		}
	}
	return nil
}

// ErrSuggestedConstraintsContradict is returned when suggested_user_constraints
// contradict profile constraints / signature (ADR §6.2 rule 9).
var ErrSuggestedConstraintsContradict = errors.New("suggested_user_constraints contradict profile constraints")

// ValidateSuggestedUserConstraints reports contradictions vs constraints / signature.
// A nil suggestion is valid (UI shows no pre-checks). Load remains permissive;
// explore marks the profile erroneous when this returns an error (P9b); startup
// catalogue signals are P11a.
func ValidateSuggestedUserConstraints(p *SolutionProfile) error {
	if p == nil || p.SuggestedUserConstraints == nil {
		return nil
	}
	s := p.SuggestedUserConstraints
	if !s.AllowNewWallet && p.Constraints.RequiresNewAccount {
		return fmt.Errorf("%w: allow_new_wallet=false but constraints.requires_new_account=true", ErrSuggestedConstraintsContradict)
	}
	if s.AddressContinuityRequired && !p.Constraints.AddressContinuitySupported {
		return fmt.Errorf("%w: address_continuity_required=true but constraints.address_continuity_supported=false", ErrSuggestedConstraintsContradict)
	}
	if s.KeyRotationModel != "" && s.KeyRotationModel != p.Signature.KeyRotationModel {
		return fmt.Errorf(
			"%w: key_rotation_model %q does not match signature.key_rotation_model %q",
			ErrSuggestedConstraintsContradict, s.KeyRotationModel, p.Signature.KeyRotationModel,
		)
	}
	return nil
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
