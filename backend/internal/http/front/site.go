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
	PublicURL string       `json:"publicUrl"`
	Hero      siteHeroView `json:"hero"`
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
	})
}
