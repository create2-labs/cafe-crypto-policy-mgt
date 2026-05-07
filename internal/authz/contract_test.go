package authz

import "testing"

func TestPrincipalValidate(t *testing.T) {
	t.Run("valid principal", func(t *testing.T) {
		p := Principal{UserID: "user-1", Subject: "sub-1"}
		if err := p.Validate(); err != nil {
			t.Fatalf("expected valid principal, got error: %v", err)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		p := Principal{Subject: "sub-1"}
		if err := p.Validate(); err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}

func TestAPIErrorValidate(t *testing.T) {
	ok := APIError{Code: "AUTH_UNAUTHORIZED", Message: "missing bearer token"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid error contract, got: %v", err)
	}

	invalid := APIError{Code: "", Message: "x"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid APIError")
	}
}

func TestScanAccessContracts(t *testing.T) {
	req := ScanAccessCheckRequest{
		ScanID: "scan-1",
		Principal: Principal{
			UserID:  "user-1",
			Subject: "sub-1",
		},
		RequestID: "req-1",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid scan access request, got: %v", err)
	}

	resp := ScanAccessCheckResponse{Decision: ScanAccessDecisionAllow}
	if err := resp.Validate(); err != nil {
		t.Fatalf("expected valid scan access response, got: %v", err)
	}

	invalid := ScanAccessCheckResponse{Decision: "maybe"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid decision error")
	}
}

func TestRouteClassValidation(t *testing.T) {
	if !IsValidRouteClass(RouteClassPublicHealth) {
		t.Fatal("expected public health route class to be valid")
	}
	if IsValidRouteClass("unknown") {
		t.Fatal("did not expect unknown route class to be valid")
	}
}
