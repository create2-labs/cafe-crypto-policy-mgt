package authz

import (
	"errors"
	"strings"
)

const (
	// AuthContractVersion is the frozen AUTH-00 contract version.
	AuthContractVersion = "cpm-auth-contract-v1"
)

const (
	RouteClassPublicHealth       = "public_health"
	RouteClassAuthenticated      = "authenticated_business"
	RouteClassDeprecatedDisabled = "deprecated_disabled"
	// RouteClassInternalService is gated by a static service token (not user JWT).
	// Used for service-to-service endpoints such as Discovery → CPM policy reference checks.
	RouteClassInternalService = "internal_service_token"
)

type Principal struct {
	UserID   string
	Subject  string
	TenantID string
	Claims   map[string]any
}

func (p Principal) Validate() error {
	if strings.TrimSpace(p.UserID) == "" {
		return errors.New("principal user_id is required")
	}
	if strings.TrimSpace(p.Subject) == "" {
		return errors.New("principal subject is required")
	}
	return nil
}

type APIError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	RequestID string         `json:"request_id,omitempty"`
}

func IsValidRouteClass(value string) bool {
	switch value {
	case RouteClassPublicHealth, RouteClassAuthenticated, RouteClassDeprecatedDisabled, RouteClassInternalService:
		return true
	default:
		return false
	}
}
