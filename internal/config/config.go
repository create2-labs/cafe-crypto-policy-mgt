package config

import (
	"os"
	"strconv"
)

const (
	defaultHTTPAddr                    = ":8080"
	defaultLogLevel                    = "info"
	defaultService                     = "cafe-cpm"
	defaultAuthRequired                = true
	defaultAuthClockSkewSec            = 30
	defaultSessionValidationTimeoutSec = 3
	defaultScanAuthorizationTimeoutSec = 3
	defaultPolicyCatalogPath           = "/app/policy/policy_graph_catalog_valid.json"
	defaultPolicyTemplatePaths         = "/app/policy/crypto_policy_template_valid.json"
	defaultPolicyInstancePaths         = "/app/policy/crypto_policy_instance_valid.json"
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
	PolicyCatalogPath             string
	PolicyTemplatePaths           []string
	PolicyInstancePaths           []string
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
		PolicyCatalogPath:             getEnv("CPM_POLICY_CATALOG_PATH", defaultPolicyCatalogPath),
		PolicyTemplatePaths:           parseCommaList(getEnv("CPM_POLICY_TEMPLATE_PATHS", defaultPolicyTemplatePaths)),
		PolicyInstancePaths:           parseCommaList(getEnv("CPM_POLICY_INSTANCE_PATHS", defaultPolicyInstancePaths)),
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
