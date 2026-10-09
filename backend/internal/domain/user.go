package domain

import "time"

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleAdmin    UserRole = "admin"
	RoleSupport  UserRole = "support"
	RoleCatalog  UserRole = "catalog"
	RoleFinance  UserRole = "finance"
)

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

// User is the account aggregate root. It never stores a raw password, only a
// hash produced by the PasswordHasher port. PasswordHash is never serialised.
type User struct {
	ID              string
	Email           string
	Phone           string
	PasswordHash    string `json:"-"`
	Name            string
	Role            UserRole
	Status          UserStatus
	EmailVerified   bool
	EmailVerifiedAt *time.Time
	// FailedAttempts and LockedUntil implement per-account throttling.
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Locked reports whether the account is temporarily locked out.
func (u *User) Locked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

func (u *User) CanLogin() bool { return u.Status == UserActive }

// UserFilter selects a page of users for the ops console.
type UserFilter struct {
	Keyword string
	Role    *UserRole
	Status  *UserStatus
	// OpsOnly restricts the result to the roles that may operate the console,
	// which is what the staff screen shows.
	OpsOnly  bool
	Page     int
	PageSize int
}
