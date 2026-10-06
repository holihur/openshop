package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

func newAuthFixture() (*AuthService, *fakeCache, *fakeUserRepo) {
	users := newFakeUserRepo()
	cache := newFakeCache()
	svc := NewAuthService(users, fakeHasher{}, fakeTokens{}, cache, &seqIDs{},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		AuthConfig{AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour},
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
