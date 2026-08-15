package provider

import (
	"fmt"
	"strings"
)

// Stable hard-finding codes (ADR §7 / CPM-P4). Soft findings: soft_compat.go.
const (
	FindingCodeWalletType = "incompatible.provider.wallet_type"
	FindingCodeNewWallet  = "incompatible.provider.new_wallet"
	FindingCodeContinuity = "incompatible.provider.continuity"
	FindingCodeChain      = "incompatible.provider.chain"
	FindingCodeRotation   = "incompatible.provider.rotation"
	FindingCodePosture    = "incompatible.provider.posture"
	FindingCodeUnresolved = "incompatible.provider.unresolved"
)

// HardFinding is an explainable provider hard incompatibility signal.
type HardFinding struct {
	Code    string
	Message string
	Field   string
}

// HardSelectionRequest is the subset of PolicySelectionRequest needed for ADR §7 hard checks.
type HardSelectionRequest struct {
	RequiredPosture           string // CAFE classical_only|hybrid|full_pq (from template/selection)
	TargetChainIDs            []int64
	AllowNewWallet            bool
	AddressContinuityRequired bool
	KeyRotationModel          string // "none" | "per_userop"
}

// HardObservation is the subset of wallet observation needed for ADR §7 hard checks.
type HardObservation struct {
	AccountKind string // wire account_kind, e.g. "eoa"
	ChainIDs    []int64
}

// EvaluateHardCompatibility applies ADR §7 hard constraints against a solution profile.
// An empty findings slice means hard pass (soft findings still attach on ranked candidates).
func EvaluateHardCompatibility(obs HardObservation, req HardSelectionRequest, profile *SolutionProfile) []HardFinding {
	if profile == nil {
		return []HardFinding{{
			Code:    FindingCodeUnresolved,
			Message: "solution profile is nil",
		}}
	}

	var findings []HardFinding

	// v0.1: strict equality required_posture ↔ resulting_posture (not via scheme/family).
	wantPosture := strings.TrimSpace(req.RequiredPosture)
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

	if profile.Constraints.RequiresNewAccount && !req.AllowNewWallet {
		findings = append(findings, HardFinding{
			Code:    FindingCodeNewWallet,
			Message: "solution requires a new account but selection_request.allow_new_wallet is false",
			Field:   "selection_request.allow_new_wallet",
		})
	}

	if req.AddressContinuityRequired && !profile.Constraints.AddressContinuitySupported {
		findings = append(findings, HardFinding{
			Code:    FindingCodeContinuity,
			Message: "selection_request.address_continuity_required is true but solution does not support EOA address continuity",
			Field:   "selection_request.address_continuity_required",
		})
	}

	wantRotation := KeyRotationModel(strings.TrimSpace(req.KeyRotationModel))
	if wantRotation != profile.Signature.KeyRotationModel {
		findings = append(findings, HardFinding{
			Code: FindingCodeRotation,
			Message: fmt.Sprintf(
				"selection_request.key_rotation_model %q incompatible with solution signature.key_rotation_model %q",
				wantRotation, profile.Signature.KeyRotationModel,
			),
			Field: "selection_request.key_rotation_model",
		})
	}

	targets := req.TargetChainIDs
	if len(targets) == 0 {
		targets = obs.ChainIDs
	}
	for _, chainID := range targets {
		if chainID <= 0 {
			continue
		}
		support, ok := findChainSupport(profile.ChainSupport, chainID)
		if !ok {
			findings = append(findings, HardFinding{
				Code:    FindingCodeChain,
				Message: fmt.Sprintf("target chain_id %d not listed in solution chain_support", chainID),
				Field:   "selection_request.target_chain_ids",
			})
			continue
		}
		if support.Status == ChainStatusPlanned {
			findings = append(findings, HardFinding{
				Code:    FindingCodeChain,
				Message: fmt.Sprintf("target chain_id %d has chain_support status %q (not deployable in v0.1)", chainID, support.Status),
				Field:   "selection_request.target_chain_ids",
			})
			continue
		}
		// Non-planned chains must advertise at least one capability (ADR ranked gate).
		if len(support.Capabilities) == 0 {
			findings = append(findings, HardFinding{
				Code:    FindingCodeChain,
				Message: fmt.Sprintf("target chain_id %d has no capabilities in chain_support", chainID),
				Field:   "selection_request.target_chain_ids",
			})
		}
	}

	return findings
}

func walletTypeAccepted(accountKind string, accepted []string) bool {
	normalized := normalizeWalletTypeToken(accountKind)
	if normalized == "" {
		return false
	}
	for _, a := range accepted {
		if normalizeWalletTypeToken(a) == normalized {
			return true
		}
	}
	return false
}

// normalizeWalletTypeToken maps wire account_kind / manifest wallet_types to a
// comparable token. Manifest uses "EOA"; observation uses "eoa".
func normalizeWalletTypeToken(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "eoa":
		return "eoa"
	default:
		return s
	}
}

func findChainSupport(list []ChainSupport, chainID int64) (ChainSupport, bool) {
	for _, c := range list {
		if c.ChainID == chainID {
			return c, true
		}
	}
	return ChainSupport{}, false
}
