package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// OIDCFactory builds an identity provider from discovery. It is injected so the
// service layer stays free of adapter imports (wired in bootstrap).
type OIDCFactory func(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (port.IdentityProvider, error)

// OIDCProviderInfo is the operator-visible description of a configured provider.
type OIDCProviderInfo struct {
	ID   string
	Name string
}

// oidcProviderSetting is one entry of the oidc.providers JSON setting. The
// client secret is deliberately absent: secrets live in the environment.
type oidcProviderSetting struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Issuer      string `json:"issuer"`
	ClientID    string `json:"clientId"`
	RedirectURL string `json:"redirectUrl"`
	Scopes      string `json:"scopes"`
	Enabled     *bool  `json:"enabled"`
}

// OIDCService resolves OpenID Connect providers from runtime settings. Several
// providers may be configured at once (e.g. Google and an internal Keycloak);
// each is discovered lazily and cached until its configuration changes.
type OIDCService struct {
	settings    *SettingsService
	secret      string
	secrets     map[string]string
	newProvider OIDCFactory

	mu    sync.Mutex
	cache map[string]oidcCachedProvider
}

type oidcCachedProvider struct {
	fingerprint string
	provider    port.IdentityProvider
}

func NewOIDCService(settings *SettingsService, clientSecret string, secrets map[string]string, newProvider OIDCFactory) *OIDCService {
	return &OIDCService{
		settings: settings, secret: clientSecret, secrets: secrets, newProvider: newProvider,
		cache: map[string]oidcCachedProvider{},
	}
}

// providerConfigs returns the enabled, fully-configured providers in a stable
// order. A legacy single-provider configuration (issuer + client id) is
// synthesised when no provider list is set.
func (s *OIDCService) providerConfigs(ctx context.Context) []oidcProviderSetting {
	var out []oidcProviderSetting
	raw := strings.TrimSpace(s.settings.String(ctx, "oidc.providers"))
	if raw != "" {
		var list []oidcProviderSetting
		if err := json.Unmarshal([]byte(raw), &list); err == nil {
			for _, p := range list {
				if p.Enabled != nil && !*p.Enabled {
					continue
				}
				if strings.TrimSpace(p.Issuer) == "" || strings.TrimSpace(p.ClientID) == "" {
					continue
				}
				if p.ID == "" {
					p.ID = slugify(p.Name)
				}
				if p.ID == "" {
					continue
				}
				out = append(out, p)
			}
		}
	}
	if len(out) == 0 {
		// Backwards compatibility with the single-provider settings.
		issuer := strings.TrimSpace(s.settings.String(ctx, "oidc.issuer"))
		clientID := strings.TrimSpace(s.settings.String(ctx, "oidc.client_id"))
		if issuer != "" && clientID != "" {
			out = append(out, oidcProviderSetting{
				ID: "default", Name: "SSO", Issuer: issuer, ClientID: clientID,
				RedirectURL: s.settings.String(ctx, "oidc.redirect_url"),
				Scopes:      s.settings.String(ctx, "oidc.scopes"),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Providers lists the configured providers for the sign-in page.
func (s *OIDCService) Providers(ctx context.Context) []OIDCProviderInfo {
	if !s.Enabled(ctx) {
		return nil
	}
	configs := s.providerConfigs(ctx)
	out := make([]OIDCProviderInfo, 0, len(configs))
	for _, p := range configs {
		name := p.Name
		if name == "" {
			name = p.ID
		}
		out = append(out, OIDCProviderInfo{ID: p.ID, Name: name})
	}
	return out
}

// Enabled reports whether at least one provider is configured and switched on.
func (s *OIDCService) Enabled(ctx context.Context) bool {
	return s.newProvider != nil &&
		s.settings.Bool(ctx, "oidc.enabled") &&
		len(s.providerConfigs(ctx)) > 0
}

func (s *OIDCService) find(ctx context.Context, id string) (oidcProviderSetting, bool) {
	for _, p := range s.providerConfigs(ctx) {
		if p.ID == id {
			return p, true
		}
	}
	return oidcProviderSetting{}, false
}

// DefaultProviderID is the provider used when none is named (backwards
// compatibility with the single-provider /auth/oidc/start route).
func (s *OIDCService) DefaultProviderID(ctx context.Context) string {
	configs := s.providerConfigs(ctx)
	if len(configs) == 0 {
		return ""
	}
	return configs[0].ID
}

func (s *OIDCService) providerFor(ctx context.Context, id string) (port.IdentityProvider, error) {
	cfg, ok := s.find(ctx, id)
	if !ok || s.newProvider == nil {
		return nil, domain.ErrNotFound
	}
	scopes := strings.Fields(cfg.Scopes)
	secret := s.secrets[id]
	if secret == "" {
		secret = s.secret
	}
	fingerprint := strings.Join([]string{
		cfg.Issuer, cfg.ClientID, cfg.RedirectURL, strings.Join(scopes, " "), secret,
	}, "|")

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok := s.cache[id]; ok && cached.fingerprint == fingerprint {
		return cached.provider, nil
	}
	provider, err := s.newProvider(ctx, cfg.Issuer, cfg.ClientID, secret, cfg.RedirectURL, scopes)
	if err != nil {
		return nil, err
	}
	s.cache[id] = oidcCachedProvider{fingerprint: fingerprint, provider: provider}
	return provider, nil
}

// AuthCodeURL builds the provider redirect for the given CSRF state.
func (s *OIDCService) AuthCodeURL(ctx context.Context, id, state string) (string, error) {
	provider, err := s.providerFor(ctx, id)
	if err != nil {
		return "", err
	}
	return provider.AuthCodeURL(state), nil
}

// Exchange swaps the callback code for a verified identity.
func (s *OIDCService) Exchange(ctx context.Context, id, code string) (*port.Identity, error) {
	provider, err := s.providerFor(ctx, id)
	if err != nil {
		return nil, err
	}
	return provider.Exchange(ctx, code)
}
