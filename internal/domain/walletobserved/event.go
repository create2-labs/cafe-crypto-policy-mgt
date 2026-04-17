package walletobserved

import (
	"time"

	"github.com/create2-labs/cafe-cpm/internal/domain/vocabulary"
)

// Event is the normalized discovery.wallet.observed envelope (contract v0.1).
// It is the canonical inbound shape CPM consumes from Discovery over NATS or APIs.
type Event struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	EventVersion  string    `json:"event_version"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	Producer      string    `json:"producer"`
	Subject       Subject   `json:"subject"`
	Payload       Payload   `json:"payload"`
}

// Subject identifies the wallet (or future subject types) for this observation.
type Subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Payload holds observation fields exported to CPM. Policy-relevant observations are
// listed first; derived fields are documented separately (see field comments).
type Payload struct {
	// Observed — direct or scanner-derived observation inputs used by policy code.
	ChainIDs         []int64   `json:"chain_ids"`
	AccountKind      string    `json:"account_kind"`
	CurrentAlgorithm string    `json:"current_algorithm"`
	PublicKeyExposed bool      `json:"public_key_exposed"`
	IsMultichain     bool      `json:"is_multichain"`
	ObservedAt       time.Time `json:"observed_at"`

	// Derived — deterministic summary computed on the Discovery export path (not persisted by CPM as Discovery-owned workflow state).
	CurrentPQPosture string `json:"current_pq_posture"`
}

// Validate checks normative vocabulary and envelope fields for a v0.1 event.
// It does not enforce business policy rules beyond exported vocabulary.
func (e *Event) Validate() error {
	if e.EventID == "" {
		return errEventID
	}
	if e.EventType != EventTypeWalletObserved {
		return errEventType
	}
	if e.EventVersion != EventVersionV01 {
		return errEventVersion
	}
	if e.Producer != ProducerCafeDiscovery {
		return errProducer
	}
	st := vocabulary.SubjectType(e.Subject.Type)
	if !st.IsValid() {
		return errSubjectType
	}
	if e.Subject.ID == "" {
		return errSubjectID
	}
	if !vocabulary.AccountKind(e.Payload.AccountKind).IsValid() {
		return errAccountKind
	}
	if !vocabulary.IsValidAlgorithmID(e.Payload.CurrentAlgorithm) {
		return errAlgorithmID
	}
	if !vocabulary.CurrentPQPosture(e.Payload.CurrentPQPosture).IsValid() {
		return errPQPosture
	}
	return nil
}
