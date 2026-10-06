// Package invoice renders invoices to PDF. It is a dependency-free renderer:
// text is drawn with the built-in Helvetica fonts, so no external assets or
// libraries are needed and the output is deterministic.
package invoice

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/holihur/openshop/internal/port"
)

const (
	pageWidth  = 595.0
	pageHeight = 842.0
	marginX    = 50.0
)

// PDFRenderer implements port.InvoiceRenderer using a minimal PDF writer.
type PDFRenderer struct{}

func NewPDFRenderer() *PDFRenderer { return &PDFRenderer{} }

func (PDFRenderer) ContentType() string { return "application/pdf" }

// Render produces a single-page A4 PDF invoice.
func (PDFRenderer) Render(data port.InvoiceData) ([]byte, error) {
	content := buildContent(data)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefPos := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefPos)
	return buf.Bytes(), nil
}

type canvas struct {
	b strings.Builder
	y float64
}

func (c *canvas) text(x float64, bold bool, size float64, s string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&c.b, "BT /%s %.1f Tf 1 0 0 1 %.1f %.1f Tm (%s) Tj ET\n", font, size, x, c.y, escape(s))
}

func (c *canvas) rule() {
	fmt.Fprintf(&c.b, "0.6 w %.1f %.1f m %.1f %.1f l S\n", marginX, c.y, pageWidth-marginX, c.y)
}

func (c *canvas) row(label, value string, bold bool) {
	c.text(marginX, false, 10, label)
	c.text(pageWidth-marginX-140, bold, 10, value)
}

func buildContent(data port.InvoiceData) string {
	c := &canvas{y: pageHeight - 70}

	// Header.
	c.text(marginX, true, 22, "OpenShop")
	c.text(pageWidth-marginX-140, true, 22, "INVOICE")
	c.y -= 30

	c.row("Order", data.OrderNumber, false)
	c.y -= 15
	c.row("Date", data.IssuedAt.Format("2006-01-02 15:04"), false)
	c.y -= 15
	c.row("Status", strings.ToUpper(data.Status), false)
	c.y -= 25

	c.text(marginX, true, 12, "Billed to")
	c.y -= 16
	c.text(marginX, false, 10, data.Customer)
	c.y -= 14
	if data.Email != "" {
		c.text(marginX, false, 10, data.Email)
		c.y -= 14
	}
	for _, line := range wrap(data.ShippingAddress, 80) {
		c.text(marginX, false, 10, line)
		c.y -= 14
	}

	c.y -= 10
	c.text(marginX, true, 10, "Item")
	c.text(pageWidth-marginX-230, true, 10, "SKU")
	c.text(pageWidth-marginX-130, true, 10, "Qty")
	c.text(pageWidth-marginX-70, true, 10, "Amount")
	c.y -= 6
	c.rule()
	c.y -= 16

	for _, it := range data.Items {
		name := it.Name
		if len(name) > 52 {
			name = name[:49] + "..."
		}
		c.text(marginX, false, 10, name)
		c.text(pageWidth-marginX-230, false, 10, it.SKU)
		c.text(pageWidth-marginX-130, false, 10, fmt.Sprintf("%d", it.Quantity))
		c.text(pageWidth-marginX-70, false, 10, money(data.Currency, it.TotalCents))
		c.y -= 16
	}

	c.y -= 6
	c.rule()
	c.y -= 18
	c.row("Subtotal", money(data.Currency, data.SubtotalCents), false)
	c.y -= 15
	if data.DiscountCents > 0 {
		c.row("Discount", "-"+money(data.Currency, data.DiscountCents), false)
		c.y -= 15
	}
	c.row("Shipping", money(data.Currency, data.ShippingCents), false)
	c.y -= 15
	c.row("Tax", money(data.Currency, data.TaxCents), false)
	c.y -= 15
	c.row("Total", money(data.Currency, data.TotalCents), true)

	c.y = 60
	c.text(marginX, false, 9, "Thank you for your order. This invoice was generated automatically by OpenShop.")

	return c.b.String()
}

func money(currency string, cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%s %.2f", sign, currency, float64(cents)/100)
}

func wrap(s string, width int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var lines []string
	for len(s) > width {
		cut := strings.LastIndexByte(s[:width], ' ')
		if cut <= 0 {
			cut = width
		}
		lines = append(lines, s[:cut])
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		lines = append(lines, s)
	}
	return lines
}

// escape makes a string safe for a PDF literal, mapping characters outside the
// WinAnsi range (for example CJK) to '?' since the built-in fonts cannot render
// them.
func escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\r', '\n', '\t':
			b.WriteByte(' ')
		default:
			if r < 32 || r > 255 {
				b.WriteByte('?')
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
