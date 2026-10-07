package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

func newAuthFixture() (*AuthService, *fakeCache, *fakeUserRepo) {
	users := newFakeUserRepo()
	cache := newFakeCache()
	svc := NewAuthService(users, fakeHasher{}, fakeTokens{}, cache, &seqIDs{},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, &fakeMailer{},
		AuthConfig{AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour},
		newTestSettings(map[string]string{"auth.password_reset_url": "http://test/reset"}),
	)
	return svc, cache, users
}

func TestRegisterAndLogin(t *testing.T) {
	svc, _, _ := newAuthFixture()
	ctx := context.Background()

	res, err := svc.Register(ctx, RegisterInput{Email: "Buyer@Example.com", Password: "supersecret", Name: "Buyer"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.User.Email != "buyer@example.com" {
		t.Fatalf("email not normalised: %s", res.User.Email)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatal("tokens not issued")
	}

	// Duplicate registration is rejected.
	if _, err := svc.Register(ctx, RegisterInput{Email: "buyer@example.com", Password: "supersecret"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate err = %v, want ErrConflict", err)
	}

	// Wrong password.
	if _, err := svc.Login(ctx, LoginInput{Identifier: "buyer@example.com", Password: "wrong"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("wrong password err = %v, want ErrUnauthorized", err)
	}

	// Correct login.
	if _, err := svc.Login(ctx, LoginInput{Identifier: "buyer@example.com", Password: "supersecret"}); err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	svc, cache, _ := newAuthFixture()
	ctx := context.Background()

	res, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "supersecret"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	refreshed, err := svc.Refresh(ctx, res.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.RefreshToken == res.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	// The old token must no longer work.
	if _, err := svc.Refresh(ctx, res.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("reused refresh err = %v, want ErrUnauthorized", err)
	}
	_ = cache
}

func newAuthFixtureWithMailer() (*AuthService, *fakeMailer, *fakeCache) {
	users := newFakeUserRepo()
	cache := newFakeCache()
	mailer := &fakeMailer{}
	svc := NewAuthService(users, fakeHasher{}, fakeTokens{}, cache, &seqIDs{},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, mailer,
		AuthConfig{AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour},
		newTestSettings(map[string]string{"auth.password_reset_url": "http://test/reset"}),
	)
	return svc, mailer, cache
}

func extractResetToken(html string) string {
	const marker = "token="
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(marker):]
	if j := strings.IndexAny(rest, "\"&<"); j >= 0 {
		return rest[:j]
	}
	return rest
}

func TestPasswordResetFlow(t *testing.T) {
	svc, mailer, _ := newAuthFixtureWithMailer()
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "oldpassword"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.RequestPasswordReset(ctx, "a@b.com"); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	msg, ok := mailer.last()
	if !ok {
		t.Fatal("no reset email sent")
	}
	token := extractResetToken(msg.HTML)
	if token == "" {
		t.Fatal("no token in reset email")
	}

	if err := svc.ResetPassword(ctx, token, "newpassword"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Old password no longer works; new one does.
	if _, err := svc.Login(ctx, LoginInput{Identifier: "a@b.com", Password: "oldpassword"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("old password still valid: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Identifier: "a@b.com", Password: "newpassword"}); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}

	// Existing sessions are revoked by the reset.
	if _, err := svc.Refresh(ctx, registered.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("session survived reset: %v", err)
	}

	// The reset token is single-use.
	if err := svc.ResetPassword(ctx, token, "anotherpassword"); !errors.Is(err, domain.ErrTokenInvalid) {
		t.Fatalf("reset token reused: %v", err)
	}
}

func TestRequestPasswordResetIsSilentForUnknownEmail(t *testing.T) {
	svc, mailer, _ := newAuthFixtureWithMailer()
	if err := svc.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("unknown email should not error: %v", err)
	}
	if _, ok := mailer.last(); ok {
		t.Fatal("no email should be sent for an unknown address")
	}
}

func TestRefreshReuseRevokesWholeFamily(t *testing.T) {
	svc, _, _ := newAuthFixture()
	ctx := context.Background()

	res, err := svc.Register(ctx, RegisterInput{Email: "c@d.com", Password: "supersecret"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Legitimate rotation produces a new token in the same family.
	rotated, err := svc.Refresh(ctx, res.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// An attacker replays the original (already used) token.
	if _, err := svc.Refresh(ctx, res.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("replay err = %v, want ErrUnauthorized", err)
	}

	// Reuse detection must revoke the whole family, including the rotated token.
	if _, err := svc.Refresh(ctx, rotated.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("family not revoked: err = %v, want ErrUnauthorized", err)
	}
}
