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

// jsonMap persists a map[string]string as a JSONB column.
type jsonMap map[string]string

func (m jsonMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]string(m))
}

func (m *jsonMap) Scan(src any) error {
	if src == nil {
		*m = nil
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("jsonMap: unsupported type %T", src)
	}
	if len(raw) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(raw, (*map[string]string)(m))
}

type userModel struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	Email           string `gorm:"size:255;uniqueIndex;not null"`
	Phone           string `gorm:"size:32;index"`
	PasswordHash    string `gorm:"size:255;not null"`
	Name            string `gorm:"size:128"`
	Role            string `gorm:"size:32;index;not null;default:customer"`
	Status          string `gorm:"size:32;not null;default:active"`
	EmailVerified   bool   `gorm:"not null;default:false"`
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
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
	WeightGrams int        `gorm:"not null;default:0"`
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
	ID                 string     `gorm:"type:uuid;primaryKey"`
	OrderNo            string     `gorm:"size:64;uniqueIndex;not null"`
	UserID             uuidString `gorm:"type:uuid;index"`
	GuestEmail         string     `gorm:"size:255;not null;default:''"`
	GuestPhone         string     `gorm:"size:32;not null;default:''"`
	AccessToken        string     `gorm:"size:128;not null;default:''"`
	Status             string     `gorm:"size:32;index;not null"`
	Currency           string     `gorm:"size:8;not null"`
	SubtotalCents      int64      `gorm:"not null;default:0"`
	DiscountCents      int64      `gorm:"not null;default:0"`
	CouponID           uuidString `gorm:"type:uuid;index"`
	CouponCode         string     `gorm:"size:64;not null;default:''"`
	TotalCents         int64      `gorm:"not null"`
	RefundedCents      int64      `gorm:"not null;default:0"`
	ShippingCents      int64      `gorm:"not null;default:0"`
	TaxCents           int64      `gorm:"not null;default:0"`
	ShippingMethodID   uuidString `gorm:"type:uuid;index"`
	ShippingMethodName string     `gorm:"size:128;not null;default:''"`
	PaymentID          uuidString `gorm:"type:uuid;index"`
	ShippingJSON       []byte     `gorm:"column:shipping_address;type:jsonb"`
	TrackingNo         string     `gorm:"size:128;not null;default:''"`
	ShippedAt          *time.Time
	CompletedAt        *time.Time
	ExpiresAt          time.Time        `gorm:"index;not null"`
	PaidAt             *time.Time       `gorm:"index"`
	Items              []orderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	CreatedAt          time.Time        `gorm:"not null"`
	UpdatedAt          time.Time        `gorm:"not null"`
}

func (orderModel) TableName() string { return "orders" }

type orderItemModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	OrderID     string     `gorm:"type:uuid;index;not null"`
	ProductID   string     `gorm:"type:uuid;index;not null"`
	VariantID   uuidString `gorm:"type:uuid;index"`
	VariantName string     `gorm:"size:160;not null;default:''"`
	SKU         string     `gorm:"size:64;not null;default:''"`
	Title       string     `gorm:"size:255;not null"`
	PriceCents  int64      `gorm:"not null"`
	Quantity    int        `gorm:"not null"`
	Subtotal    int64      `gorm:"not null"`
}

func (orderItemModel) TableName() string { return "order_items" }

type refundModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	OrderID     string     `gorm:"type:uuid;index;not null"`
	PaymentID   uuidString `gorm:"type:uuid;index"`
	AmountCents int64      `gorm:"not null"`
	Reason      string     `gorm:"size:512;not null;default:''"`
	Restock     bool       `gorm:"not null;default:false"`
	CreatedAt   time.Time  `gorm:"not null"`
}

func (refundModel) TableName() string { return "refunds" }

func toRefund(m *refundModel) *domain.Refund {
	return &domain.Refund{
		ID: m.ID, OrderID: m.OrderID, PaymentID: string(m.PaymentID),
		AmountCents: m.AmountCents, Reason: m.Reason, Restock: m.Restock, CreatedAt: m.CreatedAt,
	}
}

