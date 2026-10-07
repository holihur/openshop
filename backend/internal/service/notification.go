package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Notifier creates in-app notifications. It is implemented by
// NotificationService and injected into the services that raise them.
type Notifier interface {
	Notify(ctx context.Context, in NotifyInput) error
}

// NotifyInput is one in-app notification.
type NotifyInput struct {
	UserID string
	Type   domain.NotificationType
	Title  string
	Body   string
	Link   string
	// Data carries template variables so the client can render the message in
	// the reader's own language.
	Data map[string]any
}

// NotificationService manages a user's in-app inbox.
type NotificationService struct {
	repo  port.NotificationRepository
	ids   port.IDGenerator
	clock port.Clock
}

func NewNotificationService(repo port.NotificationRepository, ids port.IDGenerator, clock port.Clock) *NotificationService {
	return &NotificationService{repo: repo, ids: ids, clock: clock}
}

// Notify stores a notification. An empty user or title is ignored, so business
// code can fire-and-forget without guarding every call site.
func (s *NotificationService) Notify(ctx context.Context, in NotifyInput) error {
	if in.UserID == "" || strings.TrimSpace(in.Title) == "" {
		return nil
	}
	ntype := in.Type
	if ntype == "" {
		ntype = domain.NotificationSystem
	}
	data := ""
	if len(in.Data) > 0 {
		if encoded, err := json.Marshal(in.Data); err == nil {
			data = string(encoded)
		}
	}
	return s.repo.Create(ctx, &domain.Notification{
		ID: s.ids.NewID(), UserID: in.UserID, Type: ntype, Title: in.Title,
		Body: in.Body, Link: in.Link, Data: data, CreatedAt: s.clock.Now(),
	})
}

// List returns a page of the user's notifications, newest first.
func (s *NotificationService) List(ctx context.Context, userID string, unreadOnly bool, page, pageSize int) (domain.Page[domain.Notification], error) {
	return s.repo.List(ctx, domain.NotificationFilter{UserID: userID, UnreadOnly: unreadOnly, Page: page, PageSize: pageSize})
}

// UnreadCount returns the size of the user's unread badge.
func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}

// MarkRead marks one notification read.
func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkRead(ctx, id, userID, s.clock.Now())
}

// MarkAllRead clears the user's unread badge.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID, s.clock.Now())
}

// Broadcast sends a notification to every customer and returns the count.
func (s *NotificationService) Broadcast(ctx context.Context, title, body, link string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, domain.ErrInvalidArgument
	}
	return s.repo.Broadcast(ctx, &domain.Notification{
		Type: domain.NotificationSystem, Title: title, Body: body, Link: link, CreatedAt: s.clock.Now(),
	})
}
