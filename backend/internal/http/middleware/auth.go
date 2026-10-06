package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// Auth verifies the bearer token and rejects revoked tokens. Verification is
// purely local (JWT) plus a Redis deny-list lookup, so no sticky sessions are
// required and any replica can authenticate any request.
func Auth(tokens port.TokenIssuer, auth *service.AuthService, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c)
		if raw == "" {
			if optional {
				c.Next()
				return
			}
			unauthorized(c, "missing token")
			return
		}
		claims, err := tokens.Verify(raw)
		if err != nil {
			unauthorized(c, "invalid or expired token")
			return
		}
		revoked, err := auth.IsRevoked(c.Request.Context(), claims.ID)
		if err != nil {
			unauthorized(c, "cannot validate token")
			return
		}
		if revoked {
			unauthorized(c, "token revoked")
			return
		}
		c.Set(ctxUserID, claims.Subject)
		c.Set(ctxRole, claims.Role)
		c.Set(ctxClaims, claims)
		c.Next()
	}
}

// RequireAdmin guards an endpoint behind the admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAdmin(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "forbidden", "message": "admin role required"},
			})
			return
		}
		c.Next()
	}
}

func unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "unauthorized", "message": msg},
	})
}
