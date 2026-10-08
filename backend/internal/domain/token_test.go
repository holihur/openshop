package domain

import (
	"testing"
	"time"
)

func TestPersonalAccessTokenAllowsIP(t *testing.T) {
	cases := []struct {
		name  string
		cidrs []string
		ip    string
		want  bool
	}{
		{"no list allows any address", nil, "203.0.113.9", true},
		{"exact address matches", []string{"203.0.113.9"}, "203.0.113.9", true},
		{"exact address differs", []string{"203.0.113.9"}, "203.0.113.10", false},
		{"inside the network", []string{"10.0.0.0/8"}, "10.1.2.3", true},
		{"outside the network", []string{"10.0.0.0/8"}, "11.1.2.3", false},
		{"any entry in the list matches", []string{"10.0.0.0/8", "203.0.113.7"}, "203.0.113.7", true},
		{"blank entries are ignored", []string{" ", "10.0.0.0/8"}, "10.0.0.1", true},
		{"unparseable client address is rejected", []string{"10.0.0.0/8"}, "not-an-ip", false},
		{"ipv6 network", []string{"2001:db8::/32"}, "2001:db8::1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := PersonalAccessToken{CIDRs: tc.cidrs}
			if got := token.AllowsIP(tc.ip); got != tc.want {
				t.Errorf("AllowsIP(%q) with %v = %v, want %v", tc.ip, tc.cidrs, got, tc.want)
			}
		})
	}
}

func TestPersonalAccessTokenActive(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if !(&PersonalAccessToken{}).Active(now) {
		t.Error("a token with no expiry and no revocation must be active")
	}
	if (&PersonalAccessToken{RevokedAt: &past}).Active(now) {
		t.Error("a revoked token must not be active")
	}
	if (&PersonalAccessToken{ExpiresAt: &past}).Active(now) {
		t.Error("an expired token must not be active")
	}
	if !(&PersonalAccessToken{ExpiresAt: &future}).Active(now) {
		t.Error("an unexpired token must be active")
	}
}

func TestScopeSetAllows(t *testing.T) {
	read := ScopeSet{ScopeOrdersRead}
	if !read.Allows(ScopeOrdersRead) {
		t.Error("a granted scope must be allowed")
	}
	if read.Allows(ScopeOrdersWrite) {
		t.Error("an ungranted scope must be denied")
	}
	wildcard := ScopeSet{ScopeAll}
	if !wildcard.Allows(ScopeOrdersWrite) || !wildcard.Allows(ScopeSettingsWrite) {
		t.Error("the wildcard scope must allow everything")
	}
	if (ScopeSet{}).Allows(ScopeOrdersRead) {
		t.Error("an empty scope set must allow nothing")
	}
}

func TestParseScopesIgnoresUnknown(t *testing.T) {
	scopes := ParseScopes("orders:read, orders:write made-up:scope\nsettings:read")
	want := []Scope{ScopeOrdersRead, ScopeOrdersWrite, ScopeSettingsRead}
	if len(scopes) != len(want) {
		t.Fatalf("got %v, want %v", scopes, want)
	}
	for i := range want {
		if scopes[i] != want[i] {
			t.Errorf("scope %d = %q, want %q", i, scopes[i], want[i])
		}
	}
}

// Every scope offered in the console must be usable, otherwise an operator can
// grant a scope that no route ever accepts.
func TestScopeRegistryIsValid(t *testing.T) {
	if len(ScopeRegistry) == 0 {
		t.Fatal("scope registry is empty")
	}
	for _, info := range ScopeRegistry {
		if !ValidScope(info.Scope) {
			t.Errorf("registry entry %q is not valid", info.Scope)
		}
		if info.Group == "" || info.Description == "" {
			t.Errorf("scope %q is missing display metadata", info.Scope)
		}
	}
	if ValidScope("nope:nope") {
		t.Error("an unregistered scope must not validate")
	}
}
