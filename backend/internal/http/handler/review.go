package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type reviewView struct {
	ID               string `json:"id"`
	ProductID        string `json:"productId"`
	UserID           string `json:"userId"`
	Rating           int    `json:"rating"`
	Title            string `json:"title"`
	Body             string `json:"body"`
	VerifiedPurchase bool   `json:"verifiedPurchase"`
	CreatedAt        string `json:"createdAt"`
}

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
	page, size := parsePage(c, 10)
	reviews, err := h.Reviews.List(c.Request.Context(), c.Param("id"), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]reviewView, 0, len(reviews.Items))
	for _, r := range reviews.Items {
		out = append(out, toReviewView(r))
	}
	response.Paginated(c, out, reviews.Total, reviews.Page, reviews.PageSize)
}

// ListAllReviews powers the admin moderation view.
func (h *Handler) ListAllReviews(c *gin.Context) {
	page, size := parsePage(c, 20)
	reviews, err := h.Reviews.ListAll(c.Request.Context(), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]reviewView, 0, len(reviews.Items))
	for _, r := range reviews.Items {
		out = append(out, toReviewView(r))
	}
	response.Paginated(c, out, reviews.Total, reviews.Page, reviews.PageSize)
}

func (h *Handler) AddReview(c *gin.Context) {
	var req addReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
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
	response.Created(c, toReviewView(*review))
}

func (h *Handler) UpdateReview(c *gin.Context) {
	var req updateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	review, err := h.Reviews.Update(c.Request.Context(), middleware.UserID(c), c.Param("id"), service.UpdateReviewInput{
		Rating: req.Rating, Title: req.Title, Body: req.Body,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toReviewView(*review))
}

func (h *Handler) DeleteReview(c *gin.Context) {
	if err := h.Reviews.Delete(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	h.audit(c, "review.delete", "review", c.Param("id"), nil)
	response.NoContent(c)
}

func toReviewView(r domain.Review) reviewView {
	return reviewView{
		ID: r.ID, ProductID: r.ProductID, UserID: r.UserID, Rating: r.Rating,
		Title: r.Title, Body: r.Body, VerifiedPurchase: r.VerifiedPurchase,
		CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
