// Package tracing provides an OpenTelemetry implementation of port.Tracer plus
// provider setup. When no OTLP endpoint is configured the tracer is a no-op, so
// the application carries no vendor dependency at runtime.
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/holihur/openshop/internal/port"
)

// Config controls tracing setup.
type Config struct {
	ServiceName string
	InstanceID  string
	// Endpoint is an OTLP/HTTP endpoint (host:port). Empty disables tracing.
	Endpoint    string
	Insecure    bool
	SampleRatio float64
}

// Setup installs a global tracer provider and returns the port.Tracer plus a
// shutdown function. With no endpoint it returns a NoopTracer and a no-op
// shutdown, so callers never special-case tracing.
func Setup(ctx context.Context, cfg Config) (port.Tracer, func(context.Context) error, error) {
	if cfg.Endpoint == "" {
		return port.NoopTracer{}, func(context.Context) error { return nil }, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("tracing: otlp exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.instance.id", cfg.InstanceID),
	))
	if err != nil {
		return nil, nil, fmt.Errorf("tracing: resource: %w", err)
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 1
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(3*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := NewOTel(tp.Tracer(cfg.ServiceName), propagation.TraceContext{})
	return tracer, tp.Shutdown, nil
}
