package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/port"
)

// RequestID assigns a correlation id from a proxy header or a fresh one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = c.GetHeader("X-Correlation-Id")
		}
		if id == "" {
			id = generateID()
		}
		c.Set(ctxRequestID, id)
		c.Writer.Header().Set("X-Request-Id", id)
		c.Next()
	}
}

// Logger records one structured line per request.
func Logger(log port.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latencyMs", time.Since(start).Milliseconds(),
			"requestId", RequestIDOf(c),
			"clientIp", c.ClientIP(),
		)
	}
}

// Recovery converts panics into 500 responses instead of crashing the process.
func Recovery(log port.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "error", r, "path", c.Request.URL.Path, "requestId", RequestIDOf(c))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "internal_error", "message": "something went wrong"},
				})
			}
		}()
		c.Next()
	}
}

// CORS allows the configured SPA origins. It is written by hand to avoid
// binding the API to a third-party middleware and to support credentials.
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	wildcard := false
	for _, o := range origins {
		if o == "*" {
			wildcard = true
		}
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok || wildcard {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-Id")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RateLimit implements a sliding-window limiter backed by the shared rate
// limiter, so limits are enforced across all replicas rather than per instance.
// At most rps requests are allowed in any trailing one-second window. The limit
// is resolved per request so it can be changed at runtime from the ops console.
func RateLimit(limiter port.RateLimiter, rps func(ctx context.Context) int) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		if rps != nil {
			if v := rps(c.Request.Context()); v > 0 {
				limit = v
			}
		}
		allowed, retryAfter, err := limiter.Allow(c.Request.Context(), "ratelimit:ip:"+c.ClientIP(), limit, time.Second)
		if err == nil && !allowed {
			rejectRateLimited(c, retryAfter)
			return
		}
		c.Next()
	}
}

// RateLimitUser enforces a per-authenticated-user sliding window using the
// shared rate limiter. It runs after Auth so it can key on the user id,
// complementing the per-IP limiter that protects unauthenticated traffic.
func RateLimitUser(limiter port.RateLimiter, rps func(ctx context.Context) int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := UserID(c)
		if userID == "" {
			c.Next()
			return
		}
		limit := 100
		if rps != nil {
			if v := rps(c.Request.Context()); v > 0 {
				limit = v
			}
		}
		allowed, retryAfter, err := limiter.Allow(c.Request.Context(), "ratelimit:user:"+userID, limit, time.Second)
		if err == nil && !allowed {
			rejectRateLimited(c, retryAfter)
			return
		}
		c.Next()
	}
}

// rejectRateLimited aborts with 429 and a Retry-After hint (at least one
// second, since the header has one-second resolution).
func rejectRateLimited(c *gin.Context, retryAfter time.Duration) {
	secs := int(retryAfter.Seconds())
	if secs < 1 {
		secs = 1
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{"code": "rate_limited", "message": "too many requests"},
	})
}

// generateID is a tiny helper kept local to avoid an extra import cycle.
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.Itoa(randInt())
}

func randInt() int {
	return int(time.Now().UnixNano() % 1_000_000)
}

// bearer extracts a token from the Authorization header.
func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
