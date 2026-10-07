package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type addReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type updateReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func (h *Handler) ListReviews(c *gin.Context) {
	page, size := handler.ParsePage(c, 10)
	reviews, err := h.Reviews.List(c.Request.Context(), c.Param("id"), page, size)
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

func (h *Handler) AddReview(c *gin.Context) {
	var req addReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	review, err := h.Reviews.Add(c.Request.Context(), service.AddReviewInput{
		UserID: middleware.UserID(c), ProductID: c.Param("id"),
		Rating: req.Rating, Title: req.Title, Body: req.Body,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToReviewView(*review))
}

func (h *Handler) UpdateReview(c *gin.Context) {
	var req updateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	review, err := h.Reviews.Update(c.Request.Context(), middleware.UserID(c), c.Param("id"), service.UpdateReviewInput{
		Rating: req.Rating, Title: req.Title, Body: req.Body,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToReviewView(*review))
}

func (h *Handler) DeleteReview(c *gin.Context) {
	if err := h.Reviews.Delete(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "review.delete", "review", c.Param("id"), nil)
	response.NoContent(c)
}
