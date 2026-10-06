package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// stringList persists a []string as a JSONB column without pulling in an extra
// dependency such as pq.Array.
type stringList []string

func (s stringList) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

func (s *stringList) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("stringList: unsupported type %T", src)
	}
	if len(raw) == 0 {
		*s = nil
		return nil
	}
	return json.Unmarshal(raw, (*[]string)(s))
}

// uuidString is a UUID column that maps the empty string to SQL NULL.
// PostgreSQL's uuid type rejects "", so nullable foreign keys use this type.
type uuidString string

func (u uuidString) Value() (driver.Value, error) {
	if u == "" {
		return nil, nil
	}
	return string(u), nil
}

func (u *uuidString) Scan(src any) error {
	if src == nil {
		*u = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		*u = uuidString(v)
	case []byte:
		*u = uuidString(string(v))
	default:
		return fmt.Errorf("uuidString: unsupported type %T", src)
	}
	return nil
}

// nullableUUID converts an empty id to SQL NULL for nullable uuid columns in
// map-based updates, where driver.Valuer is not applied.
func nullableUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type userModel struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"size:255;uniqueIndex;not null"`
	Phone        string    `gorm:"size:32;index"`
	PasswordHash string    `gorm:"size:255;not null"`
	Name         string    `gorm:"size:128"`
	Role         string    `gorm:"size:32;index;not null;default:customer"`
	Status       string    `gorm:"size:32;not null;default:active"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (userModel) TableName() string { return "users" }

type categoryModel struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	Name      string     `gorm:"size:128;not null"`
	Slug      string     `gorm:"size:160;uniqueIndex;not null"`
	ParentID  uuidString `gorm:"type:uuid;index"`
	Sort      int        `gorm:"not null;default:0"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (categoryModel) TableName() string { return "categories" }

type productModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	CategoryID  uuidString `gorm:"type:uuid;index"`
	Title       string     `gorm:"size:255;not null"`
	Slug        string     `gorm:"size:255;uniqueIndex;not null"`
	Description string     `gorm:"type:text"`
	PriceCents  int64      `gorm:"not null;default:0"`
	Currency    string     `gorm:"size:8;not null;default:CNY"`
	CoverImage  string     `gorm:"size:512"`
	Images      stringList `gorm:"type:jsonb"`
	Status      string     `gorm:"size:32;index;not null;default:draft"`
	Stock       int        `gorm:"not null;default:0"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (productModel) TableName() string { return "products" }

type orderModel struct {
	ID         string           `gorm:"type:uuid;primaryKey"`
	OrderNo    string           `gorm:"size:64;uniqueIndex;not null"`
	UserID     string           `gorm:"type:uuid;index;not null"`
	Status     string           `gorm:"size:32;index;not null"`
	Currency   string           `gorm:"size:8;not null"`
	TotalCents int64            `gorm:"not null"`
	PaymentID  uuidString       `gorm:"type:uuid;index"`
	ExpiresAt  time.Time        `gorm:"index;not null"`
	PaidAt     *time.Time       `gorm:"index"`
	Items      []orderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	CreatedAt  time.Time        `gorm:"not null"`
	UpdatedAt  time.Time        `gorm:"not null"`
}

func (orderModel) TableName() string { return "orders" }

type orderItemModel struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	OrderID    string `gorm:"type:uuid;index;not null"`
	ProductID  string `gorm:"type:uuid;index;not null"`
	Title      string `gorm:"size:255;not null"`
	PriceCents int64  `gorm:"not null"`
	Quantity   int    `gorm:"not null"`
	Subtotal   int64  `gorm:"not null"`
}

func (orderItemModel) TableName() string { return "order_items" }

