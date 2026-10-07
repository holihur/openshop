package handler

import "github.com/holihur/openshop/internal/domain"

type ReviewView struct {
	ID               string `json:"id"`
	ProductID        string `json:"productId"`
	UserID           string `json:"userId"`
	Rating           int    `json:"rating"`
	Title            string `json:"title"`
	Body             string `json:"body"`
	VerifiedPurchase bool   `json:"verifiedPurchase"`
	CreatedAt        string `json:"createdAt"`
}

func ToReviewView(r domain.Review) ReviewView {
	return ReviewView{
		ID: r.ID, ProductID: r.ProductID, UserID: r.UserID, Rating: r.Rating,
		Title: r.Title, Body: r.Body, VerifiedPurchase: r.VerifiedPurchase,
		CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
