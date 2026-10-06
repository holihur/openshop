package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestOTelPropagatesTraceContext verifies that a trace context survives the
// async hop: the traceparent injected by the producer is extracted by the
// consumer and the resulting span is a child of the producer's span. This is the
// mechanism that stitches checkout and the outbox consumer into one trace.
func TestOTelPropagatesTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tr := NewOTel(tp.Tracer("test"), propagation.TraceContext{})

	parentCtx, parentSpan := tr.Start(context.Background(), "order.checkout")
	parentID := parentSpan.(*otelSpan).span.SpanContext().SpanID()
	parentSpan.End()

	// The producer injects a traceparent that is persisted with the outbox event.
	traceParent := tr.Inject(parentCtx)
	if traceParent == "" {
		t.Fatal("expected a non-empty traceparent")
	}

	// The relay/consumer extracts it and starts a child span.
	childCtx := tr.Extract(context.Background(), traceParent)
	_, childSpan := tr.Start(childCtx, "consume order.created")
	child := childSpan.(*otelSpan).span.SpanContext()
	childSpan.End()

	if child.TraceID() != parentSpan.(*otelSpan).span.SpanContext().TraceID() {
		t.Fatal("child span is not part of the same trace")
	}

	_ = tp.ForceFlush(context.Background())
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("exported %d spans, want 2", len(spans))
	}
	if spans[1].Parent.SpanID() != parentID {
		t.Fatalf("child parent span id = %s, want %s", spans[1].Parent.SpanID(), parentID)
	}
}

func TestOTelExtractEmptyIsNoop(t *testing.T) {
	tr := NewOTel(nil, propagation.TraceContext{})
	ctx := context.Background()
	if got := tr.Extract(ctx, ""); got != ctx {
		t.Fatal("empty traceparent should return the context unchanged")
	}
}
