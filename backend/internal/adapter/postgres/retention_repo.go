package postgres

import (
	"context"
	"time"
)

// RetentionRepository prunes published outbox events and old audit logs.
type RetentionRepository struct{ db *DB }

func NewRetentionRepository(db *DB) *RetentionRepository { return &RetentionRepository{db: db} }

func (r *RetentionRepository) DeletePublishedOutboxBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	res := r.db.session(ctx).Exec(
		`DELETE FROM outbox_events WHERE id IN (
			SELECT id FROM outbox_events
			WHERE status = 'published' AND created_at < ?
			ORDER BY created_at
			LIMIT ?
		)`, before, limit)
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	return res.RowsAffected, nil
}

func (r *RetentionRepository) DeleteAuditBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	res := r.db.session(ctx).Exec(
		`DELETE FROM audit_logs WHERE ctid IN (
			SELECT ctid FROM audit_logs
			WHERE created_at < ?
			ORDER BY created_at
			LIMIT ?
		)`, before, limit)
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	return res.RowsAffected, nil
}
