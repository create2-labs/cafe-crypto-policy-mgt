package persistence

import (
	"errors"
	"time"
)

var (
	ErrPrincipalRequired   = errors.New("principal is required")
	ErrPolicyNotFound      = errors.New("policy not found")
	ErrPolicyAlreadyExists = errors.New("active policy already exists for wallet")
	ErrForbidden           = errors.New("forbidden")
)

type PolicyRecord struct {
	ID            string
	OwnerUserID   string
	TenantID      string
	ScanID        string
	Payload       map[string]any
	PayloadSHA256 string
	WalletAddress string
	ChainID       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreatePolicyInput is the CPM→persistence create body after EIP-191 + gates (RD-P5).
type CreatePolicyInput struct {
	ScanID                  string
	WalletAddress           string
	ChainID                 int64
	Payload                 map[string]any
	PayloadSHA256           string
	SignedMessageHash       string
	WalletControlMethod     string
	WalletControlVerifiedAt time.Time
	ChallengeIssuedAt       *time.Time
	ChallengeExpiresAt      *time.Time
}

// CreatePolicyResult is the durable outcome of a successful CreatePolicy.
type CreatePolicyResult struct {
	PolicyID      string
	ScanID        string
	WalletAddress string
	ChainID       int64
	PayloadSHA256 string
	PersistedAt   time.Time
}
