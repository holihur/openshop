package postgres

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

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
	PrevHash     string     `gorm:"size:64;not null;default:''"`
	Hash         string     `gorm:"size:64;not null;default:''"`
	CreatedAt    time.Time  `gorm:"not null"`
}

func (auditModel) TableName() string { return "audit_logs" }

// auditChainLockID serialises appends to the hash chain.
const auditChainLockID int64 = 815420193

func toAudit(m *auditModel) domain.AuditLog {
	return domain.AuditLog{
		ID: m.ID, ActorID: string(m.ActorID), ActorRole: m.ActorRole, Action: m.Action,
		ResourceType: m.ResourceType, ResourceID: m.ResourceID,
		Metadata: map[string]string(m.Metadata), IP: m.IP,
		PrevHash: m.PrevHash, Hash: m.Hash, CreatedAt: m.CreatedAt,
	}
}

func (r *AuditRepository) Create(ctx context.Context, entry *domain.AuditLog) error {
	return r.db.session(ctx).Transaction(func(tx *gorm.DB) error {
		// Only one writer may append at a time, so the chain is never forked.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", auditChainLockID).Error; err != nil {
			return translate(err)
		}
		var prev string
		if err := tx.Raw("SELECT hash FROM audit_logs ORDER BY seq DESC LIMIT 1").Scan(&prev).Error; err != nil {
			return translate(err)
		}
		entry.PrevHash = prev
		// Postgres keeps microseconds, so the value that is hashed must be the
		// value that will be read back, otherwise verification never matches.
		entry.CreatedAt = entry.CreatedAt.UTC().Truncate(time.Microsecond)
		entry.Hash = domain.AuditHash(prev, entry)
		m := &auditModel{
			ID: entry.ID, ActorID: uuidString(entry.ActorID), ActorRole: entry.ActorRole,
			Action: entry.Action, ResourceType: entry.ResourceType, ResourceID: entry.ResourceID,
			Metadata: jsonMap(entry.Metadata), IP: entry.IP,
			PrevHash: entry.PrevHash, Hash: entry.Hash, CreatedAt: entry.CreatedAt,
		}
		return translate(tx.Create(m).Error)
	})
}

// auditChainRow is the projection used when verifying the chain.
type auditChainRow struct {
	ID           string
	ActorID      uuidString
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     []byte
	IP           string
	PrevHash     string
	Hash         string
	CreatedAt    time.Time
}

// VerifyChain walks the retained entries in append order and recomputes every
// hash. It returns how many entries were verified and the id of the first broken
// one (empty when the chain is intact). The boundary entry's prev_hash is not
// checked, because pruning removes the entries before it.
func (r *AuditRepository) VerifyChain(ctx context.Context) (int64, string, error) {
	var rows []auditChainRow
	if err := r.db.session(ctx).Raw(
		`SELECT id, actor_id, actor_role, action, resource_type, resource_id,
		        metadata, ip, prev_hash, hash, created_at
		 FROM audit_logs ORDER BY seq ASC`).Scan(&rows).Error; err != nil {
		return 0, "", translate(err)
	}
	var checked int64
	var prevHash string
	for i := range rows {
		r := &rows[i]
		if r.Hash == "" {
			// Entries written before the chain existed cannot be verified.
			continue
		}
		metadata := map[string]string{}
		if len(r.Metadata) > 0 {
			_ = json.Unmarshal(r.Metadata, &metadata)
		}
		entry := domain.AuditLog{
			ID: r.ID, ActorID: string(r.ActorID), ActorRole: r.ActorRole, Action: r.Action,
			ResourceType: r.ResourceType, ResourceID: r.ResourceID, Metadata: metadata,
			IP: r.IP, PrevHash: r.PrevHash, Hash: r.Hash, CreatedAt: r.CreatedAt,
		}
		if domain.AuditHash(entry.PrevHash, &entry) != entry.Hash {
			return checked, entry.ID, nil
		}
		if checked > 0 && entry.PrevHash != prevHash {
			return checked, entry.ID, nil
		}
		prevHash = entry.Hash
		checked++
	}
	return checked, "", nil
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
		items = append(items, toAudit(m))
	}
	return domain.Page[domain.AuditLog]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

var _ port.AuditRepository = (*AuditRepository)(nil)
