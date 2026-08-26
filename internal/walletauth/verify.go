package walletauth

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifyInput is the expected binding set for persist-time authorization checks.
// PayloadSHA256 is required (normative RD-P4+ message binding).
type VerifyInput struct {
	Domain        string
	WalletAddress string
	ChainID       int64
	ScanID        string
	PayloadSHA256 string
	SignedMessage string
	Signature     string
	Now           time.Time
	ClockSkew     time.Duration
}

// VerifyAuthorization validates signed_message freshness, bindings and EIP-191 signature.
func VerifyAuthorization(in VerifyInput) error {
	if strings.TrimSpace(in.SignedMessage) == "" || strings.TrimSpace(in.Signature) == "" {
		return verificationError(CodeWalletControlProofRequired, "signed wallet authorization is required")
	}

	parsed, err := ParseMessage(in.SignedMessage)
	if err != nil {
		return verificationError(CodeInvalidWalletSignature, "signed message is not a valid canonical authorization message")
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	skew := in.ClockSkew
	if skew <= 0 {
		skew = DefaultClockSkew
	}

	if err := validateFreshness(parsed.IssuedAt, parsed.ExpiresAt, now, skew); err != nil {
		return err
	}

	if err := compareBindings(parsed, in); err != nil {
		return err
	}

	expected := ExpectedMessageFromFields(Fields{
		Domain:        in.Domain,
		Action:        ActionPersistCryptoPolicy,
		WalletAddress: normalizedWalletForMessage(in.WalletAddress),
		ChainID:       in.ChainID,
		ScanID:        strings.TrimSpace(in.ScanID),
		PayloadSHA256: strings.TrimSpace(in.PayloadSHA256),
		IssuedAt:      parsed.IssuedAt,
		ExpiresAt:     parsed.ExpiresAt,
	})
	if in.SignedMessage != expected {
		return verificationError(CodeInvalidWalletSignature, "signed_message does not match the canonical authorization message")
	}

	recovered, err := RecoverSignerAddress(in.SignedMessage, in.Signature)
	if err != nil {
		return verificationError(CodeInvalidWalletSignature, "wallet signature is invalid")
	}
	wantWallet, err := persistence.NormalizeWalletTargetAddress(in.WalletAddress)
	if err != nil {
		return verificationError(CodeWalletAuthorizationWalletMismatch, "wallet address is invalid")
	}
	if !addressesEqual(recovered, wantWallet) {
		return verificationError(CodeWalletSignatureAddressMismatch, "recovered signer does not match wallet_address")
	}
	return nil
}

func validateFreshness(issuedAt, expiresAt, now time.Time, skew time.Duration) error {
	if expiresAt.Sub(issuedAt) > MaxValidityWindow {
		return verificationError(CodeWalletAuthorizationValidityLong, "signed authorization validity window exceeds 10 minutes")
	}
	if now.After(expiresAt) {
		return verificationError(CodeWalletAuthorizationExpired, "signed authorization has expired")
	}
	if issuedAt.After(now.Add(skew)) {
		return verificationError(CodeWalletAuthorizationNotYetValid, "signed authorization is not yet valid")
	}
	return nil
}

func compareBindings(parsed Fields, in VerifyInput) error {
	if parsed.Action != ActionPersistCryptoPolicy {
		return verificationError(CodeWalletAuthorizationActionMismatch, "signed authorization action mismatch")
	}
	wantSHA := strings.TrimSpace(in.PayloadSHA256)
	gotSHA := strings.TrimSpace(parsed.PayloadSHA256)
	if wantSHA == "" || gotSHA == "" || !strings.EqualFold(wantSHA, gotSHA) {
		return verificationError(CodePayloadSHA256Mismatch, "signed authorization payload_sha256 mismatch")
	}
	if strings.TrimSpace(parsed.ScanID) != strings.TrimSpace(in.ScanID) {
		return verificationError(CodeWalletAuthorizationScanMismatch, "signed authorization scan mismatch")
	}
	if parsed.ChainID != in.ChainID {
		return verificationError(CodeWalletAuthorizationChainMismatch, "signed authorization chain mismatch")
	}
	wantWallet, err := persistence.NormalizeWalletTargetAddress(in.WalletAddress)
	if err != nil {
		return verificationError(CodeWalletAuthorizationWalletMismatch, "wallet address is invalid")
	}
	gotWallet, err := persistence.NormalizeWalletTargetAddress(parsed.WalletAddress)
	if err != nil {
		return verificationError(CodeWalletAuthorizationWalletMismatch, "signed authorization wallet mismatch")
	}
	if !addressesEqual(gotWallet, wantWallet) {
		return verificationError(CodeWalletAuthorizationWalletMismatch, "signed authorization wallet mismatch")
	}
	return nil
}

// RecoverSignerAddress verifies an EIP-191 / personal_sign signature and returns the normalized EOA address.
func RecoverSignerAddress(message, signature string) (string, error) {
	sig, err := decodeSignature(signature)
	if err != nil {
		return "", err
	}
	hash := accounts.TextHash([]byte(message))
	if len(sig) != crypto.SignatureLength {
		return "", fmt.Errorf("invalid signature length")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	pub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return "", err
	}
	addr := crypto.PubkeyToAddress(*pub)
	return persistence.NormalizeWalletTargetAddress(addr.Hex())
}

func decodeSignature(signature string) ([]byte, error) {
	raw := strings.TrimSpace(signature)
	if raw == "" {
		return nil, fmt.Errorf("signature is required")
	}
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		raw = raw[2:]
	}
	sig, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("signature must be hex-encoded")
	}
	if len(sig) != crypto.SignatureLength {
		return nil, fmt.Errorf("signature must be 65 bytes")
	}
	return sig, nil
}

func normalizedWalletForMessage(address string) string {
	norm, err := persistence.NormalizeWalletTargetAddress(address)
	if err != nil {
		return strings.TrimSpace(address)
	}
	return norm
}

func addressesEqual(a, b string) bool {
	na, errA := persistence.NormalizeWalletTargetAddress(a)
	nb, errB := persistence.NormalizeWalletTargetAddress(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return na == nb
}
