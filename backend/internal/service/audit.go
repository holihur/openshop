package service

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AuditService writes and reads the append-only audit trail. Recording is
// best-effort: an audit write must never fail the business operation it
// describes.
type AuditService struct {
	repo   port.AuditRepository
	ids    port.IDGenerator
	clock  port.Clock
	logger port.Logger
}

func NewAuditService(repo port.AuditRepository, ids port.IDGenerator, clock port.Clock, logger port.Logger) *AuditService {
	return &AuditService{repo: repo, ids: ids, clock: clock, logger: logger}
}

// Entry describes an action to record.
type Entry struct {
	ActorID      string
	ActorRole    domain.UserRole
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]string
	IP           string
}

func (s *AuditService) Record(ctx context.Context, e Entry) {
	entry := &domain.AuditLog{
		ID: s.ids.NewID(), ActorID: e.ActorID, ActorRole: string(e.ActorRole),
		Action: e.Action, ResourceType: e.ResourceType, ResourceID: e.ResourceID,
		Metadata: e.Metadata, IP: e.IP, CreatedAt: s.clock.Now(),
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		s.logger.Warn("audit write failed", "action", e.Action, "error", err)
	}
}

func (s *AuditService) List(ctx context.Context, f domain.AuditFilter) (domain.Page[domain.AuditLog], error) {
	f.Page, f.PageSize = clampPage(f.Page, f.PageSize, 20)
	return s.repo.List(ctx, f)
}

// VerifyChain recomputes the audit hash chain, so tampering with a stored entry
// is detectable.
func (s *AuditService) VerifyChain(ctx context.Context) (int64, string, error) {
	return s.repo.VerifyChain(ctx)
}
