// Точка входа backend: читает настройки и запускает app.Run. Вся логика – в internal/.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ai-buddy/backend/internal/app"
	"ai-buddy/backend/internal/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Error("bad config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, log, nil); err != nil {
		log.Error("backend failed", "err", err)
		os.Exit(1)
	}
}
