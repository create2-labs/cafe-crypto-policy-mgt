package contract

import (
	"encoding/json"
	"testing"
)

// Discovery v1 list/detail shape checks (openapi/discovery-v1.yaml PaginatedScanList, WalletScanDetail).
// HTTP 401 and owner isolation are covered in cafe-discovery/internal/contract (same Option A matrix).

func TestDiscoveryV1WalletScanListEnvelope(t *testing.T) {
	var body map[string]any
	if err := json.Unmarshal(contractFixture(t, "discovery_v1_wallet_scan_list.json"), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"items", "total", "limit", "offset"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("list response missing %q", key)
		}
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("items: want non-empty array, got %T %#v", body["items"], body["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("list item is not an object")
	}
	for _, key := range []string{"scan_id", "created_at", "status"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("ScanListItem missing %q", key)
		}
	}
}

func TestDiscoveryV1WalletScanDetailStableFields(t *testing.T) {
	detail := loadJSONFixture(t, "discovery_v1_wallet_scan_detail.json")
	for _, key := range []string{"scan_id", "status", "result"} {
		if _, ok := detail[key]; !ok {
			t.Fatalf("WalletScanDetail missing %q", key)
		}
	}
	result, ok := detail["result"].(map[string]any)
	if !ok {
		t.Fatal("result is not an object")
	}
	for _, key := range []string{
		"target_address",
		"chain_ids",
		"wallet_type",
		"current_pq_posture",
	} {
		if _, ok := result[key]; !ok {
			t.Fatalf("WalletScanResult missing %q", key)
		}
	}
	chainIDs, ok := result["chain_ids"].([]any)
	if !ok {
		t.Fatalf("chain_ids type = %T", result["chain_ids"])
	}
	if len(chainIDs) == 0 {
		t.Fatal("chain_ids must not be fabricated when absent in source; fixture expects [1]")
	}
}
