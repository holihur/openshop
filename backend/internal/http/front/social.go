package front

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

const socialStatePrefix = "social:state:"

// ListSocialProviders reports the sign-in providers a shopper can use, so the
// sign-in page only shows buttons that work.
func (h *Handler) ListSocialProviders(c *gin.Context) {
	if h.Social == nil {
		response.OK(c, []domain.SocialProviderInfo{})
		return
	}
	response.OK(c, h.Social.Providers(c.Request.Context()))
}

// SocialStart begins the authorization-code flow: it stores a CSRF state token
// mapped to the provider id and redirects the browser to the provider.
func (h *Handler) SocialStart(c *gin.Context) {
	ctx := c.Request.Context()
	if h.Social == nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	providerID := strings.TrimSpace(c.Param("provider"))
	state, err := randomState()
	if err != nil {
		response.Fail(c, err)
		return
	}
	_ = h.Cache.Set(ctx, socialStatePrefix+state, providerID, 10*time.Minute)
	redirect, err := h.Social.AuthCodeURL(ctx, providerID, state)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Redirect(302, redirect)
}

// SocialCallback completes the flow. The provider id stored with the state is
// authoritative, so a tampered ?provider= cannot redirect the code exchange.
func (h *Handler) SocialCallback(c *gin.Context) {
	ctx := c.Request.Context()
	if h.Social == nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	state := c.Query("state")
	if state == "" {
		response.Fail(c, domain.ErrUnauthorized)
		return
	}
	providerID, err := h.Cache.Get(ctx, socialStatePrefix+state)
	if err != nil {
		response.Fail(c, domain.ErrUnauthorized)
		return
	}
	_ = h.Cache.Delete(ctx, socialStatePrefix+state)

	identity, err := h.Social.Exchange(ctx, providerID, c.Query("code"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.Auth.Social(ctx, identity)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "auth.social_login", "user", result.User.ID,
		map[string]string{"provider": providerID})

	// Tokens are handed back in the URL fragment, which a browser never sends to
	// a server, so they do not end up in access logs or referrers.
	base := strings.TrimRight(h.Settings.String(ctx, "store.public_url"), "/")
	fragment := url.Values{}
	fragment.Set("access_token", result.AccessToken)
	fragment.Set("refresh_token", result.RefreshToken)
	fragment.Set("expires_in", strconv.FormatInt(result.ExpiresIn, 10))
	c.Redirect(302, base+"/oidc/callback#"+fragment.Encode())
}
