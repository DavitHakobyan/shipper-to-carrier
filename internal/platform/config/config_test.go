package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_NAME", "")
	t.Setenv("APP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SESSION_TTL", "")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.Addr != ":8080" {
		t.Fatalf("Addr = %q, want %q", config.Addr, ":8080")
	}

	if config.SessionTTL != 24*time.Hour {
		t.Fatalf("SessionTTL = %s, want %s", config.SessionTTL, 24*time.Hour)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_NAME", "Marketplace")
	t.Setenv("APP_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SESSION_COOKIE_NAME", "marketplace_session")
	t.Setenv("SESSION_TTL", "48h")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.AppName != "Marketplace" {
		t.Fatalf("AppName = %q, want %q", config.AppName, "Marketplace")
	}

	if config.SessionCookieName != "marketplace_session" {
		t.Fatalf("SessionCookieName = %q, want %q", config.SessionCookieName, "marketplace_session")
	}

	if config.SessionTTL != 48*time.Hour {
		t.Fatalf("SessionTTL = %s, want %s", config.SessionTTL, 48*time.Hour)
	}
}
