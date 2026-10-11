package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// RoleRepository stores operator-defined roles.
type RoleRepository struct{ db *DB }

func NewRoleRepository(db *DB) *RoleRepository { return &RoleRepository{db: db} }

type roleModel struct {
	Key         string     `gorm:"primaryKey;size:64"`
	Name        string     `gorm:"size:128;not null"`
	Description string     `gorm:"size:512;not null;default:''"`
	Permissions stringList `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (roleModel) TableName() string { return "roles" }

func toRole(m *roleModel) *domain.Role {
	perms := make([]domain.Permission, 0, len(m.Permissions))
	for _, p := range m.Permissions {
		perms = append(perms, domain.Permission(p))
	}
	return &domain.Role{
		Key: m.Key, Name: m.Name, Description: m.Description, Permissions: perms,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func (r *RoleRepository) List(ctx context.Context) ([]domain.Role, error) {
	var models []roleModel
	if err := r.db.session(ctx).Order("key asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Role, 0, len(models))
	for i := range models {
		out = append(out, *toRole(&models[i]))
	}
	return out, nil
}

func (r *RoleRepository) Find(ctx context.Context, key string) (*domain.Role, error) {
	var m roleModel
	if err := r.db.session(ctx).Where("key = ?", key).First(&m).Error; err != nil {
		return nil, translate(err)
	}
	return toRole(&m), nil
}

func (r *RoleRepository) Upsert(ctx context.Context, role *domain.Role) error {
	perms := make(stringList, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		perms = append(perms, string(p))
	}
	m := &roleModel{
		Key: role.Key, Name: role.Name, Description: role.Description, Permissions: perms,
		CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
	}
	// The key is the identity, so an edit keeps its members.
	if err := r.db.session(ctx).Save(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

func (r *RoleRepository) Delete(ctx context.Context, key string) error {
	res := r.db.session(ctx).Where("key = ?", key).Delete(&roleModel{})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ port.RoleRepository = (*RoleRepository)(nil)
