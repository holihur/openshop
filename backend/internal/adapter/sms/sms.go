// Package sms contains port.SMS adapters.
package sms

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// Log is the development adapter: it records messages instead of sending.
type Log struct {
	logger port.Logger
	sign   string
}

func NewLog(logger port.Logger, sign string) *Log { return &Log{logger: logger, sign: sign} }

func (l *Log) Send(_ context.Context, phone, template string, params map[string]string) error {
	l.logger.Info("sms sent (log driver)", "phone", phone, "template", template, "params", params)
	return nil
}

var _ port.SMS = (*Log)(nil)

// Aliyun is a placeholder showing how a real provider slots in behind the port.
type Aliyun struct {
	sign string
}

func (a *Aliyun) Send(_ context.Context, phone, template string, params map[string]string) error {
	return fmt.Errorf("sms aliyun: not configured")
}

// New selects the SMS driver declared in configuration.
func New(cfg config.SMSConfig, logger port.Logger) port.SMS {
	switch cfg.Driver {
	case "aliyun":
		return &Aliyun{sign: cfg.Sign}
	default:
		return NewLog(logger, cfg.Sign)
	}
}
