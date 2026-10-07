package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

type walletModel struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	UserID       string    `gorm:"type:uuid;uniqueIndex;not null"`
	Currency     string    `gorm:"size:8;not null"`
	BalanceCents int64     `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (walletModel) TableName() string { return "wallets" }

type walletTxModel struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	WalletID      string    `gorm:"type:uuid;index;not null"`
	UserID        string    `gorm:"type:uuid;index;not null"`
	Type          string    `gorm:"size:32;index;not null"`
	AmountCents   int64     `gorm:"not null"`
	BalanceAfter  int64     `gorm:"not null"`
	ReferenceType string    `gorm:"size:32;not null;default:''"`
	ReferenceID   string    `gorm:"size:64;not null;default:''"`
	Description   string    `gorm:"size:512;not null;default:''"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (walletTxModel) TableName() string { return "wallet_transactions" }

type pointsAccountModel struct {
	ID             string    `gorm:"type:uuid;primaryKey"`
	UserID         string    `gorm:"type:uuid;uniqueIndex;not null"`
	Balance        int64     `gorm:"not null;default:0"`
	LifetimeEarned int64     `gorm:"not null;default:0"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (pointsAccountModel) TableName() string { return "points_accounts" }

type pointsTxModel struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	UserID        string    `gorm:"type:uuid;index;not null"`
	Type          string    `gorm:"size:32;index;not null"`
	Points        int64     `gorm:"not null"`
	BalanceAfter  int64     `gorm:"not null"`
	ReferenceType string    `gorm:"size:32;not null;default:''"`
	ReferenceID   string    `gorm:"size:64;not null;default:''"`
	Description   string    `gorm:"size:512;not null;default:''"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (pointsTxModel) TableName() string { return "points_transactions" }

type referralCodeModel struct {
	UserID    string    `gorm:"type:uuid;primaryKey"`
	Code      string    `gorm:"size:32;uniqueIndex;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (referralCodeModel) TableName() string { return "referral_codes" }

type referralModel struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	ReferrerID string    `gorm:"type:uuid;index;not null"`
	RefereeID  string    `gorm:"type:uuid;uniqueIndex;not null"`
	Code       string    `gorm:"size:32;not null;default:''"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (referralModel) TableName() string { return "referrals" }

type commissionModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	ReferrerID  string     `gorm:"type:uuid;index;not null"`
	RefereeID   string     `gorm:"type:uuid;index;not null"`
	OrderID     string     `gorm:"type:uuid;uniqueIndex;not null"`
	BaseCents   int64      `gorm:"not null"`
	RateBps     int        `gorm:"not null"`
	AmountCents int64      `gorm:"not null"`
	Status      string     `gorm:"size:16;index;not null;default:'pending'"`
	HoldUntil   time.Time  `gorm:"not null"`
	ApprovedAt  *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null"`
}

func (commissionModel) TableName() string { return "commissions" }

func toWallet(m *walletModel) *domain.Wallet {
	return &domain.Wallet{
		ID: m.ID, UserID: m.UserID, Currency: m.Currency, BalanceCents: m.BalanceCents,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toWalletTx(m *walletTxModel) domain.WalletTransaction {
	return domain.WalletTransaction{
		ID: m.ID, WalletID: m.WalletID, UserID: m.UserID, Type: domain.WalletTransactionType(m.Type),
		AmountCents: m.AmountCents, BalanceAfter: m.BalanceAfter, ReferenceType: m.ReferenceType,
		ReferenceID: m.ReferenceID, Description: m.Description, CreatedAt: m.CreatedAt,
	}
}

func toCommission(m *commissionModel) domain.Commission {
	return domain.Commission{
		ID: m.ID, ReferrerID: m.ReferrerID, RefereeID: m.RefereeID, OrderID: m.OrderID,
		BaseCents: m.BaseCents, RateBps: m.RateBps, AmountCents: m.AmountCents,
		Status: domain.CommissionStatus(m.Status), HoldUntil: m.HoldUntil, ApprovedAt: m.ApprovedAt,
		CreatedAt: m.CreatedAt,
	}
}

// WalletRepository stores stored-value balances and their ledger.
type WalletRepository struct{ db *DB }

func NewWalletRepository(db *DB) *WalletRepository { return &WalletRepository{db: db} }

func (r *WalletRepository) Ensure(ctx context.Context, userID, currency string) (*domain.Wallet, error) {
	if currency == "" {
		currency = "USD"
	}
	if err := translate(r.db.session(ctx).Exec(
		`INSERT INTO wallets (id, user_id, currency, balance_cents, created_at, updated_at)
		 VALUES (gen_random_uuid(), ?, ?, 0, now(), now())
		 ON CONFLICT (user_id) DO NOTHING`, userID, currency).Error); err != nil {
		return nil, err
	}
	return r.FindByUser(ctx, userID)
}

func (r *WalletRepository) FindByUser(ctx context.Context, userID string) (*domain.Wallet, error) {
	var m walletModel
	if err := r.db.session(ctx).First(&m, "user_id = ?", userID).Error; err != nil {
		return nil, translate(err)
	}
	return toWallet(&m), nil
}

func (r *WalletRepository) AddBalance(ctx context.Context, userID string, delta int64, allowNegative bool) (int64, error) {
	query := `UPDATE wallets SET balance_cents = balance_cents + ?, updated_at = now() WHERE user_id = ?`
	args := []any{delta, userID}
	if delta < 0 && !allowNegative {
		query += ` AND balance_cents + ? >= 0`
		args = append(args, delta)
	}
	query += ` RETURNING balance_cents`

	var balance int64
	row := r.db.session(ctx).Raw(query, args...).Row()
	if err := row.Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrInsufficientFunds
		}
		return 0, translate(err)
	}
	return balance, nil
}

func (r *WalletRepository) AddTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	return translate(r.db.session(ctx).Create(&walletTxModel{
		ID: tx.ID, WalletID: tx.WalletID, UserID: tx.UserID, Type: string(tx.Type),
		AmountCents: tx.AmountCents, BalanceAfter: tx.BalanceAfter, ReferenceType: tx.ReferenceType,
		ReferenceID: tx.ReferenceID, Description: tx.Description, CreatedAt: tx.CreatedAt,
	}).Error)
}

