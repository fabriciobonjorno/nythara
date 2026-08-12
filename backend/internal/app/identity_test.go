package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/security"
)

type identityStore struct {
	domain.Store
	user          domain.User
	oauthUser     domain.User
	oauthLookup   error
	avatar        string
	passwordHash  string
	passwordAt    time.Time
	ticketHash    []byte
	ticketExpires time.Time
	ticketUserID  string
	sessionUserID string
}

func (s *identityStore) UserByID(context.Context, string) (domain.User, error) { return s.user, nil }
func (s *identityStore) UserByOAuth(context.Context, string, string) (domain.User, error) {
	return s.oauthUser, s.oauthLookup
}
func (s *identityStore) UpdateProfileAvatar(_ context.Context, _ string, avatarID string) (domain.User, error) {
	s.avatar = avatarID
	s.user.AvatarID = avatarID
	return s.user, nil
}
func (s *identityStore) ChangePassword(_ context.Context, _ string, hash string, at time.Time) error {
	s.passwordHash, s.passwordAt = hash, at
	return nil
}
func (s *identityStore) SaveOAuthTicket(_ context.Context, hash []byte, userID string, expires time.Time) error {
	s.ticketHash, s.ticketUserID, s.ticketExpires = hash, userID, expires
	return nil
}
func (s *identityStore) ConsumeOAuthTicket(_ context.Context, hash []byte, _ time.Time) (string, error) {
	if string(hash) != string(s.ticketHash) {
		return "", domain.ErrInvalidCredentials
	}
	return s.ticketUserID, nil
}
func (s *identityStore) CreateSession(_ context.Context, session domain.NewSession) error {
	s.sessionUserID = session.UserID
	return nil
}

type fakeGoogleOAuth struct{ identity domain.OAuthIdentity }

func (f fakeGoogleOAuth) AuthorizationURL(state, verifier string) string {
	return state + ":" + verifier
}
func (f fakeGoogleOAuth) Identity(context.Context, string, string) (domain.OAuthIdentity, error) {
	return f.identity, nil
}

func TestUpdateProfileAvatarUsesPublishedAvatarOnly(t *testing.T) {
	store := &identityStore{user: domain.User{ID: "player", DisplayName: "Duelista", Role: domain.RolePlayer}}
	service := New(store)
	principal := domain.Principal{UserID: "player", DisplayName: "Duelista", Role: domain.RolePlayer}
	if _, err := service.UpdateProfileAvatar(context.Background(), principal, "fora-do-catalogo"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("emblema inexistente aceito: %v", err)
	}
	updated, err := service.UpdateProfileAvatar(context.Background(), principal, "CH-CI-01")
	if err != nil || store.avatar != "CH-CI-01" || updated.AvatarID != "CH-CI-01" || updated.DisplayName != "Duelista" {
		t.Fatalf("perfil não preservou identidade: updated=%+v err=%v", updated, err)
	}
}

func TestChangePasswordRequiresCurrentAndRevokesThroughStore(t *testing.T) {
	now := time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC)
	oldHash, _ := security.HashPassword("senha-atual-segura")
	store := &identityStore{user: domain.User{ID: "player", PasswordHash: oldHash, PasswordSet: true}}
	service := NewWithClock(store, func() time.Time { return now })
	principal := domain.Principal{UserID: "player", PasswordSet: true}
	if err := service.ChangePassword(context.Background(), principal, "senha-errada-segura", "nova-senha-segura"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("senha atual incorreta aceita: %v", err)
	}
	if store.passwordHash != "" {
		t.Fatal("store foi chamado antes de validar a senha atual")
	}
	if err := service.ChangePassword(context.Background(), principal, "senha-atual-segura", "nova-senha-segura"); err != nil {
		t.Fatal(err)
	}
	if store.passwordAt != now || !security.VerifyPassword(store.passwordHash, "nova-senha-segura") {
		t.Fatal("nova credencial não foi derivada e entregue atomicamente ao store")
	}
}

func TestGoogleOAuthReturnsOneUseLocalExchangeTicket(t *testing.T) {
	now := time.Date(2026, 8, 12, 21, 0, 0, 0, time.UTC)
	user := domain.User{ID: "oauth-player", Email: "player@gmail.com", DisplayName: "player", Role: domain.RolePlayer}
	store := &identityStore{user: user, oauthUser: user}
	service := NewWithClock(store, func() time.Time { return now })
	service.googleOAuth = fakeGoogleOAuth{identity: domain.OAuthIdentity{Provider: "google", Subject: "google-sub",
		Email: "player@gmail.com", EmailVerified: true}}
	ticket, err := service.CompleteGoogleOAuth(context.Background(), "authorization-code", "pkce-verifier")
	if err != nil {
		t.Fatal(err)
	}
	if ticket == "" || string(store.ticketHash) == ticket || store.ticketExpires != now.Add(oauthTicketTTL) {
		t.Fatal("ticket OAuth não foi persistido somente como hash com expiração curta")
	}
	loggedIn, tokens, err := service.ExchangeOAuthTicket(context.Background(), ticket)
	if err != nil || loggedIn.ID != user.ID || store.sessionUserID != user.ID || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("troca do ticket falhou: user=%+v session=%s err=%v", loggedIn, store.sessionUserID, err)
	}
}
