package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppName           string
	Addr              string
	DatabaseURL       string
	SessionCookieName string
	SessionTTL        time.Duration
}

func Load() (Config, error) {
	sessionTTL := 24 * time.Hour
	if value := os.Getenv("SESSION_TTL"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse SESSION_TTL: %w", err)
		}

		sessionTTL = duration
	}

	return Config{
		AppName:           envOrDefault("APP_NAME", "Shipper to Carrier"),
		Addr:              envOrDefault("APP_ADDR", ":8080"),
		DatabaseURL:       envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shipper_to_carrier?sslmode=disable"),
		SessionCookieName: envOrDefault("SESSION_COOKIE_NAME", "shipper_to_carrier_session"),
		SessionTTL:        sessionTTL,
	}, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
