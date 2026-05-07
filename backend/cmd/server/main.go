package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"restaurantops/backend/internal/application"
	"restaurantops/backend/internal/config"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	settings, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	app, err := application.New(ctx, settings)
	if err != nil {
		slog.Error("initialization failed", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Run(ctx); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}
