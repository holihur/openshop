package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// fixtureNow matches the clock the fixture injects, so session timestamps in the
// tests line up with the revocation marker the service writes.
var fixtureNow = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func newStaffFixture() (*StaffService, *fakeUserRepo, *AuthService) {
	users := newFakeUserRepo()
	auth := NewAuthService(users, fakeHasher{}, fakeTokens{}, newFakeCache(), &seqIDs{},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, &fakeMailer{},
		AuthConfig{AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour}, nil, nil,
	)
	return NewStaffService(users, fakeHasher{}, &seqIDs{}, fixedClock{t: fixtureNow}, auth), users, auth
}

func TestCreateStaffGeneratesAPasswordOnce(t *testing.T) {
	svc, users, _ := newStaffFixture()
	ctx := context.Background()

	result, err := svc.Create(ctx, CreateStaffInput{Email: "Agent@Example.com", Role: domain.RoleSupport})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if result.GeneratedPassword == "" {
		t.Fatal("a password must be generated when none is supplied")
	}
	if len(result.GeneratedPassword) < 16 {
		t.Errorf("generated password is too short: %d", len(result.GeneratedPassword))
	}
	if result.User.Role != domain.RoleSupport || result.User.Status != domain.UserActive {
		t.Errorf("user = %+v", result.User)
	}
	// The address is normalised and the stored hash is not the password.
	if result.User.Email != "agent@example.com" {
		t.Errorf("email = %q", result.User.Email)
	}
	if result.User.PasswordHash == result.GeneratedPassword {
		t.Error("the password must be hashed, never stored as given")
	}
	if _, ok := users.byID[result.User.ID]; !ok {
		t.Error("the user was not persisted")
	}
}

