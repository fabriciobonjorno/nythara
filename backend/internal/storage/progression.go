package storage

import (
	"context"
	"database/sql"
	"fmt"

	"veurubro/backend/internal/domain"
)

// P1 — progressão. RecordMatchProgress grava tudo numa transação com trava de
// idempotência por partida: rituais, fragmentos (com trilha em
// economy_transactions), maestria e rating (lido com FOR UPDATE).

func (p *Postgres) RecordMatchProgress(ctx context.Context, progress domain.MatchProgress) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `INSERT INTO match_progress_log(match_id)
		VALUES($1) ON CONFLICT DO NOTHING`, progress.MatchID)
	if err != nil {
		return false, mapError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return false, nil // partida já creditada
	}

	for _, player := range progress.Players {
		reward := 0
		for _, inc := range player.Rituals {
			if inc.Delta <= 0 {
				continue
			}
			var completedBefore, completedAfter bool
			err := tx.QueryRowContext(ctx, `INSERT INTO player_rituals(user_id,day,ritual_id,progress,target,reward)
				VALUES($1,$2,$3,LEAST($4::int,$5::int),$5,$6)
				ON CONFLICT (user_id,day,ritual_id) DO UPDATE
				SET progress = LEAST(player_rituals.progress + $4::int, $5::int)
				RETURNING (player_rituals.completed_at IS NOT NULL), progress >= target`,
				player.UserID, player.RitualDay, inc.RitualID, inc.Delta, inc.Target, inc.Reward).
				Scan(&completedBefore, &completedAfter)
			if err != nil {
				return false, mapError(err)
			}
			if !completedBefore && completedAfter {
				if _, err := tx.ExecContext(ctx, `UPDATE player_rituals SET completed_at=now()
					WHERE user_id=$1 AND day=$2 AND ritual_id=$3 AND completed_at IS NULL`,
					player.UserID, player.RitualDay, inc.RitualID); err != nil {
					return false, mapError(err)
				}
				reward += inc.Reward
			}
		}
		if reward > 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO player_wallets(user_id,fragments)
				VALUES($1,$2) ON CONFLICT (user_id) DO UPDATE
				SET fragments = player_wallets.fragments + $2, updated_at = now()`,
				player.UserID, reward); err != nil {
				return false, mapError(err)
			}
			payload := fmt.Sprintf(`{"match_id":%q,"fragments":%d}`, progress.MatchID, reward)
			if _, err := tx.ExecContext(ctx, `INSERT INTO economy_transactions(id,user_id,kind,source,payload)
				VALUES(gen_random_uuid(),$1,'fragment_grant','ritual',$2)`,
				player.UserID, payload); err != nil {
				return false, mapError(err)
			}
		}
		winInc := 0
		if player.Won {
			winInc = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_champion_mastery(user_id,champion_id,xp,games,wins)
			VALUES($1,$2,$3,1,$4) ON CONFLICT (user_id,champion_id) DO UPDATE
			SET xp = player_champion_mastery.xp + $3,
			    games = player_champion_mastery.games + 1,
			    wins = player_champion_mastery.wins + $4,
			    updated_at = now()`,
			player.UserID, player.ChampionID, player.MasteryXP, winInc); err != nil {
			return false, mapError(err)
		}
	}

	if progress.Ranked && progress.SeasonID != "" && len(progress.Players) == 2 {
		ratings := [2]int{}
		for i, player := range progress.Players {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ranked_ratings(user_id,season_id)
				VALUES($1,$2) ON CONFLICT DO NOTHING`, player.UserID, progress.SeasonID); err != nil {
				return false, mapError(err)
			}
			if err := tx.QueryRowContext(ctx, `SELECT rating FROM ranked_ratings
				WHERE user_id=$1 AND season_id=$2 FOR UPDATE`,
				player.UserID, progress.SeasonID).Scan(&ratings[i]); err != nil {
				return false, mapError(err)
			}
		}
		for i, player := range progress.Players {
			delta := domain.EloDelta(ratings[i], ratings[1-i], player.Won)
			winInc := 0
			if player.Won {
				winInc = 1
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ranked_ratings
				SET rating = GREATEST(rating + $3, 0), games = games + 1,
				    wins = wins + $4, updated_at = now()
				WHERE user_id=$1 AND season_id=$2`,
				player.UserID, progress.SeasonID, delta, winInc); err != nil {
				return false, mapError(err)
			}
		}
	}
	return true, tx.Commit()
}

