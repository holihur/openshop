package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/port"
)

// Tracing starts a server span per request, continuing an inbound W3C trace
// context when present, and records the outcome.
func Tracing(t port.Tracer) gin.HandlerFunc {
	if t == nil {
		t = port.NoopTracer{}
	}
	return func(c *gin.Context) {
		ctx := t.Extract(c.Request.Context(), c.GetHeader("traceparent"))
		ctx, span := t.Start(ctx, c.Request.Method+" "+c.Request.URL.Path,
			port.Attribute{Key: "http.method", Value: c.Request.Method},
			port.Attribute{Key: "http.target", Value: c.Request.URL.Path},
			port.Attribute{Key: "request.id", Value: RequestIDOf(c)},
			port.Attribute{Key: "client.ip", Value: c.ClientIP()},
		)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		span.SetAttributes(
			port.Attribute{Key: "http.status_code", Value: c.Writer.Status()},
			port.Attribute{Key: "http.route", Value: c.FullPath()},
		)
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last())
		}
		span.End()
	}
}
