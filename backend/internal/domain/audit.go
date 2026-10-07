package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// AuditLog is an append-only record of a security- or admin-relevant action.
// PrevHash and Hash form a hash chain, so altering or deleting an entry is
// detectable.
type AuditLog struct {
	ID           string
	ActorID      string
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]string
	IP           string
	PrevHash     string
	Hash         string
	CreatedAt    time.Time
}

// AuditHash computes the chain hash for an entry given the previous hash. The
// serialisation is canonical (sorted metadata, RFC3339Nano UTC time) so the same
// entry always hashes the same way regardless of map iteration order.
func AuditHash(prev string, e *AuditLog) string {
	keys := make([]string, 0, len(e.Metadata))
	for k := range e.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var meta strings.Builder
	for _, k := range keys {
		meta.WriteString(k)
		meta.WriteByte('=')
		meta.WriteString(e.Metadata[k])
		meta.WriteByte('\n')
	}
	canonical := strings.Join([]string{
		prev, e.ID, e.ActorID, e.ActorRole, e.Action, e.ResourceType, e.ResourceID, e.IP,
		e.CreatedAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano), meta.String(),
	}, "\x1f")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// AuditFilter is a storage-agnostic query for the audit trail.
type AuditFilter struct {
	ActorID      string
	Action       string
	ResourceType string
	Page         int
	PageSize     int
}
