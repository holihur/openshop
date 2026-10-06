// Package logger provides a slog-backed implementation of port.Logger. It is
// the only place the standard logging package is referenced; all other packages
// depend on the port interface.
package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/holihur/openshop/internal/port"
)

type Slog struct {
	l *slog.Logger
}

// New builds a structured logger. JSON is used in production, text locally.
func New(level string, production bool, service string, fields ...any) *Slog {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if production {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	base := slog.New(h).With("service", service)
	if len(fields) > 0 {
		base = base.With(fields...)
	}
	return &Slog{l: base}
}

func (s *Slog) Debug(msg string, kv ...any) { s.l.Debug(msg, kv...) }
func (s *Slog) Info(msg string, kv ...any)  { s.l.Info(msg, kv...) }
func (s *Slog) Warn(msg string, kv ...any)  { s.l.Warn(msg, kv...) }
func (s *Slog) Error(msg string, kv ...any) { s.l.Error(msg, kv...) }

func (s *Slog) With(kv ...any) port.Logger {
	return &Slog{l: s.l.With(kv...)}
}

var _ port.Logger = (*Slog)(nil)
