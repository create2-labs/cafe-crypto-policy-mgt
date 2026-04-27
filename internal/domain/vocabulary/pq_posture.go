package vocabulary

// CurrentPQPosture summarizes the wallet's post-quantum readiness as exported on the
// Discovery → CPM boundary. Values are normative for cafe.discovery.wallet.observed v0.1 payloads;
// derivation rules live in Discovery (see execution pack PR3).
type CurrentPQPosture string

const (
	PQPostureClassicalOnly CurrentPQPosture = "classical_only"
	PQPostureHybrid        CurrentPQPosture = "hybrid"
	PQPostureFullPQ        CurrentPQPosture = "full_pq"
	PQPostureUnknown       CurrentPQPosture = "unknown"
)

func (p CurrentPQPosture) String() string { return string(p) }

// IsValid reports whether p is a known exported posture value.
func (p CurrentPQPosture) IsValid() bool {
	switch p {
	case PQPostureClassicalOnly, PQPostureHybrid, PQPostureFullPQ, PQPostureUnknown:
		return true
	default:
		return false
	}
}
