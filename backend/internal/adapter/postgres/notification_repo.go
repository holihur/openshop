package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

type notificationModel struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	UserID    string     `gorm:"type:uuid;index;not null"`
	Type      string     `gorm:"size:32;index;not null;default:'system'"`
	Title     string     `gorm:"size:200;not null"`
	Body      string     `gorm:"size:1000;not null;default:''"`
	Link      string     `gorm:"size:300;not null;default:''"`
	Data      string     `gorm:"type:jsonb;not null;default:'{}'"`
	ReadAt    *time.Time `gorm:"index"`
	CreatedAt time.Time  `gorm:"not null"`
}

func (notificationModel) TableName() string { return "notifications" }

// dataOrEmpty keeps the JSONB column valid when no template variables exist.
func dataOrEmpty(data string) string {
	if data == "" {
		return "{}"
	}
	return data
}

func toNotification(m *notificationModel) domain.Notification {
	return domain.Notification{
		ID: m.ID, UserID: m.UserID, Type: domain.NotificationType(m.Type), Title: m.Title,
		Body: m.Body, Link: m.Link, Data: m.Data, ReadAt: m.ReadAt, CreatedAt: m.CreatedAt,
	}
}

// NotificationRepository stores in-app notifications.
type NotificationRepository struct{ db *DB }

func NewNotificationRepository(db *DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	return translate(r.db.session(ctx).Create(&notificationModel{
		ID: n.ID, UserID: n.UserID, Type: string(n.Type), Title: n.Title, Body: n.Body,
		Link: n.Link, Data: n.Data, CreatedAt: n.CreatedAt,
	}).Error)
}

func (r *NotificationRepository) List(ctx context.Context, f domain.NotificationFilter) (domain.Page[domain.Notification], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&notificationModel{}).Where("user_id = ?", f.UserID)
	if f.UnreadOnly {
		q = q.Where("read_at IS NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Notification]{}, translate(err)
	}
	var models []notificationModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Notification]{}, translate(err)
	}
	items := make([]domain.Notification, 0, len(models))
	for i := range models {
		items = append(items, toNotification(&models[i]))
	}
	return domain.Page[domain.Notification]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var n int64
	if err := r.db.session(ctx).Model(&notificationModel{}).
		Where("user_id = ? AND read_at IS NULL", userID).Count(&n).Error; err != nil {
		return 0, translate(err)
	}
	return n, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string, at time.Time) error {
	res := r.db.session(ctx).Model(&notificationModel{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string, at time.Time) error {
	return translate(r.db.session(ctx).Model(&notificationModel{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", at).Error)
}

// Broadcast inserts one notification per customer. A single INSERT ... SELECT
// keeps it cheap even for large stores.
func (r *NotificationRepository) Broadcast(ctx context.Context, n *domain.Notification) (int64, error) {
	res := r.db.session(ctx).Exec(
		`INSERT INTO notifications (id, user_id, type, title, body, link, data, created_at)
		 SELECT gen_random_uuid(), id, ?, ?, ?, ?, ?::jsonb, ?
		 FROM users WHERE role = 'customer'`,
		string(n.Type), n.Title, n.Body, n.Link, dataOrEmpty(n.Data), n.CreatedAt)
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	return res.RowsAffected, nil
}
