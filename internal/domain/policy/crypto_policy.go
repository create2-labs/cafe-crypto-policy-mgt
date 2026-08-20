package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

var (
	// ErrCryptoPolicyIDRequired indicates a missing crypto policy identifier.
	ErrCryptoPolicyIDRequired = errors.New("crypto policy id is required")
	// ErrCryptoPolicyNameRequired indicates a missing crypto policy name.
	ErrCryptoPolicyNameRequired = errors.New("crypto policy name is required")
	// ErrCryptoPolicyVersionRequired indicates a missing crypto policy version.
	ErrCryptoPolicyVersionRequired = errors.New("crypto policy version is required")
	// ErrCryptoPolicyRequiredPostureRequired indicates a missing required posture.
	ErrCryptoPolicyRequiredPostureRequired = errors.New("crypto policy required_posture is required")
	// ErrCryptoPolicyRequiredPostureInvalid indicates an invalid required posture.
	ErrCryptoPolicyRequiredPostureInvalid = errors.New("crypto policy required_posture is invalid")
	// ErrCryptoPolicyAllowedProvidersRequired indicates an empty allowed_providers list.
	ErrCryptoPolicyAllowedProvidersRequired = errors.New("crypto policy allowed_providers is required")
	// ErrCryptoPolicyAllowedProviderEmpty indicates a blank allowed_providers entry.
	ErrCryptoPolicyAllowedProviderEmpty = errors.New("crypto policy allowed_providers must not contain empty values")
)

// CryptoPolicy is a catalogued CAFE intention: required posture and allowed providers.
// It does not bind a solution_profile_ref; that binding is chosen at persist time.
type CryptoPolicy struct {
	ID               string                      `json:"id"`
	Name             string                      `json:"name"`
	Version          string                      `json:"version"`
	Description      string                      `json:"description,omitempty"`
	RequiredPosture  vocabulary.CurrentPQPosture `json:"required_posture"`
	AllowedProviders []string                    `json:"allowed_providers"`
}

// LoadCryptoPolicyFromFile reads, decodes, normalizes, and validates a crypto policy.
func LoadCryptoPolicyFromFile(path string) (*CryptoPolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read crypto policy file: %w", err)
	}
	var cp CryptoPolicy
	if err := json.Unmarshal(raw, &cp); err != nil {
		return nil, fmt.Errorf("decode crypto policy file: %w", err)
	}
	if err := cp.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	return &cp, nil
}

// Normalize applies deterministic canonicalization for stable catalogue reads.
func (c *CryptoPolicy) Normalize() {
	if c == nil {
		return
	}
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.TrimSpace(c.Name)
	c.Version = strings.TrimSpace(c.Version)
	c.Description = strings.TrimSpace(c.Description)
	c.AllowedProviders = normalizeStringsPreserveOrder(c.AllowedProviders)
}

// Validate ensures crypto policy catalogue integrity (ADR §5.2).
func (c *CryptoPolicy) Validate() error {
	if c == nil {
		return errors.New("crypto policy is nil")
	}
	if c.ID == "" {
		return ErrCryptoPolicyIDRequired
	}
	if c.Name == "" {
		return ErrCryptoPolicyNameRequired
	}
	if c.Version == "" {
		return ErrCryptoPolicyVersionRequired
	}
	if c.RequiredPosture == "" {
		return ErrCryptoPolicyRequiredPostureRequired
	}
	if !c.RequiredPosture.IsValid() || c.RequiredPosture == vocabulary.PQPostureUnknown {
		return fmt.Errorf("%w: %q", ErrCryptoPolicyRequiredPostureInvalid, c.RequiredPosture)
	}
	if len(c.AllowedProviders) == 0 {
		return ErrCryptoPolicyAllowedProvidersRequired
	}
	for _, providerID := range c.AllowedProviders {
		if strings.TrimSpace(providerID) == "" {
			return ErrCryptoPolicyAllowedProviderEmpty
		}
	}
	return nil
}

// NormalizeAndValidate applies normalization then validates the crypto policy.
func (c *CryptoPolicy) NormalizeAndValidate() error {
	c.Normalize()
	return c.Validate()
}
