package domain

import (
	"net"
	"strings"
	"time"
)

// Scope is a capability carried by a personal access token. The vocabulary is
// "resource:action"; it mirrors the ops permissions so a token can never grant
// more than the owner's role, only less.
type Scope string

const (
	// ScopeAll grants every scope. It is only offered to administrators.
	ScopeAll Scope = "*"

	ScopeOrdersRead         Scope = "orders:read"
	ScopeOrdersWrite        Scope = "orders:write"
	ScopePaymentsWrite      Scope = "payments:write"
	ScopeRefundsWrite       Scope = "refunds:write"
	ScopeProductsRead       Scope = "products:read"
	ScopeProductsWrite      Scope = "products:write"
	ScopeCategoriesWrite    Scope = "categories:write"
	ScopeCustomersRead      Scope = "customers:read"
	ScopeCustomersWrite     Scope = "customers:write"
	ScopeWalletRead         Scope = "wallet:read"
	ScopeWalletWrite        Scope = "wallet:write"
	ScopePointsRead         Scope = "points:read"
	ScopePointsWrite        Scope = "points:write"
	ScopeLoyaltyRead        Scope = "loyalty:read"
	ScopeLoyaltyWrite       Scope = "loyalty:write"
	ScopeTicketsRead        Scope = "tickets:read"
	ScopeTicketsWrite       Scope = "tickets:write"
	ScopeReturnsRead        Scope = "returns:read"
	ScopeReturnsWrite       Scope = "returns:write"
	ScopeWithdrawalsRead    Scope = "withdrawals:read"
	ScopeWithdrawalsWrite   Scope = "withdrawals:write"
	ScopeCartRead           Scope = "cart:read"
	ScopeCartWrite          Scope = "cart:write"
	ScopeAddressesRead      Scope = "addresses:read"
	ScopeAddressesWrite     Scope = "addresses:write"
	ScopeWishlistRead       Scope = "wishlist:read"
	ScopeWishlistWrite      Scope = "wishlist:write"
	ScopeReviewsRead        Scope = "reviews:read"
	ScopeReviewsWrite       Scope = "reviews:write"
	ScopeProfileRead        Scope = "profile:read"
	ScopeProfileWrite       Scope = "profile:write"
	ScopeNotificationsRead  Scope = "notifications:read"
	ScopeNotificationsWrite Scope = "notifications:write"
	ScopeCouponsRead        Scope = "coupons:read"
	ScopeCouponsWrite       Scope = "coupons:write"
	ScopeShippingRead       Scope = "shipping:read"
	ScopeShippingWrite      Scope = "shipping:write"
	ScopeCurrencyWrite      Scope = "currency:write"
	ScopeSettingsRead       Scope = "settings:read"
	ScopeSettingsWrite      Scope = "settings:write"
	ScopeAnalyticsRead      Scope = "analytics:read"
	ScopeAuditRead          Scope = "audit:read"
)

// ScopeInfo describes one grantable scope for the console.
type ScopeInfo struct {
	Scope       Scope
	Group       string
	Description string
}

