package provider

// Stable soft-finding codes (ADR §7 / CPM-P5). These do not block ranking.
// Persist acceptance of these codes is enforced by policy.ValidatePayloadForPersist (CPM-P6).
// Wallet-control proof is persist-only and is never emitted as an explore finding.
const (
	FindingCodeRequiresBundler          = "requires_bundler"
	FindingCodeRequiresLocalSignerState = "requires_local_signer_state"
)

// SoftFinding is an explainable non-blocking provider constraint that ranked
// candidates must expose (accepted later at persist).
type SoftFinding struct {
	Code    string
	Message string
	Field   string
}

// EvaluateSoftFindings derives ADR §7 soft findings from a solution profile.
// False flags produce no finding. requires_wallet_control_proof is omitted.
func EvaluateSoftFindings(profile *SolutionProfile) []SoftFinding {
	if profile == nil {
		return nil
	}

	var findings []SoftFinding
	if profile.AccountModel.RequiresBundler {
		findings = append(findings, SoftFinding{
			Code:    FindingCodeRequiresBundler,
			Message: "solution account_model requires an ERC-4337 bundler",
			Field:   "account_model.requires_bundler",
		})
	}
	if profile.Constraints.RequiresLocalSignerState {
		findings = append(findings, SoftFinding{
			Code:    FindingCodeRequiresLocalSignerState,
			Message: "solution constraints require local signer state to exercise key rotation",
			Field:   "constraints.requires_local_signer_state",
		})
	}
	return findings
}
