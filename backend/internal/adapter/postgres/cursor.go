package postgres

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// encodeCursor packs a (created_at, id) keyset position into an opaque token.
// The id breaks ties so pagination is stable even with equal timestamps.
func encodeCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", domain.ErrInvalidArgument
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", domain.ErrInvalidArgument
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", domain.ErrInvalidArgument
	}
	return createdAt, parts[1], nil
}
