package policy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/vocabulary"
)

func TestLoadCryptoPolicyFromFile_Valid(t *testing.T) {
	cp, err := LoadCryptoPolicyFromFile(filepath.Join("testdata", "crypto_policy_pq_account_validation_v1.json"))
	if err != nil {
		t.Fatalf("LoadCryptoPolicyFromFile: %v", err)
	}

	if cp.ID != "cpm_pq_account_validation_v1" {
		t.Fatalf("id: got %q", cp.ID)
	}
	if cp.RequiredPosture != vocabulary.PQPostureHybrid {
		t.Fatalf("required_posture: got %q", cp.RequiredPosture)
	}
	if !reflect.DeepEqual(cp.AllowedProviders, []string{"nicetry"}) {
		t.Fatalf("allowed_providers: %#v", cp.AllowedProviders)
	}
}

func TestCryptoPolicy_Validate_AllowedProvidersRequired(t *testing.T) {
	cp := &CryptoPolicy{
		ID:              "cpm_empty_providers",
		Name:            "Empty providers",
		Version:         "v0.1",
		RequiredPosture: vocabulary.PQPostureHybrid,
	}
	if err := cp.Validate(); !errors.Is(err, ErrCryptoPolicyAllowedProvidersRequired) {
		t.Fatalf("error = %v, want %v", err, ErrCryptoPolicyAllowedProvidersRequired)
	}
}

func TestCryptoPolicy_Validate_RequiredPostureInvalid(t *testing.T) {
	cp := &CryptoPolicy{
		ID:               "cpm_bad_posture",
		Name:             "Bad posture",
		Version:          "v0.1",
		RequiredPosture:  vocabulary.PQPostureUnknown,
		AllowedProviders: []string{"nicetry"},
	}
	if err := cp.Validate(); !errors.Is(err, ErrCryptoPolicyRequiredPostureInvalid) {
		t.Fatalf("error = %v, want %v", err, ErrCryptoPolicyRequiredPostureInvalid)
	}
}
