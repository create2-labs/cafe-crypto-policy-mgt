package nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	"github.com/create2-labs/cafe-cpm/internal/domain/policy"
)

func TestPublishAssessmentCompletedPublishesExpectedEvent(t *testing.T) {
	t.Parallel()

	publisher := &publisherStub{}
	store := newMemoryStore()
	producer := newProducerForTests(t, publisher, store)

	err := producer.PublishAssessmentCompleted(context.Background(), AssessmentCompletedInput{
		EventID:       "evt_assess_1",
		OccurredAt:    time.Date(2026, time.April, 28, 9, 0, 0, 0, time.UTC),
		CorrelationID: "corr_1",
		CausationID:   "evt_req_1",
		SubjectPolicyInstanceID: "CP001",
		AssessmentID:  "assess_1",
		InstanceID:    "CP001",
		Status:        policy.AssessmentStatusCompatibleAndDeployable,
		FindingCount:  2,
	})
	if err != nil {
		t.Fatalf("publish assessment completed: %v", err)
	}

	if len(publisher.calls) != 1 {
		t.Fatalf("expected one publish call, got %d", len(publisher.calls))
	}
	call := publisher.calls[0]
	if call.subject != cafenatsv01.NATSSubjectPolicyAssessmentCompletedV01 {
		t.Fatalf("unexpected subject: %s", call.subject)
	}

	var event cafenatsv01.PolicyAssessmentCompleted
	if err := json.Unmarshal(call.payload, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("event validation failed: %v", err)
	}
	if event.Payload.Status != cafenatsv01.PolicyAssessmentStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", event.Payload.Status)
	}
}

func TestPublishRemediationRequestedSupportsAutoStartBranches(t *testing.T) {
	t.Parallel()

	t.Run("auto-start true keeps correlation ref untouched", func(t *testing.T) {
		publisher := &publisherStub{}
		store := newMemoryStore()
		producer := newProducerForTests(t, publisher, store)

		err := producer.PublishRemediationRequested(context.Background(), RemediationRequestedInput{
			EventID:              "evt_rem_1",
			OccurredAt:           time.Date(2026, time.April, 28, 9, 1, 0, 0, time.UTC),
			CorrelationID:        "corr_1",
			CausationID:          "evt_assess_1",
			SubjectPolicyInstanceID: "CP001",
			InstanceID:           "CP001",
			RemediationID:        "rem_1",
			ReasonCode:           "non_compliant",
			RequestedBy:          "cpm",
			CorrelationRef:       "assessment_id=assess_1",
			AutoStartRemediation: true,
		})
		if err != nil {
			t.Fatalf("publish remediation requested: %v", err)
		}

		event := decodeRemediationEvent(t, publisher.calls[0].payload)
		if event.Payload.CorrelationRef != "assessment_id=assess_1" {
			t.Fatalf("unexpected correlation_ref: %s", event.Payload.CorrelationRef)
		}
	})

	t.Run("auto-start false adds informational marker", func(t *testing.T) {
		publisher := &publisherStub{}
		store := newMemoryStore()
		producer := newProducerForTests(t, publisher, store)

		err := producer.PublishRemediationRequested(context.Background(), RemediationRequestedInput{
			EventID:              "evt_rem_2",
			OccurredAt:           time.Date(2026, time.April, 28, 9, 2, 0, 0, time.UTC),
			CorrelationID:        "corr_2",
			CausationID:          "evt_assess_2",
			SubjectPolicyInstanceID: "CP002",
			InstanceID:           "CP002",
			RemediationID:        "rem_2",
			ReasonCode:           "needs_rotation",
			RequestedBy:          "cpm",
			CorrelationRef:       "assessment_id=assess_2",
			AutoStartRemediation: false,
		})
		if err != nil {
			t.Fatalf("publish remediation requested: %v", err)
		}

		event := decodeRemediationEvent(t, publisher.calls[0].payload)
		want := "assessment_id=assess_2;informational_only=true"
		if event.Payload.CorrelationRef != want {
			t.Fatalf("unexpected correlation_ref: got=%s want=%s", event.Payload.CorrelationRef, want)
		}
	})
}

