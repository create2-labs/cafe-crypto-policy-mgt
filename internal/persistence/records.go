package persistence

import (
	"errors"
	"time"
)

var (
	ErrPrincipalRequired     = errors.New("principal is required")
	ErrDraftNotFound         = errors.New("draft not found")
	ErrDraftAlreadyPersisted = errors.New("draft already persisted")
	ErrPolicyNotFound        = errors.New("policy not found")
	ErrForbidden             = errors.New("forbidden")
)

type DraftRecord struct {
	ID          string
	OwnerUserID string
	TenantID    string
	ScanID      string
	Payload     map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyRecord struct {
	ID          string
	OwnerUserID string
	TenantID    string
	ScanID      string
	Payload     map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PersistDraftInput carries wallet ownership metadata applied to the persisted policy payload.
type PersistDraftInput struct {
	WalletAddress string
	ChainID       int64
	VerifiedAt    time.Time
}

// PersistDraftResult is the durable outcome of a successful draft persist transition.
type PersistDraftResult struct {
	PolicyID      string
	DraftID       string
	ScanID        string
	WalletAddress string
	ChainID       int64
	PersistedAt   time.Time
}
