package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

type ticketModel struct {
	ID         string     `gorm:"type:uuid;primaryKey"`
	Number     int64      `gorm:"not null;uniqueIndex:uni_tickets_number"`
	UserID     uuidString `gorm:"type:uuid;index"`
	Email      string     `gorm:"size:320;not null;default:''"`
	Name       string     `gorm:"size:200;not null;default:''"`
	Subject    string     `gorm:"size:300;not null"`
	Kind       string     `gorm:"size:16;index;not null;default:'other'"`
	Priority   string     `gorm:"size:16;index;not null;default:'normal'"`
	Status     string     `gorm:"size:16;index;not null;default:'open'"`
	OrderID    uuidString `gorm:"type:uuid;index"`
	ProductID  uuidString `gorm:"type:uuid;index"`
	AssigneeID uuidString `gorm:"type:uuid;index"`
	CreatedAt  time.Time  `gorm:"not null"`
	UpdatedAt  time.Time  `gorm:"not null"`
}

func (ticketModel) TableName() string { return "tickets" }

type ticketMessageModel struct {
	ID         string     `gorm:"type:uuid;primaryKey"`
	TicketID   string     `gorm:"type:uuid;index;not null"`
	AuthorID   uuidString `gorm:"type:uuid;index"`
	AuthorRole string     `gorm:"size:16;not null;default:'customer'"`
	Body       string     `gorm:"type:text;not null"`
	Internal   bool       `gorm:"not null;default:false"`
	CreatedAt  time.Time  `gorm:"not null"`
}

func (ticketMessageModel) TableName() string { return "ticket_messages" }

func toTicket(m *ticketModel) *domain.Ticket {
	return &domain.Ticket{
		ID: m.ID, Number: m.Number, UserID: string(m.UserID), Email: m.Email, Name: m.Name,
		Subject: m.Subject, Kind: domain.TicketKind(m.Kind), Priority: domain.TicketPriority(m.Priority),
		Status: domain.TicketStatus(m.Status), OrderID: string(m.OrderID), ProductID: string(m.ProductID),
		AssigneeID: string(m.AssigneeID), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromTicket(t *domain.Ticket) *ticketModel {
	return &ticketModel{
		ID: t.ID, Number: t.Number, UserID: uuidString(t.UserID), Email: t.Email, Name: t.Name,
		Subject: t.Subject, Kind: string(t.Kind), Priority: string(t.Priority), Status: string(t.Status),
		OrderID: uuidString(t.OrderID), ProductID: uuidString(t.ProductID), AssigneeID: uuidString(t.AssigneeID),
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toTicketMessage(m *ticketMessageModel) *domain.TicketMessage {
	return &domain.TicketMessage{
		ID: m.ID, TicketID: m.TicketID, AuthorID: string(m.AuthorID), AuthorRole: m.AuthorRole,
		Body: m.Body, Internal: m.Internal, CreatedAt: m.CreatedAt,
	}
}

// TicketRepository stores support tickets and their message threads.
type TicketRepository struct{ db *DB }

func NewTicketRepository(db *DB) *TicketRepository { return &TicketRepository{db: db} }

func (r *TicketRepository) Create(ctx context.Context, t *domain.Ticket) error {
	session := r.db.session(ctx)
	if t.Number == 0 {
		var n int64
		if err := session.Raw("SELECT nextval('ticket_number_seq')").Scan(&n).Error; err != nil {
			return translate(err)
		}
		t.Number = n
	}
	return translate(session.Create(fromTicket(t)).Error)
}

func (r *TicketRepository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	var m ticketModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toTicket(&m), nil
}

func (r *TicketRepository) Update(ctx context.Context, t *domain.Ticket) error {
	res := r.db.session(ctx).Model(&ticketModel{}).Where("id = ?", t.ID).Updates(map[string]any{
		"subject":     t.Subject,
		"kind":        string(t.Kind),
		"priority":    string(t.Priority),
		"status":      string(t.Status),
		"assignee_id": uuidString(t.AssigneeID),
		"updated_at":  t.UpdatedAt,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TicketRepository) List(ctx context.Context, f domain.TicketFilter) (domain.Page[domain.Ticket], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)
	q := r.db.session(ctx).Model(&ticketModel{})
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.Kind != nil {
		q = q.Where("kind = ?", string(*f.Kind))
	}
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.Assignee != nil {
		if *f.Assignee == "" {
			q = q.Where("assignee_id IS NULL")
		} else {
			q = q.Where("assignee_id = ?", *f.Assignee)
		}
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("subject ILIKE ? OR email ILIKE ? OR name ILIKE ?", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Ticket]{}, translate(err)
	}
	var models []ticketModel
	if err := q.Order("updated_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Ticket]{}, translate(err)
	}
	items := make([]domain.Ticket, 0, len(models))
	for i := range models {
		items = append(items, *toTicket(&models[i]))
	}
	return domain.Page[domain.Ticket]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (r *TicketRepository) AddMessage(ctx context.Context, m *domain.TicketMessage) error {
	if err := translate(r.db.session(ctx).Create(&ticketMessageModel{
		ID: m.ID, TicketID: m.TicketID, AuthorID: uuidString(m.AuthorID), AuthorRole: m.AuthorRole,
		Body: m.Body, Internal: m.Internal, CreatedAt: m.CreatedAt,
	}).Error); err != nil {
		return err
	}
	// Bump the ticket so the queue is ordered by recent activity.
	return translate(r.db.session(ctx).Model(&ticketModel{}).Where("id = ?", m.TicketID).
		Update("updated_at", m.CreatedAt).Error)
}

func (r *TicketRepository) ListMessages(ctx context.Context, ticketID string, includeInternal bool) ([]domain.TicketMessage, error) {
	var models []ticketMessageModel
	q := r.db.session(ctx).Where("ticket_id = ?", ticketID)
	if !includeInternal {
		q = q.Where("internal = false")
	}
	if err := q.Order("created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.TicketMessage, 0, len(models))
	for i := range models {
		out = append(out, *toTicketMessage(&models[i]))
	}
	return out, nil
}

// CountOpen counts tickets that still need attention (open or pending).
func (r *TicketRepository) CountOpen(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.session(ctx).Model(&ticketModel{}).
		Where("status IN ?", []string{"open", "pending"}).Count(&n).Error; err != nil {
		return 0, translate(err)
	}
	return n, nil
}
