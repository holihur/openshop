package front

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type siteHeroView struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Image    string `json:"image"`
	CtaURL   string `json:"ctaUrl"`
}

type siteView struct {
	PublicURL         string               `json:"publicUrl"`
	Hero              siteHeroView         `json:"hero"`
	Announcement      siteAnnouncementView `json:"announcement"`
	Features          []siteFeatureView    `json:"features"`
	Tagline           string               `json:"tagline"`
	ThemeColor        string               `json:"themeColor"`
	AllowRegistration bool                 `json:"allowRegistration"`
	OIDCEnabled       bool                 `json:"oidcEnabled"`
	// Wallet withdrawal hints let the storefront hide the form and show the
	// operator's instructions without a second request.
	WithdrawalEnabled      bool   `json:"withdrawalEnabled"`
	WithdrawalMinCents     int64  `json:"withdrawalMinCents"`
	WithdrawalInstructions string `json:"withdrawalInstructions"`
	// OIDCProviders lists the single sign-on buttons to render.
	OIDCProviders []oidcProviderView `json:"oidcProviders"`
	// SocialProviders lists the WeChat/Alipay sign-in buttons to render. An
	// unconfigured or switched-off provider is absent rather than broken.
	SocialProviders []domain.SocialProviderInfo `json:"socialProviders"`
}

type oidcProviderView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// socialViews lists the configured WeChat/Alipay sign-in providers.
func socialViews(ctx context.Context, h *Handler) []domain.SocialProviderInfo {
	if h.Social == nil {
		return []domain.SocialProviderInfo{}
	}
	return h.Social.Providers(ctx)
}

// oidcProviderViews adapts the service's provider list to the API shape.
func (h *Handler) oidcProviderViews(ctx context.Context) []oidcProviderView {
	if h.OIDC == nil {
		return nil
	}
	providers := h.OIDC.Providers(ctx)
	out := make([]oidcProviderView, 0, len(providers))
	for _, p := range providers {
		out = append(out, oidcProviderView{ID: p.ID, Name: p.Name})
	}
	return out
}

type siteFeatureView struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type siteAnnouncementView struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

// GetSite exposes the public, non-sensitive site configuration the storefront
// renders (banner copy, public URL). It reads runtime settings, so operators
// can change the homepage from the ops console without a redeploy.
func (h *Handler) GetSite(c *gin.Context) {
	ctx := c.Request.Context()
	response.OK(c, siteView{
		PublicURL: h.Settings.String(ctx, "store.public_url"),
		Hero: siteHeroView{
			Title:    h.Settings.String(ctx, "store.hero_title"),
			Subtitle: h.Settings.String(ctx, "store.hero_subtitle"),
			Image:    h.Settings.String(ctx, "store.hero_image_url"),
			CtaURL:   h.Settings.String(ctx, "store.hero_cta_url"),
		},
		Announcement: siteAnnouncementView{
			Message: h.Settings.String(ctx, "store.announcement"),
			URL:     h.Settings.String(ctx, "store.announcement_url"),
		},
		Features: []siteFeatureView{
			{Title: h.Settings.String(ctx, "store.feature1_title"), Text: h.Settings.String(ctx, "store.feature1_text")},
			{Title: h.Settings.String(ctx, "store.feature2_title"), Text: h.Settings.String(ctx, "store.feature2_text")},
			{Title: h.Settings.String(ctx, "store.feature3_title"), Text: h.Settings.String(ctx, "store.feature3_text")},
		},
		Tagline:                h.Settings.String(ctx, "store.tagline"),
		ThemeColor:             h.Settings.String(ctx, "store.theme_color"),
		AllowRegistration:      h.Settings.Bool(ctx, "auth.allow_registration"),
		OIDCEnabled:            h.OIDC != nil && h.OIDC.Enabled(ctx),
		OIDCProviders:          h.oidcProviderViews(ctx),
		SocialProviders:        socialViews(ctx, h),
		WithdrawalEnabled:      h.Settings.Bool(ctx, "wallet.withdrawal_enabled"),
		WithdrawalMinCents:     int64(h.Settings.Int(ctx, "wallet.min_withdrawal_cents")),
		WithdrawalInstructions: h.Settings.String(ctx, "wallet.withdrawal_instructions"),
	})
}
