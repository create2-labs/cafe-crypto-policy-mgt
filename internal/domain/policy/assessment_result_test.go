package policy

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestAssessmentStatus_IsValid(t *testing.T) {
	cases := []struct {
		name   string
		status AssessmentStatus
		want   bool
	}{
		{name: "pending", status: AssessmentStatusPending, want: true},
		{name: "compatible and deployable", status: AssessmentStatusCompatibleAndDeployable, want: true},
		{name: "compatible but not deployable", status: AssessmentStatusCompatibleButNotDeployable, want: true},
		{name: "incompatible", status: AssessmentStatusIncompatible, want: true},
		{name: "error", status: AssessmentStatusError, want: true},
		{name: "invalid", status: AssessmentStatus("invalid"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.IsValid()
			if got != tc.want {
				t.Fatalf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssessmentFindingSeverity_IsValid(t *testing.T) {
	cases := []struct {
		name     string
		severity AssessmentFindingSeverity
		want     bool
	}{
		{name: "info", severity: AssessmentFindingSeverityInfo, want: true},
		{name: "warning", severity: AssessmentFindingSeverityWarning, want: true},
		{name: "blocking", severity: AssessmentFindingSeverityBlocking, want: true},
		{name: "invalid", severity: AssessmentFindingSeverity("error"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.severity.IsValid()
			if got != tc.want {
				t.Fatalf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewCryptoPolicyAssessmentResult(t *testing.T) {
	walletRef := AssessmentWalletReference{
		Address: "0x1234",
		ChainID: 8453,
	}

	result := NewCryptoPolicyAssessmentResult(
		"assessment_001",
		"cpx_001",
		walletRef,
		AssessmentStatusPending,
	)

	if result.ID != "assessment_001" {
		t.Fatalf("id: got %q", result.ID)
	}
	if result.CryptoPolicyInstanceID != "cpx_001" {
		t.Fatalf("crypto_policy_instance_id: got %q", result.CryptoPolicyInstanceID)
	}
	if result.WalletRef.Address != walletRef.Address {
		t.Fatalf("wallet_ref.address: got %q", result.WalletRef.Address)
	}
	if result.Status != AssessmentStatusPending {
		t.Fatalf("status: got %q", result.Status)
	}
	if result.EvaluatedAt.IsZero() {
		t.Fatal("evaluated_at is zero")
	}
	if result.EvaluatedAt.Location() != time.UTC {
		t.Fatalf("evaluated_at location: got %v want UTC", result.EvaluatedAt.Location())
	}
	if result.Findings == nil {
		t.Fatal("findings is nil")
	}
	if result.Warnings == nil {
		t.Fatal("warnings is nil")
	}
}

func TestCryptoPolicyAssessmentResult_Validate(t *testing.T) {
	base := CryptoPolicyAssessmentResult{
		ID:                     "assessment_001",
		CryptoPolicyInstanceID: "cpx_001",
		WalletRef: AssessmentWalletReference{
			Address: "0x1234",
			ChainID: 8453,
		},
		Status:      AssessmentStatusCompatibleAndDeployable,
		EvaluatedAt: time.Date(2026, time.April, 24, 10, 0, 0, 0, time.UTC),
		Findings: []AssessmentFinding{
			{
				Code:     "target_posture_matched",
				Severity: AssessmentFindingSeverityInfo,
			},
		},
	}

	t.Run("valid", func(t *testing.T) {
		err := base.Validate()
		if err != nil {
			t.Fatalf("Validate(): %v", err)
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		var nilResult *CryptoPolicyAssessmentResult
		err := nilResult.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing id", func(t *testing.T) {
		input := base
		input.ID = ""
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentResultIDRequired) {
			t.Fatalf("expected ErrAssessmentResultIDRequired, got %v", err)
		}
	})

	t.Run("missing instance id", func(t *testing.T) {
		input := base
		input.CryptoPolicyInstanceID = ""
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentResultInstanceIDRequired) {
			t.Fatalf("expected ErrAssessmentResultInstanceIDRequired, got %v", err)
		}
	})

	t.Run("missing wallet address", func(t *testing.T) {
		input := base
		input.WalletRef.Address = ""
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentResultWalletAddressRequired) {
			t.Fatalf("expected ErrAssessmentResultWalletAddressRequired, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		input := base
		input.Status = "bad_status"
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentResultStatusInvalid) {
			t.Fatalf("expected ErrAssessmentResultStatusInvalid, got %v", err)
		}
	})

	t.Run("missing evaluated at", func(t *testing.T) {
		input := base
		input.EvaluatedAt = time.Time{}
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentResultEvaluatedAtRequired) {
			t.Fatalf("expected ErrAssessmentResultEvaluatedAtRequired, got %v", err)
		}
	})

	t.Run("finding code required", func(t *testing.T) {
		input := base
		input.Findings = []AssessmentFinding{{
			Code:     "",
			Severity: AssessmentFindingSeverityWarning,
		}}
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentFindingCodeRequired) {
			t.Fatalf("expected ErrAssessmentFindingCodeRequired, got %v", err)
		}
	})

	t.Run("finding severity invalid", func(t *testing.T) {
		input := base
		input.Findings = []AssessmentFinding{{
			Code:     "missing_posture",
			Severity: "bad",
		}}
		err := input.Validate()
		if !errors.Is(err, ErrAssessmentFindingSeverityInvalid) {
			t.Fatalf("expected ErrAssessmentFindingSeverityInvalid, got %v", err)
		}
	})
}

func TestCryptoPolicyAssessmentResult_JSONRoundTrip(t *testing.T) {
	input := CryptoPolicyAssessmentResult{
		ID:                     "assessment_001",
		CryptoPolicyInstanceID: "cpx_001",
		TemplateID:             "tpl_hybrid_v1",
		WalletRef: AssessmentWalletReference{
			Address: "0x1234",
			ChainID: 8453,
		},
		Status:      AssessmentStatusCompatibleButNotDeployable,
		EvaluatedAt: time.Date(2026, time.April, 24, 10, 0, 0, 0, time.UTC),
		Findings: []AssessmentFinding{
			{
				Code:     "chain_not_supported",
				Message:  "requested chain is not supported by policy constraints",
				Severity: AssessmentFindingSeverityBlocking,
				Field:    "scope.chain_ids",
				Details: map[string]string{
					"requested_chain_id": "137",
				},
			},
		},
		Warnings:        []string{"paymaster currently unavailable"},
		CatalogVersion:  "v0.1",
		TemplateVersion: "1.0.0",
		CorrelationID:   "corr_001",
	}

	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got CryptoPolicyAssessmentResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if err := got.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if got.ID != input.ID {
		t.Fatalf("id: got %q want %q", got.ID, input.ID)
	}
	if got.Status != input.Status {
		t.Fatalf("status: got %q want %q", got.Status, input.Status)
	}
	if len(got.Findings) != 1 {
		t.Fatalf("findings length: got %d want 1", len(got.Findings))
	}
	if got.Findings[0].Code != "chain_not_supported" {
		t.Fatalf("finding code: got %q", got.Findings[0].Code)
	}
}
