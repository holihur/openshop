package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// UserRepository implements port.UserRepository.
type UserRepository struct{ db *DB }

func NewUserRepository(db *DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	m := fromUser(u)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	u.CreatedAt, u.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	m := fromUser(u)
	res := r.db.session(ctx).Model(&userModel{}).Where("id = ?", u.ID).
		Select("email", "phone", "password_hash", "name", "role", "status", "email_verified", "email_verified_at", "updated_at").
		Updates(m)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	res := r.db.session(ctx).Delete(&userModel{}, "id = ?", id)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var m userModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toUser(&m), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m userModel
	if err := r.db.session(ctx).First(&m, "lower(email) = lower(?)", email).Error; err != nil {
		return nil, translate(err)
	}
	return toUser(&m), nil
}

func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var m userModel
	if err := r.db.session(ctx).First(&m, "phone = ?", phone).Error; err != nil {
		return nil, translate(err)
	}
	return toUser(&m), nil
}

var _ port.UserRepository = (*UserRepository)(nil)
