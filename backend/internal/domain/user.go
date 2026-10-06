package domain

import "time"

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleAdmin    UserRole = "admin"
)

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

// User is the account aggregate root. It never stores a raw password, only a
// hash produced by the PasswordHasher port.
type User struct {
	ID           string
	Email        string
	Phone        string
	PasswordHash string
	Name         string
	Role         UserRole
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

func (u *User) CanLogin() bool { return u.Status == UserActive }
