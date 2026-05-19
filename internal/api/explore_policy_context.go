package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v01 "github.com/create2-labs/cafe-contracts/observation/wallet/v01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

// walletPolicyContextWire is the Discovery façade shape for POST .../decisions/explore (Option A).
// It is converted server-side into walletobserved.Payload for the policy evaluator.
// TargetAddress mirrors Discovery v1 GET …/wallets/scans/{scan_id} → result.target_address (openapi WalletScanResult).
type walletPolicyContextWire struct {
	ScanID           string  `json:"scan_id,omitempty"`
	WalletAddress    string  `json:"wallet_address,omitempty"`
	TargetAddress    string  `json:"target_address,omitempty"`
	WalletType       string  `json:"wallet_type,omitempty"`
	ChainIDs         []int64 `json:"chain_ids,omitempty"`
	CurrentAlgorithm string  `json:"current_algorithm,omitempty"`
	CurrentPQPosture string  `json:"current_pq_posture,omitempty"`
	ScannedAt        string  `json:"scanned_at,omitempty"`
	Status           string  `json:"status,omitempty"`
	KeyExposed       bool    `json:"key_exposed,omitempty"`
}

// walletScanResultV1Wire matches Discovery v1 WalletScanResult (subset used for explore).
type walletScanResultV1Wire struct {
	TargetAddress    string  `json:"target_address,omitempty"`
	ChainIDs         []int64 `json:"chain_ids,omitempty"`
	WalletType       string  `json:"wallet_type,omitempty"`
	CurrentPQPosture string  `json:"current_pq_posture,omitempty"`
	CurrentAlgorithm string  `json:"current_algorithm,omitempty"`
	Algorithm        string  `json:"algorithm,omitempty"`
	ScannedAt        string  `json:"scanned_at,omitempty"`
	KeyExposed       bool    `json:"key_exposed,omitempty"`
}

// parsePolicyContextFlexible accepts either the legacy flat policy_context or a Discovery v1
// wallet scan detail envelope (scan_id, status, result) per openapi/discovery-v1.yaml WalletScanDetail.
func parsePolicyContextFlexible(raw []byte) (*walletPolicyContextWire, error) {
	if len(bytesTrimSpaceJSON(raw)) == 0 {
		return nil, fmt.Errorf("policy_context is required")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("policy_context: %w", err)
	}
	if sub, ok := top["result"]; ok && len(sub) > 0 && string(sub) != "null" {
		var res walletScanResultV1Wire
		if err := json.Unmarshal(sub, &res); err != nil {
			return nil, fmt.Errorf("policy_context.result: %w", err)
		}
		out := &walletPolicyContextWire{
			WalletAddress:    strings.TrimSpace(res.TargetAddress),
			TargetAddress:    strings.TrimSpace(res.TargetAddress),
			WalletType:       res.WalletType,
			ChainIDs:         append([]int64(nil), res.ChainIDs...),
			CurrentAlgorithm: pickAlgorithm(res.CurrentAlgorithm, res.Algorithm),
			CurrentPQPosture: res.CurrentPQPosture,
			ScannedAt:        res.ScannedAt,
			KeyExposed:       res.KeyExposed,
		}
		if sid, ok := top["scan_id"]; ok {
			var s string
			_ = json.Unmarshal(sid, &s)
			out.ScanID = strings.TrimSpace(s)
		}
		if st, ok := top["status"]; ok {
			var s string
			_ = json.Unmarshal(st, &s)
			out.Status = strings.TrimSpace(s)
		}
		return out, nil
	}
	var flat walletPolicyContextWire
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, fmt.Errorf("policy_context: %w", err)
	}
	return &flat, nil
}

func bytesTrimSpaceJSON(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func observationFromWalletPolicyContext(pc *walletPolicyContextWire) (walletobserved.Payload, error) {
	if pc == nil {
		return walletobserved.Payload{}, fmt.Errorf("policy_context is nil")
	}
	kind := normalizeWireAccountKind(pc.WalletType)
	if !v01.AccountKind(kind).IsValid() {
		return walletobserved.Payload{}, fmt.Errorf("policy_context.wallet_type resolves to invalid account_kind %q", kind)
	}

	algo := normalizeWireAlgorithmID(pc.CurrentAlgorithm)
	if algo == "" {
		algo = string(v01.AlgorithmSecp256k1ECRecover)
	}
	if !v01.IsValidAlgorithmID(algo) {
		return walletobserved.Payload{}, fmt.Errorf("policy_context current_algorithm invalid: %q", algo)
	}

	pq, err := mapWirePQPostureToV01Exported(pc.CurrentPQPosture)
	if err != nil {
		return walletobserved.Payload{}, err
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
		PublicKeyExposed: pc.KeyExposed,
		IsMultichain:     len(chains) > 1,
		ObservedAt:       observedAt,
		CurrentPQPosture: pq,
	}
	// WalletAddress / Status / ScanID are binding metadata for AUTH/logging; evaluator uses Payload only today.
	return payload, nil
}

func mapWirePQPostureToV01Exported(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return string(v01.PQPostureUnknown), nil
	}
	ls := strings.ToLower(s)
	if v01.CurrentPQPosture(ls).IsValid() {
		return ls, nil
	}
	// Discovery v1 wallet scan detail labels (WalletScanResult in discovery-v1.yaml).
	switch ls {
	case "pq_ready":
		return string(v01.PQPostureFullPQ), nil
	case "not_pq_ready":
		return string(v01.PQPostureClassicalOnly), nil
	case "hybrid", "unknown":
		return ls, nil
	default:
		return "", fmt.Errorf("policy_context.current_pq_posture invalid: %q", raw)
	}
}

func normalizeWireAccountKind(walletType string) string {
	s := strings.TrimSpace(walletType)
	switch strings.ToUpper(s) {
	case "EOA":
		return string(v01.AccountKindEOA)
	case "AA":
		return string(v01.AccountKindERC4337SmartAccount)
	case "CONTRACT":
		return string(v01.AccountKindContractAccount)
	case "SMART_ACCOUNT":
		return string(v01.AccountKindERC4337SmartAccount)
	}
	lt := strings.ToLower(s)
	switch lt {
	case "eoa":
		return string(v01.AccountKindEOA)
	case "aa", "smart_account":
		return string(v01.AccountKindERC4337SmartAccount)
	case "contract":
		return string(v01.AccountKindContractAccount)
	case "unknown":
		return string(v01.AccountKindUnknown)
	}
	if lt != "" && v01.AccountKind(lt).IsValid() {
		return lt
	}
	return string(v01.AccountKindUnknown)
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

func pickAlgorithm(current, algorithm string) string {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current)
	}
	return strings.TrimSpace(algorithm)
}

func normalizeWireAlgorithmID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	ls := strings.ToLower(s)
	if v01.IsValidAlgorithmID(ls) {
		return ls
	}
	switch strings.NewReplacer("-", "_", " ", "_").Replace(ls) {
	case "ecdsa_secp256k1", "secp256k1":
		return string(v01.AlgorithmSecp256k1ECRecover)
	default:
		return s
	}
}
