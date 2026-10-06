package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ReviewRepository implements port.ReviewRepository.
type ReviewRepository struct{ db *DB }

func NewReviewRepository(db *DB) *ReviewRepository { return &ReviewRepository{db: db} }

func (r *ReviewRepository) Create(ctx context.Context, rev *domain.Review) error {
	m := fromReview(rev)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	rev.CreatedAt, rev.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *ReviewRepository) Update(ctx context.Context, rev *domain.Review) error {
	res := r.db.session(ctx).Model(&reviewModel{}).Where("id = ?", rev.ID).Updates(map[string]any{
		"rating": rev.Rating,
		"title":  rev.Title,
		"body":   rev.Body,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ReviewRepository) Delete(ctx context.Context, id string) error {
	res := r.db.session(ctx).Delete(&reviewModel{}, "id = ?", id)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ReviewRepository) FindByID(ctx context.Context, id string) (*domain.Review, error) {
	var m reviewModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toReview(&m), nil
}

func (r *ReviewRepository) FindByUserAndProduct(ctx context.Context, userID, productID string) (*domain.Review, error) {
	var m reviewModel
	if err := r.db.session(ctx).First(&m, "user_id = ? AND product_id = ?", userID, productID).Error; err != nil {
		return nil, translate(err)
	}
	return toReview(&m), nil
}

func (r *ReviewRepository) ListByProduct(ctx context.Context, f domain.ReviewFilter) (domain.Page[domain.Review], error) {
	return r.list(ctx, f, true)
}

func (r *ReviewRepository) List(ctx context.Context, f domain.ReviewFilter) (domain.Page[domain.Review], error) {
	return r.list(ctx, f, false)
}

func (r *ReviewRepository) list(ctx context.Context, f domain.ReviewFilter, requireProduct bool) (domain.Page[domain.Review], error) {
	page, size := normalizePage(f.Page, f.PageSize, 10)
	q := r.db.session(ctx).Model(&reviewModel{})
	if f.ProductID != "" {
		q = q.Where("product_id = ?", f.ProductID)
	} else if requireProduct {
		return domain.Page[domain.Review]{}, domain.ErrInvalidArgument
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Review]{}, translate(err)
	}

	var models []reviewModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Review]{}, translate(err)
	}
	items := make([]domain.Review, 0, len(models))
	for i := range models {
		items = append(items, *toReview(&models[i]))
	}
	return domain.Page[domain.Review]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (r *ReviewRepository) Summary(ctx context.Context, productID string) (domain.ReviewSummary, error) {
	var row struct {
		Count   int64
		Average float64
	}
	err := r.db.session(ctx).Model(&reviewModel{}).
		Select("count(*) as count, coalesce(avg(rating), 0) as average").
		Where("product_id = ?", productID).
		Scan(&row).Error
	if err != nil {
		return domain.ReviewSummary{}, translate(err)
	}
	return domain.ReviewSummary{Count: row.Count, Average: row.Average}, nil
}

var _ port.ReviewRepository = (*ReviewRepository)(nil)
