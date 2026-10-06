// Package service contains the application's business logic. Services depend
// exclusively on port interfaces (repositories, cache, lock, event bus,
// payment, mail...) and never import a concrete adapter, HTTP framework or
// SQL driver. This is what keeps third-party technology swappable and the
// binary horizontally scalable.
package service

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

// randomToken returns a URL-safe, high-entropy opaque token.
func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func clampPage(page, size, defSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defSize
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// ttlFrom returns the duration until t, or zero if t is in the past.
func ttlFrom(now, t time.Time) time.Duration {
	if d := t.Sub(now); d > 0 {
		return d
	}
	return 0
}

// slugify produces a URL-friendly identifier. Non-alphanumeric runs collapse to
// a single dash.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r > 127:
			// Preserve non-ASCII (e.g. CJK) characters as-is; PostgreSQL and the
			// frontend handle UTF-8 slugs fine.
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
