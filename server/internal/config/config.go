package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
)

type Config struct {
	Service              appmeta.ServiceName
	Version              string
	GinMode              string
	HTTPAddr             string
	DatabaseURL          string
	RedisAddr            string
	NATSURL              string
	ConnectTimeout       time.Duration
	ShutdownTimeout      time.Duration
	ReadHeaderTimeout    time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	AccessTokenSecret    string
	RefreshTokenSecret   string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	EmailVerifyTTL       time.Duration
	CookieSecure         bool
	AdminUsernames       []string
	CodeforcesAPIBaseURL string
	CodeforcesAPITimeout time.Duration
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

	readHeaderTimeout, err := durationValue("ACMRANK_HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationValue("ACMRANK_HTTP_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue("ACMRANK_HTTP_WRITE_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationValue("ACMRANK_HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}

	accessTokenTTL, err := durationValue("ACMRANK_AUTH_ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	refreshTokenTTL, err := durationValue("ACMRANK_AUTH_REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	emailVerifyTTL, err := durationValue("ACMRANK_AUTH_EMAIL_VERIFY_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	codeforcesAPITimeout, err := durationValue("ACMRANK_CODEFORCES_API_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	cookieSecure, err := boolValue("ACMRANK_AUTH_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Service:              service,
		Version:              stringValue("ACMRANK_VERSION", "dev"),
		GinMode:              stringValue("GIN_MODE", "release"),
		HTTPAddr:             stringValue(serviceHTTPAddrEnv(service), defaultHTTPAddr(service)),
		DatabaseURL:          stringValue("ACMRANK_DATABASE_URL", "postgres://acmrank:acmrank_dev@127.0.0.1:5432/acmrank?sslmode=disable"),
		RedisAddr:            stringValue("ACMRANK_REDIS_ADDR", "127.0.0.1:6379"),
		NATSURL:              stringValue("ACMRANK_NATS_URL", "nats://127.0.0.1:4222"),
		ConnectTimeout:       connectTimeout,
		ShutdownTimeout:      shutdownTimeout,
		ReadHeaderTimeout:    readHeaderTimeout,
		ReadTimeout:          readTimeout,
		WriteTimeout:         writeTimeout,
		IdleTimeout:          idleTimeout,
		AccessTokenSecret:    stringValue("ACMRANK_AUTH_ACCESS_TOKEN_SECRET", "acmrank-dev-access-secret"),
		RefreshTokenSecret:   stringValue("ACMRANK_AUTH_REFRESH_TOKEN_SECRET", "acmrank-dev-refresh-secret"),
		AccessTokenTTL:       accessTokenTTL,
		RefreshTokenTTL:      refreshTokenTTL,
		EmailVerifyTTL:       emailVerifyTTL,
		CookieSecure:         cookieSecure,
		AdminUsernames:       csvValue("ACMRANK_ADMIN_USERNAMES"),
		CodeforcesAPIBaseURL: stringValue("ACMRANK_CODEFORCES_API_BASE_URL", "https://codeforces.com/api"),
		CodeforcesAPITimeout: codeforcesAPITimeout,
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

func boolValue(key string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func csvValue(key string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}

	values := strings.Split(raw, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		result = append(result, trimmed)
	}

	return result
}
