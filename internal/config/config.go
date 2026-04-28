package config

import "os"

const (
	defaultHTTPAddr = ":8080"
	defaultLogLevel = "info"
	defaultService  = "cafe-cpm"
)

type Config struct {
	ServiceName         string
	HTTPAddr            string
	LogLevel            string
	PolicyCatalogPath   string
	PolicyTemplatePaths []string
	PolicyInstancePaths []string
}

func LoadFromEnv() Config {
	return Config{
		ServiceName:         getEnv("CPM_SERVICE_NAME", defaultService),
		HTTPAddr:            getEnv("CPM_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:            getEnv("CPM_LOG_LEVEL", defaultLogLevel),
		PolicyCatalogPath:   getEnv("CPM_POLICY_CATALOG_PATH", ""),
		PolicyTemplatePaths: parseCommaList(getEnv("CPM_POLICY_TEMPLATE_PATHS", "")),
		PolicyInstancePaths: parseCommaList(getEnv("CPM_POLICY_INSTANCE_PATHS", "")),
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
