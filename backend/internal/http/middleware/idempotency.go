package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/port"
)

// idempotencyRecord is the cached outcome of a completed request.
type idempotencyRecord struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

// bodyCapture tees the response into a buffer so it can be cached.
type bodyCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyCapture) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Idempotency makes POST endpoints safe to retry. A client sends a unique
// Idempotency-Key header; the first response is cached in Redis and replayed
// for subsequent requests with the same key. Concurrent duplicates are rejected
// with 409 while the first request is still in flight.
//
// This complements the checkout lock: the lock protects against concurrent
// submissions, idempotency protects against client retries of a completed one.
func Idempotency(cache port.Cache, ttl time.Duration) gin.HandlerFunc {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		userID := UserID(c)
		if key == "" || userID == "" {
			c.Next()
			return
		}

		cacheKey := "idem:" + userID + ":" + key
		ctx := c.Request.Context()

		// Replay a previously stored response.
		if raw, err := cache.Get(ctx, cacheKey); err == nil {
			var rec idempotencyRecord
			if json.Unmarshal([]byte(raw), &rec) == nil {
				c.Header("Idempotent-Replay", "true")
				c.Data(rec.Status, "application/json; charset=utf-8", []byte(rec.Body))
				c.Abort()
				return
			}
		}

		// Reserve the key so concurrent duplicates do not both execute.
		lockKey := cacheKey + ":lock"
		if n, err := cache.Incr(ctx, lockKey, 1); err == nil {
			if n == 1 {
				_ = cache.Expire(ctx, lockKey, 30*time.Second)
			} else {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"error": gin.H{
						"code":    "request_in_progress",
						"message": "a request with this idempotency key is already in progress",
					},
				})
				return
			}
		}

		capture := &bodyCapture{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = capture
		c.Next()

		// Cache only successful outcomes; errors should be retryable.
		if capture.Status() >= 200 && capture.Status() < 300 && capture.body.Len() > 0 {
			rec := idempotencyRecord{Status: capture.Status(), Body: capture.body.String()}
			if raw, err := json.Marshal(rec); err == nil {
				_ = cache.Set(ctx, cacheKey, string(raw), ttl)
			}
			_ = cache.Delete(ctx, lockKey)
		} else {
			// Allow a retry after a failed attempt.
			_ = cache.Delete(ctx, lockKey)
		}
	}
}
