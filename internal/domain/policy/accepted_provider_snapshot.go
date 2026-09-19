package policy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/payloadhash"
)

var (
	// ErrSnapshotAssistProfileNotFound indicates the solution_profile_ref is unknown
	// or manifest_version does not match the loaded registry.
	ErrSnapshotAssistProfileNotFound = errors.New("solution profile not found in provider registry")
	// ErrSnapshotAssistProviderNotAllowed indicates the provider is not in the CP allowed_providers.
	ErrSnapshotAssistProviderNotAllowed = errors.New("provider is not allowed by crypto policy")
	// ErrSnapshotAssistChainNotSupported indicates chain_id is absent or planned on the profile.
	ErrSnapshotAssistChainNotSupported = errors.New("chain is not persistable for this solution profile")
	// ErrSnapshotAssistChainRequired indicates a missing/invalid chain_id.
	ErrSnapshotAssistChainRequired = errors.New("chain_id is required")
	// ErrSnapshotAssistCryptoPolicyRequired indicates a missing crypto_policy_id / nil CP.
	ErrSnapshotAssistCryptoPolicyRequired = errors.New("crypto policy is required")
	// ErrSnapshotAssistFindingsUnknown indicates a client finding code outside the soft-findings enum.
	ErrSnapshotAssistFindingsUnknown = errors.New("accepted_findings contains an unknown code")
)

// BuildAcceptedProviderSnapshotInput is the CFB-P5 assist input (Option B).
// CPM builds the canonical accepted_provider_snapshot from registry facts so
// clients need no provider fixture / GET /providers join.
type BuildAcceptedProviderSnapshotInput struct {
	CryptoPolicy       *CryptoPolicy
	SolutionProfileRef SolutionProfileRef
	ChainID            int64
	// AcceptedFindings are client-supplied soft findings; required profile soft
	// findings are always merged in (authoritative).
	AcceptedFindings []string
	Registry         *provider.Registry
}

// BuildAcceptedProviderSnapshotResult is the assist output: registry-authoritative
// snapshot plus the normalized solution_profile_ref used for persist.
type BuildAcceptedProviderSnapshotResult struct {
	SolutionProfileRef       SolutionProfileRef
	AcceptedProviderSnapshot AcceptedProviderSnapshot
}

// BuildAcceptedProviderSnapshot derives accepted_provider_snapshot from the loaded
// provider registry (ADR US-PERSIST-SNAP / CFB-P5 Option B).
func BuildAcceptedProviderSnapshot(in BuildAcceptedProviderSnapshotInput) (BuildAcceptedProviderSnapshotResult, error) {
	if in.CryptoPolicy == nil {
		return BuildAcceptedProviderSnapshotResult{}, ErrSnapshotAssistCryptoPolicyRequired
	}
	ref := in.SolutionProfileRef
	ref.ProviderID = strings.TrimSpace(ref.ProviderID)
	ref.SolutionProfileID = strings.TrimSpace(ref.SolutionProfileID)
	ref.ManifestVersion = strings.TrimSpace(ref.ManifestVersion)
	ref.VerificationDate = strings.TrimSpace(ref.VerificationDate)
	if err := ref.Validate(); err != nil {
		return BuildAcceptedProviderSnapshotResult{}, fmt.Errorf("%w: %v", ErrCryptoPolicyPayloadInvalid, err)
	}
	if in.ChainID < 1 {
		return BuildAcceptedProviderSnapshotResult{}, ErrSnapshotAssistChainRequired
	}
	if !providerAllowedByCryptoPolicy(in.CryptoPolicy, ref.ProviderID) {
		return BuildAcceptedProviderSnapshotResult{}, fmt.Errorf(
			"%w: provider_id %q not in allowed_providers for %s",
			ErrSnapshotAssistProviderNotAllowed, ref.ProviderID, in.CryptoPolicy.ID,
		)
	}
	if in.Registry == nil {
		return BuildAcceptedProviderSnapshotResult{}, ErrSnapshotAssistProfileNotFound
	}
	resolved, ok := in.Registry.Lookup(provider.ProfileRef{
		ProviderID:        ref.ProviderID,
		SolutionProfileID: ref.SolutionProfileID,
		ManifestVersion:   ref.ManifestVersion,
	})
	if !ok || resolved == nil {
		return BuildAcceptedProviderSnapshotResult{}, fmt.Errorf(
			"%w: %s/%s (manifest_version %q)",
			ErrSnapshotAssistProfileNotFound, ref.ProviderID, ref.SolutionProfileID, ref.ManifestVersion,
		)
	}

	chain, err := pickPersistableChainSupport(resolved.Profile.ChainSupport, in.ChainID)
	if err != nil {
		return BuildAcceptedProviderSnapshotResult{}, err
	}

	findings, err := mergeAssistAcceptedFindings(&resolved.Profile, in.AcceptedFindings)
	if err != nil {
		return BuildAcceptedProviderSnapshotResult{}, err
	}

	profile := &resolved.Profile
	snap := AcceptedProviderSnapshot{
		Maturity:          profile.Maturity,
		ClaimStatus:       profile.ClaimStatus,
		ResultingPosture:  profile.ResultingPosture,
		InputRequirements: cloneInputRequirements(profile.InputRequirements),
		Signature:         profile.Signature,
		AccountModel:      cloneAccountModel(profile.AccountModel),
		Constraints:       profile.Constraints,
		ChainSupportUsed: SnapshotChainSupport{
			ChainID:      FlexibleChainID(chain.ChainID),
			Status:       chain.Status,
			Capabilities: append([]string(nil), chain.Capabilities...),
		},
		References:       cloneReferences(profile.References),
		AcceptedFindings: findings,
		AcceptedRiskNotes: append([]string(nil), profile.RiskNotes...),
	}
	if len(snap.AcceptedRiskNotes) == 0 {
		snap.AcceptedRiskNotes = nil
	}

	outRef := SolutionProfileRef{
		ProviderID:        resolved.ProviderID,
		SolutionProfileID: profile.SolutionProfileID,
		ManifestVersion:   resolved.ProviderVersion,
	}
	if ref.VerificationDate != "" {
		outRef.VerificationDate = ref.VerificationDate
	}

	return BuildAcceptedProviderSnapshotResult{
		SolutionProfileRef:       outRef,
		AcceptedProviderSnapshot: snap,
	}, nil
}

