package nats

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
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
type AssessmentRequestInput struct {
	Command          cafenatsv01.PolicyAssessmentRequested
	Observation      walletobserved.Event
	SelectionRequest policy.PolicySelectionRequest
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
func (c *AssessmentRequestConsumer) HandleMessage(ctx context.Context, subject string, data []byte) error {
	if subject != cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01 {
		return nil
	}

	var event cafenatsv01.PolicyAssessmentRequested
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
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
		Command:          event,
		Observation:      event.Payload.Observation,
		SelectionRequest: mapSelectionRequest(event.Payload.SelectionRequest),
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

func mapSelectionRequest(in cafenatsv01.PolicySelectionRequestWire) policy.PolicySelectionRequest {
	out := policy.PolicySelectionRequest{
		TargetPosture:             vocabulary.CurrentPQPosture(in.TargetPosture),
		TargetChainIDs:            append([]int64(nil), in.TargetChainIDs...),
		RequireMultichain:         in.RequireMultichain,
		AllowNewWallet:            in.AllowNewWallet,
		AddressContinuityRequired: in.AddressContinuityRequired,
		KeyRotationRequired:       in.KeyRotationRequired,
		RecoveryRequired:          in.RecoveryRequired,
		MinimumMaturity:           in.MinimumMaturity,
		AllowResearch:             in.AllowResearch,
		PreferredFamilies:         append([]string(nil), in.PreferredFamilies...),
		PreferredProviders:        append([]string(nil), in.PreferredProviders...),
		RequireBundlerAvailable:   in.RequireBundlerAvailable,
		RequirePaymasterAvailable: in.RequirePaymasterAvailable,
		ApprovalMode:              policy.ApprovalMode(in.ApprovalMode),
	}

	if len(in.AllowedProviderModes) > 0 {
		out.AllowedProviderModes = make([]policy.ProviderMode, 0, len(in.AllowedProviderModes))
		for _, mode := range in.AllowedProviderModes {
			out.AllowedProviderModes = append(out.AllowedProviderModes, policy.ProviderMode(mode))
		}
	}
	out.Normalize()
	return out
}
