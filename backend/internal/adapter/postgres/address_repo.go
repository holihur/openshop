package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AddressRepository implements port.AddressRepository.
type AddressRepository struct{ db *DB }

func NewAddressRepository(db *DB) *AddressRepository { return &AddressRepository{db: db} }

func (r *AddressRepository) Create(ctx context.Context, a *domain.Address) error {
	m := fromAddress(a)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	a.CreatedAt, a.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *AddressRepository) Update(ctx context.Context, a *domain.Address) error {
	res := r.db.session(ctx).Model(&addressModel{}).Where("id = ?", a.ID).Updates(map[string]any{
		"recipient":   a.Recipient,
		"phone":       a.Phone,
		"province":    a.Province,
		"city":        a.City,
		"district":    a.District,
		"line1":       a.Line1,
		"postal_code": a.PostalCode,
		"is_default":  a.Default,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AddressRepository) Delete(ctx context.Context, id string) error {
	res := r.db.session(ctx).Delete(&addressModel{}, "id = ?", id)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AddressRepository) FindByID(ctx context.Context, id string) (*domain.Address, error) {
	var m addressModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toAddress(&m), nil
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]domain.Address, error) {
	var models []addressModel
	if err := r.db.session(ctx).Where("user_id = ?", userID).
		Order("is_default desc, created_at desc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Address, 0, len(models))
	for i := range models {
		out = append(out, *toAddress(&models[i]))
	}
	return out, nil
}

func (r *AddressRepository) ClearDefault(ctx context.Context, userID string) error {
	return translate(r.db.session(ctx).Model(&addressModel{}).
		Where("user_id = ? AND is_default", userID).
		Update("is_default", false).Error)
}

var _ port.AddressRepository = (*AddressRepository)(nil)
