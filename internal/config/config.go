package config

import "os"

const (
	defaultHTTPAddr = ":8080"
	defaultLogLevel = "info"
	defaultService  = "cafe-cpm"
)

type Config struct {
	ServiceName string
	HTTPAddr    string
	LogLevel    string
}

func LoadFromEnv() Config {
	return Config{
		ServiceName: getEnv("CPM_SERVICE_NAME", defaultService),
		HTTPAddr:    getEnv("CPM_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:    getEnv("CPM_LOG_LEVEL", defaultLogLevel),
	}
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
