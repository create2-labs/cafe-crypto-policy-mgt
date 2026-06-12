package walletauth

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Fields are the signed-message bindings for CP-PERSIST V1 (§12).
// user_id and tenant_id are intentionally excluded.
type Fields struct {
	Domain        string
	Action        string
	WalletAddress string
	ChainID       int64
	ScanID        string
	DraftID       string
	IssuedAt      time.Time
	ExpiresAt     time.Time
}

// BuildMessage renders the canonical authorization message per CP_PERSIST.md §12.
func BuildMessage(fields Fields) string {
	issued := fields.IssuedAt.UTC().Format(time.RFC3339)
	expires := fields.ExpiresAt.UTC().Format(time.RFC3339)
	return strings.Join([]string{
		messageTitle,
		"",
		"Domain: " + strings.TrimSpace(fields.Domain),
		"Action: " + strings.TrimSpace(fields.Action),
		"Wallet: " + strings.TrimSpace(fields.WalletAddress),
		"Chain ID: " + strconv.FormatInt(fields.ChainID, 10),
		"Scan ID: " + strings.TrimSpace(fields.ScanID),
		"Draft ID: " + strings.TrimSpace(fields.DraftID),
		"Issued At: " + issued,
		"Expiration Time: " + expires,
		"",
		messageFooter,
	}, "\n")
}

// ParseMessage extracts signed-message bindings from a canonical authorization message.
func ParseMessage(message string) (Fields, error) {
	lines := strings.Split(message, "\n")
	if len(lines) < 11 {
		return Fields{}, fmt.Errorf("canonical message is too short")
	}
	if strings.TrimSpace(lines[0]) != messageTitle {
		return Fields{}, fmt.Errorf("canonical message title mismatch")
	}
	if strings.TrimSpace(lines[len(lines)-1]) != messageFooter {
		return Fields{}, fmt.Errorf("canonical message footer mismatch")
	}

	readField := func(prefix string) (string, error) {
		for _, line := range lines {
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix)), nil
			}
		}
		return "", fmt.Errorf("missing %s", strings.TrimSuffix(prefix, ": "))
	}

	domain, err := readField("Domain: ")
	if err != nil {
		return Fields{}, err
	}
	action, err := readField("Action: ")
	if err != nil {
		return Fields{}, err
	}
	wallet, err := readField("Wallet: ")
	if err != nil {
		return Fields{}, err
	}
	chainRaw, err := readField("Chain ID: ")
	if err != nil {
		return Fields{}, err
	}
	chainID, err := strconv.ParseInt(chainRaw, 10, 64)
	if err != nil || chainID < 1 {
		return Fields{}, fmt.Errorf("invalid chain id in canonical message")
	}
	scanID, err := readField("Scan ID: ")
	if err != nil {
		return Fields{}, err
	}
	draftID, err := readField("Draft ID: ")
	if err != nil {
		return Fields{}, err
	}
	issuedRaw, err := readField("Issued At: ")
	if err != nil {
		return Fields{}, err
	}
	expiresRaw, err := readField("Expiration Time: ")
	if err != nil {
		return Fields{}, err
	}
	issuedAt, err := time.Parse(time.RFC3339, issuedRaw)
	if err != nil {
		return Fields{}, fmt.Errorf("invalid issued_at in canonical message")
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresRaw)
	if err != nil {
		return Fields{}, fmt.Errorf("invalid expires_at in canonical message")
	}

	return Fields{
		Domain:        domain,
		Action:        action,
		WalletAddress: wallet,
		ChainID:       chainID,
		ScanID:        scanID,
		DraftID:       draftID,
		IssuedAt:      issuedAt.UTC(),
		ExpiresAt:     expiresAt.UTC(),
	}, nil
}

// ExpectedMessage rebuilds the canonical message for the same bindings and validity window.
func ExpectedMessage(domain string, walletAddress string, chainID int64, scanID, draftID string, issuedAt, expiresAt time.Time) string {
	return BuildMessage(Fields{
		Domain:        domain,
		Action:        ActionPersistCryptoPolicy,
		WalletAddress: walletAddress,
		ChainID:       chainID,
		ScanID:        scanID,
		DraftID:       draftID,
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
	})
}
