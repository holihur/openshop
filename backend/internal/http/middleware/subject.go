package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ctxSubject = "cart.subject"

// Subject returns the cart owner: the authenticated user id, or the guest id
// supplied via the X-Guest-Id header for anonymous shoppers.
func Subject(c *gin.Context) string {
	if v, ok := c.Get(ctxSubject); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ResolveSubject derives the cart owner from the request. It must run after an
// optional Auth middleware so a signed-in user always wins over a guest id.
func ResolveSubject() gin.HandlerFunc {
	return func(c *gin.Context) {
		subject := UserID(c)
		if subject == "" {
			subject = strings.TrimSpace(c.GetHeader("X-Guest-Id"))
		}
		c.Set(ctxSubject, subject)
		c.Next()
	}
}

// RequireSubject rejects requests with neither a user nor a guest id.
func RequireSubject() gin.HandlerFunc {
	return func(c *gin.Context) {
		if Subject(c) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "missing_cart_owner",
					"message": "sign in or send an X-Guest-Id header",
				},
			})
			return
		}
		c.Next()
	}
}
