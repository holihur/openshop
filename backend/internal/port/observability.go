package port

import "context"

// Logger is the logging port. Application code never imports a concrete
// logging library, so the backend can swap slog/zap/zerolog freely.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
	With(kv ...any) Logger
}

// Metrics is the observability port. A no-op implementation is used when no
// exporter is configured so nothing binds to a specific vendor.
type Metrics interface {
	Counter(name string, delta float64, labels map[string]string)
	Histogram(name string, value float64, labels map[string]string)
	Gauge(name string, value float64, labels map[string]string)
}

// NopMetrics discards all observations.
type NopMetrics struct{}

func (NopMetrics) Counter(string, float64, map[string]string)   {}
func (NopMetrics) Histogram(string, float64, map[string]string) {}
func (NopMetrics) Gauge(string, float64, map[string]string)     {}

// TxManager runs a callback inside a storage transaction. It keeps multi-write
// operations (stock + order) atomic without leaking GORM/SQL into services.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