func fromRefund(r *domain.Refund) *refundModel {
	return &refundModel{
		ID: r.ID, OrderID: r.OrderID, PaymentID: uuidString(r.PaymentID),
		AmountCents: r.AmountCents, Reason: r.Reason, Restock: r.Restock, CreatedAt: r.CreatedAt,
	}
}

type returnModel struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	OrderID   string     `gorm:"type:uuid;index;not null"`
	UserID    uuidString `gorm:"type:uuid;index"`
	Reason    string     `gorm:"size:512;not null;default:''"`
	Status    string     `gorm:"size:32;index;not null;default:'requested'"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (returnModel) TableName() string { return "return_requests" }

func toReturn(m *returnModel) *domain.ReturnRequest {
	return &domain.ReturnRequest{
		ID: m.ID, OrderID: m.OrderID, UserID: string(m.UserID),
		Reason: m.Reason, Status: domain.ReturnStatus(m.Status),
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromReturn(r *domain.ReturnRequest) *returnModel {
	return &returnModel{
		ID: r.ID, OrderID: r.OrderID, UserID: uuidString(r.UserID),
		Reason: r.Reason, Status: string(r.Status),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

type paymentModel struct {
	ID            string     `gorm:"type:uuid;primaryKey"`
	OrderID       string     `gorm:"type:uuid;index;not null"`
	UserID        uuidString `gorm:"type:uuid;index"`
	Provider      string     `gorm:"size:64;index;not null"`
	ProviderRef   string     `gorm:"size:128;index"`
	AmountCents   int64      `gorm:"not null"`
	Currency      string     `gorm:"size:8;not null"`
	Status        string     `gorm:"size:32;index;not null"`
	FailureReason string     `gorm:"size:512"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
}

func (paymentModel) TableName() string { return "payments" }

