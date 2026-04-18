package walletobserved

import "errors"

var (
	errEventID      = errors.New("walletobserved: event_id is required")
	errEventType    = errors.New("walletobserved: event_type must be discovery.wallet.observed")
	errEventVersion = errors.New("walletobserved: event_version must be v0.1")
	errProducer     = errors.New("walletobserved: producer must be cafe-discovery for this contract revision")
	errSubjectType  = errors.New("walletobserved: subject.type must be a known exported subject type")
	errSubjectID    = errors.New("walletobserved: subject.id is required")
	errAccountKind  = errors.New("walletobserved: payload.account_kind must be a known exported account kind")
	errAlgorithmID  = errors.New("walletobserved: payload.current_algorithm must be a known algorithm id or hybrid_*")
	errPQPosture    = errors.New("walletobserved: payload.current_pq_posture must be a known exported posture")
)
