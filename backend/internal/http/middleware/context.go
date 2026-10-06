package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const (
	ctxUserID    = "auth.userID"
	ctxRole      = "auth.role"
	ctxClaims    = "auth.claims"
	ctxRequestID = "request.id"
)

// UserID returns the authenticated user id, or "" when anonymous.
func UserID(c *gin.Context) string {
	if v, ok := c.Get(ctxUserID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Role returns the authenticated role.
func Role(c *gin.Context) domain.UserRole {
	if v, ok := c.Get(ctxRole); ok {
		if r, ok := v.(domain.UserRole); ok {
			return r
		}
	}
	return ""
}

// IsAdmin reports whether the caller has the admin role.
func IsAdmin(c *gin.Context) bool { return Role(c) == domain.RoleAdmin }

// Claims returns the verified token claims.
func Claims(c *gin.Context) *port.TokenClaims {
	if v, ok := c.Get(ctxClaims); ok {
		if cl, ok := v.(*port.TokenClaims); ok {
			return cl
		}
	}
	return nil
}

// RequestIDOf returns the per-request correlation id.
func RequestIDOf(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
