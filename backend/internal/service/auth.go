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
	refreshKeyPrefix      = "auth:refresh:"
	usedKeyPrefix         = "auth:used:"
	familyKeyPrefix       = "auth:family:"
	userFamiliesKeyPrefix = "auth:user_families:"
	resetKeyPrefix        = "auth:reset:"
	verifyKeyPrefix       = "auth:verify:"
	denyListKeyPrefix     = "auth:denylist:"
	refreshTokenBytes     = 32
	resetTokenBytes       = 32
	verifyTokenBytes      = 32
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
	mailer     port.Mailer
	accessTTL  time.Duration
	refreshTTL time.Duration
	settings   *SettingsService
	// allowedRoles, when non-empty, restricts this surface's sessions to those
	// roles so a storefront token cannot be used on ops and vice versa.
	allowedRoles []domain.UserRole
	// social links external identities; it is optional so the ops binary, which
	// has no social sign-in, can build the same service.
	social port.SocialAccountRepository
}

type AuthConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	// AllowedRoles restricts this surface's sessions to a set of roles so the
	// storefront and the ops console cannot be crossed (empty allows any).
	AllowedRoles []domain.UserRole
}

func NewAuthService(
	users port.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	cache port.Cache,
	ids port.IDGenerator,
	clock port.Clock,
	mailer port.Mailer,
	cfg AuthConfig,
	settings *SettingsService,
	social port.SocialAccountRepository,
) *AuthService {
	return &AuthService{
		users: users, hasher: hasher, tokens: tokens, cache: cache, ids: ids, clock: clock, mailer: mailer,
		accessTTL: cfg.AccessTTL, refreshTTL: cfg.RefreshTTL, settings: settings,
		allowedRoles: cfg.AllowedRoles, social: social,
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
	if s.settings != nil && !s.settings.Bool(ctx, "auth.allow_registration") {
		return nil, domain.ErrRegistrationDisabled
	}
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
	_ = s.SendVerification(ctx, user)
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
	// Per-account throttling: a distributed attack cannot be stopped by the
	// per-IP limiter alone.
	if user.Locked(s.clock.Now()) {
		return nil, domain.ErrAccountLocked
	}
	if !s.roleAllowed(user.Role) {
		// Wrong surface for this account (e.g. an admin on the storefront).
		return nil, domain.ErrUnauthorized
	}
	if !s.hasher.Compare(user.PasswordHash, in.Password) {
		s.recordLoginFailure(ctx, user)
		return nil, domain.ErrUnauthorized
	}
	if s.settings != nil && s.settings.Bool(ctx, "auth.require_email_verification") && !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}
	if user.FailedAttempts > 0 {
		_ = s.users.ClearLoginFailures(ctx, user.ID)
	}
	return s.issue(ctx, user, "")
}

// recordLoginFailure counts a wrong password and locks the account once the
// configured threshold is reached. Failures are best-effort: a storage error
// must not turn a failed sign-in into a 500.
func (s *AuthService) recordLoginFailure(ctx context.Context, user *domain.User) {
	maxAttempts := 10
	lockMinutes := 15
	if s.settings != nil {
		if v := s.settings.Int(ctx, "security.max_failed_attempts"); v > 0 {
			maxAttempts = v
		}
		if v := s.settings.Int(ctx, "security.lockout_minutes"); v > 0 {
			lockMinutes = v
		}
	}
	lockUntil := s.clock.Now().Add(time.Duration(lockMinutes) * time.Minute)
	_, _ = s.users.RecordLoginFailure(ctx, user.ID, maxAttempts, lockUntil)
}

