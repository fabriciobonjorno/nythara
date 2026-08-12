package app

import (
	"context"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// Arena — histórico pessoal e crônica pós-partida. A crônica expõe o log
// completo de eventos, então a autorização é estrita: só participantes (ou
// admin) e só depois do fim — informação oculta deixa de ser oculta quando a
// partida termina, nunca antes.

// MatchHistory devolve as últimas partidas encerradas do jogador.
func (s *Service) MatchHistory(ctx context.Context, principal domain.Principal,
	limit int) ([]domain.MatchSummary, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.store.MatchHistory(ctx, principal.UserID, limit)
}

// MatchReplay devolve a crônica de uma partida encerrada do solicitante.
func (s *Service) MatchReplay(ctx context.Context, principal domain.Principal,
	matchID string) (domain.MatchReplayData, error) {
	replay, err := s.store.MatchReplay(ctx, matchID)
	if err != nil {
		return domain.MatchReplayData{}, err
	}
	participant := replay.Players[0].UserID == principal.UserID ||
		replay.Players[1].UserID == principal.UserID
	if !participant && !principal.Role.IsAdmin() {
		return domain.MatchReplayData{}, domain.ErrForbidden
	}
	if replay.Status != "finished" {
		// O log completo revela mão e baralho: nada de espiar partida viva.
		return domain.MatchReplayData{}, domain.ErrForbidden
	}
	if rs, err := engine.RulesetByVersion(replay.RulesetVersion); err == nil && rs.IsConfront() {
		replay.StartingVitality = rs.ConfrontRules.StartingVitality
	}
	return replay, nil
}
