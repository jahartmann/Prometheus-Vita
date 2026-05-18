package notification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

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

func (s *EmailSender) Send(ctx context.Context, subject, body string) error {
	// Strip CR/LF from the subject to prevent SMTP header injection. An
	// attacker who can craft an alert subject could otherwise inject
	// additional headers (Bcc, MIME boundaries) by embedding "\r\n".
	subject = sanitiseSubject(subject)

	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.config.FromAddress,
		strings.Join(s.config.ToAddresses, ", "),
		subject,
		body,
	)

	d := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	tlsAvailable, _ := c.Extension("STARTTLS")
	if tlsAvailable {
		if err := c.StartTLS(&tls.Config{ServerName: s.config.SMTPHost}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	} else if s.config.SMTPUser != "" {
		// Hard-fail: if auth is configured but the server refuses STARTTLS,
		// we would otherwise transmit username + password in cleartext on
		// the open network. That's an unacceptable default. Operators who
		// genuinely need plain-auth (test/staging) can use a no-auth
		// channel.
		return fmt.Errorf("smtp server does not advertise STARTTLS but auth is configured — refusing to send credentials in plaintext")
	}

	if s.config.SMTPUser != "" {
		auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPass, s.config.SMTPHost)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := c.Mail(s.config.FromAddress); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	var rcptErrors []string
	var anyAccepted bool
	for _, to := range s.config.ToAddresses {
		if err := c.Rcpt(to); err != nil {
			rcptErrors = append(rcptErrors, fmt.Sprintf("%s: %v", to, err))
			continue
		}
		anyAccepted = true
	}
	if !anyAccepted {
		return fmt.Errorf("smtp rcpt: no recipient accepted (%s)", strings.Join(rcptErrors, "; "))
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}

// sanitiseSubject removes carriage returns and line feeds from a subject
// line to prevent SMTP header injection (CVE-class issue where an attacker
// who controls subject text could inject Bcc:, additional MIME parts, or
// terminate headers early).
func sanitiseSubject(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
