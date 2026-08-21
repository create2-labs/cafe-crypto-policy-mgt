package nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	walletv01 "github.com/create2-labs/cafe-contracts/observation/wallet/v01"
)

func TestAssessmentRequestConsumer_FirstDeliveryAndDuplicate(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	handler := &fakeAssessmentHandler{}
	consumer, err := NewAssessmentRequestConsumer(store, handler)
	if err != nil {
		t.Fatalf("NewAssessmentRequestConsumer() error = %v", err)
	}

	event := validAssessmentRequest("evt_pol_req_1")
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		t.Fatalf("first HandleMessage() error = %v", err)
	}
	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		t.Fatalf("duplicate HandleMessage() error = %v", err)
	}

	if got := handler.calls; got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
	if got := handler.last.CryptoPolicyID; got != "cpm_pq_account_validation_v1" {
		t.Fatalf("crypto_policy_id = %q, want catalogue id", got)
	}
	if got := handler.last.Command.Subject.ID; got != "wallet:0x742d35cc6634c0532925a3b844bc454e4438f44e" {
		t.Fatalf("command subject.id = %q, want canonical lowercase wallet subject id", got)
	}
	if got := handler.last.Observation.Subject.ID; got != "wallet:0x742d35cc6634c0532925a3b844bc454e4438f44e" {
		t.Fatalf("observation subject.id = %q, want canonical lowercase wallet subject id", got)
	}
}

func TestAssessmentRequestConsumer_LegacySelectionRequestValidationError(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	handler := &fakeAssessmentHandler{}
	consumer, err := NewAssessmentRequestConsumer(store, handler)
	if err != nil {
		t.Fatalf("NewAssessmentRequestConsumer() error = %v", err)
	}

	raw := map[string]any{
		"event_id":       "evt_pol_req_legacy",
		"event_type":     cafenatsv01.EventTypePolicyAssessmentRequested,
		"event_version":  cafenatsv01.EventVersionV01,
		"occurred_at":    time.Date(2026, time.April, 17, 10, 0, 2, 0, time.UTC),
		"correlation_id": "corr_scan_0001",
		"causation_id":   "ui_action_0001",
		"producer":       cafenatsv01.ProducerCafeDiscovery,
		"subject": map[string]any{
			"type": cafenatsv01.SubjectTypeWallet,
			"id":   "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		"payload": map[string]any{
			"crypto_policy_id": "cpm_pq_account_validation_v1",
			"observation":      validObservationEvent("evt_obs_legacy"),
			"selection_request": map[string]any{
				"target_posture": "hybrid",
			},
		},
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload)
	if err == nil {
		t.Fatal("expected validation error for legacy selection_request")
	}
	if !errors.Is(err, cafenatsv01.ErrLegacyAssessmentField) {
		t.Fatalf("error = %v, want ErrLegacyAssessmentField", err)
	}
	if handler.calls != 0 {
		t.Fatalf("handler must not run on validation error, calls=%d", handler.calls)
	}
}

func TestAssessmentRequestConsumer_ReplayAfterReload(t *testing.T) {
	store := NewInMemoryIdempotencyStoreWithProcessed("evt_pol_req_2")
	handler := &fakeAssessmentHandler{}
	consumer, err := NewAssessmentRequestConsumer(store, handler)
	if err != nil {
		t.Fatalf("NewAssessmentRequestConsumer() error = %v", err)
	}

	event := validAssessmentRequest("evt_pol_req_2")
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if got := handler.calls; got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

func TestAssessmentRequestConsumer_RetryAfterFailure(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	handler := &fakeAssessmentHandler{
		failuresRemaining: 1,
	}
	consumer, err := NewAssessmentRequestConsumer(store, handler)
	if err != nil {
		t.Fatalf("NewAssessmentRequestConsumer() error = %v", err)
	}

	event := validAssessmentRequest("evt_pol_req_3")
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err == nil {
		t.Fatalf("first HandleMessage() expected error")
	}
	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		t.Fatalf("retry HandleMessage() error = %v", err)
	}

	if got := handler.calls; got != 2 {
		t.Fatalf("handler calls = %d, want 2", got)
	}
}

func TestAssessmentRequestConsumer_ObservationSubjectIsNonTriggering(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	handler := &fakeAssessmentHandler{}
	consumer, err := NewAssessmentRequestConsumer(store, handler)
	if err != nil {
		t.Fatalf("NewAssessmentRequestConsumer() error = %v", err)
	}

	observation := validObservationEvent("evt_obs_1")
	payload, err := json.Marshal(observation)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := consumer.HandleMessage(context.Background(), cafenatsv01.NATSSubjectDiscoveryWalletObservedV01, payload); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if got := handler.calls; got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

type fakeAssessmentHandler struct {
	calls             int
	failuresRemaining int
	last              AssessmentRequestInput
}

func (h *fakeAssessmentHandler) HandleAssessmentRequest(_ context.Context, input AssessmentRequestInput) error {
	h.calls++
	h.last = input
	if h.failuresRemaining > 0 {
		h.failuresRemaining--
		return errors.New("transient failure")
	}
	return nil
}

func validAssessmentRequest(eventID string) cafenatsv01.PolicyAssessmentRequested {
	return cafenatsv01.PolicyAssessmentRequested{
		EnvelopeV01: cafenatsv01.EnvelopeV01{
			EventID:       eventID,
			EventType:     cafenatsv01.EventTypePolicyAssessmentRequested,
			EventVersion:  cafenatsv01.EventVersionV01,
			OccurredAt:    time.Date(2026, time.April, 17, 10, 0, 2, 0, time.UTC),
			CorrelationID: "corr_scan_0001",
			CausationID:   "ui_action_0001",
			Producer:      cafenatsv01.ProducerCafeDiscovery,
		},
		Subject: cafenatsv01.SubjectRef{
			Type: cafenatsv01.SubjectTypeWallet,
			ID:   "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		Payload: cafenatsv01.PolicyAssessmentRequestedPayload{
			CryptoPolicyID:  "cpm_pq_account_validation_v1",
			Observation:     validObservationEvent("evt_obs_2"),
			ClientRequestID: "req_0001",
		},
	}
}

func validObservationEvent(eventID string) walletv01.Event {
	return walletv01.Event{
		EventID:       eventID,
		EventType:     walletv01.EventTypeWalletObserved,
		EventVersion:  walletv01.EventVersion,
		OccurredAt:    time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC),
		CorrelationID: "corr_scan_0001",
		CausationID:   "scan_job_0001",
		Producer:      walletv01.ProducerCafeDiscovery,
		Subject: walletv01.Subject{
			Type: string(walletv01.SubjectTypeWallet),
			ID:   "wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		Payload: walletv01.Payload{
			ChainIDs:         []int64{1, 8453},
			AccountKind:      string(walletv01.AccountKindEOA),
			CurrentAlgorithm: string(walletv01.AlgorithmSecp256k1ECRecover),
			CurrentPQPosture: string(walletv01.PQPostureClassicalOnly),
			PublicKeyExposed: true,
			IsMultichain:     true,
			ObservedAt:       time.Date(2026, time.April, 17, 9, 59, 58, 0, time.UTC),
		},
	}
}
