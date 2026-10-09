package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// StaffService manages the people who operate the console and the roles they
// hold. It is separate from CustomerService on purpose: shoppers and staff have
// different rules, and the guards below (never lose the last admin, never lock
// yourself out) only make sense for staff.
type StaffService struct {
	users  port.UserRepository
	hasher port.PasswordHasher
	ids    port.IDGenerator
	clock  port.Clock
	auth   *AuthService
}

func NewStaffService(users port.UserRepository, hasher port.PasswordHasher, ids port.IDGenerator, clock port.Clock, auth *AuthService) *StaffService {
	return &StaffService{users: users, hasher: hasher, ids: ids, clock: clock, auth: auth}
}

// RoleView describes a role and what it can do, for the console's role matrix.
type RoleView struct {
	Role        domain.UserRole     `json:"role"`
	Permissions []domain.Permission `json:"permissions"`
	Description string              `json:"description"`
	Members     int                 `json:"members"`
	Catalog     []domain.Permission `json:"catalog"`
}

// CreateStaffInput is a new console user.
type CreateStaffInput struct {
	Email    string
	Name     string
	Role     domain.UserRole
	Password string
}

// UpdateStaffInput changes an existing console user. Nil fields are untouched.
type UpdateStaffInput struct {
	Name   *string
	Role   *domain.UserRole
	Status *domain.UserStatus
}

// StaffResult carries the one-time password when the service generated one.
type StaffResult struct {
	User              *domain.User
	GeneratedPassword string
}

func (s *StaffService) List(ctx context.Context, f domain.UserFilter) (domain.Page[domain.User], error) {
	f.OpsOnly = true
	return s.users.List(ctx, f)
}

// Roles describes every console role and its permissions, plus how many people
// hold it. The permission list is the same one the middleware enforces, so the
// screen cannot drift from the truth.
func (s *StaffService) Roles(ctx context.Context) ([]RoleView, error) {
	page, err := s.users.List(ctx, domain.UserFilter{OpsOnly: true, PageSize: 200})
	if err != nil {
		return nil, err
	}
	counts := map[domain.UserRole]int{}
	for _, u := range page.Items {
		counts[u.Role]++
	}

	catalog := make([]domain.Permission, 0, len(domain.AllPermissions()))
	catalog = append(catalog, domain.AllPermissions()...)

	out := make([]RoleView, 0, len(domain.OpsRoles()))
	for _, role := range domain.OpsRoles() {
		out = append(out, RoleView{
			Role:        role,
			Permissions: domain.Permissions(role),
			Description: roleDescription(role),
			Members:     counts[role],
			Catalog:     catalog,
		})
	}
	return out, nil
}

func roleDescription(role domain.UserRole) string {
	switch role {
	case domain.RoleAdmin:
		return "Full access, including settings, staff and the audit log."
	case domain.RoleSupport:
		return "Orders, refunds, returns, tickets, customers and their wallet or points."
	case domain.RoleCatalog:
		return "Products, categories, shipping, currency and storefront reviews."
	case domain.RoleFinance:
		return "Refunds, withdrawals, currency rates and the audit log."
	}
	return ""
}

// Create adds a console user. When no password is supplied a strong one is
// generated and returned exactly once, so an operator can hand it over without
// the service ever storing a weak default.
func (s *StaffService) Create(ctx context.Context, in CreateStaffInput) (*StaffResult, error) {
	email := normalizeEmail(in.Email)
	if email == "" {
		return nil, fmt.Errorf("%w: an email address is required", domain.ErrInvalidArgument)
	}
	if !domain.IsOpsRole(in.Role) {
		return nil, fmt.Errorf("%w: %q cannot sign in to the console", domain.ErrInvalidArgument, in.Role)
	}
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, fmt.Errorf("%w: an account with that email already exists", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	password := in.Password
	generated := ""
	if strings.TrimSpace(password) == "" {
		generated = GeneratePassword()
		password = generated
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("%w: the password must be at least 8 characters", domain.ErrInvalidArgument)
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = email
	}
	user := &domain.User{
		ID: s.ids.NewID(), Email: email, Name: name, PasswordHash: hash,
		Role: in.Role, Status: domain.UserActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return &StaffResult{User: user, GeneratedPassword: generated}, nil
}

// Update changes a console user's name, role or status. It refuses changes that
// would lock the operator out of their own console or leave the shop with no
// active administrator.
func (s *StaffService) Update(ctx context.Context, actorID, id string, in UpdateStaffInput) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Role != nil && *in.Role != user.Role {
		if !domain.IsOpsRole(*in.Role) {
			return nil, fmt.Errorf("%w: %q cannot sign in to the console", domain.ErrInvalidArgument, *in.Role)
		}
		if id == actorID {
			// Changing your own role is how an administrator accidentally
			// removes their own access.
			return nil, fmt.Errorf("%w: you cannot change your own role", domain.ErrInvalidArgument)
		}
		if user.Role == domain.RoleAdmin {
			if err := s.ensureAnotherAdmin(ctx, id); err != nil {
				return nil, err
			}
		}
		user.Role = *in.Role
	}
	if in.Status != nil && *in.Status != user.Status {
		switch *in.Status {
		case domain.UserActive, domain.UserDisabled:
		default:
			return nil, fmt.Errorf("%w: invalid status", domain.ErrInvalidArgument)
		}
		if id == actorID {
			return nil, fmt.Errorf("%w: you cannot change your own status", domain.ErrInvalidArgument)
		}
		if *in.Status == domain.UserDisabled && user.Role == domain.RoleAdmin {
			if err := s.ensureAnotherAdmin(ctx, id); err != nil {
				return nil, err
			}
		}
		user.Status = *in.Status
	}
	if in.Name != nil {
		user.Name = strings.TrimSpace(*in.Name)
	}

	user.UpdatedAt = s.clock.Now()
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ResetPassword issues a new password and signs the user out everywhere, so a
// reset also ends any session that may have been the reason for it.
func (s *StaffService) ResetPassword(ctx context.Context, id string) (*StaffResult, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	password := GeneratePassword()
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hash
	user.UpdatedAt = s.clock.Now()
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if s.auth != nil {
		// Best effort: the password is already changed, and the audit entry
		// records the reset either way.
		_ = s.auth.RevokeSessions(ctx, user.ID)
	}
	return &StaffResult{User: user, GeneratedPassword: password}, nil
}

// ensureAnotherAdmin fails when the only active administrator would be removed.
func (s *StaffService) ensureAnotherAdmin(ctx context.Context, excluding string) error {
	admins, err := s.users.List(ctx, domain.UserFilter{
		Role: rolePtr(domain.RoleAdmin), Status: statusPtr(domain.UserActive), PageSize: 200,
	})
	if err != nil {
		return err
	}
	for _, admin := range admins.Items {
		if admin.ID != excluding {
			return nil
		}
	}
	return fmt.Errorf("%w: the shop must keep at least one active administrator", domain.ErrConflict)
}

func rolePtr(role domain.UserRole) *domain.UserRole { return &role }

func statusPtr(status domain.UserStatus) *domain.UserStatus { return &status }

// GeneratePassword returns a URL-safe random password of 24 characters, well
// above the minimum length policy.
func GeneratePassword() string {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read only fails when the system source is unavailable, in which
		// case there is no safe password to invent.
		panic("openshop: no entropy for a password")
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
