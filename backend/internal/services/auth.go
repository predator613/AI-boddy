// Package services – бизнес-логика. Здесь: вход, bcrypt, выдача и проверка JWT.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Ошибки сервиса авторизации.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

// UserStore – то, что сервису нужно от репозитория. Интерфейс позволяет подменить базу в тестах.
type UserStore interface {
	GetByLogin(ctx context.Context, login string) (*models.UserWithPassword, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	ListTestUsers(ctx context.Context) ([]models.TestUser, error)
}

// Claims – содержимое JWT: id пользователя (sub) и роль.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Auth – сервис авторизации.
type Auth struct {
	users  UserStore
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// dummyHash нужен, чтобы при несуществующем логине тоже тратить время на bcrypt
// и не подсказывать по времени ответа, какие логины есть.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// NewAuth создает сервис авторизации.
func NewAuth(users UserStore, secret string, ttl time.Duration) *Auth {
	return &Auth{users: users, secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Login проверяет логин и пароль и возвращает токен и пользователя.
func (a *Auth) Login(ctx context.Context, login, password string) (string, *models.User, error) {
	u, err := a.users.GetByLogin(ctx, login)
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, fmt.Errorf("get user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := a.IssueToken(u.ID, u.Role)
	if err != nil {
		return "", nil, err
	}
	return token, &u.User, nil
}

// IssueToken выдает подписанный JWT (HS256).
func (a *Auth) IssueToken(userID, role string) (string, error) {
	now := a.now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
			Issuer:    "ai-buddy-backend",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// ParseToken проверяет подпись и срок действия и возвращает claims.
func (a *Auth) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return a.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(a.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// CurrentUser возвращает пользователя по id из токена.
func (a *Auth) CurrentUser(ctx context.Context, userID string) (*models.User, error) {
	u, err := a.users.GetByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	return u, err
}

// TestUsers возвращает список тестовых пользователей для экрана входа.
func (a *Auth) TestUsers(ctx context.Context) ([]models.TestUser, error) {
	return a.users.ListTestUsers(ctx)
}
