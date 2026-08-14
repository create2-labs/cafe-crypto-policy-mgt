package config

import (
	"os"
	"strconv"
)

const (
	defaultHTTPAddr                    = ":8082"
	defaultLogLevel                    = "info"
	defaultService                     = "cafe-cpm"
	defaultAuthRequired                = true
	defaultAuthClockSkewSec            = 30
	defaultDiscoveryHTTPTimeoutSec     = 5
	defaultSessionValidationTimeoutSec = 3
	defaultScanAuthorizationTimeoutSec = 3
	defaultPolicyTemplatePaths         = "/app/policy/crypto_policy_template_pq_account_validation_v1.json"
	defaultPolicyInstancePaths         = "/app/policy/crypto_policy_instance_pq_account_validation_v1.json"
	defaultProviderManifestPaths       = "/app/policy/provider_manifest_nicetry_v0_1.json"
	defaultCPMStore                    = "persistence"
	defaultPersistenceTimeoutSec       = 15
)

type Config struct {
	ServiceName                   string
	HTTPAddr                      string
	LogLevel                      string
	AuthRequired                  bool
	SessionValidationURL          string
	SessionValidationTimeoutSec   int
	SessionValidationServiceToken string
	ScanAuthorizationURL          string
	ScanAuthorizationTimeoutSec   int
	ScanAuthorizationServiceToken string
	AuthClockSkewSec              int
	// DiscoveryHTTPBaseURL is the Discovery service origin for server-side GET /discovery/v1/... (PR13g).
	DiscoveryHTTPBaseURL    string
	DiscoveryHTTPTimeoutSec int
	// NATSURL enables publishing policy.assessment.requested from POST …/policies/assessment/request (PR13g).
	NATSURL             string
	PolicyTemplatePaths []string
	PolicyInstancePaths []string
	// ProviderManifestPaths lists ProviderManifest JSON files (CPM_PROVIDER_MANIFEST_PATHS).
	// Loaded into the explore registry for ADR §7 hard compatibility (CPM-P4).
	ProviderManifestPaths []string
	// WalletAuthDomain is embedded in CP-PERSIST canonical messages (§12); falls back to request Host.
	WalletAuthDomain string
	// Store selects CP storage backend. Runtime (deployed images): persistence only.
	// CPM_STORE=memory is compiled only with -tags dev for unit/handler tests — not a runtime mode.
	Store string
	// PersistenceURL is the cafe-persistence service origin (e.g. http://cafe-persistence:8082).
	PersistenceURL string
	// PersistenceTimeoutSec bounds HTTP calls to cafe-persistence (ADR §5.5).
	PersistenceTimeoutSec int
	// PersistenceServiceToken is the CAFE_PERSISTENCE_SERVICE_TOKEN bearer for internal/cp/v1.
	PersistenceServiceToken string
}

func LoadFromEnv() Config {
	return Config{
		ServiceName:                   getEnv("CPM_SERVICE_NAME", defaultService),
		HTTPAddr:                      getEnv("CPM_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:                      getEnv("CPM_LOG_LEVEL", defaultLogLevel),
		AuthRequired:                  getEnvBool("CPM_AUTH_REQUIRED", defaultAuthRequired),
		SessionValidationURL:          getEnv("CAFE_SESSION_JWT_VALIDATION_URL", ""),
		SessionValidationTimeoutSec:   getEnvInt("CAFE_SESSION_JWT_VALIDATION_TIMEOUT_SEC", defaultSessionValidationTimeoutSec),
		SessionValidationServiceToken: getEnv("CAFE_SESSION_JWT_VALIDATION_SERVICE_TOKEN", ""),
		ScanAuthorizationURL:          getEnv("CAFE_SCAN_AUTHORIZATION_URL", ""),
		ScanAuthorizationTimeoutSec:   getEnvInt("CAFE_SCAN_AUTHORIZATION_TIMEOUT_SEC", defaultScanAuthorizationTimeoutSec),
		ScanAuthorizationServiceToken: getEnv("CAFE_SCAN_AUTHORIZATION_SERVICE_TOKEN", ""),
		AuthClockSkewSec:              getEnvInt("CPM_AUTH_CLOCK_SKEW_SEC", defaultAuthClockSkewSec),
		DiscoveryHTTPBaseURL:          getEnv("CAFE_DISCOVERY_HTTP_BASE", ""),
		DiscoveryHTTPTimeoutSec:       getEnvInt("CAFE_DISCOVERY_HTTP_TIMEOUT_SEC", defaultDiscoveryHTTPTimeoutSec),
		NATSURL:                       getEnv("CPM_NATS_URL", ""),
		PolicyTemplatePaths:           parseCommaList(getEnv("CPM_POLICY_TEMPLATE_PATHS", defaultPolicyTemplatePaths)),
		PolicyInstancePaths:           parseCommaList(getEnv("CPM_POLICY_INSTANCE_PATHS", defaultPolicyInstancePaths)),
		ProviderManifestPaths:         parseCommaList(getEnv("CPM_PROVIDER_MANIFEST_PATHS", defaultProviderManifestPaths)),
		WalletAuthDomain:              getEnv("CPM_WALLET_AUTH_DOMAIN", ""),
		Store:                         getEnv("CPM_STORE", defaultCPMStore),
		PersistenceURL:                getEnv("CPM_PERSISTENCE_URL", ""),
		PersistenceTimeoutSec:         getEnvInt("CPM_PERSISTENCE_TIMEOUT_SEC", defaultPersistenceTimeoutSec),
		PersistenceServiceToken:       getEnv("CAFE_PERSISTENCE_SERVICE_TOKEN", ""),
	}
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func parseCommaList(value string) []string {
	if value == "" {
		return nil
	}
	out := make([]string, 0, 1)
	current := make([]rune, 0, len(value))
	flush := func() {
		if len(current) == 0 {
			return
		}
		out = append(out, string(current))
		current = current[:0]
	}
	for _, r := range value {
		if r == ',' {
			flush()
			continue
		}
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			current = append(current, r)
		}
	}
	flush()
	if len(out) == 0 {
		return nil
	}
	return out
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
