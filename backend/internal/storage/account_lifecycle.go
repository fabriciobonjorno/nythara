package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"veurubro/backend/internal/domain"
)

// DeactivateAccount preserva a identidade, encerra todas as sessões e impede
// que contas privilegiadas usem uma rota de autosserviço para desaparecer.
func (p *Postgres) DeactivateAccount(ctx context.Context, userID, expectedPasswordHash string, now time.Time) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var role domain.Role
	var passwordHash string
	var deleted sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT role,password_hash,deleted_at FROM users WHERE id=$1 FOR UPDATE`, userID).
		Scan(&role, &passwordHash, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return mapError(err)
	}
	if role != domain.RolePlayer || userID == domain.BotUserID {
		return domain.ErrForbidden
	}
	if deleted.Valid {
		return domain.ErrConflict
	}
	if passwordHash != expectedPasswordHash {
		return domain.ErrInvalidCredentials
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET deleted_at=$2,reactivated_at=NULL,
		reactivation_reset_pending=false,updated_at=$2 WHERE id=$1`, userID, now); err != nil {
		return mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$2
		WHERE user_id=$1 AND revoked_at IS NULL`, userID, now); err != nil {
		return mapError(err)
	}
	return mapError(tx.Commit())
}

// ResolveAccountReactivation fecha uma pendência uma única vez. O reset cria
// um novo ciclo jogável sem remover as linhas que sustentam partidas alheias.
func (p *Postgres) ResolveAccountReactivation(ctx context.Context, userID string, resetData bool,
	starterRuleset string, now time.Time) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var role domain.Role
	var deleted sql.NullTime
	var pending bool
	err = tx.QueryRowContext(ctx, `SELECT role,deleted_at,reactivation_reset_pending
		FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&role, &deleted, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return mapError(err)
	}
	if role != domain.RolePlayer || userID == domain.BotUserID {
		return domain.ErrForbidden
	}
	if deleted.Valid || !pending {
		return domain.ErrConflict
	}
	if !resetData {
		_, err = tx.ExecContext(ctx, `UPDATE users SET reactivation_reset_pending=false,updated_at=$2
			WHERE id=$1`, userID, now)
		if err != nil {
			return mapError(err)
		}
		return mapError(tx.Commit())
	}

	if _, err := tx.ExecContext(ctx, `UPDATE decks SET active=false,archived_at=$2,updated_at=$2
		WHERE user_id=$1 AND archived_at IS NULL`, userID, now); err != nil {
		return mapError(err)
	}
	for _, statement := range []string{
		`DELETE FROM player_cards WHERE user_id=$1`,
		`DELETE FROM player_champions WHERE user_id=$1`,
		`DELETE FROM player_account_progress WHERE user_id=$1`,
		`DELETE FROM player_rituals WHERE user_id=$1`,
		`DELETE FROM player_champion_mastery WHERE user_id=$1`,
		`DELETE FROM ranked_ratings WHERE user_id=$1`,
		`DELETE FROM player_wallets WHERE user_id=$1`,
		`DELETE FROM rewards WHERE user_id=$1`,
		`DELETE FROM idempotency_keys WHERE user_id=$1`,
		`DELETE FROM feedback WHERE user_id=$1`,
	} {
		if _, err := tx.ExecContext(ctx, statement, userID); err != nil {
			return mapError(err)
		}
	}
	if err := grantStarterGameDataTx(ctx, tx, userID, starterRuleset, "account_reactivation"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET reactivation_reset_pending=false,
		data_reset_at=$2,updated_at=$2 WHERE id=$1`, userID, now); err != nil {
		return mapError(err)
	}
	return mapError(tx.Commit())
}
