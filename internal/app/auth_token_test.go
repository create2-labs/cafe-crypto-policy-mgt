package app

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestParseAndValidateDiscoveryToken(t *testing.T) {
	valid, err := makeTokenEnvelope(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make token: %v", err)
	}
	if _, err := parseAndValidateDiscoveryToken(valid, 30); err != nil {
		t.Fatalf("expected valid discovery token, got: %v", err)
	}

	expired, err := makeTokenEnvelope(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(-2 * time.Hour).Unix(),
	}, []string{"EdDSA", "ML-DSA-65"})
	if err != nil {
		t.Fatalf("make expired token: %v", err)
	}
	if _, err := parseAndValidateDiscoveryToken(expired, 30); err == nil {
		t.Fatal("expected expired token to fail validation")
	}

	wrongAlg, err := makeTokenEnvelope(map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}, []string{"HS256"})
	if err != nil {
		t.Fatalf("make wrong alg token: %v", err)
	}
	if _, err := parseAndValidateDiscoveryToken(wrongAlg, 30); err == nil {
		t.Fatal("expected token with wrong algorithm set to fail")
	}
}

func makeTokenEnvelope(claims map[string]any, algorithms []string) (string, error) {
	payloadRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadRaw)
	signatures := make([]map[string]string, 0, len(algorithms))
	for _, alg := range algorithms {
		headerRaw, err := json.Marshal(map[string]any{"alg": alg, "typ": "JWT"})
		if err != nil {
			return "", err
		}
		signatures = append(signatures, map[string]string{
			"protected": base64.RawURLEncoding.EncodeToString(headerRaw),
			"signature": base64.RawURLEncoding.EncodeToString([]byte("sig")),
		})
	}
	envelopeRaw, err := json.Marshal(map[string]any{
		"payload":    payloadB64,
		"signatures": signatures,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(envelopeRaw), nil
}