type paymentModel struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	OrderID       string    `gorm:"type:uuid;index;not null"`
	UserID        string    `gorm:"type:uuid;index;not null"`
	Provider      string    `gorm:"size:64;index;not null"`
	ProviderRef   string    `gorm:"size:128;index"`
	AmountCents   int64     `gorm:"not null"`
	Currency      string    `gorm:"size:8;not null"`
	Status        string    `gorm:"size:32;index;not null"`
	FailureReason string    `gorm:"size:512"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (paymentModel) TableName() string { return "payments" }

// ---- mappers: persistence <-> domain ----

func toUser(m *userModel) *domain.User {
	return &domain.User{
		ID: m.ID, Email: m.Email, Phone: m.Phone, PasswordHash: m.PasswordHash,
		Name: m.Name, Role: domain.UserRole(m.Role), Status: domain.UserStatus(m.Status),
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromUser(u *domain.User) *userModel {
	return &userModel{
		ID: u.ID, Email: u.Email, Phone: u.Phone, PasswordHash: u.PasswordHash,
		Name: u.Name, Role: string(u.Role), Status: string(u.Status),
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func toCategory(m *categoryModel) *domain.Category {
	return &domain.Category{
		ID: m.ID, Name: m.Name, Slug: m.Slug, ParentID: string(m.ParentID),
		Sort: m.Sort, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromCategory(c *domain.Category) *categoryModel {
	return &categoryModel{
		ID: c.ID, Name: c.Name, Slug: c.Slug, ParentID: uuidString(c.ParentID),
		Sort: c.Sort, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toProduct(m *productModel) *domain.Product {
	return &domain.Product{
		ID: m.ID, CategoryID: string(m.CategoryID), Title: m.Title, Slug: m.Slug,
		Description: m.Description, PriceCents: m.PriceCents, Currency: m.Currency,
		CoverImage: m.CoverImage, Images: []string(m.Images), Status: domain.ProductStatus(m.Status),
		Stock: m.Stock, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromProduct(p *domain.Product) *productModel {
	return &productModel{
		ID: p.ID, CategoryID: uuidString(p.CategoryID), Title: p.Title, Slug: p.Slug,
		Description: p.Description, PriceCents: p.PriceCents, Currency: p.Currency,
		CoverImage: p.CoverImage, Images: stringList(p.Images), Status: string(p.Status),
		Stock: p.Stock, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toOrder(m *orderModel) *domain.Order {
	items := make([]domain.OrderItem, 0, len(m.Items))
	for _, it := range m.Items {
		items = append(items, domain.OrderItem{
			ID: it.ID, OrderID: it.OrderID, ProductID: it.ProductID, Title: it.Title,
			PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	return &domain.Order{
		ID: m.ID, OrderNo: m.OrderNo, UserID: m.UserID, Status: domain.OrderStatus(m.Status),
		Currency: m.Currency, TotalCents: m.TotalCents, Items: items, PaymentID: string(m.PaymentID),
		ExpiresAt: m.ExpiresAt, PaidAt: m.PaidAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromOrder(o *domain.Order) *orderModel {
	items := make([]orderItemModel, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemModel{
			ID: it.ID, OrderID: o.ID, ProductID: it.ProductID, Title: it.Title,
			PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	return &orderModel{
		ID: o.ID, OrderNo: o.OrderNo, UserID: o.UserID, Status: string(o.Status),
		Currency: o.Currency, TotalCents: o.TotalCents, PaymentID: uuidString(o.PaymentID),
		ExpiresAt: o.ExpiresAt, PaidAt: o.PaidAt, Items: items,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func toPayment(m *paymentModel) *domain.Payment {
	return &domain.Payment{
		ID: m.ID, OrderID: m.OrderID, UserID: m.UserID, Provider: m.Provider,
		ProviderRef: m.ProviderRef, AmountCents: m.AmountCents, Currency: m.Currency,
		Status: domain.PaymentStatus(m.Status), FailureReason: m.FailureReason,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromPayment(p *domain.Payment) *paymentModel {
	return &paymentModel{
		ID: p.ID, OrderID: p.OrderID, UserID: p.UserID, Provider: p.Provider,
		ProviderRef: p.ProviderRef, AmountCents: p.AmountCents, Currency: p.Currency,
		Status: string(p.Status), FailureReason: p.FailureReason,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}
