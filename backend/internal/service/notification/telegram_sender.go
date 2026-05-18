package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/antigravity/prometheus/internal/model"
)

var telegramHTTPClient = &http.Client{Timeout: 15 * time.Second}

type telegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type TelegramSender struct {
	config telegramConfig
}

func NewTelegramSender(rawConfig json.RawMessage) (*TelegramSender, error) {
	var cfg telegramConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("parse telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return nil, fmt.Errorf("telegram config requires bot_token and chat_id")
	}
	return &TelegramSender{config: cfg}, nil
}

func (s *TelegramSender) Type() model.NotificationChannelType {
	return model.ChannelTypeTelegram
}

// telegramMaxTextBytes is Telegram's hard message limit. Sending past it
// returns a confusing error from the API; truncating with a marker makes
// long bodies degrade gracefully.
const telegramMaxTextBytes = 4096

func (s *TelegramSender) Send(ctx context.Context, subject, body string) error {
	text := fmt.Sprintf("*%s*\n\n%s", subject, body)
	if len(text) > telegramMaxTextBytes {
		text = text[:telegramMaxTextBytes-20] + "\n\n... (gekürzt)"
	}

	payload := map[string]string{
		"chat_id":    s.config.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.config.BotToken)

	// Retry-with-exponential-backoff for transient errors (429 Too Many
	// Requests, 5xx). The Retry-After header on 429 is honoured up to a cap.
	// Hard-fail on 4xx (other than 429) since those are permanent
	// configuration errors — retrying won't fix them.
	const maxAttempts = 3
	backoff := 1 * time.Second
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("create telegram request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := telegramHTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send telegram message: %w", err)
			// Network error — retry.
			if !sleepWithCtx(ctx, backoff) {
				return lastErr
			}
			backoff *= 2
			continue
		}

		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil
		}

		switch {
		case resp.StatusCode == 429:
			retryAfter := backoff
			if s, ok := resp.Header["Retry-After"]; ok && len(s) > 0 {
				if v, err := strconv.Atoi(s[0]); err == nil && v > 0 && v <= 60 {
					retryAfter = time.Duration(v) * time.Second
				}
			}
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("telegram rate limit (429), retrying after %v", retryAfter)
			if !sleepWithCtx(ctx, retryAfter) {
				return lastErr
			}
		case resp.StatusCode >= 500:
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("telegram API %d: %s", resp.StatusCode, string(bodyBytes))
			if !sleepWithCtx(ctx, backoff) {
				return lastErr
			}
			backoff *= 2
		default:
			// 4xx other than 429: permanent — don't retry.
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			_ = resp.Body.Close()
			return fmt.Errorf("telegram API %d: %s", resp.StatusCode, string(bodyBytes))
		}
	}
	return lastErr
}

// sleepWithCtx sleeps for d but returns false if the context is cancelled
// first. The caller then aborts the retry loop.
func sleepWithCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
