package invoice

import (
	"bytes"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/port"
)

func TestPDFRendererProducesValidDocument(t *testing.T) {
	r := NewPDFRenderer()
	if r.ContentType() != "application/pdf" {
		t.Fatalf("content type = %q", r.ContentType())
	}
	out, err := r.Render(port.InvoiceData{
		OrderNumber:     "OS20250101ABC",
		IssuedAt:        time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Status:          "paid",
		Customer:        "Ada Lovelace",
		Email:           "ada@example.com",
		ShippingAddress: "1 Analytical Engine Way, London",
		Currency:        "CNY",
		Items: []port.InvoiceItem{
			{Name: "Widget", SKU: "SKU-1", Quantity: 2, UnitCents: 1000, TotalCents: 2000},
		},
		SubtotalCents: 2000, DiscountCents: 200, ShippingCents: 800, TaxCents: 168, TotalCents: 2768,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatalf("missing PDF header: %q", out[:8])
	}
	if !bytes.HasSuffix(bytes.TrimSpace(out), []byte("%%EOF")) {
		t.Fatal("missing EOF marker")
	}
	if !bytes.Contains(out, []byte("Ada Lovelace")) {
		t.Fatal("customer missing from document")
	}
	if !bytes.Contains(out, []byte("OS20250101ABC")) {
		t.Fatal("order number missing from document")
	}
	if !bytes.Contains(out, []byte("startxref")) {
		t.Fatal("missing xref table")
	}
}

func TestEscapeSanitisesPDFAndNonLatin(t *testing.T) {
	got := escape(`a(b)c\d 北`)
	want := `a\(b\)c\\d ?`
	if got != want {
		t.Fatalf("escape = %q, want %q", got, want)
	}
}
