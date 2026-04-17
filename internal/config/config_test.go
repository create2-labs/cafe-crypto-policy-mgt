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
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("CPM_SERVICE_NAME", "custom-cpm")
	t.Setenv("CPM_HTTP_ADDR", ":9090")
	t.Setenv("CPM_LOG_LEVEL", "debug")

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
}
