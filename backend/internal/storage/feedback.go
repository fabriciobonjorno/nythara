package storage

import (
	"context"

	"veurubro/backend/internal/domain"
)

// Recado do Alpha. Um por partida: o índice único deixa o segundo envio virar
// atualização em vez de erro na cara de quem quis ajudar.

func (p *Postgres) RecentFeedback(ctx context.Context, limit int) ([]domain.Feedback, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT f.id,f.user_id,coalesce(f.match_id::text,''),
		f.ruleset_version,f.message,f.created_at
		FROM feedback f ORDER BY f.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.Feedback
	for rows.Next() {
		var entry domain.Feedback
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.MatchID,
			&entry.RulesetVersion, &entry.Message, &entry.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (p *Postgres) SaveFeedback(ctx context.Context, entry domain.Feedback) error {
	var matchID any
	if entry.MatchID != "" {
		matchID = entry.MatchID
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO feedback
		(id,user_id,match_id,ruleset_version,message,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5)
		ON CONFLICT (user_id,match_id) WHERE match_id IS NOT NULL
		DO UPDATE SET message=EXCLUDED.message,created_at=EXCLUDED.created_at`,
		entry.UserID, matchID, entry.RulesetVersion, entry.Message, entry.CreatedAt)
	return mapError(err)
}
