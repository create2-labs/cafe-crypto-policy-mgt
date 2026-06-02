package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

const contractTestScanID = "550e8400-e29b-41d4-a716-446655440000"

func TestDraftPOSTValidationMatrix(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-contract-user")

	cases := []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "missing id and payload", body: `{}`, wantCode: draftCodeIDRequired},
		{name: "missing payload", body: `{"id":"d1"}`, wantCode: draftCodePayloadRequired},
		{name: "empty id", body: `{"id":"","payload":{}}`, wantCode: draftCodeIDRequired},
		{name: "payload not object", body: `{"id":"d1","payload":"x"}`, wantCode: draftCodePayloadRequired},
		{name: "payload null", body: `{"id":"d1","payload":null}`, wantCode: draftCodePayloadRequired},
		{name: "invalid scan_id", body: `{"id":"d1","scan_id":"not-a-uuid","payload":{}}`, wantCode: draftCodeScanIDInvalid},
		{name: "binding forbidden", body: `{"id":"d1","binding":"discovery","payload":{}}`, wantCode: draftCodeBindingForbidden},
		{name: "owner_user_id forbidden", body: `{"id":"d1","owner_user_id":"evil","payload":{}}`, wantCode: draftCodeOwnerFieldsForbidden},
		{name: "tenant_id forbidden", body: `{"id":"d1","tenant_id":"evil","payload":{}}`, wantCode: draftCodeOwnerFieldsForbidden},
		{name: "legacy draft wrapper without id", body: `{"draft":{"selected_candidate_id":"x"}}`, wantCode: draftCodeIDRequired},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			assertAuthErrorPayload(t, rec, tc.wantCode)
		})
	}
}

func TestDraftGETRequiresQueryID(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-get-user")

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, draftCodeIDRequired)
}

func TestDraftGETNotFoundStructured(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-get-missing")

	req := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id=missing-draft", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, draftCodeNotFound)
}

func TestDraftGETReturnsDraftRecordShape(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-get-shape")
	draftID := "draft-contract-get"

	post := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+draftID+`","payload":{"k":"v"}}`,
	))
	post.Header.Set("Authorization", "Bearer "+token)
	post.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, post)
	if postRec.Code != http.StatusOK {
		t.Fatalf("upsert: expected 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id="+draftID, nil)
	get.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var record map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &record); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if record["id"] != draftID {
		t.Fatalf("expected id %q, got %v", draftID, record["id"])
	}
	payload, ok := record["payload"].(map[string]any)
	if !ok || payload["k"] != "v" {
		t.Fatalf("expected payload.k=v, got %#v", record["payload"])
	}
	for _, key := range []string{"created_at", "updated_at"} {
		if s, _ := record[key].(string); strings.TrimSpace(s) == "" {
			t.Fatalf("expected non-empty %s, got %#v", key, record[key])
		}
	}
	if _, hasItem := record["item"]; hasItem {
		t.Fatalf("response must not leak Go item wrapper: %#v", record["item"])
	}
}

func TestDraftDELETERequiresQueryID(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-del-query")

	req := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertAuthErrorPayload(t, rec, draftCodeIDRequired)
}

