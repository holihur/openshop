package service

import (
	"context"
	"fmt"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const reviewSummaryKeyPrefix = "review:summary:"

// ReviewService manages product reviews and a cached rating summary.
type ReviewService struct {
	reviews  port.ReviewRepository
	products port.ProductRepository
	cache    port.Cache
	ids      port.IDGenerator
	clock    port.Clock
}

func NewReviewService(
	reviews port.ReviewRepository,
	products port.ProductRepository,
	cache port.Cache,
	ids port.IDGenerator,
	clock port.Clock,
) *ReviewService {
	return &ReviewService{reviews: reviews, products: products, cache: cache, ids: ids, clock: clock}
}

type AddReviewInput struct {
	UserID    string
	ProductID string
	Rating    int
	Title     string
	Body      string
}

func (s *ReviewService) Add(ctx context.Context, in AddReviewInput) (*domain.Review, error) {
	if !domain.ValidRating(in.Rating) {
		return nil, fmt.Errorf("%w: rating must be between 1 and 5", domain.ErrInvalidArgument)
	}
	if _, err := s.products.FindByID(ctx, in.ProductID); err != nil {
		return nil, err
	}
	if _, err := s.reviews.FindByUserAndProduct(ctx, in.UserID, in.ProductID); err == nil {
		return nil, fmt.Errorf("%w: you have already reviewed this product", domain.ErrConflict)
	} else if err != domain.ErrNotFound {
		return nil, err
	}

	now := s.clock.Now()
	review := &domain.Review{
		ID: s.ids.NewID(), ProductID: in.ProductID, UserID: in.UserID,
		Rating: in.Rating, Title: in.Title, Body: in.Body,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.reviews.Create(ctx, review); err != nil {
		return nil, err
	}
	s.invalidateSummary(ctx, in.ProductID)
	return review, nil
}

type UpdateReviewInput struct {
	Rating int
	Title  string
	Body   string
}

func (s *ReviewService) Update(ctx context.Context, userID, reviewID string, in UpdateReviewInput) (*domain.Review, error) {
	if !domain.ValidRating(in.Rating) {
		return nil, fmt.Errorf("%w: rating must be between 1 and 5", domain.ErrInvalidArgument)
	}
	review, err := s.reviews.FindByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if review.UserID != userID {
		return nil, domain.ErrNotFound
	}
	review.Rating = in.Rating
	review.Title = in.Title
	review.Body = in.Body
	review.UpdatedAt = s.clock.Now()
	if err := s.reviews.Update(ctx, review); err != nil {
		return nil, err
	}
	s.invalidateSummary(ctx, review.ProductID)
	return review, nil
}

func (s *ReviewService) Delete(ctx context.Context, userID, reviewID string, isAdmin bool) error {
	review, err := s.reviews.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if !isAdmin && review.UserID != userID {
		return domain.ErrNotFound
	}
	if err := s.reviews.Delete(ctx, reviewID); err != nil {
		return err
	}
	s.invalidateSummary(ctx, review.ProductID)
	return nil
}

func (s *ReviewService) List(ctx context.Context, productID string, page, size int) (domain.Page[domain.Review], error) {
	page, size = clampPage(page, size, 10)
	return s.reviews.ListByProduct(ctx, domain.ReviewFilter{ProductID: productID, Page: page, PageSize: size})
}

// Summary returns the cached aggregate rating, computing it on a miss.
func (s *ReviewService) Summary(ctx context.Context, productID string) (domain.ReviewSummary, error) {
	var cached domain.ReviewSummary
	if err := s.cache.GetJSON(ctx, reviewSummaryKeyPrefix+productID, &cached); err == nil {
		return cached, nil
	}
	summary, err := s.reviews.Summary(ctx, productID)
	if err != nil {
		return domain.ReviewSummary{}, err
	}
	_ = s.cache.SetJSON(ctx, reviewSummaryKeyPrefix+productID, summary, 5*time.Minute)
	return summary, nil
}

func (s *ReviewService) invalidateSummary(ctx context.Context, productID string) {
	_ = s.cache.Delete(ctx, reviewSummaryKeyPrefix+productID)
}
