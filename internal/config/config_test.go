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
	if cfg.PolicyCatalogPath != "" {
		t.Fatalf("expected empty default policy catalog path, got %q", cfg.PolicyCatalogPath)
	}
	if len(cfg.PolicyTemplatePaths) != 0 {
		t.Fatalf("expected empty default policy template paths, got %#v", cfg.PolicyTemplatePaths)
	}
	if len(cfg.PolicyInstancePaths) != 0 {
		t.Fatalf("expected empty default policy instance paths, got %#v", cfg.PolicyInstancePaths)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("CPM_SERVICE_NAME", "custom-cpm")
	t.Setenv("CPM_HTTP_ADDR", ":9090")
	t.Setenv("CPM_LOG_LEVEL", "debug")
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
