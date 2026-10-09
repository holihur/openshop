package postgres

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// SocialAccountRepository implements port.SocialAccountRepository.
type SocialAccountRepository struct{ db *DB }

func NewSocialAccountRepository(db *DB) *SocialAccountRepository {
	return &SocialAccountRepository{db: db}
}

type socialAccountModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Provider  string    `gorm:"size:40;not null"`
	Subject   string    `gorm:"size:191;not null"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	Email     string    `gorm:"size:191;not null;default:''"`
	Name      string    `gorm:"size:191;not null;default:''"`
	AvatarURL string    `gorm:"type:text;not null;default:''"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (socialAccountModel) TableName() string { return "social_accounts" }

func toSocialAccount(m *socialAccountModel) *domain.SocialAccount {
	return &domain.SocialAccount{
		ID: m.ID, Provider: m.Provider, Subject: m.Subject, UserID: m.UserID,
		Email: m.Email, Name: m.Name, AvatarURL: m.AvatarURL,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func (r *SocialAccountRepository) FindBySubject(ctx context.Context, provider, subject string) (*domain.SocialAccount, error) {
	var m socialAccountModel
	if err := r.db.session(ctx).
		Where("provider = ? AND subject = ?", provider, subject).
		First(&m).Error; err != nil {
		return nil, translate(err)
	}
	return toSocialAccount(&m), nil
}

func (r *SocialAccountRepository) ListByUser(ctx context.Context, userID string) ([]domain.SocialAccount, error) {
	var models []socialAccountModel
	if err := r.db.session(ctx).Where("user_id = ?", userID).Order("created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.SocialAccount, 0, len(models))
	for i := range models {
		out = append(out, *toSocialAccount(&models[i]))
	}
	return out, nil
}

func (r *SocialAccountRepository) Upsert(ctx context.Context, account *domain.SocialAccount) error {
	m := &socialAccountModel{
		ID: account.ID, Provider: account.Provider, Subject: account.Subject,
		UserID: account.UserID, Email: account.Email, Name: account.Name,
		AvatarURL: account.AvatarURL, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
	}
	// Re-signing in with an already linked account refreshes the cached profile.
	if err := r.db.session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}, {Name: "subject"}},
		DoUpdates: clause.AssignmentColumns([]string{"email", "name", "avatar_url", "updated_at"}),
	}).Create(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

var _ port.SocialAccountRepository = (*SocialAccountRepository)(nil)
