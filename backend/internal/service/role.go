package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// RoleService owns the permission model: the four roles defined in code plus the
// ones an operator creates.
//
// Every permission check goes through it, so a custom role is enforced by the
// same middleware as a built-in one, and a change to a role takes effect on the
// next request (behind a short cache) instead of needing a restart.
type RoleService struct {
	repo   port.RoleRepository
	clock  port.Clock
	cache  port.Cache
	ttl    time.Duration
	mu     sync.Mutex
	loaded time.Time
	custom map[string]domain.Role
}

const roleCacheKey = "rbac:roles"

func NewRoleService(repo port.RoleRepository, clock port.Clock, cache port.Cache) *RoleService {
	return &RoleService{repo: repo, clock: clock, cache: cache, ttl: 30 * time.Second,
		custom: map[string]domain.Role{}}
}

// customRoles returns the operator-defined roles, cached briefly so a permission
// check does not hit the database on every request.
func (s *RoleService) customRoles(ctx context.Context) map[string]domain.Role {
	s.mu.Lock()
	if s.custom != nil && s.clock.Now().Sub(s.loaded) < s.ttl {
		out := s.custom
		s.mu.Unlock()
		return out
	}
	s.mu.Unlock()

	roles, err := s.repo.List(ctx)
	if err != nil {
		// A read failure must not silently grant access: the last known set is
		// used, and an empty set means only built-in roles work.
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.custom == nil {
			s.custom = map[string]domain.Role{}
		}
		return s.custom
	}
	loaded := make(map[string]domain.Role, len(roles))
	for _, role := range roles {
		loaded[role.Key] = role
	}
	s.mu.Lock()
	s.custom, s.loaded = loaded, s.clock.Now()
	out := s.custom
	s.mu.Unlock()
	return out
}

// invalidate drops the cache so a role change applies immediately.
func (s *RoleService) invalidate(ctx context.Context) {
	s.mu.Lock()
	s.custom, s.loaded = nil, time.Time{}
	s.mu.Unlock()
	if s.cache != nil {
		_ = s.cache.Delete(ctx, roleCacheKey)
	}
}

// All returns every role: the built-ins first, then the custom ones by name.
func (s *RoleService) All(ctx context.Context) []domain.Role {
	out := make([]domain.Role, 0, len(domain.OpsRoles())+4)
	for _, key := range []domain.UserRole{domain.RoleSupport, domain.RoleCatalog, domain.RoleFinance, domain.RoleAdmin} {
		out = append(out, domain.Role{
			Key: string(key), Name: string(key), Description: roleDescription(key),
			Permissions: domain.Permissions(key), Builtin: true,
		})
	}
	custom := s.customRoles(ctx)
	keys := make([]string, 0, len(custom))
	for key := range custom {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		role := custom[key]
		out = append(out, role)
	}
	return out
}

// Find resolves one role by key.
func (s *RoleService) Find(ctx context.Context, key string) (domain.Role, bool) {
	key = strings.TrimSpace(key)
	for _, role := range s.All(ctx) {
		if role.Key == key {
			return role, true
		}
	}
	return domain.Role{}, false
}

// Permissions resolves a role to its permission list. It satisfies the
// middleware's resolver, so an unknown role grants nothing.
func (s *RoleService) Permissions(ctx context.Context, role string) []domain.Permission {
	found, ok := s.Find(ctx, role)
	if !ok {
		return nil
	}
	return found.Permissions
}

// HasPermission reports whether a role grants a permission.
func (s *RoleService) HasPermission(ctx context.Context, role string, perm domain.Permission) bool {
	found, ok := s.Find(ctx, role)
	if !ok {
		return false
	}
	return found.Has(perm)
}

// IsOpsRole reports whether a role may sign in to the console: the built-in ops
// roles and every role an operator has created, which is what makes a custom
// role usable at all.
func (s *RoleService) IsOpsRole(ctx context.Context, role string) bool {
	if domain.IsOpsRole(domain.UserRole(role)) {
		return true
	}
	_, ok := s.Find(ctx, role)
	return ok
}

// CreateRoleInput is a new custom role.
type CreateRoleInput struct {
	Key         string
	Name        string
	Description string
	Permissions []domain.Permission
}

