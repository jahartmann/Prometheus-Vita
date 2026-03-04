package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	maxRetries := 10
	var err error
	for i := range maxRetries {
		err = client.Ping(ctx).Err()
		if err == nil {
			slog.Info("connected to Redis")
			return client, nil
		}

		slog.Warn("failed to connect to Redis, retrying...",
			slog.Int("attempt", i+1),
			slog.Int("max_retries", maxRetries),
			slog.Any("error", err),
		)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to Redis after %d retries: %w", maxRetries, err)
}
