// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	DefaultLocale     string
	ShutdownTimeout   time.Duration
	OIDCIssuer        string
	OIDCKeyID         string
	OIDCPrivateKeyPEM string
	AccessTokenTTL    time.Duration
	IDTokenTTL        time.Duration
	AuthCodeTTL       time.Duration
	SessionTTL        time.Duration
	RedisEnabled      bool
	RedisURL          string
	WebBaseURL        string
	FeishuAppID       string
	FeishuAppSecret   string
	FeishuBaseURL     string
	FeishuRedirectURI string
}

func Load() (Config, error) {
	shutdownTimeout, err := getDurationSeconds("IDB_SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	accessTokenTTL, err := getDurationSeconds("IDB_ACCESS_TOKEN_TTL_SECONDS", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	idTokenTTL, err := getDurationSeconds("IDB_ID_TOKEN_TTL_SECONDS", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	authCodeTTL, err := getDurationSeconds("IDB_AUTH_CODE_TTL_SECONDS", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := getDurationSeconds("IDB_SESSION_TTL_SECONDS", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	redisEnabled, err := getBool("IDB_REDIS_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:          getEnv("IDB_HTTP_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DefaultLocale:     getEnv("IDB_DEFAULT_LOCALE", "en-US"),
		ShutdownTimeout:   shutdownTimeout,
		OIDCIssuer:        getEnv("IDB_OIDC_ISSUER", "http://localhost:8080"),
		OIDCKeyID:         getEnv("IDB_OIDC_KEY_ID", "dev-key-1"),
		OIDCPrivateKeyPEM: os.Getenv("IDB_OIDC_PRIVATE_KEY_PEM"),
		AccessTokenTTL:    accessTokenTTL,
		IDTokenTTL:        idTokenTTL,
		AuthCodeTTL:       authCodeTTL,
		SessionTTL:        sessionTTL,
		RedisEnabled:      redisEnabled,
		RedisURL:          os.Getenv("IDB_REDIS_URL"),
		WebBaseURL:        os.Getenv("IDB_WEB_BASE_URL"),
		FeishuAppID:       os.Getenv("IDB_FEISHU_APP_ID"),
		FeishuAppSecret:   os.Getenv("IDB_FEISHU_APP_SECRET"),
		FeishuBaseURL:     getEnv("IDB_FEISHU_BASE_URL", "https://open.feishu.cn"),
		FeishuRedirectURI: getEnv("IDB_FEISHU_REDIRECT_URI", "http://localhost:8080/api/auth/feishu/callback"),
	}

	if cfg.DefaultLocale != "en-US" && cfg.DefaultLocale != "zh-CN" {
		return Config{}, fmt.Errorf("IDB_DEFAULT_LOCALE must be en-US or zh-CN")
	}
	if err := validateAbsoluteURL("IDB_OIDC_ISSUER", cfg.OIDCIssuer); err != nil {
		return Config{}, err
	}
	if cfg.WebBaseURL != "" {
		if err := validateAbsoluteURL("IDB_WEB_BASE_URL", cfg.WebBaseURL); err != nil {
			return Config{}, err
		}
		if os.Getenv("IDB_FEISHU_REDIRECT_URI") == "" {
			cfg.FeishuRedirectURI = strings.TrimRight(cfg.WebBaseURL, "/") + "/api/auth/feishu/callback"
		}
	}
	if err := validateAbsoluteURL("IDB_FEISHU_REDIRECT_URI", cfg.FeishuRedirectURI); err != nil {
		return Config{}, err
	}
	if cfg.RedisURL != "" {
		cfg.RedisEnabled = true
	}
	if cfg.RedisEnabled && cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("IDB_REDIS_URL is required when IDB_REDIS_ENABLED is true")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDurationSeconds(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a positive integer number of seconds", key)
	}
	if seconds <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return time.Duration(seconds) * time.Second, nil
}

func getBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return parsed, nil
}

func validateAbsoluteURL(key string, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s must be an absolute URL: %w", key, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL with scheme and host", key)
	}
	return nil
}
