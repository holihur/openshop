package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// fakeRoleRepo stores custom roles in memory.
type fakeRoleRepo struct {
	byKey map[string]domain.Role
}

func newFakeRoleRepo() *fakeRoleRepo { return &fakeRoleRepo{byKey: map[string]domain.Role{}} }

func (r *fakeRoleRepo) List(context.Context) ([]domain.Role, error) {
	out := make([]domain.Role, 0, len(r.byKey))
	for _, role := range r.byKey {
		out = append(out, role)
	}
	return out, nil
}

func (r *fakeRoleRepo) Find(_ context.Context, key string) (*domain.Role, error) {
	role, ok := r.byKey[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &role, nil
}

func (r *fakeRoleRepo) Upsert(_ context.Context, role *domain.Role) error {
	r.byKey[role.Key] = *role
	return nil
}

func (r *fakeRoleRepo) Delete(_ context.Context, key string) error {
	if _, ok := r.byKey[key]; !ok {
		return domain.ErrNotFound
	}
	delete(r.byKey, key)
	return nil
}

func newRoleFixture() (*RoleService, *fakeRoleRepo) {
	repo := newFakeRoleRepo()
	return NewRoleService(repo, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, newFakeCache()), repo
}

func TestBuiltinRolesResolveWithoutTheDatabase(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()

	// Admin is a superuser, and the others come from the code matrix.
	if !svc.HasPermission(ctx, "admin", domain.PermSettingsWrite) {
		t.Error("admin must hold every permission")
	}
	if !svc.HasPermission(ctx, "support", domain.PermOrdersWrite) {
		t.Error("support must be able to work orders")
	}
	if svc.HasPermission(ctx, "support", domain.PermProductsWrite) {
		t.Error("support must not touch the catalog")
	}
	if svc.HasPermission(ctx, "nonsense", domain.PermOrdersRead) {
		t.Error("an unknown role must grant nothing")
	}
}

func TestCreateCustomRoleAndUseIt(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()

	role, err := svc.Create(ctx, "admin", CreateRoleInput{
		Key: "Warehouse", Name: "Warehouse",
		Permissions: []domain.Permission{domain.PermOrdersRead, domain.PermOrdersWrite, domain.PermOrdersRead},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if role.Key != "warehouse" {
		t.Errorf("key = %q, want it lower-cased", role.Key)
	}
	// Duplicates collapse, and the stored order follows the catalogue.
	if len(role.Permissions) != 2 {
		t.Errorf("permissions = %v", role.Permissions)
	}
	if !svc.IsOpsRole(ctx, "warehouse") {
		t.Error("a custom role must be able to sign in to the console")
	}
	if !svc.HasPermission(ctx, "warehouse", domain.PermOrdersWrite) {
		t.Error("the custom role must grant what it lists")
	}
	if svc.HasPermission(ctx, "warehouse", domain.PermRefundsWrite) {
		t.Error("the custom role must not grant what it does not list")
	}
	// An edit applies to the next check.
	perms := []domain.Permission{domain.PermRefundsWrite}
	if _, err := svc.Update(ctx, "admin", "warehouse", UpdateRoleInput{Permissions: &perms}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !svc.HasPermission(ctx, "warehouse", domain.PermRefundsWrite) {
		t.Error("the edited permission was not applied")
	}
	if svc.HasPermission(ctx, "warehouse", domain.PermOrdersWrite) {
		t.Error("a removed permission is still granted")
	}
}

func TestCustomRoleCannotExceedItsAuthor(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()

	// An operator with only orders permissions must not be able to mint a role
	// that can change settings: that is privilege escalation.
	_, err := svc.Create(ctx, "support", CreateRoleInput{
		Key: "escalate", Name: "Escalate",
		Permissions: []domain.Permission{domain.PermSettingsWrite},
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	// What the author does hold is fine.
	if _, err := svc.Create(ctx, "support", CreateRoleInput{
		Key: "helper", Name: "Helper",
		Permissions: []domain.Permission{domain.PermOrdersWrite},
	}); err != nil {
		t.Fatalf("a permission the author holds must be grantable: %v", err)
	}
	// An administrator may grant anything, including a brand new combination.
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{
		Key: "opslead", Name: "Ops lead",
		Permissions: []domain.Permission{domain.PermSettingsRead, domain.PermAuditRead},
	}); err != nil {
		t.Fatalf("admin must be able to create any role: %v", err)
	}
}

func TestCustomRoleValidation(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()

	cases := []struct {
		name  string
		input CreateRoleInput
	}{
		{"bad key", CreateRoleInput{Key: "Bad Key!", Name: "x"}},
		{"empty name", CreateRoleInput{Key: "valid", Name: "  "}},
		{"unknown permission", CreateRoleInput{Key: "valid", Name: "x",
			Permissions: []domain.Permission{"orders:destroy"}}},
	}
	for _, c := range cases {
		if _, err := svc.Create(ctx, "admin", c.input); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Errorf("%s: expected invalid argument, got %v", c.name, err)
		}
	}

	if _, err := svc.Create(ctx, "admin", CreateRoleInput{Key: "dup", Name: "Dup"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{Key: "Dup", Name: "Other"}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected a conflict for a duplicate key, got %v", err)
	}
	// A custom role must not shadow a built-in one.
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{Key: "admin", Name: "Fake admin"}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected a conflict when shadowing a built-in role, got %v", err)
	}
}

func TestBuiltinRolesCannotBeEditedOrDeleted(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()
	name := "Renamed"
	if _, err := svc.Update(ctx, "admin", "support", UpdateRoleInput{Name: &name}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected a built-in role to be immutable, got %v", err)
	}
	if err := svc.Delete(ctx, "support", 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected a built-in role to be undeletable, got %v", err)
	}
}

func TestRoleWithMembersCannotBeDeleted(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{Key: "warehouse", Name: "Warehouse"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Accounts left on a role that no longer exists could not sign in.
	if err := svc.Delete(ctx, "warehouse", 2); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected a conflict while the role has members, got %v", err)
	}
	if err := svc.Delete(ctx, "warehouse", 0); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := svc.Find(ctx, "warehouse"); ok {
		t.Error("the role was not removed")
	}
}

func TestAssigningARoleChecksTheActorsOwnPermissions(t *testing.T) {
	svc, _ := newRoleFixture()
	ctx := context.Background()
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{
		Key: "accounting", Name: "Accounting",
		Permissions: []domain.Permission{domain.PermAuditRead, domain.PermCurrencyWrite},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Support cannot hand out a role that grants more than support holds.
	if err := svc.ValidateRoleForAssignment(ctx, "support", "accounting"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if err := svc.ValidateRoleForAssignment(ctx, "admin", "accounting"); err != nil {
		t.Fatalf("admin must be able to assign it: %v", err)
	}
	if err := svc.ValidateRoleForAssignment(ctx, "admin", "does-not-exist"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected an unknown role to be refused, got %v", err)
	}
}

func TestRoleCacheServesRepeatedChecks(t *testing.T) {
	repo := newFakeRoleRepo()
	svc := NewRoleService(repo, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, newFakeCache())
	ctx := context.Background()
	if _, err := svc.Create(ctx, "admin", CreateRoleInput{Key: "warehouse", Name: "Warehouse"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Warm the cache, then remove the role behind the service's back: it still
	// resolves until the cache is invalidated, which is what keeps a permission
	// check off the database on every request.
	if _, ok := svc.Find(ctx, "warehouse"); !ok {
		t.Fatal("the role should resolve before it is deleted")
	}
	if err := repo.Delete(ctx, "warehouse"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := svc.Find(ctx, "warehouse"); !ok {
		t.Error("the cached role should still resolve")
	}
	svc.invalidate(ctx)
	if _, ok := svc.Find(ctx, "warehouse"); ok {
		t.Error("an invalidated cache must reload from the repository")
	}
}
