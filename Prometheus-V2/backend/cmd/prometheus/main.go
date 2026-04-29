package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/prometheus-v2/internal/config"
	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/antigravity/prometheus-v2/internal/platform/jobs"
	"github.com/antigravity/prometheus-v2/internal/platform/log"
	"github.com/antigravity/prometheus-v2/internal/platform/metrics"
	"github.com/antigravity/prometheus-v2/internal/platform/redis"
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

	if err := db.RunMigrations(cfg.DatabaseURL, "file://db/migrations"); err != nil {
		logger.Error("db migrations failed", slog.Any("error", err))
		os.Exit(1)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := jobs.MigrateUp(ctx, pool); err != nil {
		logger.Error("river migrations failed", slog.Any("error", err))
		os.Exit(1)
	}

	riverClient, err := jobs.NewClient(ctx, pool, jobs.NewWorkers(), jobs.DefaultQueues())
	if err != nil {
		logger.Error("river client init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		// Use a fresh context with a deadline so River drains in-flight jobs
		// even after the signal-cancel context fires, but never blocks forever.
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		if err := riverClient.Stop(stopCtx); err != nil {
			logger.Error("river client stop failed", slog.Any("error", err))
		}
	}()

	redisClient, err := redis.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("redis init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	reg := metrics.New()

	server := httpserver.NewServer(httpserver.Deps{
		Logger:  logger,
		DB:      pool,
		Redis:   redisClient,
		Metrics: reg,
	})

	if err := httpserver.ListenAndServe(ctx, server, cfg.HTTPAddr, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
