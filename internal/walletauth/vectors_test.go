package walletauth_test

import (
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/walletauth"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

// Public deterministic test vector (private key used only in tests).
const (
	vectorPrivateKeyHex = "c87509a1a067e1eb07e3bcb3d0d47c41102a221097c02183ccac2fdaba05632c"
	vectorWalletAddress = "0xee387b44819eb54d7fff026a18229421738a8a24"
	vectorDomain        = "api.example.com"
	vectorScanID        = "550e8400-e29b-41d4-a716-446655440000"
	vectorDraftID       = "550e8400-e29b-41d4-a716-446655440001"
)

func signPersonalMessage(t *testing.T, message string) string {
	t.Helper()
	privKey, err := crypto.HexToECDSA(vectorPrivateKeyHex)
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	hash := accounts.TextHash([]byte(message))
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return walletauth.NormalizeSignatureHex(sig)
}

func vectorMessage(t *testing.T, issued, expires time.Time) string {
	t.Helper()
	return walletauth.BuildMessage(walletauth.Fields{
		Domain:        vectorDomain,
		Action:        walletauth.ActionPersistCryptoPolicy,
		WalletAddress: vectorWalletAddress,
		ChainID:       1,
		ScanID:        vectorScanID,
		DraftID:       vectorDraftID,
		IssuedAt:      issued,
		ExpiresAt:     expires,
	})
}

func TestDeterministicSignatureVector(t *testing.T) {
	t.Parallel()
	issued := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	expires := issued.Add(10 * time.Minute)
	message := vectorMessage(t, issued, expires)
	signature := signPersonalMessage(t, message)

	recovered, err := walletauth.RecoverSignerAddress(message, signature)
	if err != nil {
		t.Fatalf("RecoverSignerAddress: %v", err)
	}
	if recovered != vectorWalletAddress {
		t.Fatalf("recovered = %q want %q", recovered, vectorWalletAddress)
	}

	err = walletauth.VerifyAuthorization(walletauth.VerifyInput{
		Domain:        vectorDomain,
		WalletAddress: vectorWalletAddress,
		ChainID:       1,
		ScanID:        vectorScanID,
		DraftID:       vectorDraftID,
		SignedMessage: message,
		Signature:     signature,
		Now:           issued.Add(30 * time.Second),
		ClockSkew:     walletauth.DefaultClockSkew,
	})
	if err != nil {
		t.Fatalf("VerifyAuthorization: %v", err)
	}
}
