package front

import (
	"context"
	"encoding/json"
	"strings"

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
	PublicURL         string                 `json:"publicUrl"`
	Hero              siteHeroView           `json:"hero"`
	Announcement      siteAnnouncementView   `json:"announcement"`
	Announcements     []siteAnnouncementView `json:"announcements"`
	Features          []siteFeatureView      `json:"features"`
	Tagline           string                 `json:"tagline"`
	ThemeColor        string                 `json:"themeColor"`
	AllowRegistration bool                   `json:"allowRegistration"`
	OIDCEnabled       bool                   `json:"oidcEnabled"`
	// Wallet withdrawal hints let the storefront hide the form and show the
	// operator's instructions without a second request.
	WithdrawalEnabled      bool   `json:"withdrawalEnabled"`
	WithdrawalMinCents     int64  `json:"withdrawalMinCents"`
	WithdrawalInstructions string `json:"withdrawalInstructions"`
	// Wallet top-up bounds let the storefront build the form without a second
	// request, and keep the input inside what the API accepts.
	WalletTopUpEnabled  bool  `json:"walletTopUpEnabled"`
	WalletMinTopUpCents int64 `json:"walletMinTopUpCents"`
	WalletMaxTopUpCents int64 `json:"walletMaxTopUpCents"`
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

// siteAnnouncements returns the banner messages, in order. The list setting is
// the source of truth; the single-announcement setting is honoured when the list
// is empty, so an operator upgrading does not lose their message.
func (h *Handler) siteAnnouncements(ctx context.Context) []siteAnnouncementView {
	out := make([]siteAnnouncementView, 0, 4)
	if raw := strings.TrimSpace(h.Settings.String(ctx, "store.announcements")); raw != "" {
		var configured []siteAnnouncementView
		if err := json.Unmarshal([]byte(raw), &configured); err != nil {
			h.Logger.Warn("store.announcements is not valid JSON", "error", err)
		}
		for _, item := range configured {
			if strings.TrimSpace(item.Message) == "" {
				continue
			}
			out = append(out, siteAnnouncementView{
				Message: strings.TrimSpace(item.Message),
				URL:     strings.TrimSpace(item.URL),
			})
		}
	}
	if len(out) == 0 {
		if message := strings.TrimSpace(h.Settings.String(ctx, "store.announcement")); message != "" {
			out = append(out, siteAnnouncementView{
				Message: message,
				URL:     strings.TrimSpace(h.Settings.String(ctx, "store.announcement_url")),
			})
		}
	}
	return out
}

// siteFeatureViews returns only the feature cards that have content. An empty
// card would otherwise render as an empty box on the home page, which is what a
// half-configured store used to show.
func siteFeatureViews(ctx context.Context, h *Handler) []siteFeatureView {
	out := make([]siteFeatureView, 0, 3)
	for _, key := range []string{"1", "2", "3"} {
		title := strings.TrimSpace(h.Settings.String(ctx, "store.feature"+key+"_title"))
		text := strings.TrimSpace(h.Settings.String(ctx, "store.feature"+key+"_text"))
		if title == "" && text == "" {
			continue
		}
		out = append(out, siteFeatureView{Title: title, Text: text})
	}
	return out
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
	announcements := h.siteAnnouncements(ctx)
	firstAnnouncement := siteAnnouncementView{}
	if len(announcements) > 0 {
		firstAnnouncement = announcements[0]
	}
	response.OK(c, siteView{
		PublicURL: h.Settings.String(ctx, "store.public_url"),
		Hero: siteHeroView{
			Title:    h.Settings.String(ctx, "store.hero_title"),
			Subtitle: h.Settings.String(ctx, "store.hero_subtitle"),
			Image:    h.Settings.String(ctx, "store.hero_image_url"),
			CtaURL:   h.Settings.String(ctx, "store.hero_cta_url"),
		},
		Announcement:           firstAnnouncement,
		Announcements:          announcements,
		Features:               siteFeatureViews(ctx, h),
		Tagline:                h.Settings.String(ctx, "store.tagline"),
		ThemeColor:             h.Settings.String(ctx, "store.theme_color"),
		AllowRegistration:      h.Settings.Bool(ctx, "auth.allow_registration"),
		OIDCEnabled:            h.OIDC != nil && h.OIDC.Enabled(ctx),
		OIDCProviders:          h.oidcProviderViews(ctx),
		SocialProviders:        socialViews(ctx, h),
		WalletTopUpEnabled:     h.Settings.Bool(ctx, "wallet.enabled"),
		WalletMinTopUpCents:    int64(h.Settings.Int(ctx, "wallet.min_topup_cents")),
		WalletMaxTopUpCents:    int64(h.Settings.Int(ctx, "wallet.max_topup_cents")),
		WithdrawalEnabled:      h.Settings.Bool(ctx, "wallet.withdrawal_enabled"),
		WithdrawalMinCents:     int64(h.Settings.Int(ctx, "wallet.min_withdrawal_cents")),
		WithdrawalInstructions: h.Settings.String(ctx, "wallet.withdrawal_instructions"),
	})
}
