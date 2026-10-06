package domain

import "time"

// AuditLog is an append-only record of a security- or admin-relevant action.
type AuditLog struct {
	ID           string
	ActorID      string
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]string
	IP           string
	CreatedAt    time.Time
}

// AuditFilter is a storage-agnostic query for the audit trail.
type AuditFilter struct {
	ActorID      string
	Action       string
	ResourceType string
	Page         int
	PageSize     int
}
