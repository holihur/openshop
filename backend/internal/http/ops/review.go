package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

// ListAllReviews powers the admin moderation view.
func (h *Handler) ListAllReviews(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	reviews, err := h.Reviews.ListAll(c.Request.Context(), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.ReviewView, 0, len(reviews.Items))
	for _, r := range reviews.Items {
		out = append(out, handler.ToReviewView(r))
	}
	response.Paginated(c, out, reviews.Total, reviews.Page, reviews.PageSize)
}

// DeleteReview removes any review (admin moderation).
func (h *Handler) DeleteReview(c *gin.Context) {
	if err := h.Reviews.Delete(c.Request.Context(), "", c.Param("id"), true); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "review.delete", "review", c.Param("id"), nil)
	response.NoContent(c)
}
