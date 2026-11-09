package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

const secret = "test-secret-0123456789"

// fakeStore – подмена репозитория без базы.
type fakeStore struct {
	users map[string]*models.UserWithPassword
	err   error
}

func (f *fakeStore) GetByLogin(_ context.Context, login string) (*models.UserWithPassword, error) {
	if f.err != nil {
		return nil, f.err
	}
	if u, ok := f.users[login]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeStore) GetByID(_ context.Context, id string) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, u := range f.users {
		if u.ID == id {
			return &u.User, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeStore) ListTestUsers(context.Context) ([]models.TestUser, error) {
	return []models.TestUser{{Login: "intern_anna"}}, f.err
}

func newStore(t *testing.T) *fakeStore {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.MinCost)
	require.NoError(t, err)
	return &fakeStore{users: map[string]*models.UserWithPassword{
		"intern_anna": {
			User:         models.User{ID: "u-1", Login: "intern_anna", Name: "Анна", Role: models.RoleIntern},
			PasswordHash: string(hash),
		},
	}}
}

func TestLoginSuccess(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)
	token, user, err := svc.Login(context.Background(), "intern_anna", "demo123")
	require.NoError(t, err)
	assert.Equal(t, "u-1", user.ID)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "u-1", claims.Subject)
	assert.Equal(t, models.RoleIntern, claims.Role)
}

func TestLoginWrongPassword(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)
	_, _, err := svc.Login(context.Background(), "intern_anna", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLoginUnknownUser(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)
	_, _, err := svc.Login(context.Background(), "nobody", "demo123")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLoginStoreError(t *testing.T) {
	svc := NewAuth(&fakeStore{err: errors.New("db down")}, secret, time.Hour)
	_, _, err := svc.Login(context.Background(), "intern_anna", "demo123")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestParseTokenExpired(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Minute)
	svc.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	token, err := svc.IssueToken("u-1", models.RoleIntern)
	require.NoError(t, err)

	svc.now = time.Now
	_, err = svc.ParseToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseTokenWrongSecret(t *testing.T) {
	other := NewAuth(newStore(t), "another-secret-0123456789", time.Hour)
	token, err := other.IssueToken("u-1", models.RoleIntern)
	require.NoError(t, err)

	_, err = NewAuth(newStore(t), secret, time.Hour).ParseToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseTokenGarbageAndWrongAlg(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)

	_, err := svc.ParseToken("not-a-jwt")
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Токен с другим алгоритмом (HS512) должен отклоняться, даже если секрет тот же.
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject: "u-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	_, err = svc.ParseToken(hs512)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseTokenWithoutSubjectOrExpiry(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)

	noSub, err := svc.IssueToken("", models.RoleIntern)
	require.NoError(t, err)
	_, err = svc.ParseToken(noSub)
	assert.ErrorIs(t, err, ErrInvalidToken)

	noExp, err := jwt.NewWithClaims(jwt.SigningMethodHS256,
		Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u-1"}}).SignedString([]byte(secret))
	require.NoError(t, err)
	_, err = svc.ParseToken(noExp)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCurrentUser(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)

	u, err := svc.CurrentUser(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, "intern_anna", u.Login)

	_, err = svc.CurrentUser(context.Background(), "deleted")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTestUsers(t *testing.T) {
	svc := NewAuth(newStore(t), secret, time.Hour)
	list, err := svc.TestUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1)
}
