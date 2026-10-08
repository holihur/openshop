package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
)

// inferScope decides what a personal access token is allowed to do based on the
// route and the HTTP method. An unmapped resource must fall back to the wildcard
// scope, so a new endpoint is closed to tokens until it is mapped deliberately.
func TestInferScope(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   domain.Scope
	}{
		{http.MethodGet, "/api/v1/orders/:id", domain.ScopeOrdersRead},
		{http.MethodPost, "/api/v1/orders", domain.ScopeOrdersWrite},
		{http.MethodDelete, "/api/v1/orders/:id", domain.ScopeOrdersWrite},
		{http.MethodGet, "/api/v1/wallet", domain.ScopeWalletRead},
		{http.MethodPost, "/api/v1/wallet/topup", domain.ScopeWalletWrite},
		{http.MethodGet, "/api/v1/wallet/withdrawals", domain.ScopeWalletRead},
		{http.MethodGet, "/api/v1/points/transactions", domain.ScopePointsRead},
		{http.MethodGet, "/api/v1/referrals", domain.ScopeLoyaltyRead},
		{http.MethodGet, "/api/v1/referrals/commissions", domain.ScopeLoyaltyRead},
		{http.MethodGet, "/api/v1/auth/me", domain.ScopeProfileRead},
		{http.MethodDelete, "/api/v1/auth/me", domain.ScopeProfileWrite},
		{http.MethodPost, "/api/v1/payments", domain.ScopePaymentsWrite},
		{http.MethodGet, "/api/v1/cart", domain.ScopeCartRead},
		{http.MethodPatch, "/api/v1/addresses/:id", domain.ScopeAddressesWrite},
		{http.MethodGet, "/api/v1/tickets", domain.ScopeTicketsRead},
		{http.MethodGet, "/api/v1/notifications", domain.ScopeNotificationsRead},
		// Unmapped resource: only the wildcard scope may proceed.
		{http.MethodGet, "/api/v1/unknown-resource", domain.ScopeAll},
	}
	for _, tc := range cases {
		if got := inferScope(tc.method, tc.path); got != tc.want {
			t.Errorf("inferScope(%s, %s) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

// A browser session is never restricted by a token scope; a personal access
// token is.
func TestRequireScopeOnlyRestrictsTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		isPAT    bool
		scopes   domain.ScopeSet
		required domain.Scope
		want     int
	}{
		{"session passes", false, nil, domain.ScopeSettingsWrite, http.StatusOK},
		{"token with the scope passes", true, domain.ScopeSet{domain.ScopeOrdersRead}, domain.ScopeOrdersRead, http.StatusOK},
		{"token without the scope is refused", true, domain.ScopeSet{domain.ScopeOrdersRead}, domain.ScopeOrdersWrite, http.StatusForbidden},
		{"wildcard token passes", true, domain.ScopeSet{domain.ScopeAll}, domain.ScopeSettingsWrite, http.StatusOK},
		{"empty token scopes refuse everything", true, domain.ScopeSet{}, domain.ScopeOrdersRead, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.isPAT {
				c.Set(ctxIsPAT, true)
				c.Set(ctxScopes, tc.scopes)
			}
			RequireScope(tc.required)(c)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

// Managing tokens must require a real session, otherwise a leaked token could
// mint or revoke tokens (privilege escalation).
func TestRequireSessionRejectsTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/account/tokens", nil)
	c.Set(ctxIsPAT, true)
	c.Set(ctxScopes, domain.ScopeSet{domain.ScopeAll})
	RequireSession()(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("a personal access token must not manage tokens, got %d", w.Code)
	}
}
