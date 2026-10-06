package port

import (
	"context"
	"io"
	"time"
)

// ObjectStorage abstracts blob storage. Stateless, horizontally scaled servers
// must never write to local disk, so an S3/OSS (or local-dev) adapter is
// injected instead.
type ObjectStorage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (url string, err error)
	Delete(ctx context.Context, key string) error
	// URL returns a (possibly signed, time-limited) URL for a stored object.
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// Mailer abstracts transactional email delivery.
type Mailer interface {
	Send(ctx context.Context, msg Email) error
}

type Email struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// SMS abstracts text-message delivery, e.g. for OTP codes.
type SMS interface {
	Send(ctx context.Context, phone, template string, params map[string]string) error
}