func (r *WalletRepository) ListTransactions(ctx context.Context, f domain.WalletTxFilter) (domain.Page[domain.WalletTransaction], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&walletTxModel{})
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.Type != nil {
		q = q.Where("type = ?", string(*f.Type))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.WalletTransaction]{}, translate(err)
	}
	var models []walletTxModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.WalletTransaction]{}, translate(err)
	}
	items := make([]domain.WalletTransaction, 0, len(models))
	for i := range models {
		items = append(items, toWalletTx(&models[i]))
	}
	return domain.Page[domain.WalletTransaction]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// PointsRepository stores loyalty balances and their ledger.
type PointsRepository struct{ db *DB }

func NewPointsRepository(db *DB) *PointsRepository { return &PointsRepository{db: db} }

func (r *PointsRepository) Ensure(ctx context.Context, userID string) (*domain.PointsAccount, error) {
	if err := translate(r.db.session(ctx).Exec(
		`INSERT INTO points_accounts (id, user_id, balance, lifetime_earned, created_at, updated_at)
		 VALUES (gen_random_uuid(), ?, 0, 0, now(), now())
		 ON CONFLICT (user_id) DO NOTHING`, userID).Error); err != nil {
		return nil, err
	}
	return r.FindByUser(ctx, userID)
}

func (r *PointsRepository) FindByUser(ctx context.Context, userID string) (*domain.PointsAccount, error) {
	var m pointsAccountModel
	if err := r.db.session(ctx).First(&m, "user_id = ?", userID).Error; err != nil {
		return nil, translate(err)
	}
	return &domain.PointsAccount{
		ID: m.ID, UserID: m.UserID, Balance: m.Balance, LifetimeEarned: m.LifetimeEarned,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *PointsRepository) AddPoints(ctx context.Context, userID string, delta int64) (int64, error) {
	query := `UPDATE points_accounts
		SET balance = balance + ?, lifetime_earned = lifetime_earned + GREATEST(?, 0), updated_at = now()
		WHERE user_id = ?`
	args := []any{delta, delta, userID}
	if delta < 0 {
		query += ` AND balance + ? >= 0`
		args = append(args, delta)
	}
	query += ` RETURNING balance`

	var balance int64
	row := r.db.session(ctx).Raw(query, args...).Row()
	if err := row.Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrInsufficientFunds
		}
		return 0, translate(err)
	}
	return balance, nil
}

