package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/antigravity/prometheus/internal/model"
)

type emailConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	SMTPUser    string   `json:"smtp_user"`
	SMTPPass    string   `json:"smtp_password"`
	FromAddress string   `json:"from_address"`
	ToAddresses []string `json:"to_addresses"`
}

type EmailSender struct {
	config emailConfig
}

func NewEmailSender(rawConfig json.RawMessage) (*EmailSender, error) {
	var cfg emailConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("parse email config: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.FromAddress == "" || len(cfg.ToAddresses) == 0 {
		return nil, fmt.Errorf("email config requires smtp_host, from_address, and to_addresses")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	return &EmailSender{config: cfg}, nil
}

func (s *EmailSender) Type() model.NotificationChannelType {
	return model.ChannelTypeEmail
}

func (s *EmailSender) Send(_ context.Context, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.config.FromAddress,
		strings.Join(s.config.ToAddresses, ", "),
		subject,
		body,
	)

	var auth smtp.Auth
	if s.config.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPass, s.config.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.config.FromAddress, s.config.ToAddresses, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
