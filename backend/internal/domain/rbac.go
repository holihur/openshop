package domain

// Permission is a fine-grained capability granted to an operations role. The
// ops API is guarded per route by permission rather than by a single admin
// flag, so a support agent can handle orders without touching the catalog.
type Permission string

const (
	PermProductsRead    Permission = "products:read"
	PermProductsWrite   Permission = "products:write"
	PermCategoriesWrite Permission = "categories:write"
	PermOrdersRead      Permission = "orders:read"
	PermOrdersWrite     Permission = "orders:write"
	PermRefundsWrite    Permission = "refunds:write"
	PermReturnsRead     Permission = "returns:read"
	PermReturnsWrite    Permission = "returns:write"
	PermCouponsRead     Permission = "coupons:read"
	PermCouponsWrite    Permission = "coupons:write"
	PermShippingRead    Permission = "shipping:read"
	PermShippingWrite   Permission = "shipping:write"
	PermCurrencyWrite   Permission = "currency:write"
	PermReviewsRead     Permission = "reviews:read"
	PermReviewsWrite    Permission = "reviews:write"
	PermAnalyticsRead   Permission = "analytics:read"
	PermAuditRead       Permission = "audit:read"
	PermSettingsRead    Permission = "settings:read"
	PermSettingsWrite   Permission = "settings:write"
)

// allPermissions is the full catalogue, granted to the admin superuser role.
var allPermissions = []Permission{
	PermProductsRead, PermProductsWrite, PermCategoriesWrite,
	PermOrdersRead, PermOrdersWrite, PermRefundsWrite,
	PermReturnsRead, PermReturnsWrite,
	PermCouponsRead, PermCouponsWrite,
	PermShippingRead, PermShippingWrite, PermCurrencyWrite,
	PermReviewsRead, PermReviewsWrite,
	PermAnalyticsRead, PermAuditRead,
	PermSettingsRead, PermSettingsWrite,
}

// rolePermissions maps the non-superuser ops roles to their capabilities.
// RoleAdmin implicitly has every permission and is not listed here.
var rolePermissions = map[UserRole]map[Permission]bool{
	// Support: fulfilment and customer issues, no catalog or pricing.
	RoleSupport: set(
		PermOrdersRead, PermOrdersWrite, PermRefundsWrite,
		PermReturnsRead, PermReturnsWrite,
		PermReviewsRead, PermReviewsWrite,
		PermAnalyticsRead,
	),
	// Catalog: merchandising, shipping configuration and storefront reviews.
	RoleCatalog: set(
		PermProductsRead, PermProductsWrite, PermCategoriesWrite,
		PermShippingRead, PermShippingWrite, PermCurrencyWrite,
		PermReviewsRead, PermAnalyticsRead,
	),
	// Finance: money movement and audit, read-mostly on everything else.
	RoleFinance: set(
		PermOrdersRead, PermRefundsWrite, PermReturnsRead,
		PermCurrencyWrite, PermAnalyticsRead, PermAuditRead,
	),
}

func set(perms ...Permission) map[Permission]bool {
	m := make(map[Permission]bool, len(perms))
	for _, p := range perms {
		m[p] = true
	}
	return m
}

// IsOpsRole reports whether a role may sign in to the operations console.
func IsOpsRole(r UserRole) bool {
	switch r {
	case RoleAdmin, RoleSupport, RoleCatalog, RoleFinance:
		return true
	}
	return false
}

// HasPermission reports whether a role grants a permission. Admin is a
// superuser; the zero role (anonymous) grants nothing.
func HasPermission(r UserRole, p Permission) bool {
	if r == RoleAdmin {
		return true
	}
	return rolePermissions[r][p]
}

// Permissions lists a role's permissions, in a stable order. It is exposed to
// the console so it can hide actions the caller cannot perform.
func Permissions(r UserRole) []Permission {
	if r == RoleAdmin {
		out := make([]Permission, len(allPermissions))
		copy(out, allPermissions)
		return out
	}
	granted := rolePermissions[r]
	if len(granted) == 0 {
		return nil
	}
	out := make([]Permission, 0, len(granted))
	for _, p := range allPermissions {
		if granted[p] {
			out = append(out, p)
		}
	}
	return out
}
