package domain

import "time"

// Review is a product rating left by a user. A user may review a product once.
type Review struct {
	ID               string
	ProductID        string
	UserID           string
	Rating           int // 1..5
	Title            string
	Body             string
	VerifiedPurchase bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ReviewSummary is the aggregate shown on the product page.
type ReviewSummary struct {
	Count   int64
	Average float64
}

// ReviewFilter is a storage-agnostic query for a product's reviews.
type ReviewFilter struct {
	ProductID string
	Page      int
	PageSize  int
}

func ValidRating(r int) bool { return r >= 1 && r <= 5 }
