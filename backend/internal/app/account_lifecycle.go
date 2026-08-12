package app

import (
	"context"
	"fmt"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

func (s *Service) DeactivateAccount(ctx context.Context, principal domain.Principal,
	confirmation, currentPassword string) error {
	if principal.Role != domain.RolePlayer || principal.UserID == domain.BotUserID {
		return domain.ErrForbidden
	}
	if confirmation != "EXCLUIR" {
		return fmt.Errorf("%w: digite EXCLUIR para confirmar", domain.ErrInvalid)
	}
	user, err := s.store.UserByID(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if user.PasswordSet && !security.VerifyPassword(user.PasswordHash, currentPassword) {
		return domain.ErrInvalidCredentials
	}
	return s.store.DeactivateAccount(ctx, principal.UserID, user.PasswordHash, s.now().UTC())
}

func (s *Service) ResolveAccountReactivation(ctx context.Context, principal domain.Principal,
	resetData bool) (domain.Principal, error) {
	if principal.Role != domain.RolePlayer || principal.UserID == domain.BotUserID {
		return domain.Principal{}, domain.ErrForbidden
	}
	if !principal.ReactivationResetPending {
		return domain.Principal{}, domain.ErrConflict
	}
	if err := s.store.ResolveAccountReactivation(ctx, principal.UserID, resetData,
		engine.CompetitiveRulesetVersion, s.now().UTC()); err != nil {
		return domain.Principal{}, err
	}
	principal.ReactivationResetPending = false
	if resetData {
		now := s.now().UTC()
		principal.DataResetAt = &now
	}
	return principal, nil
}
