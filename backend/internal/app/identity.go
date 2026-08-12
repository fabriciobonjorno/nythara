package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

const (
	oauthTicketTTL    = 2 * time.Minute
	oauthCookieTTL    = 5 * time.Minute
	oauthOnlyPassword = "oauth-only"
)

type googleOAuthProvider interface {
	AuthorizationURL(state, verifier string) string
	Identity(context.Context, string, string) (domain.OAuthIdentity, error)
}

type googleOAuth struct {
	clientID     string
	clientSecret string
	redirectURI  string
	client       *http.Client
}

func (s *Service) ConfigureGoogleOAuth(clientID, clientSecret, publicAppURL string) error {
	clientID, clientSecret = strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return errors.New("GOOGLE_OAUTH_CLIENT_ID e GOOGLE_OAUTH_CLIENT_SECRET são obrigatórios juntos")
	}
	parsed, err := url.Parse(strings.TrimSpace(publicAppURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return errors.New("PUBLIC_APP_URL deve ser uma URL HTTP(S) absoluta")
	}
	parsed.Path, parsed.RawQuery, parsed.Fragment = "/v1/auth/google/callback", "", ""
	s.googleOAuth = &googleOAuth{clientID: clientID, clientSecret: clientSecret,
		redirectURI: parsed.String(), client: &http.Client{Timeout: 10 * time.Second}}
	return nil
}

func (s *Service) GoogleOAuthEnabled() bool { return s.googleOAuth != nil }

func (s *Service) BeginGoogleOAuth() (authorizationURL, state, verifier string, err error) {
	if s.googleOAuth == nil {
		return "", "", "", domain.ErrNotFound
	}
	state, _, err = security.NewToken()
	if err != nil {
		return "", "", "", err
	}
	verifier, _, err = security.NewToken()
	if err != nil {
		return "", "", "", err
	}
	return s.googleOAuth.AuthorizationURL(state, verifier), state, verifier, nil
}

func (s *Service) CompleteGoogleOAuth(ctx context.Context, code, verifier string) (string, error) {
	if s.googleOAuth == nil || code == "" || verifier == "" {
		return "", domain.ErrInvalidCredentials
	}
	identity, err := s.googleOAuth.Identity(ctx, code, verifier)
	if err != nil || identity.Provider != "google" || identity.Subject == "" || !identity.EmailVerified {
		return "", domain.ErrInvalidCredentials
	}
	identity.Email, err = normalizeEmail(identity.Email)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}
	user, err := s.oauthUser(ctx, identity)
	if err != nil {
		return "", err
	}
	ticket, ticketHash, err := security.NewToken()
	if err != nil {
		return "", err
	}
	if err := s.store.SaveOAuthTicket(ctx, ticketHash, user.ID, s.now().UTC().Add(oauthTicketTTL)); err != nil {
		return "", err
	}
	return ticket, nil
}

func (s *Service) ExchangeOAuthTicket(ctx context.Context, ticket string) (domain.User, domain.SessionTokens, error) {
	if len(ticket) < 32 || len(ticket) > 256 {
		return domain.User{}, domain.SessionTokens{}, domain.ErrInvalidCredentials
	}
	userID, err := s.store.ConsumeOAuthTicket(ctx, security.TokenHash(ticket), s.now().UTC())
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, domain.ErrInvalidCredentials
	}
	user, err := s.store.UserByID(ctx, userID)
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, err
	}
	tokens, err := s.newSession(ctx, user.ID)
	return user, tokens, err
}

