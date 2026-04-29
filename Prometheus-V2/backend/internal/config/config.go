package config

import (
	"os"
)

type Config struct {
	HTTPAddr    string
	LogLevel    string
	DatabaseURL string
	RedisURL    string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("PROMETHEUS_HTTP_ADDR", ":8180"),
		LogLevel:    getenv("PROMETHEUS_LOG_LEVEL", "info"),
		DatabaseURL: getenv("PROMETHEUS_DATABASE_URL", "postgres://prometheus:prometheus@localhost:5432/prometheus_v2?sslmode=disable&search_path=prom_v2"),
		RedisURL:    getenv("PROMETHEUS_REDIS_URL", "redis://localhost:6379/0"),
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
