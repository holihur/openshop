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
		"status":         string(o.Status),
		"payment_id":     nullableUUID(o.PaymentID),
		"paid_at":        o.PaidAt,
		"tracking_no":    o.TrackingNo,
		"shipped_at":     o.ShippedAt,
		"completed_at":   o.CompletedAt,
		"refunded_cents": o.RefundedCents,
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

func (r *OrderRepository) FindByAccessToken(ctx context.Context, token string) (*domain.Order, error) {
	var m orderModel
	if err := r.db.session(ctx).Preload("Items").First(&m, "access_token = ?", token).Error; err != nil {
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

// AnonymizeByUser detaches orders from a user for GDPR erasure, keeping the
// financial record but dropping identity and shipping data.
func (r *OrderRepository) AnonymizeByUser(ctx context.Context, userID string) error {
	return translate(r.db.session(ctx).Model(&orderModel{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"user_id":          nil,
			"shipping_address": nil,
			"guest_email":      "",
			"guest_phone":      "",
		}).Error)
}

// HasPurchasedProduct reports whether a user has a paid order containing the
// product. Used to mark reviews as verified purchases.
func (r *OrderRepository) HasPurchasedProduct(ctx context.Context, userID, productID string) (bool, error) {
	var count int64
	err := r.db.session(ctx).Model(&orderItemModel{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.user_id = ? AND order_items.product_id = ? AND orders.status IN ?",
			userID, productID, []string{
				string(domain.OrderPaid), string(domain.OrderShipped), string(domain.OrderCompleted),
			}).
		Count(&count).Error
	if err != nil {
		return false, translate(err)
	}
	return count > 0, nil
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
