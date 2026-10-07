package domain

import "testing"

func TestHasPermission(t *testing.T) {
	// Admin is a superuser and has everything.
	for _, p := range allPermissions {
		if !HasPermission(RoleAdmin, p) {
			t.Errorf("admin should have %s", p)
		}
	}
	// A support agent can fulfil orders but cannot edit the catalog.
	if !HasPermission(RoleSupport, PermOrdersWrite) {
		t.Error("support should write orders")
	}
	if HasPermission(RoleSupport, PermProductsWrite) {
		t.Error("support must not write products")
	}
	// Finance can refund and read audit, but not ship orders.
	if !HasPermission(RoleFinance, PermRefundsWrite) || !HasPermission(RoleFinance, PermAuditRead) {
		t.Error("finance should refund and read audit")
	}
	if HasPermission(RoleFinance, PermOrdersWrite) {
		t.Error("finance must not ship orders")
	}
	// Customers and anonymous callers get nothing.
	if HasPermission(RoleCustomer, PermOrdersRead) || HasPermission("", PermOrdersRead) {
		t.Error("non-ops roles must have no permissions")
	}
}

func TestPermissionsListing(t *testing.T) {
	if got := len(Permissions(RoleAdmin)); got != len(allPermissions) {
		t.Fatalf("admin permission count = %d, want %d", got, len(allPermissions))
	}
	if Permissions(RoleCustomer) != nil {
		t.Error("customer should have no permissions")
	}
	// Every listed permission must actually be granted (guards against drift).
	for _, p := range Permissions(RoleCatalog) {
		if !HasPermission(RoleCatalog, p) {
			t.Errorf("catalog lists %s but does not grant it", p)
		}
	}
}

func TestIsOpsRole(t *testing.T) {
	for _, r := range []UserRole{RoleAdmin, RoleSupport, RoleCatalog, RoleFinance} {
		if !IsOpsRole(r) {
			t.Errorf("%s should be an ops role", r)
		}
	}
	if IsOpsRole(RoleCustomer) || IsOpsRole("") {
		t.Error("customer/anonymous must not be ops roles")
	}
}
