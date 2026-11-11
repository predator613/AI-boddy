package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func valid() map[string]string {
	return map[string]string{
		"DATABASE_URL": "postgres://u:p@localhost:5432/db",
		"JWT_SECRET":   "0123456789abcdef-secret",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(valid()))
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 24*time.Hour, cfg.JWTTTL)
	assert.False(t, cfg.DemoMode)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.CORSOrigins)
}

func TestLoadCustomValues(t *testing.T) {
	m := valid()
	m["PORT"] = "9000"
	m["JWT_TTL"] = "30m"
	m["DEMO_MODE"] = "true"
	m["CORS_ORIGINS"] = "http://a.local, http://b.local ,"
	cfg, err := Load(env(m))
	require.NoError(t, err)
	assert.Equal(t, "9000", cfg.Port)
	assert.Equal(t, 30*time.Minute, cfg.JWTTTL)
	assert.True(t, cfg.DemoMode)
	assert.Equal(t, []string{"http://a.local", "http://b.local"}, cfg.CORSOrigins)
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]func(m map[string]string){
		"no database url": func(m map[string]string) { delete(m, "DATABASE_URL") },
		"short secret":    func(m map[string]string) { m["JWT_SECRET"] = "short" },
		"bad ttl":         func(m map[string]string) { m["JWT_TTL"] = "tomorrow" },
		"negative ttl":    func(m map[string]string) { m["JWT_TTL"] = "-1h" },
		"bad demo flag":   func(m map[string]string) { m["DEMO_MODE"] = "yes please" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			m := valid()
			mutate(m)
			_, err := Load(env(m))
			assert.Error(t, err)
		})
	}
}

func TestLoadLLMAndRequests(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef"}
	get := func(extra map[string]string) func(string) string {
		return func(k string) string {
			if v, ok := extra[k]; ok {
				return v
			}
			return base[k]
		}
	}
	cfg, err := Load(get(nil))
	if err != nil || cfg.LLMProvider != "mock" || cfg.LLMTimeout.String() != "20s" || cfg.RequestSync.String() != "5s" || cfg.MockServicesURL != "" {
		t.Fatalf("defaults: %+v %v", cfg, err)
	}
	cfg, err = Load(get(map[string]string{"LLM_PROVIDER": "openai_compat", "LLM_BASE_URL": "https://llm/v1", "LLM_MODEL": "a-vibe", "MOCK_SERVICES_URL": " http://mock:8090 "}))
	if err != nil || cfg.LLMModel != "a-vibe" || cfg.MockServicesURL != "http://mock:8090" {
		t.Fatalf("openai_compat: %+v %v", cfg, err)
	}
	for name, extra := range map[string]map[string]string{
		"provider":    {"LLM_PROVIDER": "gpt"},
		"no base url": {"LLM_PROVIDER": "openai_compat", "LLM_MODEL": "m"},
		"timeout":     {"LLM_TIMEOUT": "soon"},
		"zero sync":   {"REQUEST_SYNC_INTERVAL": "0s"},
		"negative":    {"MOCK_DECISION_DELAY": "-1s"},
	} {
		if _, err := Load(get(extra)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
