package port

import (
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// PasswordHasher abstracts password hashing so the algorithm can change without
// touching services.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}

// TokenClaims is the transport-agnostic content of an access token.
type TokenClaims struct {
	Subject  string
	Role     domain.UserRole
	IssuedAt time.Time
	Expires  time.Time
	ID       string // jti, used for the Redis deny-list on logout
}

// TokenIssuer abstracts JWT (or opaque token) creation and verification. Tokens
// are self-contained so no server-side session store is required, which is what
// lets the API scale horizontally.
type TokenIssuer interface {
	Issue(claims TokenClaims) (string, error)
	Verify(token string) (*TokenClaims, error)
}

// Clock is injected so time-dependent behaviour is testable and workers agree
// across instances.
type Clock interface {
	Now() time.Time
}

// SystemClock is the default production clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

// IDGenerator produces globally unique, sortable identifiers.
type IDGenerator interface {
	NewID() string
}
