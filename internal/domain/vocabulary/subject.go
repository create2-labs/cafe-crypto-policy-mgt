package vocabulary

// SubjectType identifies the kind of subject referenced in integration events.
type SubjectType string

const (
	// SubjectTypeWallet is the subject type for wallet-scoped observations and policy decisions.
	SubjectTypeWallet SubjectType = "wallet"
)

// IsValid reports whether s is a known exported subject type.
func (t SubjectType) IsValid() bool {
	switch t {
	case SubjectTypeWallet:
		return true
	default:
		return false
	}
}
