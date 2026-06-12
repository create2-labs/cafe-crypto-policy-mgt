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

func TestRouteClassValidation(t *testing.T) {
	if !IsValidRouteClass(RouteClassPublicHealth) {
		t.Fatal("expected public health route class to be valid")
	}
	if !IsValidRouteClass(RouteClassInternalService) {
		t.Fatal("expected internal service route class to be valid")
	}
	if IsValidRouteClass("unknown") {
		t.Fatal("did not expect unknown route class to be valid")
	}
}
