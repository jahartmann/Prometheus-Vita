package llm

import (
	"context"
	"log/slog"
	"time"
)

// CompleteWithRetry calls provider.Complete with retry logic for transient errors.
func CompleteWithRetry(ctx context.Context, provider Provider, req CompletionRequest, maxRetries int) (*CompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			slog.Warn("retrying LLM completion",
				slog.Int("attempt", attempt),
				slog.Any("last_error", lastErr),
			)
		}
		resp, err := provider.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
