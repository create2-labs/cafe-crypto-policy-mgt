package version

import (
	"os"
	"testing"
)

func TestCurrentUsesEmbeddedDefault(t *testing.T) {
	t.Setenv("APP_VERSION", "")
	version = "v1.2.3-test"
	t.Cleanup(func() { version = defaultVersion })

	if got := Current(); got != "v1.2.3-test" {
		t.Fatalf("Current() = %q, want v1.2.3-test", got)
	}
}

func TestCurrentPrefersAppVersionEnv(t *testing.T) {
	t.Setenv("APP_VERSION", "v9.9.9")
	version = "embedded"
	t.Cleanup(func() {
		version = defaultVersion
		_ = os.Unsetenv("APP_VERSION")
	})

	if got := Current(); got != "v9.9.9" {
		t.Fatalf("Current() = %q, want v9.9.9", got)
	}
}

func TestPayloadShape(t *testing.T) {
	t.Setenv("APP_VERSION", "v0.0.1")
	t.Cleanup(func() { _ = os.Unsetenv("APP_VERSION") })

	p := Payload()
	if p.Version != "v0.0.1" {
		t.Fatalf("Payload().Version = %q, want v0.0.1", p.Version)
	}
}
