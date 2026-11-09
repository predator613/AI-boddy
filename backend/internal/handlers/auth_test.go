package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/middleware"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

type fakeAuth struct {
	loginErr  error
	userErr   error
	listErr   error
	list      []models.TestUser
	gotLogin  string
	gotUserID string
}

func (f *fakeAuth) Login(_ context.Context, login, _ string) (string, *models.User, error) {
	f.gotLogin = login
	if f.loginErr != nil {
		return "", nil, f.loginErr
	}
	return "tok", &models.User{ID: "u-1", Login: login}, nil
}

func (f *fakeAuth) CurrentUser(_ context.Context, id string) (*models.User, error) {
	f.gotUserID = id
	if f.userErr != nil {
		return nil, f.userErr
	}
	return &models.User{ID: id, Login: "intern_anna"}, nil
}

func (f *fakeAuth) TestUsers(context.Context) ([]models.TestUser, error) {
	return f.list, f.listErr
}

var quietLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Error.Message)
	return body.Error.Code
}

func postLogin(h *Auth, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body)))
	return rec
}

func TestLoginOK(t *testing.T) {
	f := &fakeAuth{}
	rec := postLogin(NewAuth(f, false, quietLog), `{"login":"  intern_anna ","password":"demo123"}`)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "intern_anna", f.gotLogin, "login must be trimmed")
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var resp struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "tok", resp.Token)
	assert.Equal(t, "u-1", resp.User.ID)
}

func TestLoginValidation(t *testing.T) {
	h := NewAuth(&fakeAuth{}, false, quietLog)
	for name, body := range map[string]string{
		"not json":       `login=anna`,
		"empty body":     ``,
		"empty fields":   `{"login":"","password":""}`,
		"spaces login":   `{"login":"   ","password":"x"}`,
		"empty password": `{"login":"intern_anna","password":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := postLogin(h, body)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "validation_error", errorCode(t, rec))
		})
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	rec := postLogin(NewAuth(&fakeAuth{loginErr: services.ErrInvalidCredentials}, false, quietLog),
		`{"login":"intern_anna","password":"bad"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "invalid_credentials", errorCode(t, rec))
}

func TestLoginInternalError(t *testing.T) {
	rec := postLogin(NewAuth(&fakeAuth{loginErr: errors.New("db down")}, false, quietLog),
		`{"login":"intern_anna","password":"demo123"}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "internal_error", errorCode(t, rec))
}

func getMe(h *Auth, withIdentity bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	if withIdentity {
		req = req.WithContext(middleware.WithIdentity(req.Context(), middleware.Identity{UserID: "u-1", Role: "intern"}))
	}
	rec := httptest.NewRecorder()
	h.Me(rec, req)
	return rec
}

func TestMe(t *testing.T) {
	f := &fakeAuth{}
	rec := getMe(NewAuth(f, false, quietLog), true)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "u-1", f.gotUserID)
}

func TestMeErrors(t *testing.T) {
	rec := getMe(NewAuth(&fakeAuth{}, false, quietLog), false)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "unauthorized", errorCode(t, rec))

	rec = getMe(NewAuth(&fakeAuth{userErr: services.ErrInvalidToken}, false, quietLog), true)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = getMe(NewAuth(&fakeAuth{userErr: errors.New("db down")}, false, quietLog), true)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func getTestUsers(h *Auth) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.TestUsers(rec, httptest.NewRequest(http.MethodGet, "/api/auth/test-users", nil))
	return rec
}

func TestTestUsersDemoOff(t *testing.T) {
	rec := getTestUsers(NewAuth(&fakeAuth{}, false, quietLog))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "not_found", errorCode(t, rec))
}

func TestTestUsersDemoOn(t *testing.T) {
	rec := getTestUsers(NewAuth(&fakeAuth{list: []models.TestUser{{Login: "intern_anna"}}}, true, quietLog))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "password")

	rec = getTestUsers(NewAuth(&fakeAuth{}, true, quietLog))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())

	rec = getTestUsers(NewAuth(&fakeAuth{listErr: errors.New("db down")}, true, quietLog))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	Health(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}
