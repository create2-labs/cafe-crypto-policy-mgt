package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

const CryptoPolicySchemaVersionV02 = "cafe.crypto_policy.v0.2"

var (
	ErrCryptoPolicyPayloadInvalid   = errors.New("crypto policy persist payload is invalid")
	ErrProviderRefsUnpinned         = errors.New("provider references are unpinned or absent")
	ErrProviderSoftFindingsRequired = errors.New("provider soft findings must be listed in accepted_findings")
	ErrProviderChainPlanned         = errors.New("provider chain_support_used status planned is not persistable")
)

// CryptoPolicyPersistPayload is the normative CP body required at persist (ADR §9 / CPM-P6).
type CryptoPolicyPersistPayload struct {
	SchemaVersion            string                      `json:"schema_version"`
	PolicyKind               string                      `json:"policy_kind,omitempty"`
	TemplateID               string                      `json:"template_id"`
	RequiredPosture          vocabulary.CurrentPQPosture `json:"required_posture"`
	SolutionProfileRef       SolutionProfileRef          `json:"solution_profile_ref"`
	AcceptedProviderSnapshot AcceptedProviderSnapshot    `json:"accepted_provider_snapshot"`
}

// AcceptedProviderSnapshot freezes provider constraints accepted at persist time.
type AcceptedProviderSnapshot struct {
	Maturity          provider.Maturity           `json:"maturity"`
	ClaimStatus       provider.ClaimStatus        `json:"claim_status"`
	ResultingPosture  string                      `json:"resulting_posture"`
	InputRequirements provider.InputRequirements  `json:"input_requirements"`
	Signature         provider.SignatureProfile   `json:"signature"`
	AccountModel      provider.AccountModel       `json:"account_model"`
	Constraints       provider.ProfileConstraints `json:"constraints"`
	ChainSupportUsed  SnapshotChainSupport        `json:"chain_support_used"`
	References        []provider.Reference        `json:"references"`
	AcceptedFindings  []string                    `json:"accepted_findings"`
	AcceptedRiskNotes []string                    `json:"accepted_risk_notes,omitempty"`
}

// SnapshotChainSupport is the chain entry that qualified compatibility for this CP.
type SnapshotChainSupport struct {
	ChainID      int64                       `json:"chain_id"`
	Status       provider.ChainSupportStatus `json:"status"`
	Capabilities []string                    `json:"capabilities,omitempty"`
}

// ValidateDraftPayloadForPersist decodes and gates a draft payload before PersistDraftOnce.
// Does not re-resolve live manifests; cafe-persistence stays opaque to Nicetry logic.
func ValidateDraftPayloadForPersist(raw map[string]any) error {
	if raw == nil {
		return fmt.Errorf("%w: payload is nil", ErrCryptoPolicyPayloadInvalid)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("%w: marshal: %v", ErrCryptoPolicyPayloadInvalid, err)
	}
	var p CryptoPolicyPersistPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrCryptoPolicyPayloadInvalid, err)
	}
	return p.ValidateForPersist()
}

