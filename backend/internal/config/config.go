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

	// LLM: mock (по умолчанию, без ключа) или openai_compat (A-Vibe, OpenRouter, GigaChat, Ollama).
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
	LLMProject  string // ID каталога для Yandex AI Studio
	LLMTimeout  time.Duration

	// Система заявок: адрес mock-services. Пусто – имитация внутри backend.
	MockServicesURL string
	// Адрес mock-services для браузера: ссылки на карточки заявок. Пусто – ссылок нет, только ключ.
	MockServicesPublicURL string
	MockDecisionDelay     time.Duration // для имитации внутри backend
	RequestSync           time.Duration // как часто забирать решения по заявкам
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

	cfg.LLMProvider = withDefault(getenv("LLM_PROVIDER"), "mock")
	if cfg.LLMProvider != "mock" && cfg.LLMProvider != "openai_compat" {
		return Config{}, errors.New("LLM_PROVIDER must be mock or openai_compat")
	}
	cfg.LLMBaseURL = strings.TrimSpace(getenv("LLM_BASE_URL"))
	cfg.LLMAPIKey = strings.TrimSpace(getenv("LLM_API_KEY"))
	cfg.LLMModel = strings.TrimSpace(getenv("LLM_MODEL"))
	cfg.LLMProject = strings.TrimSpace(getenv("LLM_PROJECT"))
	if cfg.LLMProvider == "openai_compat" && (cfg.LLMBaseURL == "" || cfg.LLMModel == "") {
		return Config{}, errors.New("LLM_BASE_URL and LLM_MODEL are required when LLM_PROVIDER=openai_compat")
	}
	cfg.MockServicesURL = strings.TrimSpace(getenv("MOCK_SERVICES_URL"))
	cfg.MockServicesPublicURL = strings.TrimRight(strings.TrimSpace(getenv("MOCK_SERVICES_PUBLIC_URL")), "/")
	for _, d := range []struct {
		key, def string
		dst      *time.Duration
	}{
		{"LLM_TIMEOUT", "20s", &cfg.LLMTimeout},
		{"MOCK_DECISION_DELAY", "45s", &cfg.MockDecisionDelay},
		{"REQUEST_SYNC_INTERVAL", "5s", &cfg.RequestSync},
	} {
		v, err := time.ParseDuration(withDefault(getenv(d.key), d.def))
		if err != nil || v < 0 {
			return Config{}, fmt.Errorf("%s must be a duration like %s", d.key, d.def)
		}
		*d.dst = v
	}
	if cfg.LLMTimeout == 0 || cfg.RequestSync == 0 {
		return Config{}, errors.New("LLM_TIMEOUT and REQUEST_SYNC_INTERVAL must be positive")
	}

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