// OIDC signs in (or provisions) a customer from a verified external identity.
func (s *AuthService) OIDC(ctx context.Context, identity *port.Identity) (*AuthResult, error) {
	email := normalizeEmail(identity.Email)
	if email == "" {
		return nil, fmt.Errorf("%w: the identity provider did not return an email", domain.ErrUnauthorized)
	}
	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		if s.settings != nil && !s.settings.Bool(ctx, "auth.allow_registration") {
			return nil, domain.ErrRegistrationDisabled
		}
		now := s.clock.Now()
		name := strings.TrimSpace(identity.Name)
		if name == "" {
			name = email
		}
		user = &domain.User{
			ID: s.ids.NewID(), Email: email, Name: name, Role: domain.RoleCustomer,
			Status: domain.UserActive, EmailVerified: true, EmailVerifiedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if !s.roleAllowed(user.Role) {
		return nil, domain.ErrUnauthorized
	}
	if !user.CanLogin() {
		return nil, fmt.Errorf("%w: account is not active", domain.ErrForbidden)
	}
	return s.issue(ctx, user, "")
}

// Social signs in (or links) a shopper using an external identity such as
// WeChat or Alipay.
//
// Provider accounts are matched by their own subject first, because these
// providers do not return an email address. Only a verified email may match an
// existing account: matching an unverified address would let anybody take over
// an account by registering the right address at a provider.
func (s *AuthService) Social(ctx context.Context, identity *port.Identity) (*AuthResult, error) {
	if identity == nil || strings.TrimSpace(identity.Subject) == "" {
		return nil, domain.ErrUnauthorized
	}
	if s.social == nil {
		return nil, domain.ErrNotFound
	}
	provider := strings.TrimSpace(identity.Provider)
	if provider == "" {
		return nil, domain.ErrUnauthorized
	}
	now := s.clock.Now()

	// 1. Already linked: sign in.
	if account, err := s.social.FindBySubject(ctx, provider, identity.Subject); err == nil {
		user, err := s.users.FindByID(ctx, account.UserID)
		if err != nil {
			return nil, err
		}
		// Refresh the cached profile and record the sign-in.
		account.Name, account.AvatarURL, account.Email = identity.Name, identity.AvatarURL, identity.Email
		account.UpdatedAt = now
		_ = s.social.Upsert(ctx, account)
		if !s.roleAllowed(user.Role) || !user.CanLogin() {
			return nil, domain.ErrForbidden
		}
		return s.issue(ctx, user, "")
	}

	// 2. Not linked: find or create the shop account.
	var user *domain.User
	if email := normalizeEmail(identity.Email); email != "" && identity.EmailVerified {
		if existing, err := s.users.FindByEmail(ctx, email); err == nil {
			user = existing
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	if user == nil {
		if s.settings != nil && !s.settings.Bool(ctx, "auth.allow_registration") {
			return nil, domain.ErrRegistrationDisabled
		}
		// Only a provider-verified address may be used as the account email. An
		// unverified one could belong to somebody else, so it falls back to the
		// placeholder below; either way a new account never takes an address that
		// already belongs to another account.
		email := ""
		if identity.EmailVerified {
			email = normalizeEmail(identity.Email)
		}
		if email == "" || !s.emailAvailable(ctx, email) {
			// These providers often return no email at all. A placeholder keeps
			// the account usable and is obviously internal: the .invalid domain is
			// reserved by RFC 2606 and can never receive mail.
			email = provider + "+" + sanitizeLocalPart(identity.Subject) + "@social.invalid"
		}
		name := strings.TrimSpace(identity.Name)
		if name == "" {
			name = email
		}
		user = &domain.User{
			ID: s.ids.NewID(), Email: email, Name: name, Role: domain.RoleCustomer,
			Status: domain.UserActive, EmailVerified: identity.EmailVerified,
			CreatedAt: now, UpdatedAt: now,
		}
		if identity.EmailVerified {
			user.EmailVerifiedAt = &now
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	// 3. Link, so the next sign-in is a lookup.
	if err := s.social.Upsert(ctx, &domain.SocialAccount{
		ID: s.ids.NewID(), Provider: provider, Subject: identity.Subject, UserID: user.ID,
		Email: identity.Email, Name: identity.Name, AvatarURL: identity.AvatarURL,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return nil, err
	}
	if !s.roleAllowed(user.Role) || !user.CanLogin() {
		return nil, domain.ErrForbidden
	}
	return s.issue(ctx, user, "")
}

// emailAvailable reports whether no account uses the address yet.
func (s *AuthService) emailAvailable(ctx context.Context, email string) bool {
	if strings.TrimSpace(email) == "" {
		return false
	}
	_, err := s.users.FindByEmail(ctx, email)
	return errors.Is(err, domain.ErrNotFound)
}

// sanitizeLocalPart keeps a provider subject safe to embed in an email address.
func sanitizeLocalPart(subject string) string {
	out := make([]rune, 0, len(subject))
	for _, r := range subject {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		case r == '.', r == '-', r == '_':
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "user"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return string(out)
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
	if !s.roleAllowed(user.Role) {
		// A refresh token from the other surface cannot mint a token here.
		return nil, domain.ErrUnauthorized
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

// baseURL resolves a URL setting, trimming a trailing slash.
func (s *AuthService) baseURL(ctx context.Context, key string) string {
	if s.settings == nil {
		return ""
	}
	return strings.TrimRight(s.settings.String(ctx, key), "/")
}

// roleAllowed reports whether a user role may hold a session on this surface.
func (s *AuthService) roleAllowed(role domain.UserRole) bool {
	if len(s.allowedRoles) == 0 {
		return true
	}
	for _, r := range s.allowedRoles {
		if r == role {
			return true
		}
	}
	return false
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
	s.addUserFamily(ctx, user.ID, family)

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

// SendVerification emails a single-use link that marks the account's email as
// verified. It is best-effort: a mail failure never blocks registration.
func (s *AuthService) SendVerification(ctx context.Context, user *domain.User) error {
	if user.Email == "" {
		return nil
	}
	token, err := randomToken(verifyTokenBytes)
	if err != nil {
		return err
	}
	if err := s.cache.Set(ctx, verifyKeyPrefix+token, user.ID, 24*time.Hour); err != nil {
		return err
	}
	if s.mailer != nil {
		link := s.baseURL(ctx, "auth.email_verify_url") + "?token=" + token
		_ = s.mailer.Send(ctx, port.Email{
			To:      user.Email,
			Subject: "Verify your OpenShop email",
			HTML:    `<p>Confirm your email address to finish setting up your account.</p><p><a href="` + link + `">` + link + `</a></p>`,
		})
	}
	return nil
}

// VerifyEmail consumes a verification token.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.cache.Get(ctx, verifyKeyPrefix+token)
	if err != nil {
		if errors.Is(err, port.ErrCacheMiss) {
			return domain.ErrTokenInvalid
		}
		return err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	user.UpdatedAt = now
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, verifyKeyPrefix+token)
	return nil
}

// ResendVerification re-sends the verification email for the current user.
func (s *AuthService) ResendVerification(ctx context.Context, userID string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerified {
		return nil
	}
	return s.SendVerification(ctx, user)
}

// ResendVerificationByEmail re-sends a verification email for an address. It is
// silent for unknown or already-verified accounts so it can be exposed publicly
// without leaking account existence.
func (s *AuthService) ResendVerificationByEmail(ctx context.Context, email string) error {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if user.EmailVerified {
		return nil
	}
	return s.SendVerification(ctx, user)
}

// revokeFamily deletes every refresh token belonging to a compromised session.
func (s *AuthService) revokeFamily(ctx context.Context, family string) {
	for _, token := range s.familyTokens(ctx, family) {
		_ = s.cache.Delete(ctx, refreshKeyPrefix+token)
	}
	_ = s.cache.Delete(ctx, familyKeyPrefix+family)
}

func (s *AuthService) userFamilies(ctx context.Context, userID string) []string {
	var families []string
	if err := s.cache.GetJSON(ctx, userFamiliesKeyPrefix+userID, &families); err != nil {
		return nil
	}
	return families
}

func (s *AuthService) addUserFamily(ctx context.Context, userID, family string) {
	families := s.userFamilies(ctx, userID)
	for _, f := range families {
		if f == family {
			return
		}
	}
	families = append(families, family)
	_ = s.cache.SetJSON(ctx, userFamiliesKeyPrefix+userID, families, s.refreshTTL)
}

// RevokeAllSessions invalidates every refresh token of a user (used on password
// change/reset and account deletion).
func (s *AuthService) RevokeAllSessions(ctx context.Context, userID string) {
	s.revokeAllSessions(ctx, userID)
}

// revokeAllSessions invalidates every refresh token of a user. It is called
// after a password change or reset, so a compromised session cannot outlive the
// credential change.
func (s *AuthService) revokeAllSessions(ctx context.Context, userID string) {
	for _, family := range s.userFamilies(ctx, userID) {
		s.revokeFamily(ctx, family)
	}
	_ = s.cache.Delete(ctx, userFamiliesKeyPrefix+userID)
}

// RequestPasswordReset emails a single-use reset link. It always succeeds from
// the caller's perspective so accounts cannot be enumerated.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	token, err := randomToken(resetTokenBytes)
	if err != nil {
		return err
	}
	if err := s.cache.Set(ctx, resetKeyPrefix+token, user.ID, 30*time.Minute); err != nil {
		return err
	}
	if s.mailer != nil {
		link := s.baseURL(ctx, "auth.password_reset_url") + "?token=" + token
		_ = s.mailer.Send(ctx, port.Email{
			To:      user.Email,
			Subject: "Reset your OpenShop password",
			HTML:    `<p>Use the link below to reset your password. It expires in 30 minutes.</p><p><a href="` + link + `">` + link + `</a></p>`,
		})
	}
	return nil
}

// ResetPassword consumes a reset token and sets a new password, revoking every
// existing session.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidArgument)
	}
	userID, err := s.cache.Get(ctx, resetKeyPrefix+token)
	if err != nil {
		if errors.Is(err, port.ErrCacheMiss) {
			return domain.ErrTokenInvalid
		}
		return err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = s.clock.Now()
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, resetKeyPrefix+token)
	s.revokeAllSessions(ctx, userID)
	return nil
}

// ChangePassword verifies the current password and sets a new one, revoking all
// sessions so the user signs in again everywhere.
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidArgument)
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(user.PasswordHash, currentPassword) {
		return domain.ErrUnauthorized
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = s.clock.Now()
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	s.revokeAllSessions(ctx, userID)
	return nil
}
