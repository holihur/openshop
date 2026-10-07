package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
)

// RefundRepository stores partial refunds.
type RefundRepository struct{ db *DB }

func NewRefundRepository(db *DB) *RefundRepository { return &RefundRepository{db: db} }

func (r *RefundRepository) Create(ctx context.Context, refund *domain.Refund) error {
	return translate(r.db.session(ctx).Create(fromRefund(refund)).Error)
}

func (r *RefundRepository) ListByOrder(ctx context.Context, orderID string) ([]domain.Refund, error) {
	var models []refundModel
	if err := r.db.session(ctx).Where("order_id = ?", orderID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Refund, 0, len(models))
	for i := range models {
		out = append(out, *toRefund(&models[i]))
	}
	return out, nil
}

// ReturnRepository stores return requests (RMA).
type ReturnRepository struct{ db *DB }

func NewReturnRepository(db *DB) *ReturnRepository { return &ReturnRepository{db: db} }

func (r *ReturnRepository) Create(ctx context.Context, req *domain.ReturnRequest) error {
	return translate(r.db.session(ctx).Create(fromReturn(req)).Error)
}

func (r *ReturnRepository) FindByID(ctx context.Context, id string) (*domain.ReturnRequest, error) {
	var m returnModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toReturn(&m), nil
}

func (r *ReturnRepository) ListByOrder(ctx context.Context, orderID string) ([]domain.ReturnRequest, error) {
	var models []returnModel
	if err := r.db.session(ctx).Where("order_id = ?", orderID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.ReturnRequest, 0, len(models))
	for i := range models {
		out = append(out, *toReturn(&models[i]))
	}
	return out, nil
}

func (r *ReturnRepository) List(ctx context.Context, f domain.ReturnFilter) (domain.Page[domain.ReturnRequest], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&returnModel{})
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.ReturnRequest]{}, translate(err)
	}
	var models []returnModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.ReturnRequest]{}, translate(err)
	}
	items := make([]domain.ReturnRequest, 0, len(models))
	for i := range models {
		items = append(items, *toReturn(&models[i]))
	}
	return domain.Page[domain.ReturnRequest]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (r *ReturnRepository) Update(ctx context.Context, req *domain.ReturnRequest) error {
	res := r.db.session(ctx).Model(&returnModel{}).Where("id = ?", req.ID).Updates(map[string]any{
		"status":     string(req.Status),
		"updated_at": req.UpdatedAt,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
