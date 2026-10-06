package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// OrderRepository implements port.OrderRepository.
type OrderRepository struct{ db *DB }

func NewOrderRepository(db *DB) *OrderRepository { return &OrderRepository{db: db} }

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	m := fromOrder(o)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	o.CreatedAt, o.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *OrderRepository) Update(ctx context.Context, o *domain.Order) error {
	res := r.db.session(ctx).Model(&orderModel{}).Where("id = ?", o.ID).Updates(map[string]any{
		"status":     string(o.Status),
		"payment_id": nullableUUID(o.PaymentID),
		"paid_at":    o.PaidAt,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	var m orderModel
	if err := r.db.session(ctx).Preload("Items").First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toOrder(&m), nil
}

func (r *OrderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	var m orderModel
	if err := r.db.session(ctx).Preload("Items").First(&m, "order_no = ?", orderNo).Error; err != nil {
		return nil, translate(err)
	}
	return toOrder(&m), nil
}

func (r *OrderRepository) FindExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error) {
	if limit < 1 {
		limit = 100
	}
	var models []orderModel
	err := r.db.session(ctx).Preload("Items").
		Where("status = ? AND expires_at < ?", string(domain.OrderPendingPayment), now).
		Order("expires_at asc").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Order, 0, len(models))
	for i := range models {
		out = append(out, *toOrder(&models[i]))
	}
	return out, nil
}

func (r *OrderRepository) List(ctx context.Context, f domain.OrderFilter) (domain.Page[domain.Order], error) {
	page, size := normalizePage(f.Page, f.PageSize, 10)

	q := r.db.session(ctx).Model(&orderModel{})
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Order]{}, translate(err)
	}

	var models []orderModel
	if err := q.Preload("Items").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Order]{}, translate(err)
	}
	items := make([]domain.Order, 0, len(models))
	for i := range models {
		items = append(items, *toOrder(&models[i]))
	}
	return domain.Page[domain.Order]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

var _ port.OrderRepository = (*OrderRepository)(nil)
