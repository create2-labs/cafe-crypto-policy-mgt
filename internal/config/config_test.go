package config

import "testing"

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Setenv("CPM_SERVICE_NAME", "")
	t.Setenv("CPM_HTTP_ADDR", "")
	t.Setenv("CPM_LOG_LEVEL", "")

	cfg := LoadFromEnv()

	if cfg.ServiceName != defaultService {
		t.Fatalf("expected default service name %q, got %q", defaultService, cfg.ServiceName)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected default HTTP addr %q, got %q", defaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("expected default log level %q, got %q", defaultLogLevel, cfg.LogLevel)
	}
	if cfg.AuthRequired != defaultAuthRequired {
		t.Fatalf("expected default auth required %v, got %v", defaultAuthRequired, cfg.AuthRequired)
	}
	if cfg.AuthClockSkewSec != defaultAuthClockSkewSec {
		t.Fatalf("expected default auth clock skew %d, got %d", defaultAuthClockSkewSec, cfg.AuthClockSkewSec)
	}
	if cfg.SessionValidationURL != "" {
		t.Fatalf("expected empty session validation URL default, got %q", cfg.SessionValidationURL)
	}
	if cfg.SessionValidationTimeoutSec != defaultSessionValidationTimeoutSec {
		t.Fatalf("expected default session validation timeout %d, got %d", defaultSessionValidationTimeoutSec, cfg.SessionValidationTimeoutSec)
	}
	if cfg.SessionValidationServiceToken != "" {
		t.Fatalf("expected empty session validation service token default, got %q", cfg.SessionValidationServiceToken)
	}
	if cfg.ScanAuthorizationURL != "" {
		t.Fatalf("expected empty scan authorization URL default, got %q", cfg.ScanAuthorizationURL)
	}
	if cfg.ScanAuthorizationTimeoutSec != defaultScanAuthorizationTimeoutSec {
		t.Fatalf("expected default scan authorization timeout %d, got %d", defaultScanAuthorizationTimeoutSec, cfg.ScanAuthorizationTimeoutSec)
	}
	if cfg.ScanAuthorizationServiceToken != "" {
		t.Fatalf("expected empty scan authorization service token default, got %q", cfg.ScanAuthorizationServiceToken)
	}
	if cfg.PolicyCatalogPath != defaultPolicyCatalogPath {
		t.Fatalf("expected default policy catalog path %q, got %q", defaultPolicyCatalogPath, cfg.PolicyCatalogPath)
	}
	if len(cfg.PolicyTemplatePaths) != 2 {
		t.Fatalf("expected 2 default policy template paths, got %#v", cfg.PolicyTemplatePaths)
	}
	if cfg.PolicyTemplatePaths[0] != "/app/policy/crypto_policy_template_valid.json" ||
		cfg.PolicyTemplatePaths[1] != "/app/policy/crypto_policy_template_pq_ready_progressive.json" {
		t.Fatalf("unexpected default policy template paths: %#v", cfg.PolicyTemplatePaths)
	}
	if len(cfg.PolicyInstancePaths) != 2 {
		t.Fatalf("expected 2 default policy instance paths, got %#v", cfg.PolicyInstancePaths)
	}
	if cfg.PolicyInstancePaths[0] != "/app/policy/crypto_policy_instance_valid.json" ||
		cfg.PolicyInstancePaths[1] != "/app/policy/crypto_policy_instance_pq_ready_progressive.json" {
		t.Fatalf("unexpected default policy instance paths: %#v", cfg.PolicyInstancePaths)
	}
	if cfg.DiscoveryHTTPBaseURL != "" {
		t.Fatalf("expected empty discovery HTTP base default, got %q", cfg.DiscoveryHTTPBaseURL)
	}
	if cfg.DiscoveryHTTPTimeoutSec != defaultDiscoveryHTTPTimeoutSec {
		t.Fatalf("expected default discovery HTTP timeout %d, got %d", defaultDiscoveryHTTPTimeoutSec, cfg.DiscoveryHTTPTimeoutSec)
	}
	if cfg.NATSURL != "" {
		t.Fatalf("expected empty NATS URL default, got %q", cfg.NATSURL)
	}
	if cfg.Store != defaultCPMStore {
		t.Fatalf("expected default store %q, got %q", defaultCPMStore, cfg.Store)
	}
	if cfg.PersistenceURL != "" {
		t.Fatalf("expected empty persistence URL default, got %q", cfg.PersistenceURL)
	}
	if cfg.PersistenceTimeoutSec != defaultPersistenceTimeoutSec {
		t.Fatalf("expected default persistence timeout %d, got %d", defaultPersistenceTimeoutSec, cfg.PersistenceTimeoutSec)
	}
	if cfg.PersistenceServiceToken != "" {
		t.Fatalf("expected empty persistence service token default, got %q", cfg.PersistenceServiceToken)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("CPM_SERVICE_NAME", "custom-cpm")
	t.Setenv("CPM_HTTP_ADDR", ":9090")
	t.Setenv("CPM_LOG_LEVEL", "debug")
	t.Setenv("CPM_AUTH_REQUIRED", "false")
	t.Setenv("CAFE_SESSION_JWT_VALIDATION_URL", "http://discovery:8080/internal/auth/session/validate")
	t.Setenv("CAFE_SESSION_JWT_VALIDATION_TIMEOUT_SEC", "7")
	t.Setenv("CAFE_SESSION_JWT_VALIDATION_SERVICE_TOKEN", "service-token")
	t.Setenv("CAFE_SCAN_AUTHORIZATION_URL", "http://discovery:8080/internal/authz/scans")
	t.Setenv("CAFE_SCAN_AUTHORIZATION_TIMEOUT_SEC", "9")
	t.Setenv("CAFE_SCAN_AUTHORIZATION_SERVICE_TOKEN", "scan-authz-service-token")
	t.Setenv("CPM_AUTH_CLOCK_SKEW_SEC", "45")
	t.Setenv("CPM_POLICY_CATALOG_PATH", "catalog.json")
	t.Setenv("CPM_POLICY_TEMPLATE_PATHS", "tpl-a.json, tpl-b.json")
	t.Setenv("CPM_POLICY_INSTANCE_PATHS", "inst-a.json,inst-b.json")
	t.Setenv("CAFE_DISCOVERY_HTTP_BASE", "http://discovery:8080")
	t.Setenv("CAFE_DISCOVERY_HTTP_TIMEOUT_SEC", "11")
	t.Setenv("CPM_NATS_URL", "nats://nats:4222")
	t.Setenv("CPM_STORE", "persistence")
	t.Setenv("CPM_PERSISTENCE_URL", "http://persistence:8082")
	t.Setenv("CPM_PERSISTENCE_TIMEOUT_SEC", "20")
	t.Setenv("CAFE_PERSISTENCE_SERVICE_TOKEN", "persist-token")

	cfg := LoadFromEnv()

	if cfg.ServiceName != "custom-cpm" {
		t.Fatalf("expected service name override, got %q", cfg.ServiceName)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected HTTP addr override, got %q", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level override, got %q", cfg.LogLevel)
	}
	if cfg.AuthRequired {
		t.Fatalf("expected auth required override false, got %v", cfg.AuthRequired)
	}
	if cfg.SessionValidationURL != "http://discovery:8080/internal/auth/session/validate" {
		t.Fatalf("expected session validation URL override, got %q", cfg.SessionValidationURL)
	}
	if cfg.SessionValidationTimeoutSec != 7 {
		t.Fatalf("expected session validation timeout override, got %d", cfg.SessionValidationTimeoutSec)
	}
	if cfg.SessionValidationServiceToken != "service-token" {
		t.Fatalf("expected session validation service token override, got %q", cfg.SessionValidationServiceToken)
	}
	if cfg.ScanAuthorizationURL != "http://discovery:8080/internal/authz/scans" {
		t.Fatalf("expected scan authorization URL override, got %q", cfg.ScanAuthorizationURL)
	}
	if cfg.ScanAuthorizationTimeoutSec != 9 {
		t.Fatalf("expected scan authorization timeout override, got %d", cfg.ScanAuthorizationTimeoutSec)
	}
	if cfg.ScanAuthorizationServiceToken != "scan-authz-service-token" {
		t.Fatalf("expected scan authorization service token override, got %q", cfg.ScanAuthorizationServiceToken)
	}
	if cfg.AuthClockSkewSec != 45 {
		t.Fatalf("expected auth clock skew override, got %d", cfg.AuthClockSkewSec)
	}
	if cfg.PolicyCatalogPath != "catalog.json" {
		t.Fatalf("expected policy catalog path override, got %q", cfg.PolicyCatalogPath)
	}
	if len(cfg.PolicyTemplatePaths) != 2 || cfg.PolicyTemplatePaths[0] != "tpl-a.json" || cfg.PolicyTemplatePaths[1] != "tpl-b.json" {
		t.Fatalf("expected policy template paths override, got %#v", cfg.PolicyTemplatePaths)
	}
	if len(cfg.PolicyInstancePaths) != 2 || cfg.PolicyInstancePaths[0] != "inst-a.json" || cfg.PolicyInstancePaths[1] != "inst-b.json" {
		t.Fatalf("expected policy instance paths override, got %#v", cfg.PolicyInstancePaths)
	}
	if cfg.DiscoveryHTTPBaseURL != "http://discovery:8080" {
		t.Fatalf("expected discovery HTTP base override, got %q", cfg.DiscoveryHTTPBaseURL)
	}
	if cfg.DiscoveryHTTPTimeoutSec != 11 {
		t.Fatalf("expected discovery HTTP timeout override, got %d", cfg.DiscoveryHTTPTimeoutSec)
	}
	if cfg.NATSURL != "nats://nats:4222" {
		t.Fatalf("expected NATS URL override, got %q", cfg.NATSURL)
	}
	if cfg.Store != "persistence" {
		t.Fatalf("expected store override, got %q", cfg.Store)
	}
	if cfg.PersistenceURL != "http://persistence:8082" {
		t.Fatalf("expected persistence URL override, got %q", cfg.PersistenceURL)
	}
	if cfg.PersistenceTimeoutSec != 20 {
		t.Fatalf("expected persistence timeout override, got %d", cfg.PersistenceTimeoutSec)
	}
	if cfg.PersistenceServiceToken != "persist-token" {
		t.Fatalf("expected persistence service token override, got %q", cfg.PersistenceServiceToken)
	}
}
