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

// engagementW2Error is returned when W2 (latest completed owner-scoped scan) fails.
type engagementW2Error struct {
	code    string
	message string
	status  int
}

// ensureEngagementW2 requires scanID to be the latest completed Discovery wallet
// scan for walletAddress (owner-scoped via the caller's JWT). Fail-closed when
// Discovery is unset, times out, or returns 5xx (RD-P5).
func ensureEngagementW2(ctx context.Context, r *http.Request, cfg authConfig, requestID, walletAddress, scanID string) *engagementW2Error {
	if strings.TrimSpace(cfg.DiscoveryHTTPBaseURL) == "" {
		return &engagementW2Error{
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
		return &engagementW2Error{
			code:    walletAuthCodeDiscoveryUnavailable,
			message: "discovery is unavailable",
			status:  http.StatusServiceUnavailable,
		}
	}
	if strings.TrimSpace(latest) == "" {
		return &engagementW2Error{
			code:    walletAuthCodeScanNotLatest,
			message: "scan_id is not the latest completed wallet scan for this address",
			status:  http.StatusUnprocessableEntity,
		}
	}
	normLatest, normErr := NormalizeDiscoveryScanID(latest)
	if normErr != nil {
		return &engagementW2Error{
			code:    walletAuthCodeDiscoveryUnavailable,
			message: "discovery is unavailable",
			status:  http.StatusServiceUnavailable,
		}
	}
	if !strings.EqualFold(normLatest, strings.TrimSpace(scanID)) {
		return &engagementW2Error{
			code:    walletAuthCodeScanNotLatest,
			message: "scan_id is not the latest completed wallet scan for this address",
			status:  http.StatusUnprocessableEntity,
		}
	}
	return nil
}

