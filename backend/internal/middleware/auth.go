// Package middleware – проверка токена до того, как запрос попадет в обработчик.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/services"
)

type ctxKey struct{}

// Identity – кто делает запрос. Кладется в context запроса.
type Identity struct {
	UserID string
	Role   string
}

// TokenParser – часть сервиса авторизации, нужная middleware.
type TokenParser interface {
	ParseToken(token string) (*services.Claims, error)
}

// RequireAuth пропускает запрос дальше, только если в заголовке Authorization валидный Bearer-токен.
// Иначе отвечает 401 { "error": { "code": "unauthorized" } }.
func RequireAuth(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing or malformed Authorization header")
				return
			}

			claims, err := parser.ParseToken(strings.TrimSpace(token))
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired token")
				return
			}

			ctx := WithIdentity(r.Context(), Identity{UserID: claims.Subject, Role: claims.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithIdentity кладет Identity в context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFrom достает Identity из context.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}
