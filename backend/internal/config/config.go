package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Encryption EncryptionConfig
	CORS       CORSConfig
	LLM        LLMConfig
	SMTP       SMTPConfig
	Telegram   TelegramConfig
	Briefing   BriefingConfig
	RateLimit  RateLimitConfig
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Enabled           bool
}

type BriefingConfig struct {
	Hour    int
	Enabled bool
}

type LLMConfig struct {
	OllamaURL    string
	OpenAIKey    string
	OpenAIURL    string
	AnthropicKey string
	DefaultModel string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type TelegramConfig struct {
	BotToken     string
	PollInterval int  // seconds
	Enabled      bool
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  int // minutes
	RefreshTokenExpiry int // hours
}

type EncryptionConfig struct {
	Key string // 32 bytes hex-encoded for AES-256
}

type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "postgres"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			User:     getEnv("POSTGRES_USER", "prometheus"),
			Password: getEnv("POSTGRES_PASSWORD", "changeme_db_password"),
			DBName:   getEnv("POSTGRES_DB", "prometheus"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns: getEnvInt("POSTGRES_MAX_CONNS", 20),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "redis"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", "changeme_redis_password"),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", ""),
			AccessTokenExpiry:  getEnvInt("JWT_ACCESS_EXPIRY_MINUTES", 1440), // 24 hours
			RefreshTokenExpiry: getEnvInt("JWT_REFRESH_EXPIRY_HOURS", 168),   // 7 days
		},
		Encryption: EncryptionConfig{
			Key: getEnv("ENCRYPTION_KEY", ""),
		},
		CORS: CORSConfig{
			AllowOrigins: strings.Split(getEnv("CORS_ALLOW_ORIGINS", "*"), ","),
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowHeaders: []string{"Authorization", "Content-Type", "X-Request-ID", "X-API-Key"},
		},
		LLM: LLMConfig{
			OllamaURL:    getEnv("LLM_OLLAMA_URL", "http://localhost:11434"),
			OpenAIKey:    getEnv("LLM_OPENAI_KEY", ""),
			OpenAIURL:    getEnv("LLM_OPENAI_URL", ""),
			AnthropicKey: getEnv("LLM_ANTHROPIC_KEY", ""),
			DefaultModel: getEnv("LLM_DEFAULT_MODEL", "llama3"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvInt("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", ""),
		},
		Telegram: TelegramConfig{
			BotToken:     getEnv("TELEGRAM_BOT_TOKEN", ""),
			PollInterval: getEnvInt("TELEGRAM_POLL_INTERVAL", 3),
			Enabled:      getEnv("TELEGRAM_ENABLED", "") == "true" || getEnv("TELEGRAM_BOT_TOKEN", "") != "",
		},
		Briefing: BriefingConfig{
			Hour:    getEnvInt("BRIEFING_HOUR", 7),
			Enabled: getEnv("BRIEFING_ENABLED", "true") == "true",
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_RPM", 300),
			Enabled:           getEnv("RATE_LIMIT_ENABLED", "true") == "true",
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Encryption.Key == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}
	if len(c.Encryption.Key) != 64 {
		return fmt.Errorf("ENCRYPTION_KEY must be 64 hex characters (32 bytes)")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
