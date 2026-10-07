package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const ctxLocale = "request.locale"

// Locale resolves the request locale from ?locale= or Accept-Language and
// stores it on the context so handlers can localize content (category names).
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.Query("locale")
		if locale == "" {
			locale = parseAcceptLanguage(c.GetHeader("Accept-Language"))
		}
		c.Set(ctxLocale, normalizeLocale(locale))
		c.Next()
	}
}

// LocaleOf returns the resolved locale, or "" when none was requested.
func LocaleOf(c *gin.Context) string {
	if v, ok := c.Get(ctxLocale); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func normalizeLocale(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if i := strings.IndexAny(raw, "-_"); i > 0 {
		return raw[:i]
	}
	return raw
}

// parseAcceptLanguage returns the highest-priority language tag.
func parseAcceptLanguage(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	first := header
	if i := strings.IndexByte(header, ','); i >= 0 {
		first = header[:i]
	}
	if i := strings.IndexByte(first, ';'); i >= 0 {
		first = first[:i]
	}
	return strings.TrimSpace(first)
}
