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
}

func TestLoadHonorsOverrides(t *testing.T) {
	t.Setenv("ACMRANK_SCHEDULER_HTTP_ADDR", ":9091")
	t.Setenv("ACMRANK_CONNECT_TIMEOUT", "7s")
	t.Setenv("ACMRANK_SHUTDOWN_TIMEOUT", "19s")

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
