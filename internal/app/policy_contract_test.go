//go:build dev

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
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func TestGETPoliciesByScanIDReturnsOpenAPIPolicyRecordShape(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "u-shape")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("create policy: want 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}
	var created policyPersistResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?scan_id="+policyPersistTestScanID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET policies by scan_id: want 200, got %d body=%s", res.Code, res.Body.String())
	}

	var list struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(list.Items))
	}
	row := list.Items[0]
	for _, key := range []string{"id", "payload", "created_at", "updated_at", "payload_sha256"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("expected %q in policy row, got %#v", key, row)
		}
	}
	if row["id"] != created.PolicyID {
		t.Fatalf("id: got %#v want %s", row["id"], created.PolicyID)
	}
	if row["payload_sha256"] != digest {
		t.Fatalf("payload_sha256: got %#v", row["payload_sha256"])
	}
}

func TestGETPolicyByIDReturnsPolicyRecordNotItemWrapper(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "u-get")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	h.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusOK {
		t.Fatalf("create policy: want 200, got %d body=%s", createRes.Code, createRes.Body.String())
	}
	var created policyPersistResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Policies+"?id="+created.PolicyID, nil)
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
	if row["id"] != created.PolicyID {
		t.Fatalf("id: got %#v", row["id"])
	}
}

func TestPOSTPolicyReturnsPersistAckNotItemWrapper(t *testing.T) {
	h := newPolicyPersistTestHandler(t)
	token := mustToken(t, "u-post")
	payloadJSON, digest := policyPersistHashedPayload(t)
	now := time.Now().UTC().Truncate(time.Second)
	body := buildSignedPolicyPersistBody(t, payloadJSON, digest, now, now.Add(walletauth.MaxValidityWindow))

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, cpmroutes.Policies, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("POST policy: want 200, got %d body=%s", res.Code, res.Body.String())
	}

	var row map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, wrapped := row["item"]; wrapped {
		t.Fatalf("expected bare persist ack, got item wrapper: %#v", row)
	}
	if row["policy_id"] == nil || row["payload_sha256"] != digest {
		t.Fatalf("unexpected ack: %#v", row)
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
