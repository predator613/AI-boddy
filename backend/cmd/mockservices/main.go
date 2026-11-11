// Command mockservices – отдельный контейнер с имитацией корпоративной системы заявок.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"ai-buddy/backend/internal/mockservices"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	delay, err := time.ParseDuration(env("MOCK_DECISION_DELAY", "45s"))
	if err != nil {
		log.Error("MOCK_DECISION_DELAY must be a duration like 45s", "err", err)
		os.Exit(1)
	}
	addr := ":" + env("PORT", "8090")
	log.Info("mock-services started", "addr", addr, "decisionDelay", delay.String())
	srv := &http.Server{Addr: addr, Handler: mockservices.Handler(mockservices.NewStore(delay)), ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("mock-services stopped", "err", err)
		os.Exit(1)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