func TestPublishAssessmentCompletedSuppressesIdenticalDuplicate(t *testing.T) {
	t.Parallel()

	publisher := &publisherStub{}
	store := newMemoryStore()
	producer := newProducerForTests(t, publisher, store)
	input := AssessmentCompletedInput{
		EventID:       "evt_assess_dup",
		OccurredAt:    time.Date(2026, time.April, 28, 9, 0, 0, 0, time.UTC),
		CorrelationID: "corr_dup",
		CausationID:   "evt_req_dup",
		SubjectPolicyInstanceID: "CP003",
		AssessmentID:  "assess_dup",
		InstanceID:    "CP003",
		Status:        policy.AssessmentStatusCompatibleAndDeployable,
		FindingCount:  1,
	}

	if err := producer.PublishAssessmentCompleted(context.Background(), input); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}
	if err := producer.PublishAssessmentCompleted(context.Background(), input); err != nil {
		t.Fatalf("duplicate publish failed: %v", err)
	}

	if len(publisher.calls) != 2 {
		t.Fatalf("expected deterministic republish behavior (2 calls), got %d", len(publisher.calls))
	}
	if string(publisher.calls[0].payload) != string(publisher.calls[1].payload) {
		t.Fatal("duplicate publish payload diverged")
	}
}

func TestPublishAssessmentCompletedRejectsDivergentDuplicate(t *testing.T) {
	t.Parallel()

	publisher := &publisherStub{}
	store := newMemoryStore()
	producer := newProducerForTests(t, publisher, store)

	base := AssessmentCompletedInput{
		EventID:       "evt_assess_dup_divergent",
		OccurredAt:    time.Date(2026, time.April, 28, 9, 0, 0, 0, time.UTC),
		CorrelationID: "corr_dup_divergent",
		CausationID:   "evt_req_dup_divergent",
		SubjectPolicyInstanceID: "CP004",
		AssessmentID:  "assess_dup_divergent",
		InstanceID:    "CP004",
		Status:        policy.AssessmentStatusCompatibleAndDeployable,
		FindingCount:  1,
	}

	if err := producer.PublishAssessmentCompleted(context.Background(), base); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}

	divergent := base
	divergent.FindingCount = 42
	err := producer.PublishAssessmentCompleted(context.Background(), divergent)
	if !errors.Is(err, ErrDuplicateEventPayloadMismatch) {
		t.Fatalf("expected ErrDuplicateEventPayloadMismatch, got %v", err)
	}
}

func newProducerForTests(t *testing.T, publisher Publisher, store IdempotencyStore) *OutboundProducer {
	t.Helper()
	producer, err := NewOutboundProducer(publisher, store)
	if err != nil {
		t.Fatalf("new outbound producer: %v", err)
	}
	return producer
}

func decodeRemediationEvent(t *testing.T, payload []byte) cafenatsv01.PolicyRemediationRequested {
	t.Helper()
	var event cafenatsv01.PolicyRemediationRequested
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal remediation event: %v", err)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("validate remediation event: %v", err)
	}
	return event
}

type publishCall struct {
	subject string
	payload []byte
}

type publisherStub struct {
	calls []publishCall
}

func (p *publisherStub) Publish(_ context.Context, subject string, payload []byte) error {
	p.calls = append(p.calls, publishCall{subject: subject, payload: append([]byte(nil), payload...)})
	return nil
}

type memoryStore struct {
	values map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: make(map[string]string)}
}

func (m *memoryStore) Get(_ context.Context, key string) (string, bool, error) {
	value, ok := m.values[key]
	return value, ok, nil
}

func (m *memoryStore) Put(_ context.Context, key, payloadHash string) error {
	m.values[key] = payloadHash
	return nil
}
