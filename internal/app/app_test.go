package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerHealthz(t *testing.T) {
	h := handler("cafe-cpm")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if got := res.Body.String(); got != "cafe-cpm ok" {
		t.Fatalf("expected body %q, got %q", "cafe-cpm ok", got)
	}
}
