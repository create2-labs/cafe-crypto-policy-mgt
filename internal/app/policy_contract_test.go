package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

func TestGETPoliciesByScanIDReturnsOpenAPIPolicyRecordShape(t *testing.T) {
	h := newAuthedTestHandlerWithScanAuthz(t)
	token := mustToken(t, "u-shape")
	scanID := "550e8400-e29b-41d4-a716-446655440000"

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(
		`{"id":"pol-shape-1","scan_id":"`+scanID+`","binding":"discovery","payload":{"mode":"strict"}}`,
	))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("create policy: want 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?scan_id="+scanID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET policies by scan_id: want 200, got %d body=%s", res.Code, res.Body.String())
	}

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(body.Items))
	}
	row := body.Items[0]
	for _, key := range []string{"id", "payload", "created_at", "updated_at"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("expected %q in policy row, got %#v", key, row)
		}
	}
	if row["id"] != "pol-shape-1" {
		t.Fatalf("id: got %#v", row["id"])
	}
	if row["scan_id"] != scanID {
		t.Fatalf("scan_id: got %#v", row["scan_id"])
	}
	if _, hasPascal := row["ID"]; hasPascal {
		t.Fatalf("unexpected PascalCase ID field: %#v", row)
	}
}

func TestGETPolicyByIDReturnsPolicyRecordNotItemWrapper(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "u-get")

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(
		`{"id":"pol-get-1","binding":"fixture","payload":{"k":1}}`,
	))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("create policy: want 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id=pol-get-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET policy by id: want 200, got %d body=%s", res.Code, res.Body.String())
	}

	var row map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, wrapped := row["item"]; wrapped {
		t.Fatalf("expected bare PolicyRecord, got item wrapper: %#v", row)
	}
	if row["id"] != "pol-get-1" {
		t.Fatalf("id: got %#v", row["id"])
	}
}

func TestPOSTPolicyReturnsPolicyRecordNotItemWrapper(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "u-post")

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(
		`{"id":"pol-post-1","binding":"fixture","payload":{"k":1}}`,
	))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, create)
	if res.Code != http.StatusOK {
		t.Fatalf("POST policy: want 200, got %d body=%s", res.Code, res.Body.String())
	}

	var row map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, wrapped := row["item"]; wrapped {
		t.Fatalf("expected bare PolicyRecord, got item wrapper: %#v", row)
	}
	if row["id"] != "pol-post-1" {
		t.Fatalf("id: got %#v", row["id"])
	}
}

func TestPolicyRecordResponseFromStoreOmitsEmptyScanID(t *testing.T) {
	now := time.Date(2026, 6, 6, 20, 57, 56, 638613000, time.UTC)
	resp := policyRecordResponseFromStore(persistence.PolicyRecord{
		ID:        "p1",
		Payload:   map[string]any{"x": 1},
		CreatedAt: now,
		UpdatedAt: now,
	})
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "scan_id") {
		t.Fatalf("expected omitted scan_id, got %s", string(raw))
	}
}