func (s *Service) oauthUser(ctx context.Context, identity domain.OAuthIdentity) (domain.User, error) {
	if user, err := s.store.UserByOAuth(ctx, identity.Provider, identity.Subject); err == nil {
		return user, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}

	user, err := s.store.UserByEmail(ctx, identity.Email)
	if err == nil {
		lowerEmail := strings.ToLower(identity.Email)
		if !strings.HasSuffix(lowerEmail, "@gmail.com") && strings.TrimSpace(identity.HostedDomain) == "" {
			return domain.User{}, fmt.Errorf("%w: entre com a senha para vincular esta conta ao Google", domain.ErrConflict)
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	} else {
		user, err = s.createOAuthUser(ctx, identity)
		if err != nil {
			if !errors.Is(err, domain.ErrConflict) {
				return domain.User{}, err
			}
			user, err = s.store.UserByEmail(ctx, identity.Email)
			if err != nil {
				return domain.User{}, err
			}
		}
	}

	if err := s.store.LinkOAuth(ctx, identity.Provider, identity.Subject, user.ID); err != nil && !errors.Is(err, domain.ErrConflict) {
		return domain.User{}, err
	}
	linked, err := s.store.UserByOAuth(ctx, identity.Provider, identity.Subject)
	if err != nil {
		return domain.User{}, err
	}
	if linked.ID != user.ID {
		return domain.User{}, domain.ErrConflict
	}
	return linked, nil
}

func (s *Service) createOAuthUser(ctx context.Context, identity domain.OAuthIdentity) (domain.User, error) {
	base := oauthUsername(identity.Email)
	digest := sha256.Sum256([]byte(identity.Subject))
	suffix := base64.RawURLEncoding.EncodeToString(digest[:])[:6]
	for attempt := 0; attempt < 8; attempt++ {
		name := base
		if attempt > 0 {
			end := suffix
			if attempt > 1 {
				end = fmt.Sprintf("%s%d", suffix[:4], attempt)
			}
			name = strings.TrimRight(base[:min(len(base), 32-len(end)-1)], "-_") + "-" + end
		}
		id, err := security.NewID()
		if err != nil {
			return domain.User{}, err
		}
		user, err := s.store.CreateUser(ctx, domain.User{ID: id, Email: identity.Email,
			DisplayName: name, Role: domain.RolePlayer, PasswordHash: oauthOnlyPassword, PasswordSet: false},
			engine.CompetitiveRulesetVersion)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, domain.ErrConflict) {
			return domain.User{}, err
		}
		if existing, lookupErr := s.store.UserByEmail(ctx, identity.Email); lookupErr == nil {
			return existing, nil
		}
	}
	return domain.User{}, domain.ErrConflict
}

var oauthUsernameCharacters = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func oauthUsername(email string) string {
	local := strings.SplitN(email, "@", 2)[0]
	local = strings.Trim(oauthUsernameCharacters.ReplaceAllString(local, "-"), "-_")
	if len(local) > 24 {
		local = local[:24]
	}
	if len(local) < 2 {
		return "viajante"
	}
	return local
}

func (g *googleOAuth) AuthorizationURL(state, verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id":             {g.clientID},
		"redirect_uri":          {g.redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + query.Encode()
}

func (g *googleOAuth) Identity(ctx context.Context, code, verifier string) (domain.OAuthIdentity, error) {
	form := url.Values{"code": {code}, "client_id": {g.clientID}, "client_secret": {g.clientSecret},
		"redirect_uri": {g.redirectURI}, "grant_type": {"authorization_code"}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := g.client.Do(req)
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	defer response.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&token) != nil || token.AccessToken == "" {
		return domain.OAuthIdentity{}, domain.ErrInvalidCredentials
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err = g.client.Do(req)
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	defer response.Body.Close()
	var profile struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HostedDomain  string `json:"hd"`
		Name          string `json:"name"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&profile) != nil {
		return domain.OAuthIdentity{}, domain.ErrInvalidCredentials
	}
	return domain.OAuthIdentity{Provider: "google", Subject: profile.Subject, Email: profile.Email,
		EmailVerified: profile.EmailVerified, HostedDomain: profile.HostedDomain, DisplayName: profile.Name}, nil
}

func (s *Service) UpdateProfileAvatar(ctx context.Context, principal domain.Principal, avatarID string) (domain.Principal, error) {
	avatarID = strings.TrimSpace(avatarID)
	if engine.CompetitiveRuleset().Champions[avatarID] == nil {
		return domain.Principal{}, fmt.Errorf("%w: imagem de perfil inválida", domain.ErrInvalid)
	}
	user, err := s.store.UpdateProfileAvatar(ctx, principal.UserID, avatarID)
	if err != nil {
		return domain.Principal{}, err
	}
	return domain.Principal{UserID: user.ID, Role: user.Role, DisplayName: user.DisplayName,
		AvatarID: user.AvatarID, PasswordSet: user.PasswordSet}, nil
}

func (s *Service) ChangePassword(ctx context.Context, principal domain.Principal, currentPassword, newPassword string) error {
	user, err := s.store.UserByID(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if user.PasswordSet && !security.VerifyPassword(user.PasswordHash, currentPassword) {
		return domain.ErrInvalidCredentials
	}
	if user.PasswordSet && security.VerifyPassword(user.PasswordHash, newPassword) {
		return fmt.Errorf("%w: a nova senha deve ser diferente da atual", domain.ErrInvalid)
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	return s.store.ChangePassword(ctx, principal.UserID, hash, s.now().UTC())
}

func OAuthCookieTTL() time.Duration { return oauthCookieTTL }
