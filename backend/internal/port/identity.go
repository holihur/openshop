package port

import "context"

// Identity is a verified external identity returned by an identity provider.
//
// Subject is the provider's stable account identifier. It may be an email-less
// identifier (WeChat and Alipay do not return one), which is why accounts are
// linked through SocialAccount rather than by email alone.
type Identity struct {
	Provider  string
	Subject   string
	Email     string
	Name      string
	AvatarURL string
	// EmailVerified reports whether the provider vouched for the address. Only
	// then may it be matched against an existing account.
	EmailVerified bool
}

// IdentityProvider abstracts an OpenID Connect provider: it builds the
// authorization redirect and exchanges the callback code for a verified
// identity (ID token signature and claims are validated by the adapter).
type IdentityProvider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*Identity, error)
}
