package walletauth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func TestVerifyAuthorization_bindingAndFreshnessFailures(t *testing.T) {
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	message := vectorMessage(t, issued, expires)
	signature := signPersonalMessage(t, message)
	now := issued.Add(time.Minute)

	base := walletauth.VerifyInput{
		Domain:        vectorDomain,
		WalletAddress: vectorWalletAddress,
		ChainID:       1,
		ScanID:        vectorScanID,
		PayloadSHA256: vectorPayloadSHA256,
		SignedMessage: message,
		Signature:     signature,
		Now:           now,
		ClockSkew:     walletauth.DefaultClockSkew,
	}

	cases := []struct {
		name     string
		mutate   func(*walletauth.VerifyInput)
		wantCode string
	}{
		{
			name: "wrong wallet",
			mutate: func(in *walletauth.VerifyInput) {
				in.WalletAddress = "0x0000000000000000000000000000000000000001"
			},
			wantCode: walletauth.CodeWalletAuthorizationWalletMismatch,
		},
		{
			name: "wrong chain",
			mutate: func(in *walletauth.VerifyInput) {
				in.ChainID = 8453
			},
			wantCode: walletauth.CodeWalletAuthorizationChainMismatch,
		},
		{
			name: "wrong payload sha",
			mutate: func(in *walletauth.VerifyInput) {
				in.PayloadSHA256 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
			wantCode: walletauth.CodePayloadSHA256Mismatch,
		},
		{
			name: "wrong scan",
			mutate: func(in *walletauth.VerifyInput) {
				in.ScanID = "00000000-0000-0000-0000-000000000088"
			},
			wantCode: walletauth.CodeWalletAuthorizationScanMismatch,
		},
		{
			name: "expired message",
			mutate: func(in *walletauth.VerifyInput) {
				in.Now = expires.Add(time.Second)
			},
			wantCode: walletauth.CodeWalletAuthorizationExpired,
		},
		{
			name: "future issued_at",
			mutate: func(in *walletauth.VerifyInput) {
				in.Now = issued.Add(-time.Minute)
			},
			wantCode: walletauth.CodeWalletAuthorizationNotYetValid,
		},
		{
			name: "validity window too long",
			mutate: func(in *walletauth.VerifyInput) {
				longExpires := issued.Add(11 * time.Minute)
				in.SignedMessage = vectorMessage(t, issued, longExpires)
				in.Signature = signPersonalMessage(t, in.SignedMessage)
			},
			wantCode: walletauth.CodeWalletAuthorizationValidityLong,
		},
		{
			name: "invalid signature",
			mutate: func(in *walletauth.VerifyInput) {
				in.Signature = "0x" + strings.Repeat("11", 65)
			},
			wantCode: walletauth.CodeInvalidWalletSignature,
		},
		{
			name: "wrong action in message",
			mutate: func(in *walletauth.VerifyInput) {
				in.SignedMessage = strings.Replace(message, "Action: persist_crypto_policy", "Action: evil_action", 1)
				in.Signature = signPersonalMessage(t, in.SignedMessage)
				in.PayloadSHA256 = vectorPayloadSHA256
				in.ScanID = vectorScanID
			},
			wantCode: walletauth.CodeWalletAuthorizationActionMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			err := walletauth.VerifyAuthorization(in)
			if err == nil {
				t.Fatal("expected error")
			}
			if got := walletauth.Code(err); got != tc.wantCode {
				t.Fatalf("code = %q want %q err=%v", got, tc.wantCode, err)
			}
		})
	}
}

func TestRecoverSignerAddress_mismatch(t *testing.T) {
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	message := vectorMessage(t, issued, expires)
	signature := signPersonalMessage(t, message)

	err := walletauth.VerifyAuthorization(walletauth.VerifyInput{
		Domain:        vectorDomain,
		WalletAddress: "0x0000000000000000000000000000000000000001",
		ChainID:       1,
		ScanID:        vectorScanID,
		PayloadSHA256: vectorPayloadSHA256,
		SignedMessage: message,
		Signature:     signature,
		Now:           issued.Add(time.Minute),
		ClockSkew:     walletauth.DefaultClockSkew,
	})
	if walletauth.Code(err) != walletauth.CodeWalletAuthorizationWalletMismatch {
		t.Fatalf("expected wallet mismatch before signature check, got %v", err)
	}
}
