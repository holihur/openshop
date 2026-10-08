package security

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

func claims(audience string) port.TokenClaims {
	now := time.Now()
	return port.TokenClaims{
		Subject: "user-1", Role: domain.RoleAdmin, ID: "jti-1",
		IssuedAt: now, Expires: now.Add(time.Hour),
	}
}

func TestJWTIssueAndVerifyRoundTrip(t *testing.T) {
	issuer := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "front")
	raw, err := issuer.Issue(claims("front"))
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	got, err := issuer.Verify(raw)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Subject != "user-1" || got.Role != domain.RoleAdmin || got.ID != "jti-1" {
		t.Fatalf("unexpected claims: %+v", got)
	}
}

func TestJWTRejectsForeignSecret(t *testing.T) {
	signer := NewJWTIssuer("secret-one-long-enough-for-tests", "openshop", "front")
	verifier := NewJWTIssuer("secret-two-long-enough-for-tests", "openshop", "front")
	raw, _ := signer.Issue(claims("front"))
	if _, err := verifier.Verify(raw); !errors.Is(err, domain.ErrTokenInvalid) {
		t.Fatalf("a token signed with another secret must be rejected, got %v", err)
	}
}

// The audience binds a token to one surface: a storefront token must never be
// accepted by the operations API and vice versa.
func TestJWTRejectsWrongAudience(t *testing.T) {
	front := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "front")
	ops := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "ops")
	raw, _ := front.Issue(claims("front"))
	if _, err := ops.Verify(raw); err == nil {
		t.Fatal("a storefront token must not verify against the ops audience")
	}
}

func TestJWTRejectsWrongIssuer(t *testing.T) {
	signer := NewJWTIssuer("a-sufficiently-long-test-secret", "someone-else", "front")
	verifier := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "front")
	raw, _ := signer.Issue(claims("front"))
	if _, err := verifier.Verify(raw); err == nil {
		t.Fatal("a token from another issuer must be rejected")
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	issuer := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "front")
	past := time.Now().Add(-2 * time.Hour)
	c := claims("front")
	c.IssuedAt = past
	c.Expires = past.Add(time.Minute)
	raw, _ := issuer.Issue(c)
	if _, err := issuer.Verify(raw); !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("an expired token must report ErrTokenExpired, got %v", err)
	}
}

// An unsigned ("alg: none") token must never be accepted, which is the classic
// JWT algorithm-confusion attack.
func TestJWTRejectsUnsignedToken(t *testing.T) {
	issuer := NewJWTIssuer("a-sufficiently-long-test-secret", "openshop", "front")
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user-1", "role": "admin", "iss": "openshop",
		"aud": []string{"front"}, "exp": time.Now().Add(time.Hour).Unix(),
	})
	raw, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if _, err := issuer.Verify(raw); err == nil {
		t.Fatal("an unsigned token must be rejected")
	}
}

func TestBcryptHasher(t *testing.T) {
	h := NewBcryptHasher()
	hash, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("the hash must not equal the plaintext")
	}
	if !h.Compare(hash, "correct horse battery staple") {
		t.Error("the correct password must verify")
	}
	if h.Compare(hash, "wrong password") {
		t.Error("a wrong password must not verify")
	}
	if h.Compare("not-a-bcrypt-hash", "anything") {
		t.Error("a malformed hash must not verify")
	}
}
