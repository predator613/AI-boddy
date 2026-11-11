package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/config"
	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/migrations"
)

var quietLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func testConfig(t *testing.T, demo bool) config.Config {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	return config.Config{
		Port:        "0",
		DatabaseURL: url,
		JWTSecret:   "integration-secret-0123456789",
		JWTTTL:      time.Hour,
		DemoMode:    demo,
		CORSOrigins: []string{"http://localhost:5173"},
	}
}

// newServer поднимает настоящий роутер на настоящей базе.
func newServer(t *testing.T, demo bool) *httptest.Server {
	t.Helper()
	cfg := testConfig(t, demo)
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL, 10*time.Second)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = database.Migrate(ctx, pool, migrations.FS)
	require.NoError(t, err)

	svc, err := NewServices(cfg, pool, quietLog)
	require.NoError(t, err)
	srv := httptest.NewServer(NewRouter(cfg, svc, quietLog))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func login(t *testing.T, base, user, pass string) (int, string, models.User) {
	t.Helper()
	resp := do(t, http.MethodPost, base+"/api/auth/login", "", map[string]string{"login": user, "password": pass})
	var out struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out.Token, out.User
}

func TestFullLoginFlow(t *testing.T) {
	srv := newServer(t, true)

	code, token, user := login(t, srv.URL, "intern_anna", "demo123")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, token)
	assert.Equal(t, "intern", user.Role)
	assert.Equal(t, "Backend", user.TeamName)
	assert.NotNil(t, user.MentorID)

	resp := do(t, http.MethodGet, srv.URL+"/api/auth/me", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var me models.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&me))
	assert.Equal(t, user, me)

	// Павел получает свои данные, а не Аннины.
	_, pavelToken, _ := login(t, srv.URL, "intern_pavel", "demo123")
	resp = do(t, http.MethodGet, srv.URL+"/api/auth/me", pavelToken, nil)
	var pavel models.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&pavel))
	assert.Equal(t, "intern_pavel", pavel.Login)
	assert.Equal(t, "Frontend", pavel.TeamName)
}

func TestLoginErrorsOverHTTP(t *testing.T) {
	srv := newServer(t, true)

	code, _, _ := login(t, srv.URL, "intern_anna", "wrong")
	assert.Equal(t, http.StatusUnauthorized, code)

	code, _, _ = login(t, srv.URL, "", "")
	assert.Equal(t, http.StatusBadRequest, code)

	resp := do(t, http.MethodGet, srv.URL+"/api/auth/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp = do(t, http.MethodGet, srv.URL+"/api/auth/me", "broken.token.value", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestTestUsersFlag(t *testing.T) {
	on := newServer(t, true)
	resp := do(t, http.MethodGet, on.URL+"/api/auth/test-users", "", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list []models.TestUser
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	assert.GreaterOrEqual(t, len(list), 4)

	off := newServer(t, false)
	resp = do(t, http.MethodGet, off.URL+"/api/auth/test-users", "", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRouterExtras(t *testing.T) {
	srv := newServer(t, false)

	resp := do(t, http.MethodGet, srv.URL+"/api/health", "", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = do(t, http.MethodGet, srv.URL+"/api/unknown", "", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	resp = do(t, http.MethodGet, srv.URL+"/api/auth/login", "", nil)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// CORS preflight от фронта на Vite.
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	pre, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer pre.Body.Close()
	assert.Equal(t, "http://localhost:5173", pre.Header.Get("Access-Control-Allow-Origin"))
}

func TestRunStartsAndStops(t *testing.T) {
	cfg := testConfig(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, quietLog, ready) }()

	select {
	case addr := <-ready:
		resp, err := http.Get("http://" + addr + "/api/health")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	case err := <-done:
		t.Fatalf("Run exited early: %v", err)
	case <-time.After(15 * time.Second):
		t.Fatal("server did not start")
	}

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestRunFailures(t *testing.T) {
	cfg := testConfig(t, true)

	bad := cfg
	bad.DatabaseURL = "::not a url::"
	assert.Error(t, Run(context.Background(), bad, quietLog, nil))

	busy := cfg
	busy.Port = "-1"
	assert.Error(t, Run(context.Background(), busy, quietLog, nil))
}
