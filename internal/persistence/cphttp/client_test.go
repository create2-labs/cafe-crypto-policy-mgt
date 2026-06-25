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
	testDraftID   = "550e8400-e29b-41d4-a716-446655440001"
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

func requireBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
		t.Fatalf("authorization = %q", got)
	}
	if r.Header.Get("X-User-Id") != testPrincipal.UserID {
		t.Fatalf("X-User-Id = %q", r.Header.Get("X-User-Id"))
	}
}

func TestClientUpsertGetDeleteDraft(t *testing.T) {
	drafts := map[string]map[string]any{}
	mux := http.NewServeMux()
	base := cphttp.V1Base

	mux.HandleFunc("PUT "+base+strings.ReplaceAll(cphttp.DraftByID, "{draft_id}", testDraftID), func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		var body struct {
			ScanID  string         `json:"scan_id"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		drafts[testDraftID] = map[string]any{
			"id": testDraftID, "user_id": testPrincipal.UserID, "tenant_id": testPrincipal.TenantID,
			"scan_id": testScanID, "payload": body.Payload, "created_at": now, "updated_at": now,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(drafts[testDraftID])
	})
	mux.HandleFunc("GET "+base+strings.ReplaceAll(cphttp.DraftByID, "{draft_id}", testDraftID), func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		row, ok := drafts[testDraftID]
		if !ok {
			http.Error(w, `{"error":"DRAFT_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(row)
	})
	mux.HandleFunc("DELETE "+base+strings.ReplaceAll(cphttp.DraftByID, "{draft_id}", testDraftID), func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		delete(drafts, testDraftID)
		w.WriteHeader(http.StatusNoContent)
	})

	client := newTestClient(t, mux)
	_, err := client.SaveDraft(testPrincipal, testDraftID, testScanID, map[string]any{"wallet_address": testWallet})
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	got, err := client.GetDraft(testPrincipal, testDraftID)
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if got.ScanID != testScanID {
		t.Fatalf("scan_id = %q", got.ScanID)
	}
	if err := client.DeleteDraft(testPrincipal, testDraftID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}
	if _, err := client.GetDraft(testPrincipal, testDraftID); err != persistence.ErrDraftNotFound {
		t.Fatalf("want ErrDraftNotFound after delete, got %v", err)
	}
}

func TestClientPersistDraftMaps409(t *testing.T) {
	persistURL := cphttp.V1Base + strings.ReplaceAll(cphttp.DraftPersist, "{draft_id}", testDraftID)
	draftURL := cphttp.V1Base + strings.ReplaceAll(cphttp.DraftByID, "{draft_id}", testDraftID)
	mux := http.NewServeMux()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	mux.HandleFunc("GET "+draftURL, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": testDraftID, "user_id": testPrincipal.UserID, "scan_id": testScanID,
			"payload": map[string]any{}, "created_at": now, "updated_at": now,
		})
	})
	mux.HandleFunc("POST "+persistURL, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "DRAFT_ALREADY_PERSISTED"})
	})

	client := newTestClient(t, mux)
	_, err := client.PersistDraftOnce(testPrincipal, testDraftID, persistence.PersistDraftInput{
		WalletAddress: testWallet,
		ChainID:       1,
		VerifiedAt:    time.Now().UTC(),
	})
	if err != persistence.ErrDraftAlreadyPersisted {
		t.Fatalf("want ErrDraftAlreadyPersisted, got %v", err)
	}
}

func TestClientMaps503ToPersistenceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	url := cphttp.V1Base + strings.ReplaceAll(cphttp.DraftByID, "{draft_id}", testDraftID)
	mux.HandleFunc("GET "+url, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "PERSISTENCE_UNAVAILABLE"})
	})
	client := newTestClient(t, mux)
	_, err := client.GetDraft(testPrincipal, testDraftID)
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
		_ = json.NewEncoder(w).Encode(map[string]any{"exists": true, "policy_count": 2, "draft_count": 1})
	})
	client := newTestClient(t, mux)
	counts, err := client.CountActiveWalletCPMContext(testPrincipal, testWallet)
	if err != nil {
		t.Fatalf("CountActiveWalletCPMContext: %v", err)
	}
	if !counts.Exists || counts.PolicyCount != 2 || counts.DraftCount != 1 {
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
