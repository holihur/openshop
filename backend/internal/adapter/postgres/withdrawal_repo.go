package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

type withdrawalModel struct {
	ID            string     `gorm:"type:uuid;primaryKey"`
	UserID        string     `gorm:"type:uuid;index;not null"`
	AmountCents   int64      `gorm:"not null"`
	Currency      string     `gorm:"size:8;not null;default:''"`
	Method        string     `gorm:"size:32;not null;default:'bank'"`
	AccountName   string     `gorm:"size:200;not null;default:''"`
	AccountNo     string     `gorm:"size:200;not null;default:''"`
	Note          string     `gorm:"size:512;not null;default:''"`
	Status        string     `gorm:"size:16;index;not null;default:'requested'"`
	RejectReason  string     `gorm:"size:512;not null;default:''"`
	PaidReference string     `gorm:"size:200;not null;default:''"`
	ReviewedBy    uuidString `gorm:"type:uuid"`
	ReviewedAt    *time.Time
	PaidAt        *time.Time
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (withdrawalModel) TableName() string { return "withdrawals" }

func toWithdrawal(m *withdrawalModel) *domain.Withdrawal {
	return &domain.Withdrawal{
		ID: m.ID, UserID: m.UserID, AmountCents: m.AmountCents, Currency: m.Currency,
		Method: m.Method, AccountName: m.AccountName, AccountNo: m.AccountNo, Note: m.Note,
		Status: domain.WithdrawalStatus(m.Status), RejectReason: m.RejectReason,
		PaidReference: m.PaidReference, ReviewedBy: string(m.ReviewedBy),
		ReviewedAt: m.ReviewedAt, PaidAt: m.PaidAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// WithdrawalRepository stores wallet withdrawal requests.
type WithdrawalRepository struct{ db *DB }

func NewWithdrawalRepository(db *DB) *WithdrawalRepository { return &WithdrawalRepository{db: db} }

func (r *WithdrawalRepository) Create(ctx context.Context, w *domain.Withdrawal) error {
	return translate(r.db.session(ctx).Create(&withdrawalModel{
		ID: w.ID, UserID: w.UserID, AmountCents: w.AmountCents, Currency: w.Currency,
		Method: w.Method, AccountName: w.AccountName, AccountNo: w.AccountNo, Note: w.Note,
		Status: string(w.Status), CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}).Error)
}

func (r *WithdrawalRepository) FindByID(ctx context.Context, id string) (*domain.Withdrawal, error) {
	var m withdrawalModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toWithdrawal(&m), nil
}

func (r *WithdrawalRepository) Update(ctx context.Context, w *domain.Withdrawal) error {
	res := r.db.session(ctx).Model(&withdrawalModel{}).Where("id = ?", w.ID).Updates(map[string]any{
		"status":         string(w.Status),
		"reject_reason":  w.RejectReason,
		"paid_reference": w.PaidReference,
		"reviewed_by":    uuidString(w.ReviewedBy),
		"reviewed_at":    w.ReviewedAt,
		"paid_at":        w.PaidAt,
		"updated_at":     w.UpdatedAt,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *WithdrawalRepository) List(ctx context.Context, f domain.WithdrawalFilter) (domain.Page[domain.Withdrawal], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&withdrawalModel{})
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Withdrawal]{}, translate(err)
	}
	var models []withdrawalModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Withdrawal]{}, translate(err)
	}
	items := make([]domain.Withdrawal, 0, len(models))
	for i := range models {
		items = append(items, *toWithdrawal(&models[i]))
	}
	return domain.Page[domain.Withdrawal]{Items: items, Total: total, Page: page, PageSize: size}, nil
}
