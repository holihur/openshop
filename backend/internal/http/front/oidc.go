package front

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

const oidcStatePrefix = "oidc:state:"

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// oidcProvider resolves the provider for a request: the explicit ?provider=
// wins, otherwise the default (first configured) provider is used.
func (h *Handler) oidcProvider(c *gin.Context) string {
	ctx := c.Request.Context()
	if id := strings.TrimSpace(c.Query("provider")); id != "" {
		return id
	}
	if h.OIDC == nil {
		return ""
	}
	return h.OIDC.DefaultProviderID(ctx)
}

// OIDCStart begins the authorization-code flow for one provider: it stores a
// CSRF state token (mapped to the provider id) and redirects the browser.
func (h *Handler) OIDCStart(c *gin.Context) {
	ctx := c.Request.Context()
	if h.OIDC == nil || !h.OIDC.Enabled(ctx) {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	providerID := h.oidcProvider(c)
	state, err := randomState()
	if err != nil {
		response.Fail(c, err)
		return
	}
	_ = h.Cache.Set(ctx, oidcStatePrefix+state, providerID, 10*time.Minute)
	redirect, err := h.OIDC.AuthCodeURL(ctx, providerID, state)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Redirect(302, redirect)
}

// OIDCCallback completes the flow: it validates the state, exchanges the code
// for a verified identity, issues our own session and hands the tokens back to
// the SPA via the URL fragment (a fragment is never sent to a server).
func (h *Handler) OIDCCallback(c *gin.Context) {
	ctx := c.Request.Context()
	if h.OIDC == nil || !h.OIDC.Enabled(ctx) {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	state := c.Query("state")
	if state == "" {
		response.Fail(c, domain.ErrUnauthorized)
		return
	}
	// The provider id stored with the state is authoritative, so a tampered
	// ?provider= cannot redirect the code exchange to another provider.
	providerID, err := h.Cache.Get(ctx, oidcStatePrefix+state)
	if err != nil {
		response.Fail(c, domain.ErrUnauthorized)
		return
	}
	_ = h.Cache.Delete(ctx, oidcStatePrefix+state)

	identity, err := h.OIDC.Exchange(ctx, providerID, c.Query("code"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.Auth.OIDC(ctx, identity)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "auth.oidc_login", "user", result.User.ID, nil)

	base := strings.TrimRight(h.Settings.String(ctx, "store.public_url"), "/")
	fragment := url.Values{}
	fragment.Set("access_token", result.AccessToken)
	fragment.Set("refresh_token", result.RefreshToken)
	fragment.Set("expires_in", strconv.FormatInt(result.ExpiresIn, 10))
	c.Redirect(302, base+"/oidc/callback#"+fragment.Encode())
}
