// Package oidc adapts any OpenID Connect provider (Keycloak, Auth0, Google,
// Okta, …) behind port.IdentityProvider. It discovers the provider metadata and
// verifies the ID token signature against the provider's JWKS.
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

type Provider struct {
	verifier *coreoidc.IDTokenVerifier
	oauth    *oauth2.Config
}

// New discovers the issuer and prepares the OAuth2/OIDC client. The issuer is
// operator-supplied, so discovery runs through a guarded client that refuses to
// connect to loopback, private or link-local addresses (cloud metadata in
// particular). The resolved address is dialled explicitly, which also closes
// DNS rebinding.
func New(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (*Provider, error) {
	if err := validateIssuer(issuer); err != nil {
		return nil, err
	}
	discovered, err := coreoidc.NewProvider(coreoidc.ClientContext(ctx, guardedClient()), issuer)
	if err != nil {
		// Configuration problems are the caller's, connectivity problems are the
		// provider's: keep them distinguishable for the operator.
		if errors.Is(err, domain.ErrInvalidArgument) || errors.Is(err, domain.ErrOIDCUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrOIDCUnavailable, err)
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

// validateIssuer requires https (http is tolerated only for a loopback host so
// a local Keycloak can be used in development).
func validateIssuer(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("oidc: invalid issuer URL")
	}
	if u.Scheme != "https" {
		if u.Scheme != "http" || !isLoopback(u.Hostname()) {
			return fmt.Errorf("%w: oidc issuer must use https", domain.ErrInvalidArgument)
		}
	}
	return nil
}

func isLoopback(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

// guardedClient resolves a host, rejects any non-public address and then dials
// the exact address it validated.
func guardedClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				if isLoopback(host) {
					return dialer.DialContext(ctx, network, addr)
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, resolved := range ips {
					if !isPublicIP(resolved.IP) {
						return nil, fmt.Errorf("%w: refusing to connect to a non-public address (%s)", domain.ErrOIDCUnavailable, resolved.IP)
					}
				}
				if len(ips) == 0 {
					return nil, fmt.Errorf("oidc: %s did not resolve", host)
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
	}
}

// isPublicIP reports whether an address is routable on the public internet.
func isPublicIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified())
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
