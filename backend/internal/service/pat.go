package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// PATPrefix marks a bearer token as a personal access token, so the auth
// middleware can tell one apart from a JWT without a database round trip.
const PATPrefix = "osp_"

// CreatePATInput is a new personal access token.
type CreatePATInput struct {
	UserID    string
	Name      string
	Realm     string
	Scopes    []domain.Scope
	CIDRs     []string
	ExpiresAt *time.Time
}

// PATService issues and validates personal access tokens. Only the SHA-256 hash
// of a token is stored, so a database leak cannot be replayed. Tokens carry an
// explicit scope list and an optional CIDR allow-list.
type PATService struct {
	tokens port.PersonalAccessTokenRepository
	ids    port.IDGenerator
	clock  port.Clock
}

func NewPATService(tokens port.PersonalAccessTokenRepository, ids port.IDGenerator, clock port.Clock) *PATService {
	return &PATService{tokens: tokens, ids: ids, clock: clock}
}

// Create mints a token and returns it alongside the record. The raw token is
// only ever available here.
func (s *PATService) Create(ctx context.Context, in CreatePATInput) (*domain.PersonalAccessToken, string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || in.UserID == "" {
		return nil, "", domain.ErrInvalidArgument
	}
	realm := in.Realm
	if realm != "ops" {
		realm = "front"
	}
	scopes := dedupeScopes(in.Scopes)
	if len(scopes) == 0 {
		return nil, "", domain.ErrInvalidArgument
	}
	cidrs := normalizeCIDRs(in.CIDRs)

	prefix, err := randomToken(8)
	if err != nil {
		return nil, "", err
	}
	prefix = strings.ToLower(prefix)
	secret, err := randomToken(32)
	if err != nil {
		return nil, "", err
	}
	raw := PATPrefix + prefix + "_" + secret

	record := &domain.PersonalAccessToken{
		ID: s.ids.NewID(), UserID: in.UserID, Name: name, Prefix: prefix, Realm: realm,
		Scopes: scopes, CIDRs: cidrs, ExpiresAt: in.ExpiresAt, CreatedAt: s.clock.Now(),
	}
	if err := s.tokens.Create(ctx, record, HashPAT(raw)); err != nil {
		return nil, "", err
	}
	return record, raw, nil
}

// Authenticate resolves a raw token, enforcing revocation, expiry, realm and
// the CIDR allow-list. It returns the token and its owner's role.
func (s *PATService) Authenticate(ctx context.Context, raw, clientIP, realm string) (*domain.PersonalAccessToken, domain.UserRole, error) {
	if !strings.HasPrefix(raw, PATPrefix) {
		return nil, "", domain.ErrTokenInvalid
	}
	token, user, err := s.tokens.FindByHash(ctx, HashPAT(raw))
	if err != nil {
		return nil, "", domain.ErrTokenInvalid
	}
	if !token.Active(s.clock.Now()) {
		return nil, "", domain.ErrTokenExpired
	}
	if token.Realm != realm {
		return nil, "", domain.ErrTokenInvalid
	}
	if user.Status != domain.UserActive {
		return nil, "", domain.ErrForbidden
	}
	if !token.AllowsIP(clientIP) {
		return nil, "", domain.ErrForbidden
	}
	// Record usage without letting a failure break the request.
	go func() {
		_ = s.tokens.Touch(context.WithoutCancel(ctx), token.ID, time.Now().UTC())
	}()
	return token, user.Role, nil
}

// List returns a user's tokens (never their secrets).
func (s *PATService) List(ctx context.Context, userID string) ([]domain.PersonalAccessToken, error) {
	return s.tokens.ListByUser(ctx, userID)
}

// Revoke permanently disables a token.
func (s *PATService) Revoke(ctx context.Context, id, userID string) error {
	return s.tokens.Revoke(ctx, id, userID, s.clock.Now())
}

// HashPAT hashes a raw token for storage and lookup.
func HashPAT(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func dedupeScopes(scopes []domain.Scope) []domain.Scope {
	seen := map[domain.Scope]bool{}
	out := make([]domain.Scope, 0, len(scopes))
	for _, sc := range scopes {
		if !domain.ValidScope(sc) || seen[sc] {
			continue
		}
		seen[sc] = true
		out = append(out, sc)
	}
	return out
}

func normalizeCIDRs(cidrs []string) []string {
	out := make([]string, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}