func (r *PointsRepository) AddTransaction(ctx context.Context, tx *domain.PointsTransaction) error {
	return translate(r.db.session(ctx).Create(&pointsTxModel{
		ID: tx.ID, UserID: tx.UserID, Type: string(tx.Type), Points: tx.Points,
		BalanceAfter: tx.BalanceAfter, ReferenceType: tx.ReferenceType, ReferenceID: tx.ReferenceID,
		Description: tx.Description, CreatedAt: tx.CreatedAt,
	}).Error)
}

func (r *PointsRepository) ListTransactions(ctx context.Context, userID string, page, pageSize int) (domain.Page[domain.PointsTransaction], error) {
	page, size := normalizePage(page, pageSize, 20)
	q := r.db.session(ctx).Model(&pointsTxModel{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.PointsTransaction]{}, translate(err)
	}
	var models []pointsTxModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.PointsTransaction]{}, translate(err)
	}
	items := make([]domain.PointsTransaction, 0, len(models))
	for i := range models {
		items = append(items, domain.PointsTransaction{
			ID: models[i].ID, UserID: models[i].UserID, Type: domain.PointsTransactionType(models[i].Type),
			Points: models[i].Points, BalanceAfter: models[i].BalanceAfter,
			ReferenceType: models[i].ReferenceType, ReferenceID: models[i].ReferenceID,
			Description: models[i].Description, CreatedAt: models[i].CreatedAt,
		})
	}
	return domain.Page[domain.PointsTransaction]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ReferralRepository stores referral codes and links.
type ReferralRepository struct{ db *DB }

func NewReferralRepository(db *DB) *ReferralRepository { return &ReferralRepository{db: db} }

func (r *ReferralRepository) EnsureCode(ctx context.Context, userID, code string) (*domain.Referral, error) {
	if err := translate(r.db.session(ctx).Exec(
		`INSERT INTO referral_codes (user_id, code, created_at) VALUES (?, ?, now())
		 ON CONFLICT (user_id) DO NOTHING`, userID, code).Error); err != nil {
		return nil, err
	}
	resolved, err := r.FindCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.Referral{ReferrerID: userID, Code: resolved}, nil
}

func (r *ReferralRepository) FindCode(ctx context.Context, userID string) (string, error) {
	var m referralCodeModel
	if err := r.db.session(ctx).First(&m, "user_id = ?", userID).Error; err != nil {
		return "", translate(err)
	}
	return m.Code, nil
}

func (r *ReferralRepository) FindReferrerByCode(ctx context.Context, code string) (string, error) {
	var m referralCodeModel
	if err := r.db.session(ctx).First(&m, "code = ?", code).Error; err != nil {
		return "", translate(err)
	}
	return m.UserID, nil
}

func (r *ReferralRepository) FindByReferee(ctx context.Context, refereeID string) (*domain.Referral, error) {
	var m referralModel
	if err := r.db.session(ctx).First(&m, "referee_id = ?", refereeID).Error; err != nil {
		return nil, translate(err)
	}
	return &domain.Referral{ID: m.ID, ReferrerID: m.ReferrerID, RefereeID: m.RefereeID, Code: m.Code, CreatedAt: m.CreatedAt}, nil
}

func (r *ReferralRepository) Create(ctx context.Context, ref *domain.Referral) error {
	return translate(r.db.session(ctx).Create(&referralModel{
		ID: ref.ID, ReferrerID: ref.ReferrerID, RefereeID: ref.RefereeID, Code: ref.Code, CreatedAt: ref.CreatedAt,
	}).Error)
}

