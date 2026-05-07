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
	if cfg.PolicyCatalogPath != defaultPolicyCatalogPath {
		t.Fatalf("expected default policy catalog path %q, got %q", defaultPolicyCatalogPath, cfg.PolicyCatalogPath)
	}
	if len(cfg.PolicyTemplatePaths) != 1 || cfg.PolicyTemplatePaths[0] != defaultPolicyTemplatePaths {
		t.Fatalf("expected default policy template paths [%q], got %#v", defaultPolicyTemplatePaths, cfg.PolicyTemplatePaths)
	}
	if len(cfg.PolicyInstancePaths) != 1 || cfg.PolicyInstancePaths[0] != defaultPolicyInstancePaths {
		t.Fatalf("expected default policy instance paths [%q], got %#v", defaultPolicyInstancePaths, cfg.PolicyInstancePaths)
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
	t.Setenv("CPM_AUTH_CLOCK_SKEW_SEC", "45")
	t.Setenv("CPM_POLICY_CATALOG_PATH", "catalog.json")
	t.Setenv("CPM_POLICY_TEMPLATE_PATHS", "tpl-a.json, tpl-b.json")
	t.Setenv("CPM_POLICY_INSTANCE_PATHS", "inst-a.json,inst-b.json")

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
}
