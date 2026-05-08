package authz

import (
	"errors"
	"fmt"
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

func (e APIError) Validate() error {
	if strings.TrimSpace(e.Code) == "" {
		return errors.New("error code is required")
	}
	if strings.TrimSpace(e.Message) == "" {
		return errors.New("error message is required")
	}
	return nil
}

type ScanAccessDecision string

const (
	ScanAccessDecisionAllow       ScanAccessDecision = "allow"
	ScanAccessDecisionDeny        ScanAccessDecision = "deny"
	ScanAccessDecisionUnavailable ScanAccessDecision = "unavailable"
)

type ScanAccessCheckRequest struct {
	ScanID    string
	Principal Principal
	RequestID string
}

func (r ScanAccessCheckRequest) Validate() error {
	if strings.TrimSpace(r.ScanID) == "" {
		return errors.New("scan_id is required")
	}
	if err := r.Principal.Validate(); err != nil {
		return fmt.Errorf("principal: %w", err)
	}
	return nil
}

type ScanAccessCheckResponse struct {
	Decision ScanAccessDecision
	Reason   string
}

func (r ScanAccessCheckResponse) Validate() error {
	switch r.Decision {
	case ScanAccessDecisionAllow, ScanAccessDecisionDeny, ScanAccessDecisionUnavailable:
		return nil
	default:
		return fmt.Errorf("unsupported decision: %q", r.Decision)
	}
}

func IsValidRouteClass(value string) bool {
	switch value {
	case RouteClassPublicHealth, RouteClassAuthenticated, RouteClassDeprecatedDisabled:
		return true
	default:
		return false
	}
}