// ValidateForPersist enforces ADR §9: complete snapshot, soft findings listed, pinned refs.
func (p *CryptoPolicyPersistPayload) ValidateForPersist() error {
	if p == nil {
		return fmt.Errorf("%w: nil", ErrCryptoPolicyPayloadInvalid)
	}
	p.normalize()
	if p.SchemaVersion != CryptoPolicySchemaVersionV02 {
		return fmt.Errorf("%w: schema_version %q", ErrCryptoPolicyPayloadInvalid, p.SchemaVersion)
	}
	if p.TemplateID == "" {
		return fmt.Errorf("%w: template_id is required", ErrCryptoPolicyPayloadInvalid)
	}
	if !p.RequiredPosture.IsValid() || p.RequiredPosture == vocabulary.PQPostureUnknown {
		return fmt.Errorf("%w: required_posture %q", ErrCryptoPolicyPayloadInvalid, p.RequiredPosture)
	}
	if err := p.SolutionProfileRef.Validate(); err != nil {
		return fmt.Errorf("%w: solution_profile_ref: %v", ErrCryptoPolicyPayloadInvalid, err)
	}
	s := &p.AcceptedProviderSnapshot
	if s.Maturity == "" || s.ClaimStatus == "" || s.ResultingPosture == "" {
		return fmt.Errorf("%w: accepted_provider_snapshot maturity/claim_status/resulting_posture required", ErrCryptoPolicyPayloadInvalid)
	}
	if !oneOf(string(s.ClaimStatus), "declared", "cafe_reviewed", "externally_audited", "executed_observed") {
		return fmt.Errorf("%w: claim_status %q", ErrCryptoPolicyPayloadInvalid, s.ClaimStatus)
	}
	if !oneOf(s.ResultingPosture, "classical_only", "hybrid", "full_pq") {
		return fmt.Errorf("%w: resulting_posture %q", ErrCryptoPolicyPayloadInvalid, s.ResultingPosture)
	}
	if s.Signature.Scheme == "" || s.Signature.Family == "" {
		return fmt.Errorf("%w: accepted_provider_snapshot.signature scheme/family required", ErrCryptoPolicyPayloadInvalid)
	}
	if s.ChainSupportUsed.ChainID <= 0 || s.ChainSupportUsed.Status == "" {
		return fmt.Errorf("%w: chain_support_used chain_id/status required", ErrCryptoPolicyPayloadInvalid)
	}
	if s.ChainSupportUsed.Status == provider.ChainStatusPlanned {
		return ErrProviderChainPlanned
	}
	accepted := make(map[string]struct{}, len(s.AcceptedFindings))
	for _, code := range s.AcceptedFindings {
		accepted[code] = struct{}{}
	}
	var missing []string
	if s.AccountModel.RequiresBundler {
		if _, ok := accepted[provider.FindingCodeRequiresBundler]; !ok {
			missing = append(missing, provider.FindingCodeRequiresBundler)
		}
	}
	if s.Constraints.RequiresLocalSignerState {
		if _, ok := accepted[provider.FindingCodeRequiresLocalSignerState]; !ok {
			missing = append(missing, provider.FindingCodeRequiresLocalSignerState)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing %s", ErrProviderSoftFindingsRequired, strings.Join(missing, ", "))
	}
	if len(s.References) == 0 {
		return fmt.Errorf("%w: references required", ErrProviderRefsUnpinned)
	}
	for i, ref := range s.References {
		if ref.IsUnpinned() {
			return fmt.Errorf("%w: references[%d] commit/version is %s or empty", ErrProviderRefsUnpinned, i, provider.UnpinnedPendingFixture)
		}
	}
	return nil
}

func (p *CryptoPolicyPersistPayload) normalize() {
	p.SchemaVersion = strings.TrimSpace(p.SchemaVersion)
	p.PolicyKind = strings.TrimSpace(p.PolicyKind)
	p.TemplateID = strings.TrimSpace(p.TemplateID)
	p.RequiredPosture = vocabulary.CurrentPQPosture(strings.TrimSpace(string(p.RequiredPosture)))
	p.SolutionProfileRef.ProviderID = strings.TrimSpace(p.SolutionProfileRef.ProviderID)
	p.SolutionProfileRef.SolutionProfileID = strings.TrimSpace(p.SolutionProfileRef.SolutionProfileID)
	p.SolutionProfileRef.ManifestVersion = strings.TrimSpace(p.SolutionProfileRef.ManifestVersion)
	p.SolutionProfileRef.VerificationDate = strings.TrimSpace(p.SolutionProfileRef.VerificationDate)
	s := &p.AcceptedProviderSnapshot
	s.Maturity = provider.Maturity(strings.TrimSpace(string(s.Maturity)))
	s.ClaimStatus = provider.ClaimStatus(strings.TrimSpace(string(s.ClaimStatus)))
	s.ResultingPosture = strings.TrimSpace(s.ResultingPosture)
	s.Signature.Scheme = strings.TrimSpace(s.Signature.Scheme)
	s.Signature.Family = strings.TrimSpace(s.Signature.Family)
	s.ChainSupportUsed.Status = provider.ChainSupportStatus(strings.TrimSpace(string(s.ChainSupportUsed.Status)))
	for i := range s.References {
		s.References[i].Kind = provider.ReferenceKind(strings.TrimSpace(string(s.References[i].Kind)))
		s.References[i].URL = strings.TrimSpace(s.References[i].URL)
		s.References[i].Commit = strings.TrimSpace(s.References[i].Commit)
		s.References[i].Version = strings.TrimSpace(s.References[i].Version)
	}
	s.AcceptedFindings = trimNonEmpty(s.AcceptedFindings)
	s.AcceptedRiskNotes = trimNonEmpty(s.AcceptedRiskNotes)
}

func trimNonEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func oneOf(v string, allowed ...string) bool {
	return slices.Contains(allowed, v)
}
