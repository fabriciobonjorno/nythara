package app

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/security"
)

type recoveryStore struct {
	domain.Store
	user       domain.User
	lookupErr  error
	saved      domain.PasswordResetToken
	consumed   bool
	consumedAt time.Time
	newHash    string
}

func (s *recoveryStore) UserByEmail(context.Context, string) (domain.User, error) {
	return s.user, s.lookupErr
}
func (s *recoveryStore) SavePasswordReset(_ context.Context, reset domain.PasswordResetToken) error {
	s.saved = reset
	return nil
}
func (s *recoveryStore) ConsumePasswordReset(_ context.Context, _ []byte, now time.Time, passwordHash string) error {
	s.consumed, s.consumedAt, s.newHash = true, now, passwordHash
	return nil
}
func (s *recoveryStore) SaveEmailDeliveryEvent(context.Context, domain.EmailDeliveryEvent) error {
	return nil
}

type recoverySender struct {
	to, link, locale string
	ttl              time.Duration
}

func (s *recoverySender) SendPasswordReset(_ context.Context, to, link, locale string, ttl time.Duration) error {
	s.to, s.link, s.locale, s.ttl = to, link, locale, ttl
	return nil
}

func TestRequestPasswordResetPersistsOnlyHashAndSendsOneTimeLink(t *testing.T) {
	now := time.Date(2026, 8, 12, 17, 0, 0, 0, time.UTC)
	store := &recoveryStore{user: domain.User{ID: "00000000-0000-4000-8000-000000000123", Email: "player@example.test"}}
	sender := &recoverySender{}
	service := NewWithClock(store, func() time.Time { return now })
	if err := service.ConfigurePasswordRecovery(sender, "https://nythara.fun/base?ignored=1"); err != nil {
		t.Fatal(err)
	}
	if err := service.RequestPasswordReset(context.Background(), " PLAYER@EXAMPLE.TEST ", "es"); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(sender.link)
	if err != nil {
		t.Fatal(err)
	}
	plain := parsed.Query().Get("token")
	if parsed.Path != "/reset-password" || plain == "" || sender.to != store.user.Email || sender.locale != "es" || sender.ttl != PasswordResetTTL {
		t.Fatalf("mensagem inesperada: path=%s to=%s locale=%s ttl=%s", parsed.Path, sender.to, sender.locale, sender.ttl)
	}
	if string(store.saved.TokenHash) != string(security.TokenHash(plain)) || store.saved.ExpiresAt != now.Add(PasswordResetTTL) {
		t.Fatal("persistência não corresponde ao token opaco enviado")
	}
	if plain == string(store.saved.TokenHash) {
		t.Fatal("token em texto puro foi persistido")
	}
}

func TestRequestPasswordResetDoesNotEnumerateUnknownEmail(t *testing.T) {
	store := &recoveryStore{lookupErr: domain.ErrNotFound}
	service := NewWithClock(store, time.Now)
	if err := service.RequestPasswordReset(context.Background(), "missing@example.test", "en"); err != nil {
		t.Fatalf("conta ausente alterou resposta pública: %v", err)
	}
	if store.saved.ID != "" {
		t.Fatal("pedido foi persistido para conta ausente")
	}
}

func TestResetPasswordHashesCredentialAndRejectsBadToken(t *testing.T) {
	now := time.Date(2026, 8, 12, 18, 0, 0, 0, time.UTC)
	store := &recoveryStore{}
	service := NewWithClock(store, func() time.Time { return now })
	if err := service.ResetPassword(context.Background(), "curto", "senha-segura-2026"); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("token curto aceito: %v", err)
	}
	token, _, _ := security.NewToken()
	if err := service.ResetPassword(context.Background(), token, "senha-segura-2026"); err != nil {
		t.Fatal(err)
	}
	if !store.consumed || store.consumedAt != now || !security.VerifyPassword(store.newHash, "senha-segura-2026") {
		t.Fatal("nova senha não foi derivada e consumida corretamente")
	}
}
