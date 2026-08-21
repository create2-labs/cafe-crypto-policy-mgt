package policy

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

type captureLogger struct {
	lines []string
}

func (c *captureLogger) Printf(format string, v ...any) {
	c.lines = append(c.lines, fmt.Sprintf(format, v...))
}

func TestCheckPostureOrphanage_emptyAllowedProviders(t *testing.T) {
	logger := &captureLogger{}
	cp := &CryptoPolicy{
		ID:              "cp_empty",
		RequiredPosture: vocabulary.PQPostureHybrid,
	}
	n := CheckPostureOrphanage([]*CryptoPolicy{cp}, nil, logger)
	if n != 1 {
		t.Fatalf("want 1 orphan, got %d", n)
	}
	if len(logger.lines) != 1 || !strings.Contains(logger.lines[0], "WARN catalogue: posture orphanage") {
		t.Fatalf("want WARN orphanage log, got %#v", logger.lines)
	}
}

func TestCheckPostureOrphanage_noMatchingPosture(t *testing.T) {
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	logger := &captureLogger{}
	cp := &CryptoPolicy{
		ID:               "cp_full_pq",
		RequiredPosture:  vocabulary.PQPostureFullPQ,
		AllowedProviders: []string{"nicetry"},
	}
	n := CheckPostureOrphanage([]*CryptoPolicy{cp}, reg, logger)
	if n != 1 {
		t.Fatalf("want 1 orphan (hybrid != full_pq), got %d logs=%v", n, logger.lines)
	}
}

func TestCheckPostureOrphanage_noFalseAlertWhenOnlyPlannedChain(t *testing.T) {
	// Nicetry fixture: resulting_posture=hybrid matches CP; mainnet is planned-only.
	// Static orphanage must NOT fire (ADR §7.2.1 / CPM-P11a merge criterion).
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	cp, err := LoadCryptoPolicyFromFile(filepath.Join("testdata", "crypto_policy_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("LoadCryptoPolicyFromFile: %v", err)
	}
	logger := &captureLogger{}
	n := CheckPostureOrphanage([]*CryptoPolicy{cp}, reg, logger)
	if n != 0 {
		t.Fatalf("want no orphan for posture-OK + planned chain, got %d logs=%v", n, logger.lines)
	}
}

func TestCheckPostureOrphanage_unknownProvider(t *testing.T) {
	reg, err := provider.LoadRegistryFromFiles([]string{
		filepath.Join("..", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
	})
	if err != nil {
		t.Fatalf("LoadRegistryFromFiles: %v", err)
	}
	logger := &captureLogger{}
	cp := &CryptoPolicy{
		ID:               "cp_missing_prov",
		RequiredPosture:  vocabulary.PQPostureHybrid,
		AllowedProviders: []string{"does-not-exist"},
	}
	n := CheckPostureOrphanage([]*CryptoPolicy{cp}, reg, logger)
	if n != 1 {
		t.Fatalf("want 1 orphan for missing provider, got %d", n)
	}
}
