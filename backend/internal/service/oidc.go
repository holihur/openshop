package service

import (
	"context"
	"strings"
	"sync"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// OIDCFactory builds an identity provider from discovery. It is injected so the
// service layer stays free of adapter imports (wired in bootstrap).
type OIDCFactory func(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (port.IdentityProvider, error)

// OIDCService resolves the OIDC provider from runtime settings and rebuilds the
// discovered client when the configuration changes.
type OIDCService struct {
	settings    *SettingsService
	secret      string
	newProvider OIDCFactory

	mu          sync.Mutex
	provider    port.IdentityProvider
	fingerprint string
}

func NewOIDCService(settings *SettingsService, clientSecret string, newProvider OIDCFactory) *OIDCService {
	return &OIDCService{settings: settings, secret: clientSecret, newProvider: newProvider}
}

// Enabled reports whether OIDC is configured well enough to offer.
func (s *OIDCService) Enabled(ctx context.Context) bool {
	return s.newProvider != nil &&
		s.settings.Bool(ctx, "oidc.enabled") &&
		s.settings.String(ctx, "oidc.issuer") != "" &&
		s.settings.String(ctx, "oidc.client_id") != ""
}

func (s *OIDCService) providerFor(ctx context.Context) (port.IdentityProvider, error) {
	issuer := strings.TrimSpace(s.settings.String(ctx, "oidc.issuer"))
	clientID := strings.TrimSpace(s.settings.String(ctx, "oidc.client_id"))
	redirect := strings.TrimSpace(s.settings.String(ctx, "oidc.redirect_url"))
	scopes := strings.Fields(s.settings.String(ctx, "oidc.scopes"))
	if issuer == "" || clientID == "" || s.newProvider == nil {
		return nil, domain.ErrNotFound
	}
	fingerprint := strings.Join([]string{issuer, clientID, redirect, strings.Join(scopes, " ")}, "|")

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider != nil && s.fingerprint == fingerprint {
		return s.provider, nil
	}
	provider, err := s.newProvider(ctx, issuer, clientID, s.secret, redirect, scopes)
	if err != nil {
		return nil, err
	}
	s.provider = provider
	s.fingerprint = fingerprint
	return provider, nil
}

// AuthCodeURL builds the provider redirect for the given CSRF state.
func (s *OIDCService) AuthCodeURL(ctx context.Context, state string) (string, error) {
	provider, err := s.providerFor(ctx)
	if err != nil {
		return "", err
	}
	return provider.AuthCodeURL(state), nil
}

// Exchange swaps the callback code for a verified identity.
func (s *OIDCService) Exchange(ctx context.Context, code string) (*port.Identity, error) {
	provider, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	return provider.Exchange(ctx, code)
}