func (p *Postgres) RitualsFor(ctx context.Context, userID, day string) ([]domain.RitualState, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT ritual_id, progress, target, reward, completed_at
		FROM player_rituals WHERE user_id=$1 AND day=$2 ORDER BY ritual_id`, userID, day)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.RitualState
	for rows.Next() {
		var state domain.RitualState
		var completed sql.NullTime
		if err := rows.Scan(&state.RitualID, &state.Progress, &state.Target, &state.Reward, &completed); err != nil {
			return nil, err
		}
		if completed.Valid {
			t := completed.Time
			state.CompletedAt = &t
		}
		out = append(out, state)
	}
	return out, rows.Err()
}

// SaveRitualStates materializa os rituais sorteados do dia (idempotente) para
// o jogador ver os alvos antes de progredir.
func (p *Postgres) SaveRitualStates(ctx context.Context, userID, day string,
	states []domain.RitualState) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, state := range states {
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_rituals(user_id,day,ritual_id,progress,target,reward)
			VALUES($1,$2,$3,0,$4,$5) ON CONFLICT DO NOTHING`,
			userID, day, state.RitualID, state.Target, state.Reward); err != nil {
			return mapError(err)
		}
	}
	return tx.Commit()
}

func (p *Postgres) Fragments(ctx context.Context, userID string) (int, error) {
	var fragments int
	err := p.db.QueryRowContext(ctx,
		`SELECT COALESCE((SELECT fragments FROM player_wallets WHERE user_id=$1), 0)`, userID).
		Scan(&fragments)
	return fragments, mapError(err)
}

func (p *Postgres) MasteryFor(ctx context.Context, userID string) ([]domain.ChampionMastery, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT champion_id, xp, games, wins
		FROM player_champion_mastery WHERE user_id=$1 ORDER BY xp DESC, champion_id`, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.ChampionMastery
	for rows.Next() {
		var mastery domain.ChampionMastery
		if err := rows.Scan(&mastery.ChampionID, &mastery.XP, &mastery.Games, &mastery.Wins); err != nil {
			return nil, err
		}
		out = append(out, mastery)
	}
	return out, rows.Err()
}

func (p *Postgres) RankedStandingFor(ctx context.Context, userID, seasonID string) (domain.RankedStanding, error) {
	standing := domain.RankedStanding{SeasonID: seasonID, Rating: 1000}
	err := p.db.QueryRowContext(ctx, `SELECT rating, games, wins,
		(SELECT count(*)+1 FROM ranked_ratings other
		 WHERE other.season_id=$2 AND other.rating > ranked_ratings.rating)
		FROM ranked_ratings WHERE user_id=$1 AND season_id=$2`,
		userID, seasonID).Scan(&standing.Rating, &standing.Games, &standing.Wins, &standing.Position)
	if err == sql.ErrNoRows {
		return standing, nil // ainda sem partidas ranqueadas
	}
	return standing, mapError(err)
}

func (p *Postgres) Leaderboard(ctx context.Context, seasonID string, limit int) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := p.db.QueryContext(ctx, `SELECT r.user_id, COALESCE(pp.display_name,''),
		r.rating, r.games, r.wins
		FROM ranked_ratings r
		LEFT JOIN player_profiles pp ON pp.user_id = r.user_id
		WHERE r.season_id=$1 AND r.user_id <> $2
		ORDER BY r.rating DESC, r.updated_at ASC LIMIT $3`,
		seasonID, domain.BotUserID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.LeaderboardEntry
	position := 0
	for rows.Next() {
		position++
		entry := domain.LeaderboardEntry{Position: position}
		if err := rows.Scan(&entry.UserID, &entry.DisplayName, &entry.Rating, &entry.Games, &entry.Wins); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}
