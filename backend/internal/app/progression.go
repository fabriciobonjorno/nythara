package app

import (
	"context"
	"fmt"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// P1 — leitura e gravação de progressão. A gravação nasce exclusivamente do
// battle server (eventos authoritative); a leitura alimenta a Home.

// ProgressSummary devolve rituais do dia (materializando o sorteio na
// primeira consulta), carteira, maestria e rating da temporada.
func (s *Service) ProgressSummary(ctx context.Context, principal domain.Principal) (domain.ProgressSummary, error) {
	day := utcDay(s.now())
	defs := ritualsForDay(day, principal.UserID)
	seed := make([]domain.RitualState, 0, len(defs))
	for _, def := range defs {
		seed = append(seed, domain.RitualState{RitualID: def.ID, Target: def.Target, Reward: def.Reward})
	}
	if err := s.store.SaveRitualStates(ctx, principal.UserID, day, seed); err != nil {
		return domain.ProgressSummary{}, err
	}
	stored, err := s.store.RitualsFor(ctx, principal.UserID, day)
	if err != nil {
		return domain.ProgressSummary{}, err
	}
	byID := map[string]domain.RitualState{}
	for _, state := range stored {
		byID[state.RitualID] = state
	}
	summary := domain.ProgressSummary{Day: day}
	for _, def := range defs {
		state := byID[def.ID]
		state.RitualID = def.ID
		state.Title = def.Title
		state.Description = def.Description
		if state.Target == 0 {
			state.Target = def.Target
		}
		if state.Reward == 0 {
			state.Reward = def.Reward
		}
		summary.Rituals = append(summary.Rituals, state)
	}
	if summary.Fragments, err = s.store.Fragments(ctx, principal.UserID); err != nil {
		return domain.ProgressSummary{}, err
	}
	if summary.Mastery, err = s.store.MasteryFor(ctx, principal.UserID); err != nil {
		return domain.ProgressSummary{}, err
	}
	for i := range summary.Mastery {
		summary.Mastery[i].Level = MasteryLevel(summary.Mastery[i].XP)
		summary.Mastery[i].Title = domain.MasteryTitle(summary.Mastery[i].Level)
	}
	if season, err := s.store.ActiveSeason(ctx); err == nil {
		standing, err := s.store.RankedStandingFor(ctx, principal.UserID, season.ID)
		if err != nil {
			return domain.ProgressSummary{}, err
		}
		tier := domain.TierForRating(standing.Rating)
		standing.Tier = &tier
		summary.Ranked = &standing
	}
	return summary, nil
}

// Leaderboard devolve o topo da temporada e a posição do solicitante.
func (s *Service) Leaderboard(ctx context.Context, principal domain.Principal,
	limit int) ([]domain.LeaderboardEntry, *domain.RankedStanding, error) {
	season, err := s.store.ActiveSeason(ctx)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.store.Leaderboard(ctx, season.ID, limit)
	if err != nil {
		return nil, nil, err
	}
	standing, err := s.store.RankedStandingFor(ctx, principal.UserID, season.ID)
	if err != nil {
		return nil, nil, err
	}
	tier := domain.TierForRating(standing.Rating)
	standing.Tier = &tier
	for i := range entries {
		entries[i].Tier = domain.TierForRating(entries[i].Rating).Name
	}
	return entries, &standing, nil
}

// FinishedPlayer identifica um assento no fim de partida.
type FinishedPlayer struct {
	UserID     string
	ChampionID string
}

// RecordFinishedMatch grava a progressão de uma partida encerrada. Chamado
// pelo battle server; idempotente por partida; o bot nunca progride.
func (s *Service) RecordFinishedMatch(ctx context.Context, matchID, mode, rulesetVersion string,
	players [2]FinishedPlayer, winner *int, events []engine.Event) error {
	rs, err := engine.RulesetByVersion(rulesetVersion)
	if err != nil {
		return fmt.Errorf("ruleset da partida: %w", err)
	}
	stats := StatsFromEvents(rs, events)
	pvp := mode != "practice"
	day := utcDay(s.now())

	progress := domain.MatchProgress{MatchID: matchID, Ranked: pvp}
	if pvp {
		if season, err := s.store.ActiveSeason(ctx); err == nil {
			progress.SeasonID = season.ID
		} else {
			progress.Ranked = false
		}
	}
	for slot, player := range players {
		if player.UserID == domain.BotUserID {
			continue
		}
		won := winner != nil && *winner == slot
		entry := domain.PlayerMatchProgress{
			UserID:     player.UserID,
			ChampionID: player.ChampionID,
			Won:        won,
			MasteryXP:  MasteryXPFor(won, pvp),
			RitualDay:  day,
		}
		for _, def := range ritualsForDay(day, player.UserID) {
			delta := ritualProgressFor(def, stats[slot], won, pvp)
			if delta > 0 {
				entry.Rituals = append(entry.Rituals, domain.RitualIncrement{
					RitualID: def.ID, Delta: delta, Target: def.Target, Reward: def.Reward,
				})
			}
		}
		progress.Players = append(progress.Players, entry)
	}
	if len(progress.Players) == 0 {
		return nil
	}
	// Rating só com os dois assentos humanos (PvP real).
	progress.Ranked = progress.Ranked && len(progress.Players) == 2
	_, err = s.store.RecordMatchProgress(ctx, progress)
	return err
}
