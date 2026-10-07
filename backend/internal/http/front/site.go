package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/response"
)

type siteHeroView struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Image    string `json:"image"`
	CtaURL   string `json:"ctaUrl"`
}

type siteView struct {
	PublicURL    string               `json:"publicUrl"`
	Hero         siteHeroView         `json:"hero"`
	Announcement siteAnnouncementView `json:"announcement"`
	Features     []siteFeatureView    `json:"features"`
	Tagline      string               `json:"tagline"`
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
		Tagline: h.Settings.String(ctx, "store.tagline"),
	})
}
