package provider

import (
	"fmt"
	"strings"
)

// Capability names required for couche A scan-compatibility (ADR §5.3 / §7).
const (
	CapabilityDeploy       = "deploy"
	CapabilitySignUserOp   = "sign_userop"
	CapabilityRotateSigner = "rotate_signer"
)

// FindingCodeCapability marks missing minimum chain capabilities (couche A).
const FindingCodeCapability = "incompatible.provider.capability"

// FindingCodeErroneousSuggested marks a contradictory suggested_user_constraints block.
const FindingCodeErroneousSuggested = "erroneous.suggested_user_constraints"

// EvaluateScanCompatibility applies ADR couche A hard checks only:
// required_posture, wallet_types, and at least one scan chain with status != planned
// plus minimum capabilities (deploy, sign_userop, + rotate_signer if per_userop).
// Couche B fields (allow_new_wallet, address_continuity, user key_rotation_model)
// must not be consulted here.
func EvaluateScanCompatibility(obs HardObservation, requiredPosture string, profile *SolutionProfile) []HardFinding {
	if profile == nil {
		return []HardFinding{{
			Code:    FindingCodeUnresolved,
			Message: "solution profile is nil",
		}}
	}

	var findings []HardFinding

	wantPosture := strings.TrimSpace(requiredPosture)
	gotPosture := strings.TrimSpace(profile.ResultingPosture)
	if wantPosture != "" && gotPosture != wantPosture {
		findings = append(findings, HardFinding{
			Code: FindingCodePosture,
			Message: fmt.Sprintf(
				"required_posture %q not satisfied by solution_profile.resulting_posture %q",
				wantPosture, gotPosture,
			),
			Field: "required_posture",
		})
	}

	if !walletTypeAccepted(obs.AccountKind, profile.InputRequirements.WalletTypes) {
		findings = append(findings, HardFinding{
			Code:    FindingCodeWalletType,
			Message: fmt.Sprintf("wallet account_kind %q not accepted by solution profile wallet_types", obs.AccountKind),
			Field:   "policy_context.wallet_type",
		})
	}

	requiredCaps := minimumScanCapabilities(profile.Signature.KeyRotationModel)
	anyNonPlanned, anyCapable := scanChainDeployability(obs.ChainIDs, profile.ChainSupport, requiredCaps)
	if !anyCapable {
		if anyNonPlanned {
			findings = append(findings, HardFinding{
				Code: FindingCodeCapability,
				Message: fmt.Sprintf(
					"no scan chain advertises required capabilities %v",
					requiredCaps,
				),
				Field: "chain_support.capabilities",
			})
		} else {
			findings = append(findings, HardFinding{
				Code: FindingCodeChain,
				Message: fmt.Sprintf(
					"no scan chain is deployable (listed with status != planned) for capabilities %v",
					requiredCaps,
				),
				Field: "policy_context.chain_ids",
			})
		}
	}

	return findings
}

func minimumScanCapabilities(rotation KeyRotationModel) []string {
	caps := []string{CapabilityDeploy, CapabilitySignUserOp}
	if rotation == KeyRotationPerUserOp {
		caps = append(caps, CapabilityRotateSigner)
	}
	return caps
}

// scanChainDeployability reports whether any observed chain is non-planned and
// whether any such chain also carries the required capabilities.
func scanChainDeployability(scanChains []int64, support []ChainSupport, requiredCaps []string) (anyNonPlanned, anyCapable bool) {
	for _, chainID := range scanChains {
		if chainID <= 0 {
			continue
		}
		cs, ok := findChainSupport(support, chainID)
		if !ok || cs.Status == ChainStatusPlanned {
			continue
		}
		anyNonPlanned = true
		if chainHasCapabilities(cs.Capabilities, requiredCaps) {
			anyCapable = true
			return anyNonPlanned, anyCapable
		}
	}
	return anyNonPlanned, anyCapable
}

func chainHasCapabilities(have []string, required []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, c := range have {
		set[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
	}
	for _, need := range required {
		if _, ok := set[strings.ToLower(strings.TrimSpace(need))]; !ok {
			return false
		}
	}
	return true
}
