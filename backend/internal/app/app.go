// Package app собирает все слои вместе: роутер, middleware, запуск и остановка сервера.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"ai-buddy/backend/internal/config"
	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/handlers"
	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/middleware"
	"ai-buddy/backend/internal/repository"
	"ai-buddy/backend/internal/services"
	"ai-buddy/backend/migrations"
)

// AuthService объединяет все, что роутеру нужно от сервиса авторизации.
type AuthService interface {
	handlers.AuthService
	middleware.TokenParser
}

// NewRouter описывает все маршруты API.
func NewRouter(cfg config.Config, auth AuthService, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(10 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	authH := handlers.NewAuth(auth, cfg.DemoMode, log)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handlers.Health)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Get("/test-users", authH.TestUsers)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(auth))
				r.Get("/me", authH.Me)
			})
		})
	})
	return r
}

// Run подключается к базе, применяет миграции и запускает HTTP-сервер до отмены ctx.
// Если ready не nil, в него отправляется адрес, на котором сервер начал слушать.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger, ready chan<- string) error {
	pool, err := database.Connect(ctx, cfg.DatabaseURL, 30*time.Second)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := database.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations done", "applied", applied)

	authSvc := services.NewAuth(repository.NewUsers(pool), cfg.JWTSecret, cfg.JWTTTL)

	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{
		Handler:           NewRouter(cfg, authSvc, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	log.Info("backend started", "addr", ln.Addr().String(), "demoMode", cfg.DemoMode)
	if ready != nil {
		ready <- ln.Addr().String()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("backend stopped")
	return nil
}
