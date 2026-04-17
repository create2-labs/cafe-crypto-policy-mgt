package vocabulary

// AccountKind classifies the on-chain account model for policy compatibility.
type AccountKind string

const (
	AccountKindEOA                 AccountKind = "eoa"
	AccountKindERC4337SmartAccount AccountKind = "erc4337_smart_account"
	AccountKindDelegatedEOA7702    AccountKind = "delegated_eoa_7702"
	AccountKindContractAccount     AccountKind = "contract_account"
	AccountKindUnknown             AccountKind = "unknown"
)

func (k AccountKind) String() string { return string(k) }

// IsValid reports whether k is a known exported account kind.
func (k AccountKind) IsValid() bool {
	switch k {
	case AccountKindEOA, AccountKindERC4337SmartAccount, AccountKindDelegatedEOA7702,
		AccountKindContractAccount, AccountKindUnknown:
		return true
	default:
		return false
	}
}
