package postgres

import (
	"context"
	"strings"
	"time"

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

func (r *UserRepository) RecordLoginFailure(ctx context.Context, userID string, lockAfter int, lockUntil time.Time) (int, error) {
	var attempts int
	row := r.db.session(ctx).Raw(
		`UPDATE users
		 SET failed_attempts = failed_attempts + 1,
		     locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END,
		     updated_at = now()
		 WHERE id = ?
		 RETURNING failed_attempts`, lockAfter, lockUntil, userID).Row()
	if err := row.Scan(&attempts); err != nil {
		return 0, translate(err)
	}
	return attempts, nil
}

func (r *UserRepository) ClearLoginFailures(ctx context.Context, userID string) error {
	return translate(r.db.session(ctx).Model(&userModel{}).Where("id = ?", userID).
		Updates(map[string]any{"failed_attempts": 0, "locked_until": nil}).Error)
}

func (r *UserRepository) List(ctx context.Context, f domain.UserFilter) (domain.Page[domain.User], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&userModel{})
	if f.Role != nil {
		q = q.Where("role = ?", string(*f.Role))
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("email ILIKE ? OR name ILIKE ? OR phone ILIKE ?", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.User]{}, translate(err)
	}
	var models []userModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.User]{}, translate(err)
	}
	out := make([]domain.User, 0, len(models))
	for i := range models {
		out = append(out, *toUser(&models[i]))
	}
	return domain.Page[domain.User]{Items: out, Total: total, Page: page, PageSize: size}, nil
}

var _ port.UserRepository = (*UserRepository)(nil)
