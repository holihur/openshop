package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/holihur/openshop/internal/port"
)

// OTel implements port.Tracer with OpenTelemetry.
type OTel struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func NewOTel(tracer trace.Tracer, propagator propagation.TextMapPropagator) *OTel {
	return &OTel{tracer: tracer, propagator: propagator}
}

func (o *OTel) Start(ctx context.Context, name string, attrs ...port.Attribute) (context.Context, port.Span) {
	ctx, span := o.tracer.Start(ctx, name, trace.WithAttributes(toAttributes(attrs)...))
	return ctx, &otelSpan{span: span}
}

func (o *OTel) Inject(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	o.propagator.Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

func (o *OTel) Extract(ctx context.Context, traceParent string) context.Context {
	if traceParent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{"traceparent": traceParent}
	return o.propagator.Extract(ctx, carrier)
}

type otelSpan struct {
	span trace.Span
}

func (s *otelSpan) SetAttributes(attrs ...port.Attribute) {
	s.span.SetAttributes(toAttributes(attrs)...)
}
func (s *otelSpan) RecordError(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
}
func (s *otelSpan) End() { s.span.End() }

func toAttributes(attrs []port.Attribute) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		switch v := a.Value.(type) {
		case string:
			out = append(out, attribute.String(a.Key, v))
		case bool:
			out = append(out, attribute.Bool(a.Key, v))
		case int:
			out = append(out, attribute.Int(a.Key, v))
		case int64:
			out = append(out, attribute.Int64(a.Key, v))
		case float64:
			out = append(out, attribute.Float64(a.Key, v))
		default:
			out = append(out, attribute.String(a.Key, ""))
		}
	}
	return out
}

var _ port.Tracer = (*OTel)(nil)
