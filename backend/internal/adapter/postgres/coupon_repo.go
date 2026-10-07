package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CouponRepository implements port.CouponRepository.
type CouponRepository struct{ db *DB }

func NewCouponRepository(db *DB) *CouponRepository { return &CouponRepository{db: db} }

func (r *CouponRepository) Create(ctx context.Context, c *domain.Coupon) error {
	m := fromCoupon(c)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	c.CreatedAt, c.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *CouponRepository) Update(ctx context.Context, c *domain.Coupon) error {
	res := r.db.session(ctx).Model(&couponModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"description":        c.Description,
		"discount_value":     c.DiscountValue,
		"min_subtotal_cents": c.MinSubtotalCents,
		"max_discount_cents": c.MaxDiscountCents,
		"usage_limit":        c.UsageLimit,
		"per_user_limit":     c.PerUserLimit,
		"starts_at":          c.StartsAt,
		"ends_at":            c.EndsAt,
		"active":             c.Active,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CouponRepository) FindByID(ctx context.Context, id string) (*domain.Coupon, error) {
	var m couponModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toCoupon(&m), nil
}

func (r *CouponRepository) FindByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	var m couponModel
	if err := r.db.session(ctx).First(&m, "upper(code) = upper(?)", code).Error; err != nil {
		return nil, translate(err)
	}
	return toCoupon(&m), nil
}

func (r *CouponRepository) List(ctx context.Context, f domain.CouponFilter) (domain.Page[domain.Coupon], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&couponModel{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Coupon]{}, translate(err)
	}
	var models []couponModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Coupon]{}, translate(err)
	}
	out := make([]domain.Coupon, 0, len(models))
	for i := range models {
		out = append(out, *toCoupon(&models[i]))
	}
	return domain.Page[domain.Coupon]{Items: out, Total: total, Page: page, PageSize: size}, nil
}

// IncrementUsage atomically consumes one redemption. The WHERE guard makes the
// update a compare-and-set, so the global usage limit holds across replicas.
func (r *CouponRepository) IncrementUsage(ctx context.Context, id string) error {
	res := r.db.session(ctx).Model(&couponModel{}).
		Where("id = ? AND (usage_limit = 0 OR used_count < usage_limit)", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCouponExhausted
	}
	return nil
}

func (r *CouponRepository) CountRedemptions(ctx context.Context, couponID, userID string) (int64, error) {
	var count int64
	err := r.db.session(ctx).Model(&couponRedemptionModel{}).
		Where("coupon_id = ? AND user_id = ?", couponID, userID).
		Count(&count).Error
	if err != nil {
		return 0, translate(err)
	}
	return count, nil
}

func (r *CouponRepository) CreateRedemption(ctx context.Context, red *domain.CouponRedemption) error {
	m := &couponRedemptionModel{
		ID: red.ID, CouponID: red.CouponID, UserID: red.UserID,
		OrderID: red.OrderID, OrderNo: red.OrderNo, DiscountCents: red.DiscountCents,
		CreatedAt: red.CreatedAt,
	}
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

// ListRedemptions returns a coupon's usage history, newest first, with the
// redeeming customer's email joined in for display.
func (r *CouponRepository) ListRedemptions(ctx context.Context, couponID string, f domain.CouponFilter) (domain.Page[domain.CouponRedemption], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&couponRedemptionModel{}).Where("coupon_id = ?", couponID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.CouponRedemption]{}, translate(err)
	}
	var models []couponRedemptionModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.CouponRedemption]{}, translate(err)
	}

	userIDs := make([]string, 0, len(models))
	for _, m := range models {
		if m.UserID != "" {
			userIDs = append(userIDs, m.UserID)
		}
	}
	emails := map[string]string{}
	if len(userIDs) > 0 {
		var users []userModel
		if err := r.db.session(ctx).Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				emails[u.ID] = u.Email
			}
		}
	}

	out := make([]domain.CouponRedemption, 0, len(models))
	for _, m := range models {
		out = append(out, domain.CouponRedemption{
			ID: m.ID, CouponID: m.CouponID, UserID: m.UserID, OrderID: m.OrderID,
			OrderNo: m.OrderNo, DiscountCents: m.DiscountCents, CreatedAt: m.CreatedAt,
			UserEmail: emails[m.UserID],
		})
	}
	return domain.Page[domain.CouponRedemption]{Items: out, Total: total, Page: page, PageSize: size}, nil
}

var _ port.CouponRepository = (*CouponRepository)(nil)
