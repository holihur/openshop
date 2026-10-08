package domain

import "testing"

// The permission matrix is a security boundary: this test pins it down so an
// accidental grant (or a missing revoke) fails the build rather than shipping.
func TestPermissionMatrix(t *testing.T) {
	want := map[UserRole][]Permission{
		RoleSupport: {
			PermOrdersRead, PermOrdersWrite, PermRefundsWrite,
			PermReturnsRead, PermReturnsWrite,
			PermTicketsRead, PermTicketsWrite,
			PermLoyaltyRead, PermLoyaltyWrite,
			PermWithdrawalsRead, PermWithdrawalsWrite,
			PermNotificationsWrite,
			PermReviewsRead, PermReviewsWrite,
			PermCustomersRead, PermCustomersWrite,
			PermAnalyticsRead,
		},
		RoleCatalog: {
			PermProductsRead, PermProductsWrite, PermCategoriesWrite,
			PermShippingRead, PermShippingWrite, PermCurrencyWrite,
			PermReviewsRead, PermAnalyticsRead, PermTicketsRead,
			PermLoyaltyRead,
		},
		RoleFinance: {
			PermOrdersRead, PermRefundsWrite, PermReturnsRead,
			PermTicketsRead, PermLoyaltyRead, PermLoyaltyWrite,
			PermWithdrawalsRead, PermWithdrawalsWrite,
			PermCurrencyWrite, PermAnalyticsRead, PermAuditRead,
		},
	}

	for role, expected := range want {
		got := map[Permission]bool{}
		for _, p := range Permissions(role) {
			got[p] = true
		}
		for _, p := range expected {
			if !got[p] {
				t.Errorf("%s should have %s", role, p)
			}
			delete(got, p)
		}
		for extra := range got {
			t.Errorf("%s must not have %s", role, extra)
		}
	}

	// Every granted permission must be in the catalogue, otherwise a role could
	// hold a capability no route understands.
	valid := map[Permission]bool{}
	for _, p := range allPermissions {
		valid[p] = true
	}
	for role := range want {
		for _, p := range Permissions(role) {
			if !valid[p] {
				t.Errorf("%s holds %s, which is not in the catalogue", role, p)
			}
		}
	}

	// A customer holds nothing and is not an operations role.
	if len(Permissions(RoleCustomer)) != 0 {
		t.Error("a customer must hold no operations permission")
	}
	if IsOpsRole(RoleCustomer) {
		t.Error("a customer must not be an operations role")
	}
}
