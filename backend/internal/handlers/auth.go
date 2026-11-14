// Package handlers – HTTP-обработчики: разбирают запрос, зовут сервис, пишут JSON.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/middleware"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// AuthService – то, что обработчикам нужно от сервиса авторизации.
type AuthService interface {
	Login(ctx context.Context, login, password string) (string, *models.User, error)
	CurrentUser(ctx context.Context, userID string) (*models.User, error)
	TestUsers(ctx context.Context) ([]models.TestUser, error)
}

// Auth – обработчики /api/auth/*.
type Auth struct {
	svc      AuthService
	demoMode bool
	log      *slog.Logger
	limit    *loginLimiter
}

// NewAuth создает обработчики авторизации.
func NewAuth(svc AuthService, demoMode bool, log *slog.Logger) *Auth {
	// 10 неудачных попыток за 5 минут – вход для этой пары «адрес + логин» закрыт до конца окна.
	return &Auth{svc: svc, demoMode: demoMode, log: log, limit: newLoginLimiter(10, 5*time.Minute)}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// maxBodyBytes ограничивает размер тела запроса на вход.
const maxBodyBytes = 1 << 16

// Login – POST /api/auth/login.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err := dec.Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeValidation, "request body must be JSON: {\"login\", \"password\"}")
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeValidation, "login and password are required")
		return
	}

	key := loginKey(r, req.Login)
	if wait := h.limit.blocked(key); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		httpx.Error(w, http.StatusTooManyRequests, httpx.CodeTooManyAttempts, "too many login attempts, try again later")
		return
	}
	token, user, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if errors.Is(err, services.ErrInvalidCredentials) {
		h.limit.fail(key)
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, "invalid login or password")
		return
	}
	if err != nil {
		h.internal(w, "login failed", err)
		return
	}
	h.limit.reset(key)
	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

// Me – GET /api/auth/me. Работает только за middleware.RequireAuth.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "unauthorized")
		return
	}
	user, err := h.svc.CurrentUser(r.Context(), id.UserID)
	if errors.Is(err, services.ErrInvalidToken) {
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "user from token no longer exists")
		return
	}
	if err != nil {
		h.internal(w, "load current user failed", err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// TestUsers – GET /api/auth/test-users. Доступен только при DEMO_MODE=true, иначе 404.
func (h *Auth) TestUsers(w http.ResponseWriter, r *http.Request) {
	if !h.demoMode {
		httpx.NotFound(w, r)
		return
	}
	users, err := h.svc.TestUsers(r.Context())
	if err != nil {
		h.internal(w, "list test users failed", err)
		return
	}
	if users == nil {
		users = []models.TestUser{}
	}
	httpx.JSON(w, http.StatusOK, users)
}

func (h *Auth) internal(w http.ResponseWriter, msg string, err error) {
	h.log.Error(msg, "err", err)
	httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternal, "internal server error")
}
