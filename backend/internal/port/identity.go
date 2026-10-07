package port

import "context"

// Identity is a verified external identity returned by an OIDC provider.
type Identity struct {
	Subject string
	Email   string
	Name    string
}

// IdentityProvider abstracts an OpenID Connect provider: it builds the
// authorization redirect and exchanges the callback code for a verified
// identity (ID token signature and claims are validated by the adapter).
type IdentityProvider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*Identity, error)
}
