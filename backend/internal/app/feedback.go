package app

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// Recado do Alpha. O convite é opcional em todos os níveis: o jogador pode
// ignorá-lo, e nenhuma outra parte do produto lê esta tabela para decidir algo.
// Por isso a validação aqui é mínima e o erro é sempre legível — quem tentou
// ajudar não merece uma mensagem de sistema.

// SubmitFeedback grava um recado do jogador sobre a partida que acabou.
func (s *Service) SubmitFeedback(ctx context.Context, principal domain.Principal,
	matchID, message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fmt.Errorf("%w: o recado está vazio", domain.ErrInvalid)
	}
	if utf8.RuneCountInString(trimmed) > domain.FeedbackMaxLength {
		return fmt.Errorf("%w: o recado passa de %d caracteres",
			domain.ErrInvalid, domain.FeedbackMaxLength)
	}
	// Só aceita associar o recado a uma partida que o jogador realmente jogou:
	// o identificador vem do cliente e não pode virar ponteiro para a partida
	// de outra pessoa.
	if matchID != "" {
		replay, err := s.store.MatchReplay(ctx, matchID)
		if err != nil {
			return err
		}
		if replay.Players[0].UserID != principal.UserID && replay.Players[1].UserID != principal.UserID {
			return domain.ErrForbidden
		}
	}
	return s.store.SaveFeedback(ctx, domain.Feedback{
		UserID:         principal.UserID,
		MatchID:        matchID,
		RulesetVersion: engine.CompetitiveRulesetVersion,
		Message:        trimmed,
		CreatedAt:      s.now(),
	})
}
