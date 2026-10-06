// Package mail contains port.Mailer adapters.
package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// Log writes emails to the logger. Useful in development and tests.
type Log struct {
	logger port.Logger
}

func NewLog(logger port.Logger) *Log { return &Log{logger: logger} }

func (l *Log) Send(_ context.Context, msg port.Email) error {
	l.logger.Info("email sent (log driver)",
		"to", msg.To, "subject", msg.Subject)
	return nil
}

var _ port.Mailer = (*Log)(nil)

// SMTP sends mail through an SMTP relay using only the standard library.
type SMTP struct {
	cfg config.MailConfig
}

func NewSMTP(cfg config.MailConfig) *SMTP { return &SMTP{cfg: cfg} }

func (s *SMTP) Send(_ context.Context, msg port.Email) error {
	if s.cfg.Host == "" {
		return fmt.Errorf("mail smtp: host is not configured")
	}
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)
	}

	body := buildMessage(s.cfg.From, msg)
	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, body); err != nil {
		return fmt.Errorf("mail smtp send: %w", err)
	}
	return nil
}

func buildMessage(from string, msg port.Email) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	if msg.HTML != "" {
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	b.WriteString("\r\n")
	if msg.HTML != "" {
		b.WriteString(msg.HTML)
	} else {
		b.WriteString(msg.Text)
	}
	return []byte(b.String())
}

// New selects the mail driver declared in configuration.
func New(cfg config.MailConfig, logger port.Logger) port.Mailer {
	if cfg.Driver == "smtp" {
		return NewSMTP(cfg)
	}
	return NewLog(logger)
}
