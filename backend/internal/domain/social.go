package domain

import "time"

// SocialAccount links an external identity to a shop account. The provider's
// subject is the stable identifier; the latest profile fields are cached here so
// the account area can show which identities are linked.
type SocialAccount struct {
	ID        string
	Provider  string
	Subject   string
	UserID    string
	Email     string
	Name      string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SocialProviderInfo describes a sign-in provider for the storefront.
type SocialProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
