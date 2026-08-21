package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

var (
	// ErrAssessmentRequestHandlerRequired indicates a missing assessment flow handler.
	ErrAssessmentRequestHandlerRequired = errors.New("nats: assessment request handler is required")
	// ErrIdempotencyStoreRequired indicates a missing idempotency store.
	ErrIdempotencyStoreRequired = errors.New("nats: idempotency store is required")
)

// AssessmentRequestInput is the normalized input delegated to CPM assessment flow.
// Wire v0.2: crypto_policy_id + observation (scan context); no selection_request / couche B.
type AssessmentRequestInput struct {
	Command        cafenatsv01.PolicyAssessmentRequested
	Observation    walletobserved.Event
	CryptoPolicyID string
}

// AssessmentRequestHandler executes the domain assessment flow.
type AssessmentRequestHandler interface {
	HandleAssessmentRequest(ctx context.Context, input AssessmentRequestInput) error
}

// AssessmentRequestConsumer handles inbound policy.assessment.requested.v0.1 events.
// It ignores all non-matching subjects by design.
type AssessmentRequestConsumer struct {
	idempotency IdempotencyStore
	handler     AssessmentRequestHandler
}

// NewAssessmentRequestConsumer builds a consumer with explicit dependencies.
func NewAssessmentRequestConsumer(store IdempotencyStore, handler AssessmentRequestHandler) (*AssessmentRequestConsumer, error) {
	if store == nil {
		return nil, ErrIdempotencyStoreRequired
	}
	if handler == nil {
		return nil, ErrAssessmentRequestHandlerRequired
	}
	return &AssessmentRequestConsumer{
		idempotency: store,
		handler:     handler,
	}, nil
}

// HandleMessage processes one NATS message body for the given subject.
// Legacy selection_request / couche-B fields fail at contracts decode (ErrLegacyAssessmentField)
// and are returned as a validation error (reject/nack path) — not an HTTP 400.
func (c *AssessmentRequestConsumer) HandleMessage(ctx context.Context, subject string, data []byte) error {
	if subject != cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01 {
		return nil
	}

	var event cafenatsv01.PolicyAssessmentRequested
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("event=cpm.assessment.validation_error subject=%q err=%q", subject, err.Error())
		return err
	}
	if err := event.Validate(); err != nil {
		log.Printf("event=cpm.assessment.validation_error event_id=%q err=%q", event.EventID, err.Error())
		return err
	}
	normalizeWalletSubjects(&event)

	claim, err := c.idempotency.Claim(ctx, event.EventID)
	if err != nil {
		return err
	}
	if claim != ClaimAccepted {
		return nil
	}

	input := AssessmentRequestInput{
		Command:        event,
		Observation:    event.Payload.Observation,
		CryptoPolicyID: event.Payload.CryptoPolicyID,
	}

	if err := c.handler.HandleAssessmentRequest(ctx, input); err != nil {
		_ = c.idempotency.Release(ctx, event.EventID)
		return err
	}
	return c.idempotency.MarkProcessed(ctx, event.EventID)
}

func normalizeWalletSubjects(event *cafenatsv01.PolicyAssessmentRequested) {
	if event == nil {
		return
	}
	event.Subject.ID = persistence.NormalizeWalletSubjectID(event.Subject.ID)
	event.Payload.Observation.Subject.ID = persistence.NormalizeWalletSubjectID(event.Payload.Observation.Subject.ID)
}
