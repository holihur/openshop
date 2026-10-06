package postgres

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// WishlistRepository implements port.WishlistRepository.
type WishlistRepository struct{ db *DB }

func NewWishlistRepository(db *DB) *WishlistRepository { return &WishlistRepository{db: db} }

type wishlistModel struct {
	UserID    string    `gorm:"type:uuid;primaryKey"`
	ProductID string    `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time `gorm:"not null"`
}

func (wishlistModel) TableName() string { return "wishlists" }

func (r *WishlistRepository) Add(ctx context.Context, userID, productID string) error {
	m := &wishlistModel{UserID: userID, ProductID: productID, CreatedAt: time.Now().UTC()}
	return translate(r.db.session(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(m).Error)
}

func (r *WishlistRepository) Remove(ctx context.Context, userID, productID string) error {
	return translate(r.db.session(ctx).
		Delete(&wishlistModel{}, "user_id = ? AND product_id = ?", userID, productID).Error)
}

func (r *WishlistRepository) ListByUser(ctx context.Context, userID string) ([]domain.Product, error) {
	var models []productModel
	err := r.db.session(ctx).
		Where("id IN (?)", r.db.session(ctx).Model(&wishlistModel{}).
			Select("product_id").Where("user_id = ?", userID)).
		Order("created_at desc").
		Find(&models).Error
	if err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Product, 0, len(models))
	for i := range models {
		out = append(out, *toProduct(&models[i]))
	}
	return out, nil
}

var _ port.WishlistRepository = (*WishlistRepository)(nil)
