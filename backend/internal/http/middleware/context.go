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
	ctxScopes    = "auth.scopes"
	ctxIsPAT     = "auth.pat"
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

// IsPAT reports whether the request authenticated with a personal access token
// rather than a browser session.
func IsPAT(c *gin.Context) bool {
	if v, ok := c.Get(ctxIsPAT); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// Scopes returns the granted scopes of the calling token. Session callers have
// no scope list and are never restricted by one.
func Scopes(c *gin.Context) domain.ScopeSet {
	if v, ok := c.Get(ctxScopes); ok {
		if s, ok := v.(domain.ScopeSet); ok {
			return s
		}
	}
	return nil
}

// AllowsScope reports whether the caller may exercise a scope. A browser
// session is unrestricted; a token must grant the scope explicitly.
func AllowsScope(c *gin.Context, required domain.Scope) bool {
	if !IsPAT(c) {
		return true
	}
	return Scopes(c).Allows(required)
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
