package provider

import "testing"

func TestEvaluateUserConstraints_nicetryHappyAndContinuityKO(t *testing.T) {
	profile := &SolutionProfile{
		Signature:   SignatureProfile{KeyRotationModel: KeyRotationPerUserOp},
		Constraints: ProfileConstraints{RequiresNewAccount: true, AddressContinuitySupported: false},
	}

	ok := EvaluateUserConstraints(UserConstraints{
		AllowNewWallet: true, AddressContinuityRequired: false, KeyRotationModel: KeyRotationPerUserOp,
	}, profile)
	if len(ok) != 0 {
		t.Fatalf("expected couche B pass, got %+v", ok)
	}

	ko := EvaluateUserConstraints(UserConstraints{
		AllowNewWallet: true, AddressContinuityRequired: true, KeyRotationModel: KeyRotationPerUserOp,
	}, profile)
	if len(ko) != 1 || ko[0].Code != FindingCodeContinuity {
		t.Fatalf("expected continuity finding, got %+v", ko)
	}
}

func TestEvaluateUserConstraints_newWalletAndRotation(t *testing.T) {
	profile := &SolutionProfile{
		Signature:   SignatureProfile{KeyRotationModel: KeyRotationPerUserOp},
		Constraints: ProfileConstraints{RequiresNewAccount: true, AddressContinuitySupported: false},
	}

	if got := EvaluateUserConstraints(UserConstraints{
		AllowNewWallet: false, AddressContinuityRequired: false, KeyRotationModel: KeyRotationPerUserOp,
	}, profile); len(got) != 1 || got[0].Code != FindingCodeNewWallet {
		t.Fatalf("new wallet: %+v", got)
	}

	if got := EvaluateUserConstraints(UserConstraints{
		AllowNewWallet: true, AddressContinuityRequired: false, KeyRotationModel: KeyRotationNone,
	}, profile); len(got) != 1 || got[0].Code != FindingCodeRotation {
		t.Fatalf("rotation: %+v", got)
	}
}