func (r *ReferralRepository) ListByReferrer(ctx context.Context, referrerID string, page, pageSize int) (domain.Page[domain.Referral], error) {
	page, size := normalizePage(page, pageSize, 20)
	q := r.db.session(ctx).Model(&referralModel{}).Where("referrer_id = ?", referrerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Referral]{}, translate(err)
	}
	var models []referralModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Referral]{}, translate(err)
	}
	items := make([]domain.Referral, 0, len(models))
	for i := range models {
		items = append(items, domain.Referral{
			ID: models[i].ID, ReferrerID: models[i].ReferrerID, RefereeID: models[i].RefereeID,
			Code: models[i].Code, CreatedAt: models[i].CreatedAt,
		})
	}
	return domain.Page[domain.Referral]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// CommissionRepository stores referral commissions.
type CommissionRepository struct{ db *DB }

func NewCommissionRepository(db *DB) *CommissionRepository { return &CommissionRepository{db: db} }

func (r *CommissionRepository) Create(ctx context.Context, c *domain.Commission) error {
	return translate(r.db.session(ctx).Create(&commissionModel{
		ID: c.ID, ReferrerID: c.ReferrerID, RefereeID: c.RefereeID, OrderID: c.OrderID,
		BaseCents: c.BaseCents, RateBps: c.RateBps, AmountCents: c.AmountCents,
		Status: string(c.Status), HoldUntil: c.HoldUntil, ApprovedAt: c.ApprovedAt, CreatedAt: c.CreatedAt,
	}).Error)
}

func (r *CommissionRepository) FindByOrder(ctx context.Context, orderID string) (*domain.Commission, error) {
	var m commissionModel
	if err := r.db.session(ctx).First(&m, "order_id = ?", orderID).Error; err != nil {
		return nil, translate(err)
	}
	c := toCommission(&m)
	return &c, nil
}

func (r *CommissionRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]domain.Commission, error) {
	var models []commissionModel
	if err := r.db.session(ctx).
		Where("status = ? AND hold_until <= ?", string(domain.CommissionPending), now).
		Order("hold_until asc").Limit(limit).Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Commission, 0, len(models))
	for i := range models {
		out = append(out, toCommission(&models[i]))
	}
	return out, nil
}

func (r *CommissionRepository) MarkApproved(ctx context.Context, id string, at time.Time) error {
	res := r.db.session(ctx).Model(&commissionModel{}).
		Where("id = ? AND status = ?", id, string(domain.CommissionPending)).
		Updates(map[string]any{"status": string(domain.CommissionApproved), "approved_at": at})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (r *CommissionRepository) MarkReversed(ctx context.Context, id string) error {
	res := r.db.session(ctx).Model(&commissionModel{}).
		Where("id = ?", id).Update("status", string(domain.CommissionReversed))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CommissionRepository) SumByReferrer(ctx context.Context, referrerID string) (int64, int64, error) {
	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	if err := r.db.session(ctx).Model(&commissionModel{}).
		Select("status, COALESCE(SUM(amount_cents), 0) AS total").
		Where("referrer_id = ?", referrerID).Group("status").Scan(&rows).Error; err != nil {
		return 0, 0, translate(err)
	}
	var pending, approved int64
	for _, r := range rows {
		switch domain.CommissionStatus(r.Status) {
		case domain.CommissionPending:
			pending = r.Total
		case domain.CommissionApproved:
			approved = r.Total
		}
	}
	return pending, approved, nil
}

func (r *CommissionRepository) List(ctx context.Context, f domain.CommissionFilter) (domain.Page[domain.Commission], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&commissionModel{})
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.ReferrerID != "" {
		q = q.Where("referrer_id = ?", f.ReferrerID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Commission]{}, translate(err)
	}
	var models []commissionModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Commission]{}, translate(err)
	}
	items := make([]domain.Commission, 0, len(models))
	for i := range models {
		items = append(items, toCommission(&models[i]))
	}
	return domain.Page[domain.Commission]{Items: items, Total: total, Page: page, PageSize: size}, nil
}
