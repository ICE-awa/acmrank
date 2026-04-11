package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
)

type Config struct {
	Service         appmeta.ServiceName
	Version         string
	GinMode         string
	HTTPAddr        string
	DatabaseURL     string
	RedisAddr       string
	NATSURL         string
	ConnectTimeout  time.Duration
	ShutdownTimeout time.Duration
}

func Load(service appmeta.ServiceName) (Config, error) {
	if !appmeta.IsSupportedService(service) {
		return Config{}, fmt.Errorf("unsupported service %q", service)
	}

	connectTimeout, err := durationValue("ACMRANK_CONNECT_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue("ACMRANK_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Service:         service,
		Version:         stringValue("ACMRANK_VERSION", "dev"),
		GinMode:         stringValue("GIN_MODE", "release"),
		HTTPAddr:        stringValue(serviceHTTPAddrEnv(service), defaultHTTPAddr(service)),
		DatabaseURL:     stringValue("ACMRANK_DATABASE_URL", "postgres://acmrank:acmrank_dev@127.0.0.1:5432/acmrank?sslmode=disable"),
		RedisAddr:       stringValue("ACMRANK_REDIS_ADDR", "127.0.0.1:6379"),
		NATSURL:         stringValue("ACMRANK_NATS_URL", "nats://127.0.0.1:4222"),
		ConnectTimeout:  connectTimeout,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func serviceHTTPAddrEnv(service appmeta.ServiceName) string {
	return fmt.Sprintf("ACMRANK_%s_HTTP_ADDR", strings.ToUpper(string(service)))
}

func defaultHTTPAddr(service appmeta.ServiceName) string {
	switch service {
	case appmeta.ServiceAPI:
		return ":8080"
	case appmeta.ServiceScheduler:
		return ":8081"
	case appmeta.ServiceAggregator:
		return ":8082"
	default:
		return ":8080"
	}
}

func stringValue(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func durationValue(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}
