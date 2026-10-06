package port

import "context"

// Attribute is a key/value annotation attached to a span. Values may be string,
// bool, int, int64 or float64.
type Attribute struct {
	Key   string
	Value any
}

// Span is an in-progress unit of work.
type Span interface {
	SetAttributes(attrs ...Attribute)
	RecordError(err error)
	End()
}

// Tracer abstracts distributed tracing. Inject/Extract propagate W3C trace
// context across process boundaries — including the asynchronous outbox, so a
// checkout and the worker that reacts to it share one trace.
type Tracer interface {
	Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span)
	// Inject returns a W3C traceparent for the current context ("" if none).
	Inject(ctx context.Context) string
	// Extract continues a trace from a traceparent.
	Extract(ctx context.Context, traceParent string) context.Context
}

// NoopTracer discards all tracing. It is the default when no exporter is
// configured, so nothing binds to a vendor.
type NoopTracer struct{}

func (NoopTracer) Start(ctx context.Context, _ string, _ ...Attribute) (context.Context, Span) {
	return ctx, noopSpan{}
}
func (NoopTracer) Inject(context.Context) string                         { return "" }
func (NoopTracer) Extract(ctx context.Context, _ string) context.Context { return ctx }

type noopSpan struct{}

func (noopSpan) SetAttributes(...Attribute) {}
func (noopSpan) RecordError(error)          {}
func (noopSpan) End()                       {}
