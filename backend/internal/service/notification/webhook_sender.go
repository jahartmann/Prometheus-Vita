package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/antigravity/prometheus/internal/model"
)

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
