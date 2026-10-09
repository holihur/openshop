package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// fakeSocialRepo stores account links in memory.
type fakeSocialRepo struct {
	mu      sync.Mutex
	byKey   map[string]*domain.SocialAccount
	upserts int
}

func newFakeSocialRepo() *fakeSocialRepo {
	return &fakeSocialRepo{byKey: map[string]*domain.SocialAccount{}}
}

func (r *fakeSocialRepo) FindBySubject(_ context.Context, provider, subject string) (*domain.SocialAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if account, ok := r.byKey[provider+"|"+subject]; ok {
		cp := *account
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeSocialRepo) ListByUser(_ context.Context, userID string) ([]domain.SocialAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.SocialAccount{}
	for _, account := range r.byKey {
		if account.UserID == userID {
			out = append(out, *account)
		}
	}
	return out, nil
}

func (r *fakeSocialRepo) Upsert(_ context.Context, account *domain.SocialAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upserts++
	cp := *account
	r.byKey[account.Provider+"|"+account.Subject] = &cp
	return nil
}

// newSocialAuthFixture builds the service as the storefront surface does: a
// session may only carry the customer role.
func newSocialAuthFixture(settings map[string]string) (*AuthService, *fakeUserRepo, *fakeSocialRepo) {
	users := newFakeUserRepo()
	social := newFakeSocialRepo()
	svc := NewAuthService(users, fakeHasher{}, fakeTokens{}, newFakeCache(), &seqIDs{},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, &fakeMailer{},
		AuthConfig{
			AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour,
			AllowedRoles: []domain.UserRole{domain.RoleCustomer},
		},
		newTestSettings(settings), social,
	)
	return svc, users, social
}

func TestSocialCreatesAnAccountAndLinksIt(t *testing.T) {
	svc, users, social := newSocialAuthFixture(nil)
	ctx := context.Background()

	res, err := svc.Social(ctx, &port.Identity{
		Provider: "wechat", Subject: "UNION-1", Name: "微信用户",
	})
	if err != nil {
		t.Fatalf("social sign in: %v", err)
	}
	if res.AccessToken == "" {
		t.Fatal("no token issued")
	}
	// WeChat returns no email, so the account gets an internal placeholder that
	// can never receive mail.
	if res.User.Email != "wechat+union-1@social.invalid" {
		t.Errorf("email = %q", res.User.Email)
	}
	if res.User.EmailVerified {
		t.Error("a placeholder address must not be treated as verified")
	}
	account, err := social.FindBySubject(ctx, "wechat", "UNION-1")
	if err != nil {
		t.Fatalf("link missing: %v", err)
	}
	if account.UserID != res.User.ID || account.Name != "微信用户" {
		t.Errorf("account = %+v", account)
	}

	// Signing in again reuses the account instead of creating a second one.
	again, err := svc.Social(ctx, &port.Identity{Provider: "wechat", Subject: "UNION-1", Name: "新名字"})
	if err != nil {
		t.Fatalf("second sign in: %v", err)
	}
	if again.User.ID != res.User.ID {
		t.Errorf("second sign in created a new user: %s != %s", again.User.ID, res.User.ID)
	}
	if len(users.byID) != 1 {
		t.Errorf("expected one user, got %d", len(users.byID))
	}
}

func TestSocialLinksAVerifiedEmailToAnExistingAccount(t *testing.T) {
	svc, users, _ := newSocialAuthFixture(nil)
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterInput{Email: "buyer@example.com", Password: "supersecret"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	res, err := svc.Social(ctx, &port.Identity{
		Provider: "alipay", Subject: "2088", Email: "Buyer@Example.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("social sign in: %v", err)
	}
	if res.User.Email != "buyer@example.com" {
		t.Errorf("email = %q", res.User.Email)
	}
	// No second account: the verified address is the same person.
	if len(users.byID) != 1 {
		t.Errorf("expected the existing account to be reused, got %d users", len(users.byID))
	}
}

func TestSocialDoesNotTrustAnUnverifiedEmail(t *testing.T) {
	svc, users, _ := newSocialAuthFixture(nil)
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterInput{Email: "victim@example.com", Password: "supersecret"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// An unverified address matching an existing account must not sign in as
	// that account: the provider never confirmed it belongs to the caller.
	res, err := svc.Social(ctx, &port.Identity{
		Provider: "alipay", Subject: "999", Email: "victim@example.com", EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("social sign in: %v", err)
	}
	if res.User.Email == "victim@example.com" {
		t.Fatal("an unverified provider email must not take over an existing account")
	}
	if len(users.byID) != 2 {
		t.Errorf("expected a separate account, got %d users", len(users.byID))
	}
}

func TestSocialHonoursRegistrationToggle(t *testing.T) {
	svc, _, _ := newSocialAuthFixture(map[string]string{"auth.allow_registration": "false"})
	ctx := context.Background()

	// A new shopper cannot create an account through a provider either.
	if _, err := svc.Social(ctx, &port.Identity{Provider: "wechat", Subject: "NEW"}); !errors.Is(err, domain.ErrRegistrationDisabled) {
		t.Fatalf("expected registration to be disabled, got %v", err)
	}

	// An already linked identity still signs in.
	svc2, _, _ := newSocialAuthFixture(nil)
	if _, err := svc2.Social(ctx, &port.Identity{Provider: "wechat", Subject: "KNOWN"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc2.Social(ctx, &port.Identity{Provider: "wechat", Subject: "KNOWN"}); err != nil {
		t.Fatalf("a linked identity must keep working: %v", err)
	}
}

func TestSocialRejectsMissingSubject(t *testing.T) {
	svc, _, _ := newSocialAuthFixture(nil)
	ctx := context.Background()
	if _, err := svc.Social(ctx, &port.Identity{Provider: "wechat"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if _, err := svc.Social(ctx, &port.Identity{Subject: "x"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized without a provider, got %v", err)
	}
}

func TestSocialIgnoresAnUnsuitableRole(t *testing.T) {
	svc, users, _ := newSocialAuthFixture(nil)
	ctx := context.Background()
	// An administrator signing in through the storefront must be refused.
	if err := users.Create(ctx, &domain.User{
		ID: "admin", Email: "admin@example.com", Role: domain.RoleAdmin,
		Status: domain.UserActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if _, err := svc.Social(ctx, &port.Identity{
		Provider: "alipay", Subject: "1", Email: "admin@example.com", EmailVerified: true,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestSanitizeLocalPart(t *testing.T) {
	cases := map[string]string{
		"UNION-1":        "union-1",
		"OpenID_ABC.def": "openid_abc.def",
		"!!":             "user",
		"":               "user",
	}
	for input, want := range cases {
		if got := sanitizeLocalPart(input); got != want {
			t.Errorf("sanitizeLocalPart(%q) = %q, want %q", input, got, want)
		}
	}
}
