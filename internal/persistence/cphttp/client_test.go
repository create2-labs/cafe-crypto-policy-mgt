package cphttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence/cphttp"
)

const testToken = "test-persistence-token"

var (
	testPrincipal = authz.Principal{UserID: "11111111-1111-4111-8111-111111111111", Subject: "11111111-1111-4111-8111-111111111111", TenantID: "tenant-a"}
	testPolicyID  = "550e8400-e29b-41d4-a716-446655440001"
	testScanID    = "550e8400-e29b-41d4-a716-446655440000"
	testWallet    = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
)

func newTestClient(t *testing.T, handler http.Handler) persistence.PolicyStore {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return cphttp.NewClient(cphttp.Config{
		BaseURL:    srv.URL,
		Token:      testToken,
		HTTPClient: srv.Client(),
	})
}

func TestClientMaps503ToPersistenceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	url := cphttp.V1Base + strings.ReplaceAll(cphttp.PolicyByID, "{policy_id}", testPolicyID)
	mux.HandleFunc("GET "+url, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "PERSISTENCE_UNAVAILABLE"})
	})
	client := newTestClient(t, mux)
	_, err := client.GetPolicy(testPrincipal, testPolicyID)
	if err != persistence.ErrPersistenceUnavailable {
		t.Fatalf("want ErrPersistenceUnavailable, got %v", err)
	}
}

func TestClientListPoliciesByScan(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+cphttp.V1Base+cphttp.Policies, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("scan_id") != testScanID {
			t.Fatalf("scan_id query = %q", r.URL.Query().Get("scan_id"))
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{
				"id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "user_id": testPrincipal.UserID,
				"scan_id": testScanID, "payload": map[string]any{}, "created_at": now, "updated_at": now,
			}},
			"total": 1,
		})
	})
	client := newTestClient(t, mux)
	list, err := client.ListPersistedPoliciesForScan(testPrincipal, testScanID)
	if err != nil {
		t.Fatalf("ListPersistedPoliciesForScan: %v", err)
	}
	if len(list) != 1 || list[0].ScanID != testScanID {
		t.Fatalf("unexpected list: %#v", list)
	}
}

func TestClientCountWalletReference(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+cphttp.V1Base+cphttp.ReferenceWallet, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("wallet_address") != testWallet {
			t.Fatalf("wallet_address = %q", r.URL.Query().Get("wallet_address"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"exists": true, "policy_count": 2})
	})
	client := newTestClient(t, mux)
	counts, err := client.CountActiveWalletCPMContext(testPrincipal, testWallet)
	if err != nil {
		t.Fatalf("CountActiveWalletCPMContext: %v", err)
	}
	if !counts.Exists || counts.PolicyCount != 2 {
		t.Fatalf("counts = %#v", counts)
	}
}

func TestClientSavePolicyUnsupported(t *testing.T) {
	client := newTestClient(t, http.NewServeMux())
	_, err := client.SavePolicy(testPrincipal, "id", testScanID, nil)
	if err != persistence.ErrUnsupportedStoreOperation {
		t.Fatalf("want ErrUnsupportedStoreOperation, got %v", err)
	}
}
