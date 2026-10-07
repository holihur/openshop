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
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

func (u *User) CanLogin() bool { return u.Status == UserActive }

// UserFilter selects a page of users for the ops console.
type UserFilter struct {
	Keyword  string
	Role     *UserRole
	Status   *UserStatus
	Page     int
	PageSize int
}
