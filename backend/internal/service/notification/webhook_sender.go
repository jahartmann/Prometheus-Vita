package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/antigravity/prometheus/internal/model"
)

var webhookHTTPClient = &http.Client{Timeout: 30 * time.Second}

type webhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Secret  string            `json:"secret"`
}

type WebhookSender struct {
	config webhookConfig
}

func NewWebhookSender(rawConfig json.RawMessage) (*WebhookSender, error) {
	var cfg webhookConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("parse webhook config: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("webhook config requires url")
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	return &WebhookSender{config: cfg}, nil
}

func (s *WebhookSender) Type() model.NotificationChannelType {
	return model.ChannelTypeWebhook
}

func (s *WebhookSender) Send(ctx context.Context, subject, body string) error {
	payload := map[string]string{
		"subject": subject,
		"body":    body,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, s.config.Method, s.config.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	if s.config.Secret != "" {
		mac := hmac.New(sha256.New, []byte(s.config.Secret))
		mac.Write(data)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature", signature)
	}

	// Retry transient failures (network errors, 5xx). 4xx is permanent
	// (bad URL, auth failure) — surface it immediately without backoff.
	const maxAttempts = 3
	backoff := 1 * time.Second
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Rewind the body for retry.
		req.Body = io.NopCloser(bytes.NewReader(data))

		resp, err := webhookHTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send webhook: %w", err)
			if !sleepWithCtx(ctx, backoff) {
				return lastErr
			}
			backoff *= 2
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		if resp.StatusCode < 400 {
			return nil
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
			if !sleepWithCtx(ctx, backoff) {
				return lastErr
			}
			backoff *= 2
			continue
		}
		// Permanent failure (4xx).
		return fmt.Errorf("webhook returned status %d (permanent)", resp.StatusCode)
	}
	return lastErr
}
