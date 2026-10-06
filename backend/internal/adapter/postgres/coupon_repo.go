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

func (r *CouponRepository) List(ctx context.Context) ([]domain.Coupon, error) {
	var models []couponModel
	if err := r.db.session(ctx).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Coupon, 0, len(models))
	for i := range models {
		out = append(out, *toCoupon(&models[i]))
	}
	return out, nil
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
		OrderID: red.OrderID, CreatedAt: red.CreatedAt,
	}
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

var _ port.CouponRepository = (*CouponRepository)(nil)
