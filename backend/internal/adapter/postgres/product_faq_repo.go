package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// ProductFAQRepository implements port.ProductFAQRepository.
type ProductFAQRepository struct{ db *DB }

func NewProductFAQRepository(db *DB) *ProductFAQRepository { return &ProductFAQRepository{db: db} }

func (r *ProductFAQRepository) ListByProduct(ctx context.Context, productID string) ([]domain.ProductFAQ, error) {
	var models []productFAQModel
	if err := r.db.session(ctx).Where("product_id = ?", productID).
		Order("sort asc, created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.ProductFAQ, 0, len(models))
	for i := range models {
		out = append(out, *toProductFAQ(&models[i]))
	}
	return out, nil
}

func (r *ProductFAQRepository) Replace(ctx context.Context, productID string, faqs []domain.ProductFAQ) error {
	return r.db.WithinTx(ctx, func(txCtx context.Context) error {
		if err := r.db.session(txCtx).Where("product_id = ?", productID).
			Delete(&productFAQModel{}).Error; err != nil {
			return translate(err)
		}
		if len(faqs) == 0 {
			return nil
		}
		rows := make([]productFAQModel, 0, len(faqs))
		now := time.Now().UTC()
		for i := range faqs {
			f := faqs[i]
			rows = append(rows, productFAQModel{
				ID: f.ID, ProductID: productID, Question: f.Question, Answer: f.Answer,
				Sort: f.Sort, CreatedAt: now, UpdatedAt: now,
			})
		}
		if err := r.db.session(txCtx).Create(&rows).Error; err != nil {
			return translate(err)
		}
		return nil
	})
}
