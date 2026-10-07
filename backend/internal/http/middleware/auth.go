package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// Auth verifies the bearer credential and rejects revoked ones. Two credential
// kinds are accepted: a session JWT (browser) and a personal access token
// (programmatic). Verification of a JWT is purely local plus a Redis deny-list
// lookup, so no sticky sessions are required and any replica can authenticate
// any request. PATs additionally carry scopes and a CIDR allow-list, and are
// bound to a realm so a storefront token cannot reach the operations API.
func Auth(tokens port.TokenIssuer, auth *service.AuthService, pats *service.PATService, realm string, optional bool) gin.HandlerFunc {
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

		if pats != nil && strings.HasPrefix(raw, service.PATPrefix) {
			token, role, err := pats.Authenticate(c.Request.Context(), raw, c.ClientIP(), realm)
			if err != nil {
				// A token that is valid but not usable from here is a permission
				// problem, not an authentication one.
				if errors.Is(err, domain.ErrForbidden) {
					forbidden(c, "this token is not allowed from your address")
					return
				}
				unauthorized(c, "invalid or expired access token")
				return
			}
			c.Set(ctxUserID, token.UserID)
			c.Set(ctxRole, role)
			c.Set(ctxScopes, domain.ScopeSet(token.Scopes))
			c.Set(ctxIsPAT, true)
			c.Next()
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

// RequireOps guards an endpoint behind an operations role.
func RequireOps() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !domain.IsOpsRole(Role(c)) {
			forbidden(c, "operations role required")
			return
		}
		c.Next()
	}
}

// RequirePermission guards an endpoint behind a fine-grained permission. It must
// run after Auth so the role is available on the context. A personal access
// token must additionally grant the matching scope, so a token can never do
// more than its scope list, and never more than its owner's role.
func RequirePermission(perm domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !domain.HasPermission(Role(c), perm) {
			forbidden(c, "missing permission: "+string(perm))
			return
		}
		if !AllowsScope(c, domain.Scope(perm)) {
			forbidden(c, "missing scope: "+string(perm))
			return
		}
		c.Next()
	}
}

// RequireScope guards an endpoint behind one explicit token scope. Browser
// sessions are unaffected; only personal access tokens are restricted.
func RequireScope(scope domain.Scope) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !AllowsScope(c, scope) {
			forbidden(c, "missing scope: "+string(scope))
			return
		}
		c.Next()
	}
}

// frontScopeResources maps the first segment of a storefront API path to the
// scope resource it belongs to. Unknown resources require the wildcard scope,
// so a new endpoint is denied to tokens until it is mapped deliberately.
var frontScopeResources = map[string]string{
	"orders":        "orders",
	"payments":      "payments",
	"cart":          "cart",
	"addresses":     "addresses",
	"wishlist":      "wishlist",
	"wallet":        "wallet",
	"points":        "points",
	"referrals":     "loyalty",
	"tickets":       "tickets",
	"notifications": "notifications",
	"reviews":       "reviews",
	"products":      "products",
	"coupons":       "coupons",
	"auth":          "profile",
}

// InferScope enforces a scope for personal-access-token requests, derived from
// the route path and the HTTP method (GET reads, anything else writes). Browser
// sessions pass through untouched, so no route needs a per-endpoint guard.
func InferScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsPAT(c) {
			c.Next()
			return
		}
		if required := inferScope(c.Request.Method, c.FullPath()); !AllowsScope(c, required) {
			forbidden(c, "missing scope: "+string(required))
			return
		}
		c.Next()
	}
}

func inferScope(method, fullPath string) domain.Scope {
	path := strings.TrimPrefix(fullPath, "/api/v1")
	path = strings.TrimPrefix(path, "/ops")
	path = strings.TrimPrefix(path, "/")
	segment := strings.SplitN(path, "/", 2)[0]

	resource, ok := frontScopeResources[segment]
	if !ok {
		return domain.ScopeAll
	}
	action := "read"
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		action = "write"
	}
	return domain.Scope(resource + ":" + action)
}

// RequireSession rejects personal access tokens, so a leaked token cannot mint
// or revoke tokens (privilege escalation).
func RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsPAT(c) {
			forbidden(c, "this endpoint requires a browser session")
			return
		}
		c.Next()
	}
}

// RequireAdmin guards an endpoint behind the admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAdmin(c) {
			forbidden(c, "admin role required")
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

func forbidden(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{"code": "forbidden", "message": msg},
	})
}
