package handler

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/service"
)

// wrapBind converts a Gin binding error into a domain error so the response
// layer maps it to HTTP 400 with a helpful message.
func WrapBind(err error) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, err.Error())
}

func queryInt(c *gin.Context, key string, def int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func ParsePage(c *gin.Context, defSize int) (int, int) {
	return queryInt(c, "page", 1), queryInt(c, "pageSize", defSize)
}

// SettingLimit returns a per-request integer provider backed by a runtime
// setting, for use with the rate-limit middleware. It resolves the value on
// every request so ops-console changes take effect without a restart.
func SettingLimit(settings *service.SettingsService, key string) func(ctx context.Context) int {
	return func(ctx context.Context) int {
		if settings == nil {
			return 0
		}
		return settings.Int(ctx, key)
	}
}

// ParseOptionalCents reads a money filter in major units ("129.90") and returns
// it in cents. An unparseable or negative value is ignored rather than failing
// the whole listing.
func ParseOptionalCents(raw string) *int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return nil
	}
	cents := int64(math.Round(v * 100))
	return &cents
}

// ParseAttributes collects attribute filters from query parameters carrying the
// given prefix (attr.color=red). It lets the storefront filter on any variant
// attribute without a schema change.
func ParseAttributes(c *gin.Context, prefix string) map[string]string {
	out := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		if !strings.HasPrefix(key, prefix) || len(values) == 0 {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(key, prefix))
		value := strings.TrimSpace(values[0])
		if name != "" && value != "" {
			out[name] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
