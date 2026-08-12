package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"veurubro/backend/internal/domain"
)

// Arena — histórico pessoal e crônica de partidas. Leitura pura sobre as
// tabelas de batalha já existentes; a autorização (participante + partida
// encerrada) fica na camada de aplicação.

func (p *Postgres) MatchHistory(ctx context.Context, userID string, limit int) ([]domain.MatchSummary, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT m.id, m.mode, m.ruleset_version,
			me.slot, my_deck.champion_id,
			COALESCE(opp_profile.display_name, ''), opp_deck.champion_id,
			m.winner_slot, COALESCE(m.end_reason, ''), m.ended_at
		FROM matches m
		JOIN match_players me ON me.match_id = m.id AND me.user_id = $1
		JOIN decks my_deck ON my_deck.id = me.deck_id
		JOIN match_players opp ON opp.match_id = m.id AND opp.slot = 1 - me.slot
		JOIN decks opp_deck ON opp_deck.id = opp.deck_id
		LEFT JOIN player_profiles opp_profile ON opp_profile.user_id = opp.user_id
		JOIN users viewer ON viewer.id = me.user_id
		WHERE m.status = 'finished'
		  AND (viewer.data_reset_at IS NULL OR m.created_at >= viewer.data_reset_at)
		ORDER BY m.ended_at DESC NULLS LAST, m.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.MatchSummary
	for rows.Next() {
		var summary domain.MatchSummary
		var winner sql.NullInt64
		var finished sql.NullTime
		if err := rows.Scan(&summary.MatchID, &summary.Mode, &summary.RulesetVersion,
			&summary.MySlot, &summary.MyChampion, &summary.Opponent,
			&summary.OpponentChampion, &winner, &summary.EndReason, &finished); err != nil {
			return nil, err
		}
		if winner.Valid {
			summary.Won = int(winner.Int64) == summary.MySlot
		} else {
			summary.Draw = true
		}
		if finished.Valid {
			t := finished.Time
			summary.FinishedAt = &t
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

func (p *Postgres) MatchReplay(ctx context.Context, matchID string) (domain.MatchReplayData, error) {
	replay := domain.MatchReplayData{MatchID: matchID}
	var winner sql.NullInt64
	var endReason sql.NullString
	err := p.db.QueryRowContext(ctx, `SELECT m.mode, m.ruleset_version, m.status,
			m.winner_slot, m.end_reason, m.created_at
		FROM matches m WHERE m.id = $1`, matchID).
		Scan(&replay.Mode, &replay.RulesetVersion, &replay.Status, &winner, &endReason, &replay.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.MatchReplayData{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MatchReplayData{}, mapError(err)
	}
	if winner.Valid {
		w := int(winner.Int64)
		replay.Winner = &w
	}
	replay.EndReason = endReason.String

	seats, err := p.db.QueryContext(ctx, `SELECT mp.slot, mp.user_id,
			COALESCE(pp.display_name, ''), d.champion_id
		FROM match_players mp
		JOIN decks d ON d.id = mp.deck_id
		LEFT JOIN player_profiles pp ON pp.user_id = mp.user_id
		WHERE mp.match_id = $1 ORDER BY mp.slot`, matchID)
	if err != nil {
		return domain.MatchReplayData{}, mapError(err)
	}
	defer seats.Close()
	for seats.Next() {
		var slot int
		var player domain.ReplayPlayer
		if err := seats.Scan(&slot, &player.UserID, &player.DisplayName, &player.ChampionID); err != nil {
			return domain.MatchReplayData{}, err
		}
		if slot == 0 || slot == 1 {
			replay.Players[slot] = player
		}
	}
	if err := seats.Err(); err != nil {
		return domain.MatchReplayData{}, err
	}

	events, err := p.db.QueryContext(ctx, `SELECT event FROM match_events
		WHERE match_id = $1 ORDER BY event_seq`, matchID)
	if err != nil {
		return domain.MatchReplayData{}, mapError(err)
	}
	defer events.Close()
	for events.Next() {
		var raw json.RawMessage
		if err := events.Scan(&raw); err != nil {
			return domain.MatchReplayData{}, err
		}
		replay.Events = append(replay.Events, raw)
	}
	return replay, events.Err()
}
