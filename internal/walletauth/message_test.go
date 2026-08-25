package walletauth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

func TestBuildMessage_roundTrip(t *testing.T) {
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	fields := walletauth.Fields{
		Domain:        "api.example.com",
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
		ChainID:       1,
		ScanID:        "550e8400-e29b-41d4-a716-446655440000",
		DraftID:       "550e8400-e29b-41d4-a716-446655440001",
		IssuedAt:      issued,
		ExpiresAt:     expires,
	}
	message := walletauth.BuildMessage(fields)
	if !strings.Contains(message, "Domain: api.example.com") {
		t.Fatalf("expected domain in message: %q", message)
	}
	if !strings.Contains(message, "Draft ID: "+fields.DraftID) {
		t.Fatalf("expected legacy Draft ID in message: %q", message)
	}
	if strings.Contains(message, "user_id") || strings.Contains(message, "tenant_id") {
		t.Fatalf("canonical message must not include user or tenant bindings: %q", message)
	}

	parsed, err := walletauth.ParseMessage(message)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if parsed.Domain != fields.Domain || parsed.Action != fields.Action {
		t.Fatalf("parsed metadata mismatch: %#v", parsed)
	}
	if parsed.ChainID != fields.ChainID || parsed.DraftID != fields.DraftID {
		t.Fatalf("parsed ids mismatch: %#v", parsed)
	}
}

func TestBuildMessage_payloadSHA256RoundTrip(t *testing.T) {
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	fields := walletauth.Fields{
		Domain:        "api.example.com",
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
		ChainID:       1,
		ScanID:        "550e8400-e29b-41d4-a716-446655440000",
		PayloadSHA256: sha,
		IssuedAt:      issued,
		ExpiresAt:     expires,
	}
	message := walletauth.BuildMessage(fields)
	if strings.Contains(message, "Draft ID:") {
		t.Fatalf("payload binding must not emit Draft ID: %q", message)
	}
	if !strings.Contains(message, "Payload SHA-256: "+sha) {
		t.Fatalf("expected Payload SHA-256 line: %q", message)
	}
	parsed, err := walletauth.ParseMessage(message)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if parsed.PayloadSHA256 != sha || parsed.DraftID != "" {
		t.Fatalf("parsed payload binding mismatch: %#v", parsed)
	}
}
