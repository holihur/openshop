package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/port"
)

// Metrics records request count and latency. The route pattern (c.FullPath) is
// used instead of the raw path to keep label cardinality bounded.
func Metrics(m port.Metrics) gin.HandlerFunc {
	if m == nil {
		m = port.NopMetrics{}
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		labels := map[string]string{
			"method": c.Request.Method,
			"route":  route,
			"status": strconv.Itoa(c.Writer.Status()),
		}
		m.Counter("openshop_http_requests_total", 1, labels)
		m.Histogram("openshop_http_request_duration_seconds", time.Since(start).Seconds(), map[string]string{
			"method": c.Request.Method,
			"route":  route,
		})
	}
}
