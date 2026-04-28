package nats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	"github.com/create2-labs/cafe-cpm/internal/domain/policy"
)

var (
	// ErrNilPublisher indicates producer wiring is incomplete.
	ErrNilPublisher = errors.New("outbound producer: publisher is nil")
	// ErrNilIdempotencyStore indicates idempotent duplicate tracking is missing.
	ErrNilIdempotencyStore = errors.New("outbound producer: idempotency store is nil")
	// ErrEventIDRequired indicates missing event identity used for replay-safe behavior.
	ErrEventIDRequired = errors.New("outbound producer: event_id is required")
	// ErrCorrelationIDRequired indicates missing correlation id for traceability.
	ErrCorrelationIDRequired = errors.New("outbound producer: correlation_id is required")
	// ErrCausationIDRequired indicates missing causation id for traceability.
	ErrCausationIDRequired = errors.New("outbound producer: causation_id is required")
	// ErrSubjectPolicyInstanceIDRequired indicates missing subject policy instance id.
	ErrSubjectPolicyInstanceIDRequired = errors.New("outbound producer: subject_policy_instance_id is required")
	// ErrDuplicateEventPayloadMismatch indicates event identity reuse with divergent payload.
	ErrDuplicateEventPayloadMismatch = errors.New("outbound producer: duplicate event_id has divergent payload")
)

// Publisher is the minimal broker abstraction for outbound event publication.
type Publisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

// OutboundDedupStore tracks published event ids and payload hash for replay safety.
type OutboundDedupStore interface {
	Get(ctx context.Context, key string) (payloadHash string, found bool, err error)
	Put(ctx context.Context, key, payloadHash string) error
}

// OutboundProducer publishes CPM outbound events for remediation workflows.
type OutboundProducer struct {
	publisher Publisher
	store     OutboundDedupStore
	now       func() time.Time
}

