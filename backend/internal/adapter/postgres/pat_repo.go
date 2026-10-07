package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// splitList parses a comma or whitespace separated list, dropping empties.
func splitList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f)
	}
	return out
}

func joinList(items []string) string { return strings.Join(items, ",") }

type personalAccessTokenModel struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	UserID     string `gorm:"type:uuid;index;not null"`
	Name       string `gorm:"size:100;not null"`
	Prefix     string `gorm:"size:16;not null"`
	TokenHash  string `gorm:"size:64;uniqueIndex;not null"`
	Realm      string `gorm:"size:16;not null;default:'front'"`
	Scopes     string `gorm:"type:text;not null;default:''"`
	CIDRs      string `gorm:"column:cidrs;type:text;not null;default:''"`
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time `gorm:"not null"`
}

func (personalAccessTokenModel) TableName() string { return "personal_access_tokens" }

func toPAT(m *personalAccessTokenModel) *domain.PersonalAccessToken {
	return &domain.PersonalAccessToken{
		ID: m.ID, UserID: m.UserID, Name: m.Name, Prefix: m.Prefix, Realm: m.Realm,
		Scopes: domain.ParseScopes(m.Scopes), CIDRs: splitList(m.CIDRs),
		ExpiresAt: m.ExpiresAt, LastUsedAt: m.LastUsedAt, RevokedAt: m.RevokedAt,
		CreatedAt: m.CreatedAt,
	}
}

// PersonalAccessTokenRepository stores programmatic credentials.
type PersonalAccessTokenRepository struct{ db *DB }

func NewPersonalAccessTokenRepository(db *DB) *PersonalAccessTokenRepository {
	return &PersonalAccessTokenRepository{db: db}
}

func (r *PersonalAccessTokenRepository) Create(ctx context.Context, t *domain.PersonalAccessToken, hash string) error {
	return translate(r.db.session(ctx).Create(&personalAccessTokenModel{
		ID: t.ID, UserID: t.UserID, Name: t.Name, Prefix: t.Prefix, TokenHash: hash,
		Realm: t.Realm, Scopes: domain.ScopesString(t.Scopes), CIDRs: joinList(t.CIDRs),
		ExpiresAt: t.ExpiresAt, CreatedAt: t.CreatedAt,
	}).Error)
}

func (r *PersonalAccessTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.PersonalAccessToken, *domain.User, error) {
	var m personalAccessTokenModel
	if err := r.db.session(ctx).First(&m, "token_hash = ?", hash).Error; err != nil {
		return nil, nil, translate(err)
	}
	var user userModel
	if err := r.db.session(ctx).First(&user, "id = ?", m.UserID).Error; err != nil {
		return nil, nil, translate(err)
	}
	return toPAT(&m), toUser(&user), nil
}

func (r *PersonalAccessTokenRepository) ListByUser(ctx context.Context, userID string) ([]domain.PersonalAccessToken, error) {
	var models []personalAccessTokenModel
	if err := r.db.session(ctx).Where("user_id = ?", userID).
		Order("created_at desc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.PersonalAccessToken, 0, len(models))
	for i := range models {
		out = append(out, *toPAT(&models[i]))
	}
	return out, nil
}

func (r *PersonalAccessTokenRepository) Revoke(ctx context.Context, id, userID string, at time.Time) error {
	res := r.db.session(ctx).Model(&personalAccessTokenModel{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PersonalAccessTokenRepository) Touch(ctx context.Context, id string, at time.Time) error {
	return translate(r.db.session(ctx).Model(&personalAccessTokenModel{}).
		Where("id = ?", id).Update("last_used_at", at).Error)
}
