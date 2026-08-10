package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

var _ battle.Store = (*Postgres)(nil)

func (p *Postgres) CreateMatch(ctx context.Context, match battle.Match) error {
	config, err := json.Marshal(match.Config)
	if err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mode := match.Mode
	if mode == "" {
		mode = "pvp"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO matches
		(id,ruleset_version,seed,config,status,mode,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		match.ID, match.Config.RulesetVersion, int64(match.Config.Seed), config, match.Status, mode, match.CreatedAt); err != nil {
		return mapError(err)
	}
	for _, player := range match.Players {
		if _, err := tx.ExecContext(ctx, `INSERT INTO match_players(match_id,slot,user_id,deck_id)
			VALUES($1,$2,$3,$4)`, match.ID, player.Slot, player.UserID, player.DeckID); err != nil {
			return mapError(err)
		}
	}
	return tx.Commit()
}

func (p *Postgres) MarkReady(ctx context.Context, matchID string, slot int) error {
	result, err := p.db.ExecContext(ctx, `UPDATE match_players SET ready_at=COALESCE(ready_at,now())
		WHERE match_id=$1 AND slot=$2`, matchID, slot)
	if err != nil {
		return mapError(err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (p *Postgres) StartMatch(ctx context.Context, matchID string, events []engine.Event, snapshot battle.Snapshot) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE matches SET status='active',started_at=now()
		WHERE id=$1 AND status='waiting_ready' AND
		(SELECT count(*) FROM match_players WHERE match_id=$1 AND ready_at IS NOT NULL)=2`, matchID)
	if err != nil {
		return mapError(err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return battle.ErrNotReady
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO match_commands
		(match_id,command_index,player_slot,origin,command) VALUES($1,0,-1,'system','{"kind":"start"}')`, matchID); err != nil {
		return mapError(err)
	}
	if err := insertEvents(ctx, tx, matchID, 0, events); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO match_snapshots
		(match_id,command_index,event_seq,snapshot) VALUES($1,$2,$3,$4)`, matchID,
		snapshot.CommandIndex, snapshot.EventSeq, snapshot.Data); err != nil {
		return mapError(err)
	}
	return tx.Commit()
}

func (p *Postgres) PersistStep(ctx context.Context, step battle.Step) error {
	commandJSON, err := json.Marshal(step.Command)
	if err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if step.ClientSequence != nil {
		result, err := tx.ExecContext(ctx, `UPDATE match_players SET last_client_sequence=$1
			WHERE match_id=$2 AND slot=$3 AND last_client_sequence=$1-1`, *step.ClientSequence,
			step.MatchID, step.PlayerSlot)
		if err != nil {
			return mapError(err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return battle.ErrSequenceGap
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO match_commands
		(match_id,command_index,player_slot,client_sequence,origin,command)
		VALUES($1,$2,$3,$4,$5,$6)`, step.MatchID, step.CommandIndex, step.PlayerSlot,
		step.ClientSequence, step.Origin, commandJSON); err != nil {
		return mapError(err)
	}
	if err := insertEvents(ctx, tx, step.MatchID, step.CommandIndex, step.Events); err != nil {
		return err
	}
	if step.Snapshot != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO match_snapshots
			(match_id,command_index,event_seq,snapshot) VALUES($1,$2,$3,$4)`, step.MatchID,
			step.Snapshot.CommandIndex, step.Snapshot.EventSeq, step.Snapshot.Data); err != nil {
			return mapError(err)
		}
	}
	if step.Finished {
		result, err := tx.ExecContext(ctx, `UPDATE matches SET status='finished',winner_slot=$1,
			end_reason=$2,ended_at=now() WHERE id=$3 AND status='active'`, step.Winner, step.EndReason, step.MatchID)
		if err != nil {
			return mapError(err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return domain.ErrConflict
		}
	}
	return tx.Commit()
}

func insertEvents(ctx context.Context, tx *sql.Tx, matchID string, commandIndex int64, events []engine.Event) error {
	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO match_events(match_id,event_seq,command_index,event)
			VALUES($1,$2,$3,$4)`, matchID, event.Seq, commandIndex, raw); err != nil {
			return mapError(err)
		}
	}
	return nil
}

