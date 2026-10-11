package domain

import (
	"regexp"
	"strings"
	"time"
)

// Role is a named set of permissions. The four built-in roles are defined in
// code; the rest are created by an operator, which is why a role can be either.
type Role struct {
	Key         string
	Name        string
	Description string
	Permissions []Permission
	// Builtin marks a role defined in code, which the console may read but not
	// edit or delete: changing one would change the meaning of existing
	// sessions and audit entries.
	Builtin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// roleKeyPattern keeps a role key usable as a setting, a URL segment and a JWT
// claim without escaping.
var roleKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,63}$`)

// ValidRoleKey reports whether a key is acceptable for a custom role.
func ValidRoleKey(key string) bool {
	return roleKeyPattern.MatchString(strings.TrimSpace(key))
}

// Has reports whether the role grants a permission. The admin role is a
// superuser; a role with no permissions grants nothing.
func (r Role) Has(p Permission) bool {
	if r.Key == string(RoleAdmin) {
		return true
	}
	for _, granted := range r.Permissions {
		if granted == p {
			return true
		}
	}
	return false
}
