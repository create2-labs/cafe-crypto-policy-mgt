package nats

import (
	"context"
	"sync"
)

// ClaimStatus reports whether an inbound event_id can be processed.
type ClaimStatus int

const (
	ClaimAccepted ClaimStatus = iota
	ClaimDuplicate
	ClaimInFlight
)

// IdempotencyStore tracks inbound event_ids for duplicate suppression and replay handling.
type IdempotencyStore interface {
	Claim(ctx context.Context, eventID string) (ClaimStatus, error)
	MarkProcessed(ctx context.Context, eventID string) error
	Release(ctx context.Context, eventID string) error
}

type eventState int

const (
	stateUnknown eventState = iota
	stateInFlight
	stateProcessed
)

// InMemoryIdempotencyStore provides deterministic duplicate suppression for tests/local runs.
type InMemoryIdempotencyStore struct {
	mu     sync.Mutex
	states map[string]eventState
}

// NewInMemoryIdempotencyStore creates a blank store.
func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	return &InMemoryIdempotencyStore{
		states: make(map[string]eventState),
	}
}

// NewInMemoryIdempotencyStoreWithProcessed creates a store preloaded with processed ids.
func NewInMemoryIdempotencyStoreWithProcessed(eventIDs ...string) *InMemoryIdempotencyStore {
	store := NewInMemoryIdempotencyStore()
	for _, id := range eventIDs {
		if id == "" {
			continue
		}
		store.states[id] = stateProcessed
	}
	return store
}

func (s *InMemoryIdempotencyStore) Claim(_ context.Context, eventID string) (ClaimStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.states[eventID] {
	case stateProcessed:
		return ClaimDuplicate, nil
	case stateInFlight:
		return ClaimInFlight, nil
	default:
		s.states[eventID] = stateInFlight
		return ClaimAccepted, nil
	}
}

func (s *InMemoryIdempotencyStore) MarkProcessed(_ context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[eventID] = stateProcessed
	return nil
}

func (s *InMemoryIdempotencyStore) Release(_ context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.states[eventID] == stateInFlight {
		delete(s.states, eventID)
	}
	return nil
}
