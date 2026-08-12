package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

type accountLifecycleStore struct {
	domain.Store
	user              domain.User
	deactivated       bool
	deactivationHash  string
	deactivationAt    time.Time
	resetChoice       *bool
	resolutionRuleset string
	resolutionAt      time.Time
	sessionCreated    bool
}

func (s *accountLifecycleStore) UserByEmail(context.Context, string) (domain.User, error) {
	return s.user, nil
}
func (s *accountLifecycleStore) UserByID(context.Context, string) (domain.User, error) {
	return s.user, nil
}
func (s *accountLifecycleStore) CreateSession(context.Context, domain.NewSession) error {
	s.sessionCreated = true
	s.user.ReactivationResetPending = true
	return nil
}
func (s *accountLifecycleStore) DeactivateAccount(_ context.Context, _ string, hash string, at time.Time) error {
	s.deactivated, s.deactivationHash, s.deactivationAt = true, hash, at
	return nil
}
func (s *accountLifecycleStore) ResolveAccountReactivation(_ context.Context, _ string, reset bool,
	ruleset string, at time.Time) error {
	s.resetChoice, s.resolutionRuleset, s.resolutionAt = &reset, ruleset, at
	return nil
}

func TestDeactivateAccountRequiresPlayerConfirmationAndCurrentPassword(t *testing.T) {
	now := time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	hash, _ := security.HashPassword("senha-atual-segura")
	store := &accountLifecycleStore{user: domain.User{ID: "player", Role: domain.RolePlayer,
		PasswordHash: hash, PasswordSet: true}}
	service := NewWithClock(store, func() time.Time { return now })
	principal := domain.Principal{UserID: "player", Role: domain.RolePlayer, PasswordSet: true}

	if err := service.DeactivateAccount(context.Background(), principal, "apagar", "senha-atual-segura"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("confirmação fraca aceita: %v", err)
	}
	if err := service.DeactivateAccount(context.Background(), principal, "EXCLUIR", "senha-incorreta"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("senha incorreta aceita: %v", err)
	}
	if err := service.DeactivateAccount(context.Background(), domain.Principal{UserID: "admin", Role: domain.RoleAdmin},
		"EXCLUIR", ""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("admin conseguiu se desativar: %v", err)
	}
	if err := service.DeactivateAccount(context.Background(), principal, "EXCLUIR", "senha-atual-segura"); err != nil {
		t.Fatal(err)
	}
	if !store.deactivated || store.deactivationHash != hash || store.deactivationAt != now {
		t.Fatalf("desativação não chegou íntegra ao store: %+v", store)
	}
}

func TestLoginReloadsAutomaticallyReactivatedAccount(t *testing.T) {
	hash, _ := security.HashPassword("senha-atual-segura")
	store := &accountLifecycleStore{user: domain.User{ID: "player", Email: "player@example.test",
		Role: domain.RolePlayer, PasswordHash: hash, PasswordSet: true}}
	service := New(store)
	user, tokens, err := service.Login(context.Background(), "player@example.test", "senha-atual-segura")
	if err != nil {
		t.Fatal(err)
	}
	if !store.sessionCreated || !user.ReactivationResetPending || tokens.AccessToken == "" {
		t.Fatalf("login não devolveu o estado reativado: user=%+v session=%v", user, store.sessionCreated)
	}
}

func TestResolveReactivationIsRestrictedToPendingPlayer(t *testing.T) {
	now := time.Date(2026, 8, 12, 17, 0, 0, 0, time.UTC)
	store := &accountLifecycleStore{}
	service := NewWithClock(store, func() time.Time { return now })
	principal := domain.Principal{UserID: "player", Role: domain.RolePlayer}
	if _, err := service.ResolveAccountReactivation(context.Background(), principal, true); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("conta sem pendência conseguiu resetar: %v", err)
	}
	principal.ReactivationResetPending = true
	updated, err := service.ResolveAccountReactivation(context.Background(), principal, true)
	if err != nil {
		t.Fatal(err)
	}
	if store.resetChoice == nil || !*store.resetChoice || store.resolutionRuleset != engine.CompetitiveRulesetVersion ||
		store.resolutionAt != now || updated.ReactivationResetPending || updated.DataResetAt == nil {
		t.Fatalf("reset pendente não foi resolvido: store=%+v principal=%+v", store, updated)
	}
}

type oldReplayStore struct {
	domain.Store
	replay domain.MatchReplayData
}

func (s oldReplayStore) MatchReplay(context.Context, string) (domain.MatchReplayData, error) {
	return s.replay, nil
}

func TestResetCutoffHidesOldReplayOnlyFromResetPlayer(t *testing.T) {
	created := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	resetAt := created.Add(time.Hour)
	store := oldReplayStore{replay: domain.MatchReplayData{MatchID: "match", Status: "finished",
		CreatedAt: created, Players: [2]domain.ReplayPlayer{{UserID: "reset-player"}, {UserID: "opponent"}}}}
	service := New(store)
	if _, err := service.MatchReplay(context.Background(), domain.Principal{UserID: "reset-player",
		Role: domain.RolePlayer, DataResetAt: &resetAt}, "match"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("jogador resetado ainda abriu replay antigo: %v", err)
	}
	if _, err := service.MatchReplay(context.Background(), domain.Principal{UserID: "opponent",
		Role: domain.RolePlayer}, "match"); err != nil {
		t.Fatalf("reset alheio apagou o replay do adversário: %v", err)
	}
}
