package app

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
)

func newAuthedTestHandler(t *testing.T) http.Handler {
	t.Helper()
	store, err := api.LoadReadStore(api.ReadStoreOptions{
		CryptoPolicyPaths: []string{
			filepath.Join("..", "domain", "policy", "testdata", "crypto_policy_pq_account_validation_v1.json"),
		},
		ProviderManifestPaths: []string{
			filepath.Join("..", "domain", "provider", "testdata", "provider_manifest_nicetry_v0_1.json"),
		},
	})
	if err != nil {
		t.Fatalf("LoadReadStore: %v", err)
	}
	introspect := newDiscoveryValidationServer(t, discoveryValidationTestConfig{status: http.StatusOK})
	t.Cleanup(introspect.Close)
	h, err := testHandler(store, nil, authConfig{
		Required:                    true,
		SessionValidationURL:        introspect.URL,
		SessionValidationTimeoutSec: 3,
		ClockSkewSec:                30,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return h
}

func mustToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := makeDiscoveryHybridToken(map[string]any{
		"user_id": userID,
		"email":   userID + "@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	return token
}
