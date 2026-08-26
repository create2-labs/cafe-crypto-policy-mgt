package app

import (
	"context"
	"net/http"
	"strings"
)

const (
	walletAuthCodeScanNotLatest        = "SCAN_NOT_LATEST"
	walletAuthCodeDiscoveryUnavailable = "DISCOVERY_UNAVAILABLE"
)

// w2GateError is returned when W2 (latest completed owner-scoped scan) fails.
// Shared by explore, wallet-challenges, and signed POST /policies (RD-P6).
type w2GateError struct {
	code    string
	message string
	status  int
}

// ensureOwnerScopedW2 requires scanID to be the latest completed Discovery wallet
// scan for walletAddress (owner-scoped via the caller's JWT). Fail-closed when
// Discovery is unset, times out, or returns 5xx (ADR §3.2 norme 8 / RD-P5–P6).
func ensureOwnerScopedW2(ctx context.Context, r *http.Request, cfg authConfig, requestID, walletAddress, scanID string) *w2GateError {
	if strings.TrimSpace(cfg.DiscoveryHTTPBaseURL) == "" {
		return &w2GateError{
			code:    walletAuthCodeDiscoveryUnavailable,
			message: "discovery is unavailable",
			status:  http.StatusServiceUnavailable,
		}
	}
	authz := ""
	if r != nil {
		authz = r.Header.Get("Authorization")
	}
	latest, err := fetchWalletLatestCompletedScanID(ctx, cfg, authz, requestID, walletAddress)
	if err != nil {
		return &w2GateError{
			code:    walletAuthCodeDiscoveryUnavailable,
			message: "discovery is unavailable",
			status:  http.StatusServiceUnavailable,
		}
	}
	if strings.TrimSpace(latest) == "" {
		return &w2GateError{
			code:    walletAuthCodeScanNotLatest,
			message: "scan_id is not the latest completed wallet scan for this address",
			status:  http.StatusUnprocessableEntity,
		}
	}
	normLatest, normErr := NormalizeDiscoveryScanID(latest)
	if normErr != nil {
		return &w2GateError{
			code:    walletAuthCodeDiscoveryUnavailable,
			message: "discovery is unavailable",
			status:  http.StatusServiceUnavailable,
		}
	}
	if !strings.EqualFold(normLatest, strings.TrimSpace(scanID)) {
		return &w2GateError{
			code:    walletAuthCodeScanNotLatest,
			message: "scan_id is not the latest completed wallet scan for this address",
			status:  http.StatusUnprocessableEntity,
		}
	}
	return nil
}
