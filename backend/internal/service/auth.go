package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const (
	refreshKeyPrefix  = "auth:refresh:"
	usedKeyPrefix     = "auth:used:"
	familyKeyPrefix   = "auth:family:"
	denyListKeyPrefix = "auth:denylist:"
	refreshTokenBytes = 32
)

// refreshRecord is the server-side state for an issued refresh token. The
// family ties every rotation of a login session together, which is what enables
// theft detection: if a token that was already rotated is presented again, the
// whole family is revoked.
type refreshRecord struct {
	UserID string `json:"userId"`
	Family string `json:"family"`
}

// AuthService implements registration, login, token refresh and logout.
type AuthService struct {
	users      port.UserRepository
	hasher     port.PasswordHasher
	tokens     port.TokenIssuer
	cache      port.Cache
	ids        port.IDGenerator
	clock      port.Clock
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type AuthConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func NewAuthService(
	users port.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	cache port.Cache,
	ids port.IDGenerator,
	clock port.Clock,
	cfg AuthConfig,
) *AuthService {
	return &AuthService{
		users: users, hasher: hasher, tokens: tokens, cache: cache, ids: ids, clock: clock,
		accessTTL: cfg.AccessTTL, refreshTTL: cfg.RefreshTTL,
	}
}

type RegisterInput struct {
	Email    string
	Phone    string
	Password string
	Name     string
}

type LoginInput struct {
	// Identifier is an email or phone number.
	Identifier string
	Password   string
}

type AuthResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	email := normalizeEmail(in.Email)
	if email == "" && in.Phone == "" {
		return nil, fmt.Errorf("%w: email or phone is required", domain.ErrInvalidArgument)
	}
	if len(in.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidArgument)
	}

	if email != "" {
		if _, err := s.users.FindByEmail(ctx, email); err == nil {
			return nil, fmt.Errorf("%w: email already registered", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	if in.Phone != "" {
		if _, err := s.users.FindByPhone(ctx, in.Phone); err == nil {
			return nil, fmt.Errorf("%w: phone already registered", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	user := &domain.User{
		ID:           s.ids.NewID(),
		Email:        email,
		Phone:        in.Phone,
		PasswordHash: hash,
		Name:         in.Name,
		Role:         domain.RoleCustomer,
		Status:       domain.UserActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return s.issue(ctx, user, "")
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	id := in.Identifier
	var user *domain.User
	var err error
	if strings.Contains(id, "@") {
		user, err = s.users.FindByEmail(ctx, normalizeEmail(id))
	} else {
		user, err = s.users.FindByPhone(ctx, id)
	}
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !user.CanLogin() {
		return nil, fmt.Errorf("%w: account is not active", domain.ErrForbidden)
	}
	if !s.hasher.Compare(user.PasswordHash, in.Password) {
		return nil, domain.ErrUnauthorized
	}
	return s.issue(ctx, user, "")
}

// Refresh rotates a refresh token within its family. If an already-rotated
// token is presented again (a strong signal of theft), the entire family is
// revoked and the caller must sign in again.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, domain.ErrUnauthorized
	}
	key := refreshKeyPrefix + refreshToken
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, port.ErrCacheMiss) {
			// Was this token already used? If so, assume theft and revoke family.
			if family, used := s.wasUsed(ctx, refreshToken); used {
				s.revokeFamily(ctx, family)
			}
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	var rec refreshRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Rotate: invalidate the presented token, remember it as used.
	_ = s.cache.Delete(ctx, key)
	_ = s.cache.Set(ctx, usedKeyPrefix+refreshToken, rec.Family, s.refreshTTL)
	s.removeFromFamily(ctx, rec.Family, refreshToken)

	user, err := s.users.FindByID(ctx, rec.UserID)
	if err != nil {
		return nil, err
	}
	if !user.CanLogin() {
		return nil, domain.ErrForbidden
	}
	return s.issue(ctx, user, rec.Family)
}

// Logout revokes the current access token (deny-list) and its refresh token.
func (s *AuthService) Logout(ctx context.Context, claims *port.TokenClaims, refreshToken string) error {
	if claims != nil && claims.ID != "" {
		ttl := ttlFrom(s.clock.Now(), claims.Expires)
		if ttl > 0 {
			if err := s.cache.Set(ctx, denyListKeyPrefix+claims.ID, "1", ttl); err != nil {
				return err
			}
		}
	}
	if refreshToken != "" {
		if raw, err := s.cache.Get(ctx, refreshKeyPrefix+refreshToken); err == nil {
			var rec refreshRecord
			if json.Unmarshal([]byte(raw), &rec) == nil {
				s.removeFromFamily(ctx, rec.Family, refreshToken)
			}
		}
		if err := s.cache.Delete(ctx, refreshKeyPrefix+refreshToken); err != nil {
			return err
		}
	}
	return nil
}

// IsRevoked reports whether an access-token jti has been deny-listed.
func (s *AuthService) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	_, err := s.cache.Get(ctx, denyListKeyPrefix+jti)
	if errors.Is(err, port.ErrCacheMiss) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

// issue mints an access/refresh pair. An empty family starts a new session.
func (s *AuthService) issue(ctx context.Context, user *domain.User, family string) (*AuthResult, error) {
	now := s.clock.Now()
	access, err := s.tokens.Issue(port.TokenClaims{
		Subject:  user.ID,
		Role:     user.Role,
		IssuedAt: now,
		Expires:  now.Add(s.accessTTL),
		ID:       s.ids.NewID(),
	})
	if err != nil {
		return nil, err
	}
	if family == "" {
		family, err = randomToken(16)
		if err != nil {
			return nil, err
		}
	}
	refresh, err := randomToken(refreshTokenBytes)
	if err != nil {
		return nil, err
	}
	rec, err := json.Marshal(refreshRecord{UserID: user.ID, Family: family})
	if err != nil {
		return nil, err
	}
	if err := s.cache.Set(ctx, refreshKeyPrefix+refresh, string(rec), s.refreshTTL); err != nil {
		return nil, err
	}
	s.addToFamily(ctx, family, refresh)

	return &AuthResult{
		User:         user,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *AuthService) wasUsed(ctx context.Context, token string) (string, bool) {
	family, err := s.cache.Get(ctx, usedKeyPrefix+token)
	if err != nil || family == "" {
		return "", false
	}
	return family, true
}

func (s *AuthService) familyTokens(ctx context.Context, family string) []string {
	var tokens []string
	if err := s.cache.GetJSON(ctx, familyKeyPrefix+family, &tokens); err != nil {
		return nil
	}
	return tokens
}

func (s *AuthService) saveFamily(ctx context.Context, family string, tokens []string) {
	if len(tokens) == 0 {
		_ = s.cache.Delete(ctx, familyKeyPrefix+family)
		return
	}
	_ = s.cache.SetJSON(ctx, familyKeyPrefix+family, tokens, s.refreshTTL)
}

func (s *AuthService) addToFamily(ctx context.Context, family, token string) {
	tokens := append(s.familyTokens(ctx, family), token)
	s.saveFamily(ctx, family, tokens)
}

func (s *AuthService) removeFromFamily(ctx context.Context, family, token string) {
	tokens := s.familyTokens(ctx, family)
	out := tokens[:0]
	for _, t := range tokens {
		if t != token {
			out = append(out, t)
		}
	}
	s.saveFamily(ctx, family, out)
}

// revokeFamily deletes every refresh token belonging to a compromised session.
func (s *AuthService) revokeFamily(ctx context.Context, family string) {
	for _, token := range s.familyTokens(ctx, family) {
		_ = s.cache.Delete(ctx, refreshKeyPrefix+token)
	}
	_ = s.cache.Delete(ctx, familyKeyPrefix+family)
}