type couponModel struct {
	ID               string `gorm:"type:uuid;primaryKey"`
	Code             string `gorm:"size:64;not null"`
	Description      string `gorm:"type:text;not null;default:''"`
	DiscountType     string `gorm:"size:16;not null"`
	DiscountValue    int64  `gorm:"not null;default:0"`
	MinSubtotalCents int64  `gorm:"not null;default:0"`
	MaxDiscountCents int64  `gorm:"not null;default:0"`
	UsageLimit       int    `gorm:"not null;default:0"`
	UsedCount        int    `gorm:"not null;default:0"`
	PerUserLimit     int    `gorm:"not null;default:1"`
	StartsAt         *time.Time
	EndsAt           *time.Time
	Active           bool      `gorm:"not null;default:true"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (couponModel) TableName() string { return "coupons" }

type couponRedemptionModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	CouponID  string    `gorm:"type:uuid;index;not null"`
	UserID    string    `gorm:"type:uuid;index;not null"`
	OrderID   string    `gorm:"type:uuid;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (couponRedemptionModel) TableName() string { return "coupon_redemptions" }

type reviewModel struct {
	ID               string    `gorm:"type:uuid;primaryKey"`
	ProductID        string    `gorm:"type:uuid;index;not null"`
	UserID           string    `gorm:"type:uuid;index;not null"`
	Rating           int16     `gorm:"not null"`
	Title            string    `gorm:"size:160;not null;default:''"`
	Body             string    `gorm:"type:text;not null;default:''"`
	VerifiedPurchase bool      `gorm:"not null;default:false"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (reviewModel) TableName() string { return "reviews" }

type variantModel struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	ProductID   string    `gorm:"type:uuid;index;not null"`
	SKU         string    `gorm:"size:64;not null"`
	Name        string    `gorm:"size:160;not null"`
	PriceCents  int64     `gorm:"not null;default:0"`
	Stock       int       `gorm:"not null;default:0"`
	WeightGrams int       `gorm:"not null;default:0"`
	Attributes  jsonMap   `gorm:"type:jsonb"`
	Sort        int       `gorm:"not null;default:0"`
	Active      bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (variantModel) TableName() string { return "product_variants" }

type addressModel struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	UserID     string    `gorm:"type:uuid;index;not null"`
	Recipient  string    `gorm:"size:128;not null"`
	Phone      string    `gorm:"size:32;not null;default:''"`
	Province   string    `gorm:"size:64;not null;default:''"`
	City       string    `gorm:"size:64;not null;default:''"`
	District   string    `gorm:"size:64;not null;default:''"`
	Line1      string    `gorm:"size:255;not null"`
	PostalCode string    `gorm:"size:16;not null;default:''"`
	Default    bool      `gorm:"column:is_default;not null;default:false"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

func (addressModel) TableName() string { return "addresses" }

type shippingMethodModel struct {
	ID                 string    `gorm:"type:uuid;primaryKey"`
	Code               string    `gorm:"size:64;not null"`
	Name               string    `gorm:"size:128;not null"`
	FlatRateCents      int64     `gorm:"not null;default:0"`
	FreeThresholdCents int64     `gorm:"not null;default:0"`
	Active             bool      `gorm:"not null;default:true"`
	Sort               int       `gorm:"not null;default:0"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (shippingMethodModel) TableName() string { return "shipping_methods" }

type shippingZoneModel struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	Name      string     `gorm:"size:128;not null"`
	Provinces stringList `gorm:"type:jsonb"`
	Active    bool       `gorm:"not null;default:true"`
	Sort      int        `gorm:"not null;default:0"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (shippingZoneModel) TableName() string { return "shipping_zones" }

type shippingRateModel struct {
	ID                 string `gorm:"type:uuid;primaryKey"`
	ZoneID             string `gorm:"type:uuid;index;not null"`
	MethodID           string `gorm:"type:uuid;index;not null"`
	FlatRateCents      int64  `gorm:"not null;default:0"`
	FreeThresholdCents int64  `gorm:"not null;default:0"`
	PerKgCents         int64  `gorm:"not null;default:0"`
}

func (shippingRateModel) TableName() string { return "shipping_rates" }

// ---- mappers: persistence <-> domain ----

func toUser(m *userModel) *domain.User {
	return &domain.User{
		ID: m.ID, Email: m.Email, Phone: m.Phone, PasswordHash: m.PasswordHash,
		Name: m.Name, Role: domain.UserRole(m.Role), Status: domain.UserStatus(m.Status),
		EmailVerified: m.EmailVerified, EmailVerifiedAt: m.EmailVerifiedAt,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromUser(u *domain.User) *userModel {
	return &userModel{
		ID: u.ID, Email: u.Email, Phone: u.Phone, PasswordHash: u.PasswordHash,
		Name: u.Name, Role: string(u.Role), Status: string(u.Status),
		EmailVerified: u.EmailVerified, EmailVerifiedAt: u.EmailVerifiedAt,
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
		WeightGrams: m.WeightGrams,
		CoverImage:  m.CoverImage, Images: []string(m.Images), Status: domain.ProductStatus(m.Status),
		Stock: m.Stock, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromProduct(p *domain.Product) *productModel {
	return &productModel{
		ID: p.ID, CategoryID: uuidString(p.CategoryID), Title: p.Title, Slug: p.Slug,
		Description: p.Description, PriceCents: p.PriceCents, Currency: p.Currency,
		WeightGrams: p.WeightGrams,
		CoverImage:  p.CoverImage, Images: stringList(p.Images), Status: string(p.Status),
		Stock: p.Stock, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toOrder(m *orderModel) *domain.Order {
	items := make([]domain.OrderItem, 0, len(m.Items))
	for _, it := range m.Items {
		items = append(items, domain.OrderItem{
			ID: it.ID, OrderID: it.OrderID, ProductID: it.ProductID,
			VariantID: string(it.VariantID), VariantName: it.VariantName, SKU: it.SKU,
			Title: it.Title, PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	return &domain.Order{
		ID: m.ID, OrderNo: m.OrderNo, UserID: string(m.UserID), Status: domain.OrderStatus(m.Status),
		GuestEmail: m.GuestEmail, GuestPhone: m.GuestPhone, AccessToken: m.AccessToken,
		Currency: m.Currency, SubtotalCents: m.SubtotalCents, DiscountCents: m.DiscountCents,
		CouponID: string(m.CouponID), CouponCode: m.CouponCode, TotalCents: m.TotalCents,
		RefundedCents: m.RefundedCents,
		ShippingCents: m.ShippingCents, TaxCents: m.TaxCents,
		ShippingMethodID: string(m.ShippingMethodID), ShippingMethodName: m.ShippingMethodName,
		Items: items, PaymentID: string(m.PaymentID),
		ShippingAddress: decodeAddress(m.ShippingJSON),
		TrackingNo:      m.TrackingNo, ShippedAt: m.ShippedAt, CompletedAt: m.CompletedAt,
		ExpiresAt: m.ExpiresAt, PaidAt: m.PaidAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func decodeAddress(raw []byte) *domain.Address {
	if len(raw) == 0 {
		return nil
	}
	var a domain.Address
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil
	}
	return &a
}

func encodeAddress(a *domain.Address) []byte {
	if a == nil {
		return nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil
	}
	return b
}

func fromOrder(o *domain.Order) *orderModel {
	items := make([]orderItemModel, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemModel{
			ID: it.ID, OrderID: o.ID, ProductID: it.ProductID,
			VariantID: uuidString(it.VariantID), VariantName: it.VariantName, SKU: it.SKU,
			Title: it.Title, PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	return &orderModel{
		ID: o.ID, OrderNo: o.OrderNo, UserID: uuidString(o.UserID), Status: string(o.Status),
		GuestEmail: o.GuestEmail, GuestPhone: o.GuestPhone, AccessToken: o.AccessToken,
		Currency: o.Currency, SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents,
		CouponID: uuidString(o.CouponID), CouponCode: o.CouponCode, TotalCents: o.TotalCents,
		ShippingCents: o.ShippingCents, TaxCents: o.TaxCents,
		ShippingMethodID: uuidString(o.ShippingMethodID), ShippingMethodName: o.ShippingMethodName,
		PaymentID: uuidString(o.PaymentID), ShippingJSON: encodeAddress(o.ShippingAddress),
		TrackingNo: o.TrackingNo, ShippedAt: o.ShippedAt, CompletedAt: o.CompletedAt,
		ExpiresAt: o.ExpiresAt, PaidAt: o.PaidAt, Items: items,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func toPayment(m *paymentModel) *domain.Payment {
	return &domain.Payment{
		ID: m.ID, OrderID: m.OrderID, UserID: string(m.UserID), Provider: m.Provider,
		ProviderRef: m.ProviderRef, AmountCents: m.AmountCents, Currency: m.Currency,
		Status: domain.PaymentStatus(m.Status), FailureReason: m.FailureReason,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromPayment(p *domain.Payment) *paymentModel {
	return &paymentModel{
		ID: p.ID, OrderID: p.OrderID, UserID: uuidString(p.UserID), Provider: p.Provider,
		ProviderRef: p.ProviderRef, AmountCents: p.AmountCents, Currency: p.Currency,
		Status: string(p.Status), FailureReason: p.FailureReason,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toCoupon(m *couponModel) *domain.Coupon {
	return &domain.Coupon{
		ID: m.ID, Code: m.Code, Description: m.Description,
		DiscountType: domain.DiscountType(m.DiscountType), DiscountValue: m.DiscountValue,
		MinSubtotalCents: m.MinSubtotalCents, MaxDiscountCents: m.MaxDiscountCents,
		UsageLimit: m.UsageLimit, UsedCount: m.UsedCount, PerUserLimit: m.PerUserLimit,
		StartsAt: m.StartsAt, EndsAt: m.EndsAt, Active: m.Active,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromCoupon(c *domain.Coupon) *couponModel {
	return &couponModel{
		ID: c.ID, Code: c.Code, Description: c.Description,
		DiscountType: string(c.DiscountType), DiscountValue: c.DiscountValue,
		MinSubtotalCents: c.MinSubtotalCents, MaxDiscountCents: c.MaxDiscountCents,
		UsageLimit: c.UsageLimit, UsedCount: c.UsedCount, PerUserLimit: c.PerUserLimit,
		StartsAt: c.StartsAt, EndsAt: c.EndsAt, Active: c.Active,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toReview(m *reviewModel) *domain.Review {
	return &domain.Review{
		ID: m.ID, ProductID: m.ProductID, UserID: m.UserID, Rating: int(m.Rating),
		Title: m.Title, Body: m.Body, VerifiedPurchase: m.VerifiedPurchase, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromReview(r *domain.Review) *reviewModel {
	return &reviewModel{
		ID: r.ID, ProductID: r.ProductID, UserID: r.UserID, Rating: int16(r.Rating),
		Title: r.Title, Body: r.Body, VerifiedPurchase: r.VerifiedPurchase, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toVariant(m *variantModel) *domain.Variant {
	return &domain.Variant{
		ID: m.ID, ProductID: m.ProductID, SKU: m.SKU, Name: m.Name,
		PriceCents: m.PriceCents, Stock: m.Stock, WeightGrams: m.WeightGrams,
		Attributes: map[string]string(m.Attributes),
		Sort:       m.Sort, Active: m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromVariant(v *domain.Variant) *variantModel {
	return &variantModel{
		ID: v.ID, ProductID: v.ProductID, SKU: v.SKU, Name: v.Name,
		PriceCents: v.PriceCents, Stock: v.Stock, WeightGrams: v.WeightGrams,
		Attributes: jsonMap(v.Attributes),
		Sort:       v.Sort, Active: v.Active, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

func toAddress(m *addressModel) *domain.Address {
	return &domain.Address{
		ID: m.ID, UserID: m.UserID, Recipient: m.Recipient, Phone: m.Phone,
		Province: m.Province, City: m.City, District: m.District, Line1: m.Line1,
		PostalCode: m.PostalCode, Default: m.Default, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromAddress(a *domain.Address) *addressModel {
	return &addressModel{
		ID: a.ID, UserID: a.UserID, Recipient: a.Recipient, Phone: a.Phone,
		Province: a.Province, City: a.City, District: a.District, Line1: a.Line1,
		PostalCode: a.PostalCode, Default: a.Default, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func toShippingMethod(m *shippingMethodModel) *domain.ShippingMethod {
	return &domain.ShippingMethod{
		ID: m.ID, Code: m.Code, Name: m.Name, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, Active: m.Active, Sort: m.Sort,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromShippingMethod(m *domain.ShippingMethod) *shippingMethodModel {
	return &shippingMethodModel{
		ID: m.ID, Code: m.Code, Name: m.Name, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, Active: m.Active, Sort: m.Sort,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toShippingZone(m *shippingZoneModel) *domain.ShippingZone {
	return &domain.ShippingZone{
		ID: m.ID, Name: m.Name, Provinces: []string(m.Provinces),
		Active: m.Active, Sort: m.Sort, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromShippingZone(z *domain.ShippingZone) *shippingZoneModel {
	return &shippingZoneModel{
		ID: z.ID, Name: z.Name, Provinces: stringList(z.Provinces),
		Active: z.Active, Sort: z.Sort, CreatedAt: z.CreatedAt, UpdatedAt: z.UpdatedAt,
	}
}

func toShippingRate(m *shippingRateModel) *domain.ShippingRate {
	return &domain.ShippingRate{
		ID: m.ID, ZoneID: m.ZoneID, MethodID: m.MethodID, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, PerKgCents: m.PerKgCents,
	}
}

func fromShippingRate(r *domain.ShippingRate) *shippingRateModel {
	return &shippingRateModel{
		ID: r.ID, ZoneID: r.ZoneID, MethodID: r.MethodID, FlatRateCents: r.FlatRateCents,
		FreeThresholdCents: r.FreeThresholdCents, PerKgCents: r.PerKgCents,
	}
}
