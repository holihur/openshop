package handler

import (
	"encoding/json"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// NotificationView is the API shape of an in-app notification.
type NotificationView struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body,omitempty"`
	Link      string     `json:"link,omitempty"`
	Read      bool       `json:"read"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	// Data carries template variables so the client can localise the message.
	Data json.RawMessage `json:"data,omitempty"`
}

func ToNotificationView(n domain.Notification) NotificationView {
	view := NotificationView{
		ID: n.ID, Type: string(n.Type), Title: n.Title, Body: n.Body, Link: n.Link,
		Read: n.Read(), ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
	}
	if n.Data != "" && json.Valid([]byte(n.Data)) {
		view.Data = json.RawMessage(n.Data)
	}
	return view
}

func ToNotificationViews(ns []domain.Notification) []NotificationView {
	out := make([]NotificationView, 0, len(ns))
	for _, n := range ns {
		out = append(out, ToNotificationView(n))
	}
	return out
}
