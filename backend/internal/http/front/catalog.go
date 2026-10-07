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
