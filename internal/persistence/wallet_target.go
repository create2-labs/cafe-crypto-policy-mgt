package persistence

import (
	"errors"
	"strings"
)

// WalletSubjectPrefix is the canonical NATS / assessment subject prefix for EVM wallet targets.
const WalletSubjectPrefix = "wallet:"

// WalletTargetContextCounts is the minimal IMM-9b lookup result for a normalized wallet target_address.
type WalletTargetContextCounts struct {
	Exists          bool
	PolicyCount     int
	DraftCount      int
	PlatformDraftID string // set when DraftCount == 1 (owner GET /drafts?id= for W1 UI)
}

// NormalizeWalletTargetAddress applies the same normalization as Discovery wallet scans (0x + lowercase).
func NormalizeWalletTargetAddress(address string) (string, error) {
	a := strings.TrimSpace(address)
	if a == "" {
		return "", errors.New("target_address is required")
	}
	if strings.HasPrefix(a, "0X") {
		a = "0x" + a[2:]
	}
	if !strings.HasPrefix(a, "0x") {
		a = "0x" + a
	}
	a = strings.ToLower(a)
	if len(a) != 42 || !strings.HasPrefix(a, "0x") {
		return "", errors.New("target_address must be a normalized EVM address")
	}
	for _, c := range a[2:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", errors.New("target_address must be a normalized EVM address")
		}
	}
	return a, nil
}

// WalletSubjectIDFromAddress returns wallet:0x<40 lowercase hex> for a bare EVM address (Discovery detail, assessment HTTP).
func WalletSubjectIDFromAddress(address string) (string, error) {
	norm, err := NormalizeWalletTargetAddress(address)
	if err != nil {
		return "", err
	}
	return WalletSubjectPrefix + norm, nil
}

// NormalizeWalletSubjectID canonicalizes wallet:0x… or bare hex subject IDs (NATS assessment consumer).
// Non-wallet subject IDs are returned unchanged when hex normalization fails (legacy pass-through).
func NormalizeWalletSubjectID(subjectID string) string {
	value := strings.TrimSpace(subjectID)
	if value == "" {
		return value
	}
	lower := strings.ToLower(value)
	hexPart := value
	hasWalletPrefix := strings.HasPrefix(lower, WalletSubjectPrefix)
	if hasWalletPrefix {
		hexPart = strings.TrimSpace(value[len(WalletSubjectPrefix):])
	}
	norm, err := NormalizeWalletTargetAddress(hexPart)
	if err != nil {
		return value
	}
	return WalletSubjectPrefix + norm
}
