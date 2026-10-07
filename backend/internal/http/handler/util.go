package handler

import (
	"context"
	"fmt"
	"strconv"

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
