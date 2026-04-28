package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/antigravity/prometheus-v2/internal/config"
	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/antigravity/prometheus-v2/internal/platform/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger := log.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	server := httpserver.NewServer(httpserver.Deps{
		Logger: logger,
		DB:     nil,
		Redis:  nil,
	})

	if err := httpserver.ListenAndServe(ctx, server, cfg.HTTPAddr, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
