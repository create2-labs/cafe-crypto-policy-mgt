package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
)

var (
	// ErrWalletScanDetailTLS indicates a non-wallet scan family (TLS) — CPM assessment is wallet-only (PR13g).
	ErrWalletScanDetailTLS = errors.New("wallet scan detail is not a wallet scan")
	// ErrWalletScanDetailNoResult indicates missing `result` on the Discovery v1 wallet scan detail payload.
	ErrWalletScanDetailNoResult = errors.New("wallet scan detail has no result yet")
)

// ObservationPayloadFromDiscoveryWalletScanDetail maps Discovery GET /discovery/v1/wallets/scans/{scan_id}
// JSON into the wallet-observed v0.1 payload used inside policy.assessment.requested (PR13g).
func ObservationPayloadFromDiscoveryWalletScanDetail(detail []byte) (walletobserved.Payload, error) {
	var probe struct {
		ScanFamily string          `json:"scan_family"`
		Result     json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(detail, &probe); err != nil {
		return walletobserved.Payload{}, fmt.Errorf("wallet scan detail: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(probe.ScanFamily), "tls") {
		return walletobserved.Payload{}, ErrWalletScanDetailTLS
	}
	if len(bytesTrimSpaceJSON(probe.Result)) == 0 || string(probe.Result) == "null" {
		return walletobserved.Payload{}, ErrWalletScanDetailNoResult
	}
	pc, err := parsePolicyContextFlexible(detail)
	if err != nil {
		return walletobserved.Payload{}, err
	}
	return observationFromWalletPolicyContext(pc)
}
