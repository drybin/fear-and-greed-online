package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv           string
	AppHost          string
	AppPort          string
	BinanceBaseURL   string
	SyncBackfillDays int
	Telegram         TelegramConfig
	Postgres         PostgresConfig
}

type TelegramConfig struct {
	Enabled  bool
	BotToken string
	ChatID   string
	BaseURL  string
}

type PostgresConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	SSLMode  string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:           envOrDefault("APP_ENV", "development"),
		AppHost:          envOrDefault("APP_HOST", "0.0.0.0"),
		AppPort:          envOrDefault("APP_PORT", "80"),
		BinanceBaseURL:   envOrDefault("BINANCE_BASE_URL", "https://api.binance.com"),
		SyncBackfillDays: intEnvOrDefault("SYNC_BACKFILL_DAYS", 30),
		Telegram: TelegramConfig{
			Enabled:  boolEnvOrDefault("TG_ENABLED", false),
			BotToken: envOrDefault("TG_BOT_TOKEN", ""),
			ChatID:   envOrDefault("TG_CHAT_ID", ""),
			BaseURL:  envOrDefault("TG_BASE_URL", "http://dr54.ru/"),
		},
		Postgres: PostgresConfig{
			Host:     envOrDefault("POSTGRES_HOST", "127.0.0.1"),
			Port:     envOrDefault("POSTGRES_PORT", "5433"),
			Database: envOrDefault("POSTGRES_DB", "fear_and_greed_online"),
			User:     envOrDefault("POSTGRES_USER", "fear_and_greed"),
			Password: envOrDefault("POSTGRES_PASSWORD", "fear_and_greed"),
			SSLMode:  envOrDefault("POSTGRES_SSLMODE", "disable"),
		},
	}

	if cfg.AppPort == "" {
		return nil, fmt.Errorf("APP_PORT is required")
	}
	if cfg.SyncBackfillDays < 1 {
		return nil, fmt.Errorf("SYNC_BACKFILL_DAYS must be at least 1")
	}
	if cfg.Telegram.Enabled {
		if cfg.Telegram.BotToken == "" {
			return nil, fmt.Errorf("TG_BOT_TOKEN is required when TG_ENABLED=true")
		}
		if cfg.Telegram.ChatID == "" {
			return nil, fmt.Errorf("TG_CHAT_ID is required when TG_ENABLED=true")
		}
	}
	if cfg.Postgres.Host == "" || cfg.Postgres.Port == "" || cfg.Postgres.Database == "" || cfg.Postgres.User == "" {
		return nil, fmt.Errorf("postgres configuration is incomplete")
	}

	return cfg, nil
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host,
		c.Port,
		c.Database,
		c.User,
		c.Password,
		c.SSLMode,
	)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnvOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func boolEnvOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
