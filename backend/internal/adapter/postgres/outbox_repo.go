package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
