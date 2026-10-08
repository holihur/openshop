package domain

import (
	"testing"
	"time"
)

func TestAuditHashIsDeterministic(t *testing.T) {
	entry := AuditLog{
		ID: "id-1", ActorID: "user-1", ActorRole: "admin", Action: "order.refund",
		ResourceType: "order", ResourceID: "ord-1", IP: "203.0.113.5",
		Metadata:  map[string]string{"b": "2", "a": "1"},
		CreatedAt: time.Date(2026, 2, 3, 4, 5, 6, 123456000, time.UTC),
	}
	first := AuditHash("prev-hash", &entry)

	// Map iteration order must not affect the hash.
	reordered := entry
	reordered.Metadata = map[string]string{"a": "1", "b": "2"}
	if got := AuditHash("prev-hash", &reordered); got != first {
		t.Error("metadata order must not change the hash")
	}

	// Every field is part of the input.
	mutations := map[string]func(e *AuditLog){
		"id":         func(e *AuditLog) { e.ID = "id-2" },
		"actor":      func(e *AuditLog) { e.ActorID = "user-2" },
		"role":       func(e *AuditLog) { e.ActorRole = "support" },
		"action":     func(e *AuditLog) { e.Action = "order.ship" },
		"resource":   func(e *AuditLog) { e.ResourceType = "product" },
		"resourceID": func(e *AuditLog) { e.ResourceID = "ord-2" },
		"ip":         func(e *AuditLog) { e.IP = "203.0.113.6" },
		"metadata":   func(e *AuditLog) { e.Metadata = map[string]string{"a": "9"} },
		"createdAt":  func(e *AuditLog) { e.CreatedAt = e.CreatedAt.Add(time.Second) },
	}
	for name, mutate := range mutations {
		changed := entry
		mutate(&changed)
		if AuditHash("prev-hash", &changed) == first {
			t.Errorf("changing %s must change the hash", name)
		}
	}

	// The chain link is part of the input.
	if AuditHash("other-hash", &entry) == first {
		t.Error("the previous hash must be part of the input")
	}
}

// Postgres stores timestamps with microsecond precision, so the hash must ignore
// anything finer or verification would never match.
func TestAuditHashTruncatesSubMicrosecondTime(t *testing.T) {
	base := AuditLog{ID: "id", Action: "a", CreatedAt: time.Date(2026, 2, 3, 4, 5, 6, 123456000, time.UTC)}
	finer := base
	finer.CreatedAt = base.CreatedAt.Add(499 * time.Nanosecond)
	if AuditHash("", &base) != AuditHash("", &finer) {
		t.Error("sub-microsecond precision must not affect the hash")
	}
}
