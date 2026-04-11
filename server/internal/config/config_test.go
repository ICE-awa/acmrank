package config

import (
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
)

func TestLoadDefaultsForAPI(t *testing.T) {
	cfg, err := Load(appmeta.ServiceAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("Load() HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}

	if cfg.DatabaseURL == "" || cfg.RedisAddr == "" || cfg.NATSURL == "" {
		t.Fatal("Load() expected infrastructure addresses to be populated")
	}

	if cfg.GinMode != "release" {
		t.Fatalf("Load() GinMode = %q, want %q", cfg.GinMode, "release")
	}

	if cfg.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("Load() ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, 5*time.Second)
	}

	if cfg.ReadTimeout != 15*time.Second {
		t.Fatalf("Load() ReadTimeout = %v, want %v", cfg.ReadTimeout, 15*time.Second)
	}

	if cfg.WriteTimeout != 15*time.Second {
		t.Fatalf("Load() WriteTimeout = %v, want %v", cfg.WriteTimeout, 15*time.Second)
	}

	if cfg.IdleTimeout != 60*time.Second {
		t.Fatalf("Load() IdleTimeout = %v, want %v", cfg.IdleTimeout, 60*time.Second)
	}

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("Load() AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 15*time.Minute)
	}

	if cfg.RefreshTokenTTL != 30*24*time.Hour {
		t.Fatalf("Load() RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 30*24*time.Hour)
	}

	if cfg.EmailVerifyTTL != 24*time.Hour {
		t.Fatalf("Load() EmailVerifyTTL = %v, want %v", cfg.EmailVerifyTTL, 24*time.Hour)
	}

	if cfg.CookieSecure {
		t.Fatal("Load() CookieSecure should default to false")
	}

	if len(cfg.AdminUsernames) != 0 {
		t.Fatalf("Load() AdminUsernames len = %d, want 0", len(cfg.AdminUsernames))
	}
}

func TestLoadHonorsOverrides(t *testing.T) {
	t.Setenv("ACMRANK_SCHEDULER_HTTP_ADDR", ":9091")
	t.Setenv("ACMRANK_CONNECT_TIMEOUT", "7s")
	t.Setenv("ACMRANK_SHUTDOWN_TIMEOUT", "19s")
	t.Setenv("ACMRANK_HTTP_READ_HEADER_TIMEOUT", "6s")
	t.Setenv("ACMRANK_HTTP_READ_TIMEOUT", "21s")
	t.Setenv("ACMRANK_HTTP_WRITE_TIMEOUT", "22s")
	t.Setenv("ACMRANK_HTTP_IDLE_TIMEOUT", "75s")
	t.Setenv("ACMRANK_AUTH_ACCESS_TOKEN_TTL", "17m")
	t.Setenv("ACMRANK_AUTH_REFRESH_TOKEN_TTL", "240h")
	t.Setenv("ACMRANK_AUTH_EMAIL_VERIFY_TTL", "48h")
	t.Setenv("ACMRANK_AUTH_COOKIE_SECURE", "true")
	t.Setenv("ACMRANK_ADMIN_USERNAMES", "admin, operator ")

	cfg, err := Load(appmeta.ServiceScheduler)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":9091" {
		t.Fatalf("Load() HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9091")
	}

	if cfg.ConnectTimeout != 7*time.Second {
		t.Fatalf("Load() ConnectTimeout = %v, want %v", cfg.ConnectTimeout, 7*time.Second)
	}

	if cfg.ShutdownTimeout != 19*time.Second {
		t.Fatalf("Load() ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 19*time.Second)
	}

	if cfg.ReadHeaderTimeout != 6*time.Second {
		t.Fatalf("Load() ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, 6*time.Second)
	}

	if cfg.ReadTimeout != 21*time.Second {
		t.Fatalf("Load() ReadTimeout = %v, want %v", cfg.ReadTimeout, 21*time.Second)
	}

	if cfg.WriteTimeout != 22*time.Second {
		t.Fatalf("Load() WriteTimeout = %v, want %v", cfg.WriteTimeout, 22*time.Second)
	}

	if cfg.IdleTimeout != 75*time.Second {
		t.Fatalf("Load() IdleTimeout = %v, want %v", cfg.IdleTimeout, 75*time.Second)
	}

	if cfg.AccessTokenTTL != 17*time.Minute {
		t.Fatalf("Load() AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 17*time.Minute)
	}

	if cfg.RefreshTokenTTL != 240*time.Hour {
		t.Fatalf("Load() RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 240*time.Hour)
	}

	if cfg.EmailVerifyTTL != 48*time.Hour {
		t.Fatalf("Load() EmailVerifyTTL = %v, want %v", cfg.EmailVerifyTTL, 48*time.Hour)
	}

	if !cfg.CookieSecure {
		t.Fatal("Load() CookieSecure should honor override")
	}

	if len(cfg.AdminUsernames) != 2 || cfg.AdminUsernames[0] != "admin" || cfg.AdminUsernames[1] != "operator" {
		t.Fatalf("Load() AdminUsernames = %#v, want [admin operator]", cfg.AdminUsernames)
	}
}

func TestLoadRejectsUnsupportedService(t *testing.T) {
	if _, err := Load("worker"); err == nil {
		t.Fatal("Load() expected unsupported service error")
	}
}

func TestLoadRejectsInvalidDurationOverride(t *testing.T) {
	t.Setenv("ACMRANK_CONNECT_TIMEOUT", "7")

	if _, err := Load(appmeta.ServiceAPI); err == nil {
		t.Fatal("Load() expected invalid duration error")
	}
}

func TestLoadRejectsInvalidBoolOverride(t *testing.T) {
	t.Setenv("ACMRANK_AUTH_COOKIE_SECURE", "sometimes")

	if _, err := Load(appmeta.ServiceAPI); err == nil {
		t.Fatal("Load() expected invalid bool error")
	}
}
