package config

import (
	"testing"
	"time"
)

// Load must refuse to start production with a guessable signing key or a
// wildcard CORS policy, because both silently remove a security boundary.
func TestLoadRejectsUnsafeProductionConfig(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	t.Run("default secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "dev-insecure-change-me")
		if _, err := Load(); err == nil {
			t.Fatal("the development secret must be rejected in production")
		}
	})

	t.Run("short secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "too-short")
		if _, err := Load(); err == nil {
			t.Fatal("a short secret must be rejected in production")
		}
	})

	t.Run("wildcard cors", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "0123456789012345678901234567890123456789")
		t.Setenv("HTTP_CORS_ORIGINS", "*")
		if _, err := Load(); err == nil {
			t.Fatal("a wildcard CORS origin must be rejected in production")
		}
	})

	t.Run("valid", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "0123456789012345678901234567890123456789")
		t.Setenv("HTTP_CORS_ORIGINS", "https://shop.example.com")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if !cfg.App.IsProduction() {
			t.Error("APP_ENV=production should mark the config as production")
		}
	})
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTP.RateLimitAuthRPS <= 0 {
		t.Error("the sign-in rate limit must default to a positive value")
	}
	if cfg.Worker.CommissionSettleInterval <= 0 {
		t.Error("the commission settle interval must default to a positive duration")
	}
	if cfg.JWT.AccessTTL <= 0 || cfg.JWT.AccessTTL > 24*time.Hour {
		t.Errorf("access token TTL = %v, want a short positive duration", cfg.JWT.AccessTTL)
	}
}

func TestLoadReadsAuthRateLimit(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_AUTH_RATE_LIMIT_RPS", "7")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTP.RateLimitAuthRPS != 7 {
		t.Fatalf("auth rate limit = %d, want 7", cfg.HTTP.RateLimitAuthRPS)
	}
}

func TestLoadParsesOIDCClientSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("OIDC_CLIENT_SECRETS", `{"google":"g-secret","keycloak":"k-secret"}`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OIDC.ClientSecrets["google"] != "g-secret" || cfg.OIDC.ClientSecrets["keycloak"] != "k-secret" {
		t.Fatalf("client secrets were not parsed: %v", cfg.OIDC.ClientSecrets)
	}
	// A malformed map must not panic; it is treated as absent.
	t.Setenv("OIDC_CLIENT_SECRETS", "not json")
	if cfg, err = Load(); err != nil {
		t.Fatalf("load with bad json: %v", err)
	}
	if len(cfg.OIDC.ClientSecrets) != 0 {
		t.Error("a malformed OIDC_CLIENT_SECRETS value must be ignored")
	}
}
