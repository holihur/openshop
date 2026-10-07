package domain

import "time"

// ProductFAQ is a question/answer pair shown on a product page.
type ProductFAQ struct {
	ID        string
	ProductID string
	Question  string
	Answer    string
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
}
