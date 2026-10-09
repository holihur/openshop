package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

func (h *Handler) GetProduct(c *gin.Context) {
	p, err := h.Catalog.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	view := handler.ToProductView(*p)
	if h.Reviews != nil {
		if summary, err := h.Reviews.Summary(c.Request.Context(), p.ID); err == nil {
			view.Rating = summary.Average
			view.ReviewCount = summary.Count
		}
	}
	response.OK(c, view)
}

// ListProductFacets returns the price range and variant attributes available in
// a category, so the storefront can offer the filters that actually apply.
func (h *Handler) ListProductFacets(c *gin.Context) {
	facets, err := h.Catalog.Facets(c.Request.Context(), c.Query("categoryId"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, facets)
}