func TestCreateStaffRejectsAShopperRole(t *testing.T) {
	svc, _, _ := newStaffFixture()
	ctx := context.Background()

	// A customer role cannot operate the console, so it must not be assignable.
	if _, err := svc.Create(ctx, CreateStaffInput{Email: "x@example.com", Role: domain.RoleCustomer}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
	if _, err := svc.Create(ctx, CreateStaffInput{Email: "", Role: domain.RoleSupport}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument for a missing email, got %v", err)
	}
	if _, err := svc.Create(ctx, CreateStaffInput{Email: "y@example.com", Role: domain.RoleSupport, Password: "short"}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument for a short password, got %v", err)
	}

	if _, err := svc.Create(ctx, CreateStaffInput{Email: "dup@example.com", Role: domain.RoleSupport}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.Create(ctx, CreateStaffInput{Email: "DUP@example.com", Role: domain.RoleCatalog}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected a conflict for a duplicate email, got %v", err)
	}
}

func TestStaffListExcludesShoppers(t *testing.T) {
	svc, users, _ := newStaffFixture()
	ctx := context.Background()

	now := time.Now()
	if err := users.Create(ctx, &domain.User{
		ID: "shopper", Email: "buyer@example.com", Role: domain.RoleCustomer,
		Status: domain.UserActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed shopper: %v", err)
	}
	if _, err := svc.Create(ctx, CreateStaffInput{Email: "agent@example.com", Role: domain.RoleSupport}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	page, err := svc.List(ctx, domain.UserFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Email != "agent@example.com" {
		t.Errorf("the staff list must contain only console users, got %+v", page.Items)
	}
}

func TestStaffCannotLockTheShopOut(t *testing.T) {
	svc, users, _ := newStaffFixture()
	ctx := context.Background()
	admin := &domain.User{
		ID: "admin-1", Email: "admin@example.com", Role: domain.RoleAdmin,
		Status: domain.UserActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := users.Create(ctx, admin); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	// The only administrator cannot demote themselves...
	role := domain.RoleSupport
	if _, err := svc.Update(ctx, admin.ID, admin.ID, UpdateStaffInput{Role: &role}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected self-demotion to be refused, got %v", err)
	}
	// ...nor disable themselves...
	disabled := domain.UserDisabled
	if _, err := svc.Update(ctx, admin.ID, admin.ID, UpdateStaffInput{Status: &disabled}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected self-disable to be refused, got %v", err)
	}

	// ...and neither can another operator, because the shop would be left with
	// no one who can change settings.
	other := &domain.User{
		ID: "admin-2", Email: "second@example.com", Role: domain.RoleAdmin,
		Status: domain.UserActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := users.Create(ctx, other); err != nil {
		t.Fatalf("seed second admin: %v", err)
	}
	if _, err := svc.Update(ctx, "admin-2", "admin-1", UpdateStaffInput{Role: &role}); err != nil {
		t.Fatalf("demoting one of two administrators should work: %v", err)
	}
	// With only admin-2 left, they can no longer be demoted by anybody.
	if _, err := svc.Update(ctx, "admin-1", "admin-2", UpdateStaffInput{Role: &role}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected the last administrator to be protected, got %v", err)
	}
}

func TestStaffInvalidStatusIsRejected(t *testing.T) {
	svc, users, _ := newStaffFixture()
	ctx := context.Background()
	agent := &domain.User{
		ID: "agent", Email: "agent@example.com", Role: domain.RoleSupport,
		Status: domain.UserActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := users.Create(ctx, agent); err != nil {
		t.Fatalf("seed: %v", err)
	}
	bogus := domain.UserStatus("suspended")
	if _, err := svc.Update(ctx, "someone-else", agent.ID, UpdateStaffInput{Status: &bogus}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
	role := domain.UserRole("root")
	if _, err := svc.Update(ctx, "someone-else", agent.ID, UpdateStaffInput{Role: &role}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument for an unknown role, got %v", err)
	}
}

func TestResetPasswordRevokesSessions(t *testing.T) {
	svc, users, auth := newStaffFixture()
	ctx := context.Background()
	agent := &domain.User{
		ID: "agent", Email: "agent@example.com", Role: domain.RoleSupport,
		Status: domain.UserActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := users.Create(ctx, agent); err != nil {
		t.Fatalf("seed: %v", err)
	}

	result, err := svc.ResetPassword(ctx, agent.ID)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if result.GeneratedPassword == "" {
		t.Fatal("a password must be returned")
	}
	// A token issued before the reset must no longer be accepted.
	valid, err := auth.SessionValid(ctx, agent.ID, fixtureNow.Add(-time.Minute))
	if err != nil {
		t.Fatalf("session check: %v", err)
	}
	if valid {
		t.Error("sessions issued before a password reset must be rejected")
	}
	// One issued after the reset is fine.
	valid, err = auth.SessionValid(ctx, agent.ID, fixtureNow.Add(time.Minute))
	if err != nil {
		t.Fatalf("session check: %v", err)
	}
	if !valid {
		t.Error("a session issued after the reset must be accepted")
	}
}

func TestSessionValidWithoutRevocation(t *testing.T) {
	_, _, auth := newStaffFixture()
	valid, err := auth.SessionValid(context.Background(), "nobody", fixtureNow)
	if err != nil {
		t.Fatalf("session check: %v", err)
	}
	if !valid {
		t.Error("without a revocation marker every session is valid")
	}
}

func TestRolesMatrixMatchesTheEnforcedPermissions(t *testing.T) {
	svc, _, _ := newStaffFixture()
	roles, err := svc.Roles(context.Background())
	if err != nil {
		t.Fatalf("roles: %v", err)
	}
	// Every ops role is described, and the matrix is the enforced list.
	if len(roles) != len(domain.OpsRoles()) {
		t.Fatalf("expected %d roles, got %d", len(domain.OpsRoles()), len(roles))
	}
	for _, view := range roles {
		if len(view.Catalog) == 0 {
			t.Errorf("role %s has no permission catalogue", view.Role)
		}
		want := domain.Permissions(view.Role)
		if len(view.Permissions) != len(want) {
			t.Errorf("role %s lists %d permissions, enforced %d", view.Role, len(view.Permissions), len(want))
		}
		if view.Description == "" {
			t.Errorf("role %s has no description", view.Role)
		}
	}
	// Support must not be able to manage staff or settings.
	for _, view := range roles {
		if view.Role != domain.RoleSupport {
			continue
		}
		for _, forbidden := range []domain.Permission{domain.PermStaffWrite, domain.PermSettingsWrite, domain.PermProductsWrite} {
			for _, granted := range view.Permissions {
				if granted == forbidden {
					t.Errorf("support must not hold %s", forbidden)
				}
			}
		}
	}
}

func TestGeneratePasswordIsUniqueAndURLSafe(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		p := GeneratePassword()
		if len(p) < 16 {
			t.Fatalf("password too short: %q", p)
		}
		for _, r := range p {
			if r == '+' || r == '/' || r == '=' {
				t.Fatalf("password must be URL safe: %q", p)
			}
		}
		if seen[p] {
			t.Fatalf("duplicate password generated: %q", p)
		}
		seen[p] = true
	}
}
