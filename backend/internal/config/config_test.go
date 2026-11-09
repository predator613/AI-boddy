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
