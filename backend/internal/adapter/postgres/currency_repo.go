package postgres

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CurrencyRepository implements port.CurrencyRepository.
type CurrencyRepository struct{ db *DB }

func NewCurrencyRepository(db *DB) *CurrencyRepository { return &CurrencyRepository{db: db} }

type exchangeRateModel struct {
	Currency  string    `gorm:"size:8;primaryKey"`
	RateMicro int64     `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (exchangeRateModel) TableName() string { return "exchange_rates" }

func (r *CurrencyRepository) Upsert(ctx context.Context, currency string, rateMicro int64) error {
	m := &exchangeRateModel{Currency: currency, RateMicro: rateMicro, UpdatedAt: time.Now().UTC()}
	return translate(r.db.session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate_micro", "updated_at"}),
	}).Create(m).Error)
}

func (r *CurrencyRepository) List(ctx context.Context) ([]domain.ExchangeRate, error) {
	var models []exchangeRateModel
	if err := r.db.session(ctx).Order("currency asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.ExchangeRate, 0, len(models))
	for _, m := range models {
		out = append(out, domain.ExchangeRate{Currency: m.Currency, RateMicro: m.RateMicro, UpdatedAt: m.UpdatedAt})
	}
	return out, nil
}

func (r *CurrencyRepository) Find(ctx context.Context, currency string) (*domain.ExchangeRate, error) {
	var m exchangeRateModel
	if err := r.db.session(ctx).First(&m, "currency = ?", currency).Error; err != nil {
		return nil, translate(err)
	}
	return &domain.ExchangeRate{Currency: m.Currency, RateMicro: m.RateMicro, UpdatedAt: m.UpdatedAt}, nil
}

var _ port.CurrencyRepository = (*CurrencyRepository)(nil)
