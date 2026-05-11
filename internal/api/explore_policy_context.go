package api

import (
	"fmt"
	"strings"
	"time"

	v01 "github.com/create2-labs/cafe-contracts/observation/wallet/v01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

// walletPolicyContextWire is the Discovery façade shape for POST .../decisions/explore (Option A).
// It is converted server-side into walletobserved.Payload for the policy evaluator.
type walletPolicyContextWire struct {
	ScanID           string `json:"scan_id,omitempty"`
	WalletAddress    string `json:"wallet_address,omitempty"`
	WalletType       string `json:"wallet_type,omitempty"`
	ChainIDs         []int  `json:"chain_ids,omitempty"`
	CurrentAlgorithm string `json:"current_algorithm,omitempty"`
	CurrentPQPosture string `json:"current_pq_posture,omitempty"`
	ScannedAt        string `json:"scanned_at,omitempty"`
	Status           string `json:"status,omitempty"`
}

func observationFromWalletPolicyContext(pc *walletPolicyContextWire) (walletobserved.Payload, error) {
	if pc == nil {
		return walletobserved.Payload{}, fmt.Errorf("policy_context is nil")
	}
	kind := normalizeWireAccountKind(pc.WalletType)
	if !v01.AccountKind(kind).IsValid() {
		return walletobserved.Payload{}, fmt.Errorf("policy_context.wallet_type resolves to invalid account_kind %q", kind)
	}

	algo := strings.TrimSpace(pc.CurrentAlgorithm)
	if algo == "" {
		algo = string(v01.AlgorithmSecp256k1ECRecover)
	}
	if !v01.IsValidAlgorithmID(algo) {
		return walletobserved.Payload{}, fmt.Errorf("policy_context current_algorithm invalid: %q", algo)
	}

	pq := strings.ToLower(strings.TrimSpace(pc.CurrentPQPosture))
	if pq == "" {
		pq = string(v01.PQPostureUnknown)
	}
	if !v01.CurrentPQPosture(pq).IsValid() {
		return walletobserved.Payload{}, fmt.Errorf("policy_context.current_pq_posture invalid: %q", pc.CurrentPQPosture)
	}

	chains := make([]int64, 0, len(pc.ChainIDs))
	for _, id := range pc.ChainIDs {
		chains = append(chains, int64(id))
	}

	var observedAt time.Time
	if raw := strings.TrimSpace(pc.ScannedAt); raw != "" {
		var parsed bool
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			t, err := time.Parse(layout, raw)
			if err == nil {
				observedAt = t
				parsed = true
				break
			}
		}
		if !parsed {
			return walletobserved.Payload{}, fmt.Errorf("policy_context.scanned_at: invalid time %q", raw)
		}
	}

	payload := walletobserved.Payload{
		ChainIDs:         chains,
		AccountKind:      kind,
		CurrentAlgorithm: algo,
		PublicKeyExposed: false,
		IsMultichain:     len(chains) > 1,
		ObservedAt:       observedAt,
		CurrentPQPosture: pq,
	}
	// WalletAddress / Status / ScanID are binding metadata for AUTH/logging; evaluator uses Payload only today.
	return payload, nil
}

func normalizeWireAccountKind(walletType string) string {
	switch strings.ToUpper(strings.TrimSpace(walletType)) {
	case "EOA":
		return string(v01.AccountKindEOA)
	case "AA":
		return string(v01.AccountKindERC4337SmartAccount)
	case "CONTRACT":
		return string(v01.AccountKindContractAccount)
	default:
		lt := strings.ToLower(strings.TrimSpace(walletType))
		if lt != "" && v01.AccountKind(lt).IsValid() {
			return lt
		}
		return string(v01.AccountKindUnknown)
	}
}

// observationFromDecisionExplore derives the evaluator payload from Option A wire input only.
func observationFromDecisionExplore(req *decisionExploreRequest) (walletobserved.Payload, error) {
	if req == nil {
		return walletobserved.Payload{}, fmt.Errorf("request is nil")
	}
	if req.PolicyContext == nil {
		return walletobserved.Payload{}, fmt.Errorf("policy_context is required")
	}
	return observationFromWalletPolicyContext(req.PolicyContext)
}
