package provider

import (
	"fmt"
	"strings"
)

// UserConstraints are the user-validated couche B choices snapshotted at persist (ADR §7 / §9).
// Distinct from SuggestedUserConstraints (indicative explore defaults).
type UserConstraints struct {
	AllowNewWallet            bool             `json:"allow_new_wallet"`
	AddressContinuityRequired bool             `json:"address_continuity_required"`
	KeyRotationModel          KeyRotationModel `json:"key_rotation_model"`
}

// EvaluateUserConstraints applies ADR couche B hard checks only:
// allow_new_wallet, address_continuity_required, key_rotation_model vs profile offer.
func EvaluateUserConstraints(uc UserConstraints, profile *SolutionProfile) []HardFinding {
	if profile == nil {
		return []HardFinding{{
			Code:    FindingCodeUnresolved,
			Message: "solution profile is nil",
		}}
	}

	var findings []HardFinding

	if profile.Constraints.RequiresNewAccount && !uc.AllowNewWallet {
		findings = append(findings, HardFinding{
			Code:    FindingCodeNewWallet,
			Message: "solution requires a new account but user_constraints.allow_new_wallet is false",
			Field:   "user_constraints.allow_new_wallet",
		})
	}

	if uc.AddressContinuityRequired && !profile.Constraints.AddressContinuitySupported {
		findings = append(findings, HardFinding{
			Code:    FindingCodeContinuity,
			Message: "user_constraints.address_continuity_required is true but solution does not support EOA address continuity",
			Field:   "user_constraints.address_continuity_required",
		})
	}

	wantRotation := KeyRotationModel(strings.TrimSpace(string(uc.KeyRotationModel)))
	if wantRotation != profile.Signature.KeyRotationModel {
		findings = append(findings, HardFinding{
			Code: FindingCodeRotation,
			Message: fmt.Sprintf(
				"user_constraints.key_rotation_model %q incompatible with solution signature.key_rotation_model %q",
				wantRotation, profile.Signature.KeyRotationModel,
			),
			Field: "user_constraints.key_rotation_model",
		})
	}

	return findings
}
