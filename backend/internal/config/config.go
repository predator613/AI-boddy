// Package config читает настройки сервиса из переменных окружения.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config – все настройки backend.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	DemoMode    bool
	CORSOrigins []string
}

// Load собирает Config. getenv передается снаружи, чтобы в тестах подставлять свои значения.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:        withDefault(getenv("PORT"), "8080"),
		DatabaseURL: getenv("DATABASE_URL"),
		JWTSecret:   getenv("JWT_SECRET"),
		CORSOrigins: splitList(withDefault(getenv("CORS_ORIGINS"), "http://localhost:5173")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 16 {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 16 characters")
	}

	ttl, err := time.ParseDuration(withDefault(getenv("JWT_TTL"), "24h"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be a positive duration like 24h or 30m")
	}
	cfg.JWTTTL = ttl

	demo, err := strconv.ParseBool(withDefault(getenv("DEMO_MODE"), "false"))
	if err != nil {
		return Config{}, fmt.Errorf("DEMO_MODE must be true or false")
	}
	cfg.DemoMode = demo

	return cfg, nil
}

func withDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
