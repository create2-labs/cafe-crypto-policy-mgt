package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
)

func TestGETWalletTargetContextRequiresAddress(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "wtc-query-user")

	req := httptest.NewRequest(http.MethodGet, cpmroutes.WalletTargetContext, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGETWalletTargetContextExistsWithDraftID(t *testing.T) {
	h := newAuthedTestHandler(t)
	token := mustToken(t, "wtc-exists-user")
	addr := "0x70Af6FeA3DF8a81fA71E5E5abc2989F6880CFa21"
	draftID := "draft-wtc-single"

	post := httptest.NewRequest(http.MethodPost, cpmroutes.Drafts, strings.NewReader(
		`{"id":"`+draftID+`","payload":{"policy_context":{"result":{"target_address":"`+addr+`"}}}}`,
	))
	post.Header.Set("Authorization", "Bearer "+token)
	post.Header.Set("Content-Type", "application/json")
	pr := httptest.NewRecorder()
	h.ServeHTTP(pr, post)
	if pr.Code != http.StatusOK {
		t.Fatalf("draft post: %d %s", pr.Code, pr.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, cpmroutes.WalletTargetContext+"?target_address="+addr, nil)
	get.Header.Set("Authorization", "Bearer "+token)
	gr := httptest.NewRecorder()
	h.ServeHTTP(gr, get)
	if gr.Code != http.StatusOK {
		t.Fatalf("context get: %d %s", gr.Code, gr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(gr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["exists"] != true {
		t.Fatalf("exists = %v", out["exists"])
	}
	if out["platform_draft_id"] != draftID {
		t.Fatalf("platform_draft_id = %v, want %s", out["platform_draft_id"], draftID)
	}
}
