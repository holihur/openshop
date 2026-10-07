package front

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
)

// RobotsTxt advertises the sitemap and allows crawling.
func (h *Handler) RobotsTxt(c *gin.Context) {
	body := "User-agent: *\nAllow: /\n"
	if h.SiteURL != "" {
		body += "Sitemap: " + strings.TrimRight(h.SiteURL, "/") + "/sitemap.xml\n"
	}
	c.Data(200, "text/plain; charset=utf-8", []byte(body))
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// Sitemap renders an XML sitemap of static pages and published products. It
// pages through the catalog so it works for large stores too.
func (h *Handler) Sitemap(c *gin.Context) {
	base := strings.TrimRight(h.SiteURL, "/")
	if base == "" {
		base = "http://localhost:5173"
	}

	set := urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, path := range []string{"", "/products"} {
		set.URLs = append(set.URLs, sitemapURL{Loc: base + path})
	}

	const pageSize = 100
	const maxProducts = 5000
	for page := 1; (page-1)*pageSize < maxProducts; page++ {
		published := domain.ProductPublished
		result, err := h.Catalog.ListProducts(c.Request.Context(), domain.ProductFilter{
			Status: &published, Page: page, PageSize: pageSize,
		})
		if err != nil {
			break
		}
		for _, p := range result.Items {
			set.URLs = append(set.URLs, sitemapURL{
				Loc:     fmt.Sprintf("%s/products/%s", base, p.ID),
				LastMod: p.UpdatedAt.UTC().Format("2006-01-02"),
			})
		}
		if int64(page*pageSize) >= result.Total || len(result.Items) == 0 {
			break
		}
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		c.Status(500)
		return
	}
	c.Data(200, "application/xml; charset=utf-8", append([]byte(xml.Header), body...))
}
