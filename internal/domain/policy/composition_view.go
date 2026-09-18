package policy

import (
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

// CompositionView is the explore-derived composition fact set for Expected result
// (ADR CFB-P3 / US-COMPOSE-VIEW). It is a product projection of account / signature /
// constraints — not the admin ProviderManifest (no chain_support, references, risk_notes).
type CompositionView struct {
	AccountModel CompositionAccountModel `json:"account_model"`
	Signature    CompositionSignature    `json:"signature"`
	Constraints  CompositionConstraints  `json:"constraints"`
}

// CompositionAccountModel is the account slice of the explore composition view.
type CompositionAccountModel struct {
	Standard           string   `json:"standard"`
	ExecutionModel     string   `json:"execution_model"`
	RequiresBundler    bool     `json:"requires_bundler"`
	RequiresEntrypoint bool     `json:"requires_entrypoint"`
	EntrypointVersions []string `json:"entrypoint_versions,omitempty"`
}

// CompositionSignature is the signature slice of the explore composition view.
type CompositionSignature struct {
	Scheme           string `json:"scheme"`
	Family           string `json:"family"`
	KeyRotationModel string `json:"key_rotation_model"`
}

// CompositionConstraints is the constraint slice of the explore composition view.
type CompositionConstraints struct {
	RequiresNewAccount         bool `json:"requires_new_account"`
	AddressContinuitySupported bool `json:"address_continuity_supported"`
	RequiresLocalSignerState   bool `json:"requires_local_signer_state"`
}

// DeriveCompositionView builds the product composition view from a solution profile.
// A nil profile yields a zero view (callers should only attach on resolved candidates).
func DeriveCompositionView(profile *provider.SolutionProfile) CompositionView {
	if profile == nil {
		return CompositionView{}
	}
	return CompositionView{
		AccountModel: CompositionAccountModel{
			Standard:           profile.AccountModel.Standard,
			ExecutionModel:     profile.AccountModel.ExecutionModel,
			RequiresBundler:    profile.AccountModel.RequiresBundler,
			RequiresEntrypoint: profile.AccountModel.RequiresEntrypoint,
			EntrypointVersions: append([]string(nil), profile.AccountModel.EntrypointVersions...),
		},
		Signature: CompositionSignature{
			Scheme:           profile.Signature.Scheme,
			Family:           profile.Signature.Family,
			KeyRotationModel: string(profile.Signature.KeyRotationModel),
		},
		Constraints: CompositionConstraints{
			RequiresNewAccount:         profile.Constraints.RequiresNewAccount,
			AddressContinuitySupported: profile.Constraints.AddressContinuitySupported,
			RequiresLocalSignerState:   profile.Constraints.RequiresLocalSignerState,
		},
	}
}