func TestDraftDELETEContractSemantics(t *testing.T) {
	h := newAuthedTestHandler(t)
	ownerTok := mustToken(t, "del-owner")
	otherTok := mustToken(t, "del-other")
	draftID := "draft-del-contract"

	create := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+draftID+`","payload":{"n":1}}`,
	))
	create.Header.Set("Authorization", "Bearer "+ownerTok)
	create.Header.Set("Content-Type", "application/json")
	cr := httptest.NewRecorder()
	h.ServeHTTP(cr, create)
	if cr.Code != http.StatusOK {
		t.Fatalf("create: %d %s", cr.Code, cr.Body.String())
	}

	del := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id="+draftID, nil)
	del.Header.Set("Authorization", "Bearer "+ownerTok)
	dr := httptest.NewRecorder()
	h.ServeHTTP(dr, del)
	if dr.Code != http.StatusNoContent {
		t.Fatalf("first delete: expected 204, got %d body=%s", dr.Code, dr.Body.String())
	}
	if dr.Body.Len() != 0 {
		t.Fatalf("204 response must have empty body, got %q", dr.Body.String())
	}

	delAgain := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id="+draftID, nil)
	delAgain.Header.Set("Authorization", "Bearer "+ownerTok)
	da := httptest.NewRecorder()
	h.ServeHTTP(da, delAgain)
	if da.Code != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d", da.Code)
	}
	assertAuthErrorPayload(t, da, draftCodeNotFound)

	delCross := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id="+draftID, nil)
	delCross.Header.Set("Authorization", "Bearer "+otherTok)
	dc := httptest.NewRecorder()
	h.ServeHTTP(dc, delCross)
	if dc.Code != http.StatusNotFound {
		t.Fatalf("cross-owner delete after removed: expected 404, got %d", dc.Code)
	}
	assertAuthErrorPayload(t, dc, draftCodeNotFound)

	otherDraft := "draft-owned-by-other"
	createOther := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+otherDraft+`","payload":{"x":1}}`,
	))
	createOther.Header.Set("Authorization", "Bearer "+otherTok)
	createOther.Header.Set("Content-Type", "application/json")
	co := httptest.NewRecorder()
	h.ServeHTTP(co, createOther)
	if co.Code != http.StatusOK {
		t.Fatalf("create other: %d %s", co.Code, co.Body.String())
	}

	delOtherOwner := httptest.NewRequest(http.MethodDelete, cpmroutes.Drafts+"?id="+otherDraft, nil)
	delOtherOwner.Header.Set("Authorization", "Bearer "+ownerTok)
	do := httptest.NewRecorder()
	h.ServeHTTP(do, delOtherOwner)
	if do.Code != http.StatusNotFound {
		t.Fatalf("cross-owner delete existing: expected 404, got %d body=%s", do.Code, do.Body.String())
	}
	assertAuthErrorPayload(t, do, draftCodeNotFound)
}

func TestDraftPOSTWithScanIDWhenScanAuthzAllowed(t *testing.T) {
	h := newAuthedTestHandlerWithScanAuthz(t)
	token := mustToken(t, "draft-scan-user")
	draftID := "draft-with-scan"

	post := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+draftID+`","scan_id":"`+contractTestScanID+`","payload":{"bound":true}}`,
	))
	post.Header.Set("Authorization", "Bearer "+token)
	post.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, post)
	if postRec.Code != http.StatusOK {
		t.Fatalf("upsert with scan_id: expected 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id="+draftID, nil)
	get.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var record map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &record); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if record["scan_id"] != contractTestScanID {
		t.Fatalf("expected scan_id %q, got %v", contractTestScanID, record["scan_id"])
	}
}

func TestDraftPOSTUpsertResponseContract(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "draft-upsert-contract")
	draftID := "draft-upsert-contract-id"

	createBody := `{"id":"` + draftID + `","payload":{"v":1}}`
	create := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(createBody))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", createRec.Code, createRec.Body.String())
	}
	var createResp map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	assertDraftUpsertResponse(t, createResp, draftID)

	update := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+draftID+`","payload":{"v":2}}`,
	))
	update.Header.Set("Authorization", "Bearer "+token)
	update.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	h.ServeHTTP(updateRec, update)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updateResp map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	assertDraftUpsertResponse(t, updateResp, draftID)

	get := httptest.NewRequest(http.MethodGet, cpmroutes.Drafts+"?id="+draftID, nil)
	get.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get after update: %d %s", getRec.Code, getRec.Body.String())
	}
	var record map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &record); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	payload, ok := record["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %#v", record["payload"])
	}
	if payload["v"] != float64(2) {
		t.Fatalf("expected updated payload v=2, got %#v", payload["v"])
	}
}

func assertDraftUpsertResponse(t *testing.T, body map[string]any, wantDraftID string) {
	t.Helper()
	if body["draft_id"] != wantDraftID {
		t.Fatalf("expected draft_id %q, got %v", wantDraftID, body["draft_id"])
	}
	if body["status"] != draftStatusServerDraft {
		t.Fatalf("expected status %q, got %v", draftStatusServerDraft, body["status"])
	}
	if s, _ := body["saved_at"].(string); strings.TrimSpace(s) == "" {
		t.Fatalf("expected non-empty saved_at, got %#v", body["saved_at"])
	}
	if _, hasItem := body["item"]; hasItem {
		t.Fatalf("must not return item wrapper: %#v", body["item"])
	}
}
