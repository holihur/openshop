package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
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