func providerAllowedByCryptoPolicy(cp *CryptoPolicy, providerID string) bool {
	if cp == nil {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(providerID))
	for _, id := range cp.AllowedProviders {
		if strings.ToLower(strings.TrimSpace(id)) == want {
			return true
		}
	}
	return false
}

func pickPersistableChainSupport(entries []provider.ChainSupport, chainID int64) (provider.ChainSupport, error) {
	for _, cs := range entries {
		if cs.ChainID != chainID {
			continue
		}
		if cs.Status == provider.ChainStatusPlanned || strings.TrimSpace(string(cs.Status)) == "" {
			return provider.ChainSupport{}, fmt.Errorf(
				"%w: chain_id %d status %q", ErrSnapshotAssistChainNotSupported, chainID, cs.Status,
			)
		}
		return cs, nil
	}
	return provider.ChainSupport{}, fmt.Errorf("%w: chain_id %d", ErrSnapshotAssistChainNotSupported, chainID)
}

func mergeAssistAcceptedFindings(profile *provider.SolutionProfile, client []string) ([]string, error) {
	allowed := map[string]struct{}{
		provider.FindingCodeRequiresBundler:          {},
		provider.FindingCodeRequiresLocalSignerState: {},
	}
	merged := make([]string, 0, len(client)+2)
	for _, code := range client {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := allowed[code]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrSnapshotAssistFindingsUnknown, code)
		}
		merged = append(merged, code)
	}
	for _, soft := range provider.EvaluateSoftFindings(profile) {
		merged = append(merged, soft.Code)
	}
	return payloadhash.NormalizeAcceptedFindings(merged), nil
}

func cloneInputRequirements(in provider.InputRequirements) provider.InputRequirements {
	return provider.InputRequirements{
		WalletTypes:                append([]string(nil), in.WalletTypes...),
		RequiresWalletControlProof: in.RequiresWalletControlProof,
	}
}

func cloneAccountModel(in provider.AccountModel) provider.AccountModel {
	return provider.AccountModel{
		Standard:           in.Standard,
		ExecutionModel:     in.ExecutionModel,
		RequiresBundler:    in.RequiresBundler,
		RequiresEntrypoint: in.RequiresEntrypoint,
		EntrypointVersions: append([]string(nil), in.EntrypointVersions...),
	}
}

func cloneReferences(in []provider.Reference) []provider.Reference {
	if len(in) == 0 {
		return nil
	}
	out := make([]provider.Reference, len(in))
	copy(out, in)
	return out
}
