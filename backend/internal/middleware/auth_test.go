package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/services"
)

type fakeParser struct{}

func (fakeParser) ParseToken(token string) (*services.Claims, error) {
	if token == "good" {
		return &services.Claims{Role: "intern", RegisteredClaims: jwt.RegisteredClaims{Subject: "u-1"}}, nil
	}
	return nil, errors.New("bad token")
}

func run(t *testing.T, header string) (*httptest.ResponseRecorder, *Identity) {
	t.Helper()
	var seen *Identity
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentityFrom(r.Context())
		require.True(t, ok)
		seen = &id
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	RequireAuth(fakeParser{})(next).ServeHTTP(rec, req)
	return rec, seen
}

func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body.Error.Code)
}

func TestRequireAuthRejects(t *testing.T) {
	for name, header := range map[string]string{
		"no header":    "",
		"wrong scheme": "Basic abc",
		"empty token":  "Bearer   ",
		"no space":     "Bearergood",
		"bad token":    "Bearer broken",
	} {
		t.Run(name, func(t *testing.T) {
			rec, seen := run(t, header)
			assertUnauthorized(t, rec)
			assert.Nil(t, seen)
		})
	}
}

func TestRequireAuthPassesIdentity(t *testing.T) {
	rec, seen := run(t, "bearer good")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, seen)
	assert.Equal(t, Identity{UserID: "u-1", Role: "intern"}, *seen)
}

func TestIdentityFromEmptyContext(t *testing.T) {
	_, ok := IdentityFrom(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	assert.False(t, ok)
}
