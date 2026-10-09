package service

import (
	"context"
	"strings"
	"sync"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// SocialFactory builds a sign-in provider from its runtime configuration. It is
// injected so the service layer never imports an adapter.
type SocialFactory func(id string, opts SocialProviderOptions) (port.IdentityProvider, bool)

// SocialProviderOptions is the configuration one provider needs. Secrets are
// supplied separately from the environment, never from the settings table.
type SocialProviderOptions struct {
	ID          string
	AppID       string
	AppSecret   string
	RedirectURL string
}

// SocialService resolves the enabled social sign-in providers. Each provider is
// switched on and configured at runtime; the secret comes from the environment.
type SocialService struct {
	settings *SettingsService
	secrets  map[string]string
	factory  SocialFactory

	mu    sync.Mutex
	cache map[string]port.IdentityProvider
}

// socialProviders lists the providers this build supports, with the setting keys
// that control each one.
type socialProviderDef struct {
	id         string
	name       string
	enabledKey string
	appIDKey   string
}

var socialProviders = []socialProviderDef{
	{id: "wechat", name: "WeChat", enabledKey: "social.wechat.enabled", appIDKey: "social.wechat.app_id"},
	{id: "alipay", name: "Alipay", enabledKey: "social.alipay.enabled", appIDKey: "social.alipay.app_id"},
}

func NewSocialService(settings *SettingsService, secrets map[string]string, factory SocialFactory) *SocialService {
	return &SocialService{settings: settings, secrets: secrets, factory: factory,
		cache: map[string]port.IdentityProvider{}}
}

// callbackURL is the absolute URL the provider redirects back to. It is derived
// from the configured storefront URL so operators do not have to keep two
// settings in step, and it must match what is registered with the provider.
func (s *SocialService) callbackURL(ctx context.Context, id string) string {
	base := strings.TrimRight(strings.TrimSpace(s.settings.String(ctx, "store.public_url")), "/")
	if base == "" {
		return ""
	}
	return base + "/api/v1/auth/social/" + id + "/callback"
}

// find returns the settings of a supported provider.
func findSocialProvider(id string) (socialProviderDef, bool) {
	for _, p := range socialProviders {
		if p.id == id {
			return p, true
		}
	}
	return socialProviderDef{}, false
}

// Providers lists the providers a shopper can use, so the sign-in page never
// offers a button that cannot work.
func (s *SocialService) Providers(ctx context.Context) []domain.SocialProviderInfo {
	out := make([]domain.SocialProviderInfo, 0, len(socialProviders))
	for _, p := range socialProviders {
		if s.provider(ctx, p.id) == nil {
			continue
		}
		out = append(out, domain.SocialProviderInfo{ID: p.id, Name: p.name})
	}
	return out
}

// provider returns a ready provider, or nil when it is switched off, lacks a
// credential, or has no callback URL.
func (s *SocialService) provider(ctx context.Context, id string) port.IdentityProvider {
	p, ok := findSocialProvider(id)
	if !ok || s.factory == nil || s.settings == nil {
		return nil
	}
	if !s.settings.Bool(ctx, p.enabledKey) {
		return nil
	}
	appID := strings.TrimSpace(s.settings.String(ctx, p.appIDKey))
	redirect := s.callbackURL(ctx, id)
	if appID == "" || redirect == "" {
		return nil
	}

	opts := SocialProviderOptions{ID: id, AppID: appID, AppSecret: s.secretFor(id),
		RedirectURL: redirect}
	// The cache key includes every input, so a settings change takes effect on
	// the next request instead of needing a restart.
	fingerprint := strings.Join([]string{id, appID, redirect, s.secretFor(id)}, "|")

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok := s.cache[fingerprint]; ok {
		return cached
	}
	provider, configured := s.factory(id, opts)
	if !configured {
		return nil
	}
	s.cache = map[string]port.IdentityProvider{fingerprint: provider}
	return provider
}

// secretFor returns the environment secret of a provider.
func (s *SocialService) secretFor(id string) string {
	if id == "wechat" {
		return strings.TrimSpace(s.secrets["wechat_app_secret"])
	}
	if id == "alipay" {
		return strings.TrimSpace(s.secrets["alipay_private_key"])
	}
	return ""
}

// AuthCodeURL builds the provider redirect for a CSRF state.
func (s *SocialService) AuthCodeURL(ctx context.Context, id, state string) (string, error) {
	provider := s.provider(ctx, id)
	if provider == nil {
		return "", domain.ErrNotFound
	}
	return provider.AuthCodeURL(state), nil
}

// Exchange swaps the callback code for a verified identity.
func (s *SocialService) Exchange(ctx context.Context, id, code string) (*port.Identity, error) {
	provider := s.provider(ctx, id)
	if provider == nil {
		return nil, domain.ErrNotFound
	}
	return provider.Exchange(ctx, code)
}

// Enabled reports whether any provider is usable, for the /site payload.
func (s *SocialService) Enabled(ctx context.Context) bool {
	return len(s.Providers(ctx)) > 0
}