// Create adds a custom role. The caller's own permissions are the ceiling: a
// role manager must not be able to mint a role that outranks them.
func (s *RoleService) Create(ctx context.Context, actorRole string, in CreateRoleInput) (*domain.Role, error) {
	key := strings.ToLower(strings.TrimSpace(in.Key))
	if !domain.ValidRoleKey(key) {
		return nil, fmt.Errorf("%w: a role key is lower case letters, digits, - or _", domain.ErrInvalidArgument)
	}
	if _, exists := s.Find(ctx, key); exists {
		return nil, fmt.Errorf("%w: a role with that key already exists", domain.ErrConflict)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: a role needs a name", domain.ErrInvalidArgument)
	}
	perms, err := s.validatePermissions(ctx, actorRole, in.Permissions)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	role := &domain.Role{
		Key: key, Name: name, Description: strings.TrimSpace(in.Description),
		Permissions: perms, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Upsert(ctx, role); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return role, nil
}

// UpdateRoleInput changes a custom role. Nil fields are untouched.
type UpdateRoleInput struct {
	Name        *string
	Description *string
	Permissions *[]domain.Permission
}

// Update edits a custom role. Built-in roles are refused: their meaning is part
// of the code and existing audit entries refer to it.
func (s *RoleService) Update(ctx context.Context, actorRole, key string, in UpdateRoleInput) (*domain.Role, error) {
	role, ok := s.Find(ctx, key)
	if !ok {
		return nil, domain.ErrNotFound
	}
	if role.Builtin {
		return nil, fmt.Errorf("%w: a built-in role cannot be changed", domain.ErrInvalidArgument)
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: a role needs a name", domain.ErrInvalidArgument)
		}
		role.Name = name
	}
	if in.Description != nil {
		role.Description = strings.TrimSpace(*in.Description)
	}
	if in.Permissions != nil {
		perms, err := s.validatePermissions(ctx, actorRole, *in.Permissions)
		if err != nil {
			return nil, err
		}
		role.Permissions = perms
	}
	role.UpdatedAt = s.clock.Now()
	if err := s.repo.Upsert(ctx, &role); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return &role, nil
}

// Delete removes a custom role that nobody holds. Keying by role means a role
// with members cannot simply be dropped: those accounts would keep a role that
// grants nothing and could not sign in.
func (s *RoleService) Delete(ctx context.Context, key string, members int) error {
	role, ok := s.Find(ctx, key)
	if !ok {
		return domain.ErrNotFound
	}
	if role.Builtin {
		return fmt.Errorf("%w: a built-in role cannot be deleted", domain.ErrInvalidArgument)
	}
	if members > 0 {
		return fmt.Errorf("%w: %d account(s) still hold this role", domain.ErrConflict, members)
	}
	if err := s.repo.Delete(ctx, key); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// validatePermissions keeps a role within what its author may grant, drops
// duplicates and rejects anything outside the catalogue (a typo would otherwise
// create a permission that no route checks).
func (s *RoleService) validatePermissions(ctx context.Context, actorRole string, requested []domain.Permission) ([]domain.Permission, error) {
	known := make(map[domain.Permission]bool, len(domain.AllPermissions()))
	for _, p := range domain.AllPermissions() {
		known[p] = true
	}
	granted := make(map[domain.Permission]bool)
	if actorRole != "" {
		for _, p := range s.Permissions(ctx, actorRole) {
			granted[p] = true
		}
	}

	out := make([]domain.Permission, 0, len(requested))
	seen := map[domain.Permission]bool{}
	for _, p := range requested {
		if !known[p] {
			return nil, fmt.Errorf("%w: unknown permission %q", domain.ErrInvalidArgument, p)
		}
		if actorRole != "" && !granted[p] {
			return nil, fmt.Errorf("%w: you cannot grant %q, which you do not hold", domain.ErrForbidden, p)
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	// A stable order keeps the stored list and the console's display identical
	// to the catalogue order.
	ordered := make([]domain.Permission, 0, len(out))
	for _, p := range domain.AllPermissions() {
		if seen[p] {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

// ValidateRoleForAssignment reports whether a role may be given to a staff
// member, and that the actor may grant it.
func (s *RoleService) ValidateRoleForAssignment(ctx context.Context, actorRole, role string) error {
	target, ok := s.Find(ctx, role)
	if !ok {
		return fmt.Errorf("%w: %q is not a role", domain.ErrInvalidArgument, role)
	}
	if actorRole == "" || target.Key == string(domain.RoleAdmin) {
		return nil
	}
	// Only an administrator (or whoever holds every permission the target role
	// grants) may hand it out, so a role manager cannot escalate themselves.
	for _, p := range target.Permissions {
		if !s.HasPermission(ctx, actorRole, p) {
			return fmt.Errorf("%w: you cannot assign %q, which grants permissions you do not hold",
				domain.ErrForbidden, role)
		}
	}
	return nil
}