// ScopeRegistry is the full list of scopes a token may be granted, in display
// order. It is the single source of truth for validation and for the UI.
var ScopeRegistry = []ScopeInfo{
	{ScopeAll, "general", "Full access to everything the owner can do"},
	{ScopeProfileRead, "account", "Read the profile"},
	{ScopeProfileWrite, "account", "Update the profile, password and addresses"},
	{ScopeAddressesRead, "account", "Read shipping addresses"},
	{ScopeAddressesWrite, "account", "Create, update and delete addresses"},
	{ScopeCartRead, "shop", "Read the cart"},
	{ScopeCartWrite, "shop", "Change the cart"},
	{ScopeWishlistRead, "shop", "Read the wishlist"},
	{ScopeWishlistWrite, "shop", "Change the wishlist"},
	{ScopeOrdersRead, "orders", "Read orders"},
	{ScopeOrdersWrite, "orders", "Create, cancel and complete orders"},
	{ScopePaymentsWrite, "orders", "Start and simulate payments"},
	{ScopeRefundsWrite, "orders", "Issue refunds"},
	{ScopeReturnsRead, "orders", "Read return requests"},
	{ScopeReturnsWrite, "orders", "Decide return requests"},
	{ScopeProductsRead, "catalog", "Read products and categories"},
	{ScopeProductsWrite, "catalog", "Create and update products"},
	{ScopeCategoriesWrite, "catalog", "Create and update categories"},
	{ScopeReviewsRead, "catalog", "Read reviews"},
	{ScopeReviewsWrite, "catalog", "Moderate reviews"},
	{ScopeCustomersRead, "customers", "Read customers"},
	{ScopeCustomersWrite, "customers", "Update customers"},
	{ScopeWalletRead, "loyalty", "Read wallet balances and ledger"},
	{ScopeWalletWrite, "loyalty", "Top up and adjust the wallet"},
	{ScopePointsRead, "loyalty", "Read loyalty points"},
	{ScopePointsWrite, "loyalty", "Adjust loyalty points"},
	{ScopeLoyaltyRead, "loyalty", "Read loyalty and commission data"},
	{ScopeLoyaltyWrite, "loyalty", "Adjust loyalty and commission data"},
	{ScopeWithdrawalsRead, "loyalty", "Read withdrawal requests"},
	{ScopeWithdrawalsWrite, "loyalty", "Approve, reject and settle withdrawals"},
	{ScopeTicketsRead, "support", "Read support tickets"},
	{ScopeTicketsWrite, "support", "Reply to and triage tickets"},
	{ScopeNotificationsRead, "support", "Read notifications"},
	{ScopeNotificationsWrite, "support", "Broadcast notifications"},
	{ScopeCouponsRead, "marketing", "Read coupons"},
	{ScopeCouponsWrite, "marketing", "Create and update coupons"},
	{ScopeShippingRead, "settings", "Read shipping methods and zones"},
	{ScopeShippingWrite, "settings", "Change shipping methods and zones"},
	{ScopeCurrencyWrite, "settings", "Change currencies and rates"},
	{ScopeSettingsRead, "settings", "Read runtime settings"},
	{ScopeSettingsWrite, "settings", "Change runtime settings"},
	{ScopeAnalyticsRead, "insights", "Read dashboards and reports"},
	{ScopeAuditRead, "insights", "Read the audit log"},
}

// ValidScope reports whether a scope is in the registry.
func ValidScope(s Scope) bool {
	for _, info := range ScopeRegistry {
		if info.Scope == s {
			return true
		}
	}
	return false
}

// ParseScopes splits a stored scope string (space or comma separated).
func ParseScopes(raw string) []Scope {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\n' || r == '\t'
	})
	out := make([]Scope, 0, len(fields))
	for _, f := range fields {
		if ValidScope(Scope(f)) {
			out = append(out, Scope(f))
		}
	}
	return out
}

// ScopesString renders scopes for storage.
func ScopesString(scopes []Scope) string {
	parts := make([]string, 0, len(scopes))
	for _, s := range scopes {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, " ")
}

// ScopeSet is a token's granted scopes.
type ScopeSet []Scope

// Allows reports whether the set grants a required scope. ScopeAll grants all.
func (s ScopeSet) Allows(required Scope) bool {
	for _, granted := range s {
		if granted == ScopeAll || granted == required {
			return true
		}
	}
	return false
}

// PersonalAccessToken is a programmatic credential. Only the hash of the secret
// is persisted.
type PersonalAccessToken struct {
	ID         string
	UserID     string
	Name       string
	Prefix     string
	Realm      string // "front" | "ops"
	Scopes     []Scope
	CIDRs      []string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// Active reports whether the token may be used at the given time.
func (t PersonalAccessToken) Active(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	return t.ExpiresAt == nil || t.ExpiresAt.After(now)
}

// AllowsIP reports whether a client address is inside the token's allow-list.
// An empty list allows any address.
func (t PersonalAccessToken) AllowsIP(addr string) bool {
	if len(t.CIDRs) == 0 {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(addr))
	if ip == nil {
		return false
	}
	for _, raw := range t.CIDRs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(raw); err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}
		if parsed := net.ParseIP(raw); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}
	return false
}