// NewOutboundProducer builds a producer with explicit idempotent behavior.
func NewOutboundProducer(publisher Publisher, store OutboundDedupStore) (*OutboundProducer, error) {
	if publisher == nil {
		return nil, ErrNilPublisher
	}
	if store == nil {
		return nil, ErrNilIdempotencyStore
	}
	return &OutboundProducer{
		publisher: publisher,
		store:     store,
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

// AssessmentCompletedInput carries stable references for outbound projection.
type AssessmentCompletedInput struct {
	EventID       string
	OccurredAt    time.Time
	CorrelationID string
	CausationID   string
	SubjectPolicyInstanceID string
	AssessmentID  string
	InstanceID    string
	Status        policy.AssessmentStatus
	FindingCount  int
}

// RemediationRequestedInput carries stable references for outbound projection.
type RemediationRequestedInput struct {
	EventID              string
	OccurredAt           time.Time
	CorrelationID        string
	CausationID          string
	SubjectPolicyInstanceID string
	InstanceID           string
	RemediationID        string
	ReasonCode           string
	RequestedBy          string
	CorrelationRef       string
	AutoStartRemediation bool
}

// PublishAssessmentCompleted projects and publishes policy.assessment.completed.
func (p *OutboundProducer) PublishAssessmentCompleted(ctx context.Context, in AssessmentCompletedInput) error {
	if err := validateCommonInput(in.EventID, in.CorrelationID, in.CausationID, in.SubjectPolicyInstanceID); err != nil {
		return err
	}
	event := cafenatsv01.PolicyAssessmentCompleted{
		EnvelopeV01: cafenatsv01.EnvelopeV01{
			EventID:       in.EventID,
			EventType:     cafenatsv01.EventTypePolicyAssessmentCompleted,
			EventVersion:  cafenatsv01.EventVersionV01,
			OccurredAt:    p.occurredAt(in.OccurredAt),
			CorrelationID: in.CorrelationID,
			CausationID:   in.CausationID,
			Producer:      cafenatsv01.ProducerCafeCPM,
		},
		Subject: cafenatsv01.SubjectRef{
			Type: cafenatsv01.SubjectTypePolicyInstance,
			ID:   in.SubjectPolicyInstanceID,
		},
		Payload: cafenatsv01.PolicyAssessmentCompletedPayload{
			InstanceID:   in.InstanceID,
			AssessmentID: in.AssessmentID,
			Status:       mapAssessmentStatus(in.Status),
			FindingCount: in.FindingCount,
		},
	}
	return p.publishEvent(ctx, cafenatsv01.NATSSubjectPolicyAssessmentCompletedV01, event.EventID, &event)
}

// PublishRemediationRequested projects and publishes policy.remediation.requested.
func (p *OutboundProducer) PublishRemediationRequested(ctx context.Context, in RemediationRequestedInput) error {
	if err := validateCommonInput(in.EventID, in.CorrelationID, in.CausationID, in.SubjectPolicyInstanceID); err != nil {
		return err
	}
	correlationRef := in.CorrelationRef
	if !in.AutoStartRemediation {
		// Keep false-branch intent explicit in wire projection for downstream consumers.
		correlationRef = appendInformationalOnlyRef(correlationRef)
	}
	event := cafenatsv01.PolicyRemediationRequested{
		EnvelopeV01: cafenatsv01.EnvelopeV01{
			EventID:       in.EventID,
			EventType:     cafenatsv01.EventTypePolicyRemediationRequested,
			EventVersion:  cafenatsv01.EventVersionV01,
			OccurredAt:    p.occurredAt(in.OccurredAt),
			CorrelationID: in.CorrelationID,
			CausationID:   in.CausationID,
			Producer:      cafenatsv01.ProducerCafeCPM,
		},
		Subject: cafenatsv01.SubjectRef{
			Type: cafenatsv01.SubjectTypePolicyInstance,
			ID:   in.SubjectPolicyInstanceID,
		},
		Payload: cafenatsv01.PolicyRemediationRequestedPayload{
			InstanceID:     in.InstanceID,
			RemediationID:  in.RemediationID,
			ReasonCode:     in.ReasonCode,
			RequestedBy:    in.RequestedBy,
			CorrelationRef: correlationRef,
		},
	}
	return p.publishEvent(ctx, cafenatsv01.NATSSubjectPolicyRemediationRequestedV01, event.EventID, &event)
}

func (p *OutboundProducer) publishEvent(ctx context.Context, subject, eventID string, event interface{ Validate() error }) error {
	if err := event.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := p.checkDuplicate(ctx, subject, eventID, body); err != nil {
		return err
	}
	if err := p.publisher.Publish(ctx, subject, body); err != nil {
		return err
	}
	return p.store.Put(ctx, duplicateKey(subject, eventID), payloadHash(body))
}

func (p *OutboundProducer) checkDuplicate(ctx context.Context, subject, eventID string, payload []byte) error {
	key := duplicateKey(subject, eventID)
	existingHash, found, err := p.store.Get(ctx, key)
	if err != nil {
		return err
	}
	currentHash := payloadHash(payload)
	if !found {
		return nil
	}
	if existingHash == currentHash {
		return nil
	}
	return fmt.Errorf("%w: subject=%s event_id=%s", ErrDuplicateEventPayloadMismatch, subject, eventID)
}

func (p *OutboundProducer) occurredAt(value time.Time) time.Time {
	if value.IsZero() {
		return p.now()
	}
	return value.UTC()
}

func validateCommonInput(eventID, correlationID, causationID, subjectPolicyInstanceID string) error {
	switch {
	case eventID == "":
		return ErrEventIDRequired
	case correlationID == "":
		return ErrCorrelationIDRequired
	case causationID == "":
		return ErrCausationIDRequired
	case subjectPolicyInstanceID == "":
		return ErrSubjectPolicyInstanceIDRequired
	default:
		return nil
	}
}

func mapAssessmentStatus(status policy.AssessmentStatus) string {
	switch status {
	case policy.AssessmentStatusError:
		return cafenatsv01.PolicyAssessmentStatusFailed
	case policy.AssessmentStatusCompatibleButNotDeployable:
		return cafenatsv01.PolicyAssessmentStatusPartial
	default:
		return cafenatsv01.PolicyAssessmentStatusSucceeded
	}
}

func appendInformationalOnlyRef(ref string) string {
	const suffix = "informational_only=true"
	if ref == "" {
		return suffix
	}
	return ref + ";" + suffix
}

func duplicateKey(subject, eventID string) string {
	return subject + ":" + eventID
}

func payloadHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
