package walletauth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
)

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
	if !strings.Contains(message, "Domain: api.example.com") {
		t.Fatalf("expected domain in message: %q", message)
	}
	if strings.Contains(message, "Draft ID:") {
		t.Fatalf("payload binding must not emit Draft ID: %q", message)
	}
	if !strings.Contains(message, "Payload SHA-256: "+sha) {
		t.Fatalf("expected Payload SHA-256 line: %q", message)
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
	if parsed.ChainID != fields.ChainID || parsed.PayloadSHA256 != sha {
		t.Fatalf("parsed ids mismatch: %#v", parsed)
	}
}

func TestParseMessage_rejectsLegacyDraftID(t *testing.T) {
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	legacy := strings.Join([]string{
		"CAFE Crypto Policy Persistence",
		"",
		"Domain: api.example.com",
		"Action: persist_crypto_policy",
		"Wallet: 0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
		"Chain ID: 1",
		"Scan ID: 550e8400-e29b-41d4-a716-446655440000",
		"Draft ID: 550e8400-e29b-41d4-a716-446655440001",
		"Issued At: " + issued.Format(time.RFC3339),
		"Expiration Time: " + expires.Format(time.RFC3339),
		"",
		"By signing this message, I prove control of the wallet and authorize CAFE to persist this Crypto Policy for this wallet.",
	}, "\n")
	if _, err := walletauth.ParseMessage(legacy); err == nil {
		t.Fatal("expected legacy Draft ID message to be rejected")
	}
}
