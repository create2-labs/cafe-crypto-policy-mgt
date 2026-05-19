package api

import (
	"errors"
	"testing"
)

func TestObservationPayloadFromDiscoveryWalletScanDetail_tlsRejected(t *testing.T) {
	detail := []byte(`{"scan_family":"tls","scan_id":"705c9700-0000-4000-8000-000000000002","status":"completed","result":{"target_address":"0x1"}}`)
	_, err := ObservationPayloadFromDiscoveryWalletScanDetail(detail)
	if !errors.Is(err, ErrWalletScanDetailTLS) {
		t.Fatalf("expected ErrWalletScanDetailTLS, got %v", err)
	}
}

func TestObservationPayloadFromDiscoveryWalletScanDetail_noResult(t *testing.T) {
	detail := []byte(`{"scan_id":"705c9700-0000-4000-8000-000000000003","status":"requested"}`)
	_, err := ObservationPayloadFromDiscoveryWalletScanDetail(detail)
	if !errors.Is(err, ErrWalletScanDetailNoResult) {
		t.Fatalf("expected ErrWalletScanDetailNoResult, got %v", err)
	}
}
