package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// JWTIssuer implements port.TokenIssuer with HMAC-signed JWTs. Tokens are
// self-contained so any instance can verify them without shared session state.
type JWTIssuer struct {
	secret []byte
	issuer string
}

func NewJWTIssuer(secret, issuer string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer}
}

type jwtClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWTIssuer) Issue(c port.TokenClaims) (string, error) {
	claims := jwtClaims{
		Role: string(c.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.Subject,
			Issuer:    j.issuer,
			ID:        c.ID,
			IssuedAt:  jwt.NewNumericDate(c.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(c.Expires),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTIssuer) Verify(raw string) (*port.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrTokenInvalid
		}
		return j.secret, nil
	}, jwt.WithIssuer(j.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrTokenInvalid
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrTokenInvalid
	}
	var issuedAt, expires time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	if claims.ExpiresAt != nil {
		expires = claims.ExpiresAt.Time
	}
	return &port.TokenClaims{
		Subject:  claims.Subject,
		Role:     domain.UserRole(claims.Role),
		IssuedAt: issuedAt,
		Expires:  expires,
		ID:       claims.ID,
	}, nil
}

var _ port.TokenIssuer = (*JWTIssuer)(nil)
