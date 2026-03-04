package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/antigravity/prometheus/internal/model"
)

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

func (s *TelegramSender) Send(ctx context.Context, subject, body string) error {
	text := fmt.Sprintf("*%s*\n\n%s", subject, body)

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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}
	return nil
}
