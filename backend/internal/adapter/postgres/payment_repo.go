package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// PaymentRepository implements port.PaymentRepository.
type PaymentRepository struct{ db *DB }

func NewPaymentRepository(db *DB) *PaymentRepository { return &PaymentRepository{db: db} }

func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	m := fromPayment(p)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	p.CreatedAt, p.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *PaymentRepository) Update(ctx context.Context, p *domain.Payment) error {
	res := r.db.session(ctx).Model(&paymentModel{}).Where("id = ?", p.ID).Updates(map[string]any{
		"provider":       p.Provider,
		"provider_ref":   p.ProviderRef,
		"status":         string(p.Status),
		"failure_reason": p.FailureReason,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	var m paymentModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toPayment(&m), nil
}

func (r *PaymentRepository) FindByProviderRef(ctx context.Context, provider, ref string) (*domain.Payment, error) {
	var m paymentModel
	if err := r.db.session(ctx).First(&m, "provider = ? AND provider_ref = ?", provider, ref).Error; err != nil {
		return nil, translate(err)
	}
	return toPayment(&m), nil
}

var _ port.PaymentRepository = (*PaymentRepository)(nil)
