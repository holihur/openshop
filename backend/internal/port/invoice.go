package port

import "time"

// InvoiceItem is one line on an invoice.
type InvoiceItem struct {
	Name       string
	SKU        string
	Quantity   int
	UnitCents  int64
	TotalCents int64
}

// InvoiceData is the fully-resolved content of an invoice, independent of any
// rendering technology.
type InvoiceData struct {
	OrderNumber     string
	IssuedAt        time.Time
	Status          string
	Customer        string
	Email           string
	ShippingAddress string
	Currency        string
	Items           []InvoiceItem
	SubtotalCents   int64
	DiscountCents   int64
	ShippingCents   int64
	TaxCents        int64
	TotalCents      int64
}

// InvoiceRenderer turns InvoiceData into a document (for example a PDF). It is a
// port so the document format and library stay swappable.
type InvoiceRenderer interface {
	Render(data InvoiceData) ([]byte, error)
	ContentType() string
}