func (p *Postgres) CancelMatch(ctx context.Context, matchID, reason string) error {
	result, err := p.db.ExecContext(ctx, `UPDATE matches SET status='cancelled',end_reason=$1,ended_at=now()
		WHERE id=$2 AND status='waiting_ready'`, reason, matchID)
	if err != nil {
		return mapError(err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (p *Postgres) LoadMatch(ctx context.Context, matchID string) (battle.LoadedMatch, error) {
	var loaded battle.LoadedMatch
	var configRaw []byte
	var winner sql.NullInt16
	var started, ended sql.NullTime
	err := p.db.QueryRowContext(ctx, `SELECT id,config,status,winner_slot,COALESCE(end_reason,''),
		mode,created_at,started_at,ended_at FROM matches WHERE id=$1`, matchID).Scan(
		&loaded.Match.ID, &configRaw, &loaded.Match.Status, &winner, &loaded.Match.EndReason,
		&loaded.Match.Mode, &loaded.Match.CreatedAt, &started, &ended)
	if err != nil {
		return loaded, mapError(err)
	}
	if err := json.Unmarshal(configRaw, &loaded.Match.Config); err != nil {
		return loaded, err
	}
	if winner.Valid {
		value := int(winner.Int16)
		loaded.Match.Winner = &value
	}
	if started.Valid {
		loaded.Match.StartedAt = &started.Time
	}
	if ended.Valid {
		loaded.Match.EndedAt = &ended.Time
	}
	rows, err := p.db.QueryContext(ctx, `SELECT slot,user_id,deck_id,ready_at IS NOT NULL,last_client_sequence
		FROM match_players WHERE match_id=$1 ORDER BY slot`, matchID)
	if err != nil {
		return loaded, err
	}
	for rows.Next() {
		var player battle.Participant
		if err := rows.Scan(&player.Slot, &player.UserID, &player.DeckID, &player.Ready, &player.LastSequence); err != nil {
			rows.Close()
			return loaded, err
		}
		loaded.Match.Players[player.Slot] = player
	}
	if err := rows.Close(); err != nil {
		return loaded, err
	}
	rows, err = p.db.QueryContext(ctx, `SELECT c.command_index,c.player_slot,c.client_sequence,c.origin,c.command,
		COALESCE(min(e.event_seq),-1),COALESCE(max(e.event_seq),-1)
		FROM match_commands c LEFT JOIN match_events e
		ON e.match_id=c.match_id AND e.command_index=c.command_index
		WHERE c.match_id=$1 GROUP BY c.command_index,c.player_slot,c.client_sequence,c.origin,c.command
		ORDER BY c.command_index`, matchID)
	if err != nil {
		return loaded, err
	}
	for rows.Next() {
		var command battle.StoredCommand
		var sequence sql.NullInt64
		var raw []byte
		if err := rows.Scan(&command.Index, &command.PlayerSlot, &sequence, &command.Origin, &raw,
			&command.FirstEventSeq, &command.LastEventSeq); err != nil {
			rows.Close()
			return loaded, err
		}
		if sequence.Valid {
			value := sequence.Int64
			command.ClientSequence = &value
		}
		if command.Index > 0 {
			if err := json.Unmarshal(raw, &command.Command); err != nil {
				rows.Close()
				return loaded, err
			}
		}
		loaded.Commands = append(loaded.Commands, command)
	}
	if err := rows.Close(); err != nil {
		return loaded, err
	}
	rows, err = p.db.QueryContext(ctx, `SELECT event FROM match_events WHERE match_id=$1 ORDER BY event_seq`, matchID)
	if err != nil {
		return loaded, err
	}
	for rows.Next() {
		var raw []byte
		var event engine.Event
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return loaded, err
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			rows.Close()
			return loaded, err
		}
		loaded.Events = append(loaded.Events, event)
	}
	if err := rows.Close(); err != nil {
		return loaded, err
	}
	var snapshot battle.Snapshot
	err = p.db.QueryRowContext(ctx, `SELECT command_index,event_seq,snapshot FROM match_snapshots
		WHERE match_id=$1 ORDER BY command_index DESC LIMIT 1`, matchID).Scan(
		&snapshot.CommandIndex, &snapshot.EventSeq, &snapshot.Data)
	if err == nil {
		loaded.Snapshot = &snapshot
	} else if !errors.Is(err, sql.ErrNoRows) {
		return loaded, fmt.Errorf("carregar snapshot: %w", err)
	}
	return loaded, nil
}

func (p *Postgres) ActiveMatchForUser(ctx context.Context, userID string) (battle.Match, int, error) {
	var matchID string
	var slot int
	err := p.db.QueryRowContext(ctx, `SELECT m.id,mp.slot FROM matches m
		JOIN match_players mp ON mp.match_id=m.id
		WHERE mp.user_id=$1 AND m.status IN ('waiting_ready','active')
		ORDER BY m.created_at DESC LIMIT 1`, userID).Scan(&matchID, &slot)
	if err != nil {
		return battle.Match{}, 0, mapError(err)
	}
	loaded, err := p.LoadMatch(ctx, matchID)
	if err != nil {
		return battle.Match{}, 0, err
	}
	return loaded.Match, slot, nil
}
