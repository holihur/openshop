package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AuditRepository implements port.AuditRepository.
type AuditRepository struct{ db *DB }

func NewAuditRepository(db *DB) *AuditRepository { return &AuditRepository{db: db} }

type auditModel struct {
	ID           string     `gorm:"type:uuid;primaryKey"`
	ActorID      uuidString `gorm:"type:uuid;index"`
	ActorRole    string     `gorm:"size:32;not null;default:''"`
	Action       string     `gorm:"size:64;not null"`
	ResourceType string     `gorm:"size:64;not null;default:''"`
	ResourceID   string     `gorm:"size:64;not null;default:''"`
	Metadata     jsonMap    `gorm:"type:jsonb"`
	IP           string     `gorm:"size:64;not null;default:''"`
	CreatedAt    time.Time  `gorm:"not null"`
}

func (auditModel) TableName() string { return "audit_logs" }

func (r *AuditRepository) Create(ctx context.Context, entry *domain.AuditLog) error {
	m := &auditModel{
		ID: entry.ID, ActorID: uuidString(entry.ActorID), ActorRole: entry.ActorRole,
		Action: entry.Action, ResourceType: entry.ResourceType, ResourceID: entry.ResourceID,
		Metadata: jsonMap(entry.Metadata), IP: entry.IP, CreatedAt: entry.CreatedAt,
	}
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

func (r *AuditRepository) List(ctx context.Context, f domain.AuditFilter) (domain.Page[domain.AuditLog], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&auditModel{})
	if f.ActorID != "" {
		q = q.Where("actor_id = ?", f.ActorID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.AuditLog]{}, translate(err)
	}

	var models []auditModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.AuditLog]{}, translate(err)
	}
	items := make([]domain.AuditLog, 0, len(models))
	for i := range models {
		m := &models[i]
		items = append(items, domain.AuditLog{
			ID: m.ID, ActorID: string(m.ActorID), ActorRole: m.ActorRole, Action: m.Action,
			ResourceType: m.ResourceType, ResourceID: m.ResourceID,
			Metadata: map[string]string(m.Metadata), IP: m.IP, CreatedAt: m.CreatedAt,
		})
	}
	return domain.Page[domain.AuditLog]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

var _ port.AuditRepository = (*AuditRepository)(nil)
