package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

// ListTokenScopes lists every scope a token may be granted. It is shared by the
// storefront and the console.
func (h *Handler) ListTokenScopes(c *gin.Context) {
	response.OK(c, ScopeCatalog())
}

// PersonalAccessTokenView is the API shape of a token. The secret is never
// included; it is returned only once, at creation.
type PersonalAccessTokenView struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Realm      string     `json:"realm"`
	Scopes     []string   `json:"scopes"`
	CIDRs      []string   `json:"cidrs"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func ToPATView(t domain.PersonalAccessToken) PersonalAccessTokenView {
	scopes := make([]string, 0, len(t.Scopes))
	for _, s := range t.Scopes {
		scopes = append(scopes, string(s))
	}
	cidrs := t.CIDRs
	if cidrs == nil {
		cidrs = []string{}
	}
	return PersonalAccessTokenView{
		ID: t.ID, Name: t.Name, Prefix: t.Prefix, Realm: t.Realm, Scopes: scopes,
		CIDRs: cidrs, ExpiresAt: t.ExpiresAt, LastUsedAt: t.LastUsedAt,
		RevokedAt: t.RevokedAt, CreatedAt: t.CreatedAt,
	}
}

// CreatedPATView is returned once when a token is minted.
type CreatedPATView struct {
	PersonalAccessTokenView
	Token string `json:"token"`
}

// ScopeView describes a grantable scope.
type ScopeView struct {
	Scope       string `json:"scope"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

// ScopeCatalog returns every grantable scope.
func ScopeCatalog() []ScopeView {
	out := make([]ScopeView, 0, len(domain.ScopeRegistry))
	for _, s := range domain.ScopeRegistry {
		out = append(out, ScopeView{Scope: string(s.Scope), Group: s.Group, Description: s.Description})
	}
	return out
}
