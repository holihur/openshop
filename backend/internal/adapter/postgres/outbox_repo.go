package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// OutboxRepository implements port.Outbox on PostgreSQL. Enqueue joins the
// ambient transaction (via the context set by TxManager), which is what makes
// event publication atomic with the business write.
type OutboxRepository struct{ db *DB }

func NewOutboxRepository(db *DB) *OutboxRepository { return &OutboxRepository{db: db} }

type outboxModel struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	Subject     string    `gorm:"size:128;not null"`
	Payload     []byte    `gorm:"type:jsonb;not null"`
	Status      string    `gorm:"size:16;not null;default:pending"`
	Attempts    int       `gorm:"not null;default:0"`
	AvailableAt time.Time `gorm:"not null"`
	LockedUntil *time.Time
	LastError   string    `gorm:"type:text;not null;default:''"`
	TraceParent string    `gorm:"size:128;not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	PublishedAt *time.Time
}

func (outboxModel) TableName() string { return "outbox_events" }

func (r *OutboxRepository) Enqueue(ctx context.Context, evt port.Event) error {
	payload := evt.Payload
	if payload == nil {
		payload = []byte("{}")
	}
	now := time.Now().UTC()
	m := &outboxModel{
		ID:          evt.ID,
		Subject:     evt.Subject,
		Payload:     payload,
		Status:      "pending",
		AvailableAt: now,
		TraceParent: evt.TraceParent,
		CreatedAt:   now,
	}
	if m.ID == "" {
		return fmt.Errorf("outbox: event id is required")
	}
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	return nil
}

var _ port.OutboxInspector = (*OutboxRepository)(nil)

func toOutboxMessage(m *outboxModel) port.OutboxMessage {
	return port.OutboxMessage{
		ID: m.ID, Subject: m.Subject, Payload: m.Payload, Attempts: m.Attempts,
		TraceParent: m.TraceParent, Status: m.Status, LastError: m.LastError,
		CreatedAt: m.CreatedAt, AvailableAt: m.AvailableAt,
	}
}

// List returns a page of queue rows, newest first, for the operations console.
func (r *OutboxRepository) List(ctx context.Context, f port.OutboxFilter) (domain.Page[port.OutboxMessage], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&outboxModel{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Subject != "" {
		q = q.Where("subject = ?", f.Subject)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[port.OutboxMessage]{}, translate(err)
	}
	var models []outboxModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[port.OutboxMessage]{}, translate(err)
	}
	items := make([]port.OutboxMessage, 0, len(models))
	for i := range models {
		items = append(items, toOutboxMessage(&models[i]))
	}
	return domain.Page[port.OutboxMessage]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// Stats summarises the queue, including the dead-letter count and how long the
// oldest pending event has been waiting (the backlog age an alert would watch).
func (r *OutboxRepository) Stats(ctx context.Context) (port.OutboxStats, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.session(ctx).Model(&outboxModel{}).
		Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return port.OutboxStats{}, translate(err)
	}
	var stats port.OutboxStats
	for _, r := range rows {
		switch r.Status {
		case "pending":
			stats.Pending = r.Count
		case "processing":
			stats.Processing = r.Count
		case "published":
			stats.Published = r.Count
		case "failed":
			stats.Failed = r.Count
		}
	}

	var oldest sql.NullTime
	if err := r.db.session(ctx).
		Raw("SELECT min(available_at) FROM outbox_events WHERE status = ?", "pending").
		Scan(&oldest).Error; err != nil {
		return stats, translate(err)
	}
	if oldest.Valid {
		at := oldest.Time
		stats.OldestPending = &at
	}
	return stats, nil
}

// Replay puts a dead-lettered event back on the queue with a clean slate, so an
// operator can recover after fixing the cause instead of losing the event.
func (r *OutboxRepository) Replay(ctx context.Context, id string) error {
	res := r.db.session(ctx).Model(&outboxModel{}).
		Where("id = ? AND status = ?", id, "failed").
		Updates(map[string]any{
			"status":       "pending",
			"attempts":     0,
			"available_at": time.Now().UTC(),
			"locked_until": nil,
			"last_error":   "",
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrConflict
	}
	return nil
}

// Claim leases a batch of pending rows. The SELECT ... FOR UPDATE SKIP LOCKED
// plus the status update happen in one short transaction, so concurrent relays
// never hand out the same row.
func (r *OutboxRepository) Claim(ctx context.Context, limit int, lease time.Duration) ([]port.OutboxMessage, error) {
	if limit < 1 {
		limit = 50
	}
	if lease <= 0 {
		lease = 30 * time.Second
	}
	now := time.Now().UTC()

	var claimed []outboxModel
	err := r.db.session(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []outboxModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND available_at <= ?", "pending", now).
			Order("available_at asc, created_at asc").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]string, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].ID)
			rows[i].Status = "processing"
			rows[i].Attempts++
		}
		lockedUntil := now.Add(lease)
		if err := tx.Model(&outboxModel{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":       "processing",
				"attempts":     gorm.Expr("attempts + 1"),
				"locked_until": lockedUntil,
			}).Error; err != nil {
			return err
		}
		claimed = rows
		return nil
	})
	if err != nil {
		return nil, translate(err)
	}

	out := make([]port.OutboxMessage, 0, len(claimed))
	for _, m := range claimed {
		out = append(out, port.OutboxMessage{
			ID: m.ID, Subject: m.Subject, Payload: m.Payload, Attempts: m.Attempts,
			TraceParent: m.TraceParent,
		})
	}
	return out, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return translate(r.db.session(ctx).Model(&outboxModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":       "published",
			"published_at": now,
			"locked_until": nil,
			"last_error":   "",
		}).Error)
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, msg port.OutboxMessage, reason string, retryAt time.Time) error {
	status := "pending"
	if msg.Attempts >= 10 {
		status = "failed"
	}
	return translate(r.db.session(ctx).Model(&outboxModel{}).Where("id = ?", msg.ID).
		Updates(map[string]any{
			"status":       status,
			"available_at": retryAt,
			"locked_until": nil,
			"last_error":   reason,
		}).Error)
}

func (r *OutboxRepository) Reclaim(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.session(ctx).Model(&outboxModel{}).
		Where("status = ? AND locked_until < ?", "processing", now).
		Updates(map[string]any{
			"status":       "pending",
			"available_at": now,
			"locked_until": nil,
		})
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	return res.RowsAffected, nil
}

var _ port.Outbox = (*OutboxRepository)(nil)
