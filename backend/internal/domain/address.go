package domain

import (
	"strings"
	"time"
)

// Address is a shipping address in a user's address book. Orders snapshot the
// address at checkout so later edits never change historical orders.
type Address struct {
	ID         string
	UserID     string
	Recipient  string
	Phone      string
	Province   string
	City       string
	District   string
	Line1      string
	PostalCode string
	Default    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OneLine renders the address for display and receipts.
func (a *Address) OneLine() string {
	parts := []string{a.Province, a.City, a.District, a.Line1, a.PostalCode}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, " ")
}
