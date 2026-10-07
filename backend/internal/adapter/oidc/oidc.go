// Package oidc adapts any OpenID Connect provider (Keycloak, Auth0, Google,
// Okta, …) behind port.IdentityProvider. It discovers the provider metadata and
// verifies the ID token signature against the provider's JWKS.
package oidc

import (
	"context"
	"fmt"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/holihur/openshop/internal/port"
)

type Provider struct {
	verifier *coreoidc.IDTokenVerifier
	oauth    *oauth2.Config
}

// New discovers the issuer and prepares the OAuth2/OIDC client.
func New(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (*Provider, error) {
	discovered, err := coreoidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	if len(scopes) == 0 {
		scopes = []string{coreoidc.ScopeOpenID, "profile", "email"}
	}
	return &Provider{
		verifier: discovered.Verifier(&coreoidc.Config{ClientID: clientID}),
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     discovered.Endpoint(),
			RedirectURL:  redirectURL,
			Scopes:       scopes,
		},
	}, nil
}

func (p *Provider) AuthCodeURL(state string) string {
	return p.oauth.AuthCodeURL(state)
}

func (p *Provider) Exchange(ctx context.Context, code string) (*port.Identity, error) {
	token, err := p.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oidc exchange: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, fmt.Errorf("oidc: response has no id_token")
	}
	idToken, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("oidc verify: %w", err)
	}
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	_ = idToken.Claims(&claims)
	return &port.Identity{Subject: idToken.Subject, Email: claims.Email, Name: claims.Name}, nil
}

var _ port.IdentityProvider = (*Provider)(nil)
