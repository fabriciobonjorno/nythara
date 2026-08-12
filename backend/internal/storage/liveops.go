package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

func (p *Postgres) AdminOverview(ctx context.Context) (domain.AdminOverview, error) {
	var overview domain.AdminOverview
	err := p.db.QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM users WHERE role='player' AND id<>$1),
		(SELECT count(*) FROM users WHERE role IN ('admin','owner')),
		(SELECT count(*) FROM player_bans WHERE lifted_at IS NULL),
		(SELECT count(*) FROM users WHERE role='player' AND id<>$1 AND created_at>=now()-interval '7 days'),
		(SELECT count(DISTINCT s.user_id) FROM auth_sessions s JOIN users u ON u.id=s.user_id
			WHERE u.role='player' AND u.id<>$1 AND s.created_at>=now()-interval '7 days'),
		(SELECT count(DISTINCT s.user_id) FROM auth_sessions s JOIN users u ON u.id=s.user_id
			WHERE u.role='player' AND u.id<>$1 AND s.created_at>=now()-interval '30 days')`, domain.BotUserID).
		Scan(&overview.TotalPlayers, &overview.TotalAdmins, &overview.BannedPlayers,
			&overview.NewPlayers7D, &overview.ActivePlayers7D, &overview.ActivePlayers30D)
	if err != nil {
		return domain.AdminOverview{}, mapError(err)
	}
	err = p.db.QueryRowContext(ctx, `SELECT count(*),
		count(*) FILTER (WHERE status IN ('waiting_ready','active')),
		count(*) FILTER (WHERE status='finished'),
		count(*) FILTER (WHERE status='cancelled'),
		count(*) FILTER (WHERE mode='pvp'),
		count(*) FILTER (WHERE mode='practice') FROM matches`).
		Scan(&overview.TotalMatches, &overview.ActiveMatches, &overview.FinishedMatches,
			&overview.CancelledMatches, &overview.PVPMatches, &overview.PracticeMatches)
	return overview, mapError(err)
}

func (p *Postgres) ListAdminPlayers(ctx context.Context, query string, limit int) ([]domain.AdminPlayer, error) {
	pattern := "%" + query + "%"
	rows, err := p.db.QueryContext(ctx, `WITH session_stats AS (
			SELECT user_id,max(created_at) last_session_at FROM auth_sessions GROUP BY user_id
		), match_stats AS (
			SELECT mp.user_id,count(*) matches,
				count(*) FILTER (WHERE m.status='finished' AND m.winner_slot=mp.slot) wins
			FROM match_players mp JOIN matches m ON m.id=mp.match_id GROUP BY mp.user_id
		)
		SELECT u.id,u.email,p.display_name,u.role,COALESCE(ap.xp,0),u.created_at,
			s.last_session_at,COALESCE(ms.matches,0),COALESCE(ms.wins,0),b.created_at,b.reason
		FROM users u JOIN player_profiles p ON p.user_id=u.id
		LEFT JOIN player_account_progress ap ON ap.user_id=u.id
		LEFT JOIN session_stats s ON s.user_id=u.id
		LEFT JOIN match_stats ms ON ms.user_id=u.id
		LEFT JOIN player_bans b ON b.user_id=u.id AND b.lifted_at IS NULL
		WHERE u.id<>$1 AND ($2='' OR u.email ILIKE $3 OR p.display_name ILIKE $3)
		ORDER BY (b.id IS NOT NULL) DESC,u.created_at DESC LIMIT $4`, domain.BotUserID, query, pattern, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	players := make([]domain.AdminPlayer, 0)
	for rows.Next() {
		var player domain.AdminPlayer
		var lastSession, bannedAt sql.NullTime
		var banReason sql.NullString
		if err := rows.Scan(&player.ID, &player.Email, &player.DisplayName, &player.Role, &player.AccountXP,
			&player.CreatedAt, &lastSession, &player.MatchCount, &player.Wins, &bannedAt, &banReason); err != nil {
			return nil, err
		}
		if lastSession.Valid {
			value := lastSession.Time
			player.LastSessionAt = &value
		}
		if bannedAt.Valid {
			value := bannedAt.Time
			player.BannedAt = &value
			player.BannedReason = banReason.String
		}
		players = append(players, player)
	}
	return players, rows.Err()
}

func (p *Postgres) ListAdminMatches(ctx context.Context, limit int) ([]domain.AdminMatch, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT m.id,m.mode,m.ruleset_version,m.status,
		COALESCE(json_agg(json_build_object('user_id',u.id,'display_name',p.display_name,'slot',mp.slot)
			ORDER BY mp.slot) FILTER (WHERE mp.user_id IS NOT NULL),'[]'::json),
		m.winner_slot,m.end_reason,m.created_at,m.started_at,m.ended_at,
		COALESCE(EXTRACT(EPOCH FROM (m.ended_at-m.started_at))::integer,0)
		FROM matches m LEFT JOIN match_players mp ON mp.match_id=m.id
		LEFT JOIN users u ON u.id=mp.user_id LEFT JOIN player_profiles p ON p.user_id=u.id
		GROUP BY m.id ORDER BY m.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	matches := make([]domain.AdminMatch, 0)
	for rows.Next() {
		var match domain.AdminMatch
		var playersJSON []byte
		var winner sql.NullInt64
		var endReason sql.NullString
		var started, ended sql.NullTime
		if err := rows.Scan(&match.ID, &match.Mode, &match.RulesetVersion, &match.Status, &playersJSON,
			&winner, &endReason, &match.CreatedAt, &started, &ended, &match.DurationSeconds); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(playersJSON, &match.Players); err != nil {
			return nil, err
		}
		if winner.Valid {
			value := int(winner.Int64)
			match.WinnerSlot = &value
		}
		match.EndReason = endReason.String
		if started.Valid {
			value := started.Time
			match.StartedAt = &value
		}
		if ended.Valid {
			value := ended.Time
			match.EndedAt = &value
		}
		matches = append(matches, match)
	}
	return matches, rows.Err()
}

func (p *Postgres) BanPlayer(ctx context.Context, ban domain.PlayerBan, audit domain.AuditEntry) (domain.PlayerBan, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlayerBan{}, err
	}
	defer tx.Rollback()
	var role domain.Role
	if err := tx.QueryRowContext(ctx, `SELECT role FROM users WHERE id=$1 FOR UPDATE`, ban.UserID).Scan(&role); err != nil {
		return domain.PlayerBan{}, mapError(err)
	}
	if role != domain.RolePlayer || ban.UserID == domain.BotUserID {
		return domain.PlayerBan{}, domain.ErrForbidden
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO player_bans(id,user_id,reason,created_by)
		VALUES($1,$2,$3,$4) RETURNING created_at`, ban.ID, ban.UserID, ban.Reason, ban.CreatedBy).
		Scan(&ban.CreatedAt)
	if err != nil {
		return domain.PlayerBan{}, mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=now()
		WHERE user_id=$1 AND revoked_at IS NULL`, ban.UserID); err != nil {
		return domain.PlayerBan{}, mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.PlayerBan{}, err
	}
	return ban, mapError(tx.Commit())
}

func (p *Postgres) LiftPlayerBan(ctx context.Context, userID, liftedBy string, audit domain.AuditEntry) (domain.PlayerBan, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlayerBan{}, err
	}
	defer tx.Rollback()
	var ban domain.PlayerBan
	err = tx.QueryRowContext(ctx, `UPDATE player_bans SET lifted_by=$2,lifted_at=now()
		WHERE user_id=$1 AND lifted_at IS NULL
		RETURNING id,user_id,reason,created_by,created_at,lifted_by,lifted_at`, userID, liftedBy).
		Scan(&ban.ID, &ban.UserID, &ban.Reason, &ban.CreatedBy, &ban.CreatedAt, &ban.LiftedBy, &ban.LiftedAt)
	if err != nil {
		return domain.PlayerBan{}, mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.PlayerBan{}, err
	}
	return ban, mapError(tx.Commit())
}

func (p *Postgres) CreateAdminInvite(ctx context.Context, invite domain.AdminInvite, audit domain.AuditEntry) (domain.AdminInvite, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AdminInvite{}, err
	}
	defer tx.Rollback()
	var owner bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='owner')`, invite.CreatedBy).Scan(&owner); err != nil {
		return domain.AdminInvite{}, mapError(err)
	}
	if !owner {
		return domain.AdminInvite{}, domain.ErrForbidden
	}
	// Serializa convites do mesmo e-mail sem bloquear emissões independentes.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, invite.Email); err != nil {
		return domain.AdminInvite{}, mapError(err)
	}
	var emailExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, invite.Email).Scan(&emailExists); err != nil {
		return domain.AdminInvite{}, mapError(err)
	}
	if emailExists {
		return domain.AdminInvite{}, fmt.Errorf("%w: já existe uma conta com este e-mail", domain.ErrConflict)
	}
	var activeInvite bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM admin_invites
		WHERE email=$1 AND used_at IS NULL AND expires_at>now())`, invite.Email).Scan(&activeInvite); err != nil {
		return domain.AdminInvite{}, mapError(err)
	}
	if activeInvite {
		return domain.AdminInvite{}, fmt.Errorf("%w: já existe um convite ativo para este e-mail", domain.ErrConflict)
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO admin_invites(id,email,token_hash,created_by,expires_at)
		VALUES($1,$2,$3,$4,$5)
		RETURNING created_at`, invite.ID, invite.Email, invite.TokenHash, invite.CreatedBy, invite.ExpiresAt).
		Scan(&invite.CreatedAt)
	if err != nil {
		return domain.AdminInvite{}, mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.AdminInvite{}, err
	}
	return invite, mapError(tx.Commit())
}

func (p *Postgres) ListAdminInvites(ctx context.Context, limit int) ([]domain.AdminInvite, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,email,created_by,expires_at,used_at,used_by,created_at
		FROM admin_invites ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	invites := make([]domain.AdminInvite, 0)
	for rows.Next() {
		var invite domain.AdminInvite
		var usedAt sql.NullTime
		var usedBy sql.NullString
		if err := rows.Scan(&invite.ID, &invite.Email, &invite.CreatedBy, &invite.ExpiresAt,
			&usedAt, &usedBy, &invite.CreatedAt); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			value := usedAt.Time
			invite.UsedAt = &value
			invite.UsedBy = usedBy.String
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

// Fase 7 — Admin/LiveOps. Toda mutação grava a entrada de auditoria na mesma
// transação: sem auditoria, sem mudança.

func appendAudit(ctx context.Context, tx *sql.Tx, audit domain.AuditEntry) error {
	payload := audit.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO admin_audit(actor,action,subject,payload)
		VALUES($1,$2,$3,$4)`, audit.Actor, audit.Action, audit.Subject, payload)
	return mapError(err)
}

func (p *Postgres) ListRulesets(ctx context.Context) ([]domain.RulesetInfo, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT version, active, created_at FROM rulesets ORDER BY created_at, version`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.RulesetInfo
	for rows.Next() {
		var info domain.RulesetInfo
		if err := rows.Scan(&info.Version, &info.Active, &info.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, rows.Err()
}

func (p *Postgres) RulesetPayload(ctx context.Context, version string) (domain.RulesetPayload, error) {
	var payload domain.RulesetPayload
	err := p.db.QueryRowContext(ctx, `SELECT version, cards, champions, effects
		FROM ruleset_payloads WHERE version=$1`, version).
		Scan(&payload.Version, &payload.Cards, &payload.Champions, &payload.Effects)
	return payload, mapError(err)
}

func (p *Postgres) ListRulesetPayloads(ctx context.Context) ([]domain.RulesetPayload, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT version, cards, champions, effects FROM ruleset_payloads ORDER BY created_at`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.RulesetPayload
	for rows.Next() {
		var payload domain.RulesetPayload
		if err := rows.Scan(&payload.Version, &payload.Cards, &payload.Champions, &payload.Effects); err != nil {
			return nil, err
		}
		out = append(out, payload)
	}
	return out, rows.Err()
}

// PublishRuleset grava uma versão completa e imutável: linha em rulesets
// (inativa), snapshot compilável e linhas consultáveis de catálogo — e marca
// o draft de origem como publicado.
func (p *Postgres) PublishRuleset(ctx context.Context, payload domain.RulesetPayload,
	draftID string, audit domain.AuditEntry) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var metadata struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(payload.Effects, &metadata); err != nil {
		return fmt.Errorf("%w: metadados do ruleset: %v", domain.ErrInvalid, err)
	}
	if metadata.Mode == "" {
		metadata.Mode = engine.RulesModeLegacy
	}
	if metadata.Mode != engine.RulesModeLegacy && metadata.Mode != engine.RulesModeConfront {
		return fmt.Errorf("%w: modo do ruleset desconhecido %q", domain.ErrInvalid, metadata.Mode)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rulesets(version,active,mode) VALUES($1,false,$2)`, payload.Version, metadata.Mode); err != nil {
		return mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ruleset_payloads(version,cards,champions,effects)
		VALUES($1,$2,$3,$4)`, payload.Version, payload.Cards, payload.Champions, payload.Effects); err != nil {
		return mapError(err)
	}

	var cards []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Faction      string `json:"faction"`
		Type         string `json:"type"`
		Rarity       string `json:"rarity"`
		Cost         int    `json:"cost"`
		EclipseShift int    `json:"eclipse_shift"`
		Sigil        string `json:"sigil"`
		RulesText    string `json:"rules_text"`
	}
	if err := json.Unmarshal(payload.Cards, &cards); err != nil {
		return fmt.Errorf("%w: cartas do payload: %v", domain.ErrInvalid, err)
	}
	var rawCards []json.RawMessage
	if err := json.Unmarshal(payload.Cards, &rawCards); err != nil {
		return fmt.Errorf("%w: cartas do payload: %v", domain.ErrInvalid, err)
	}
	for i, c := range cards {
		if _, err := tx.ExecContext(ctx, `INSERT INTO card_definitions
			(id,ruleset_version,name,faction,card_type,rarity,cost,eclipse_shift,sigil,rules_text,definition)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			c.ID, payload.Version, c.Name, c.Faction, c.Type, c.Rarity, c.Cost,
			c.EclipseShift, c.Sigil, c.RulesText, rawCards[i]); err != nil {
			return mapError(err)
		}
	}

	var champs []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Faction  string `json:"faction"`
		Vitality int    `json:"vitality"`
	}
	if err := json.Unmarshal(payload.Champions, &champs); err != nil {
		return fmt.Errorf("%w: campeões do payload: %v", domain.ErrInvalid, err)
	}
	var rawChamps []json.RawMessage
	if err := json.Unmarshal(payload.Champions, &rawChamps); err != nil {
		return fmt.Errorf("%w: campeões do payload: %v", domain.ErrInvalid, err)
	}
	for i, ch := range champs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO champions
			(id,ruleset_version,name,faction,vitality,definition) VALUES($1,$2,$3,$4,$5,$6)`,
			ch.ID, payload.Version, ch.Name, ch.Faction, ch.Vitality, rawChamps[i]); err != nil {
			return mapError(err)
		}
	}

	if draftID != "" {
		result, err := tx.ExecContext(ctx, `UPDATE card_drafts
			SET status='published', published_version=$2, updated_at=now()
			WHERE id=$1 AND status IN ('draft','validated')`, draftID, payload.Version)
		if err != nil {
			return mapError(err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("%w: draft %s não está aberto", domain.ErrConflict, draftID)
		}
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Postgres) ActivateRuleset(ctx context.Context, version string, audit domain.AuditEntry) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT true FROM rulesets WHERE version=$1`, version).Scan(&exists); err != nil {
		return mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE rulesets SET active=false WHERE active`); err != nil {
		return mapError(err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rulesets SET active=true WHERE version=$1`, version); err != nil {
		return mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Drafts ---

const draftColumns = `id,card_id,base_version,status,note,card,effects,
	COALESCE(last_validation,'null'::jsonb),COALESCE(published_version,''),created_by,created_at,updated_at`

func scanDraft(scanner interface{ Scan(...any) error }) (domain.CardDraft, error) {
	var d domain.CardDraft
	err := scanner.Scan(&d.ID, &d.CardID, &d.BaseVersion, &d.Status, &d.Note, &d.Card,
		&d.Effects, &d.LastValidation, &d.PublishedVersion, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	return d, mapError(err)
}

func (p *Postgres) CreateDraft(ctx context.Context, draft domain.CardDraft,
	audit domain.AuditEntry) (domain.CardDraft, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CardDraft{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `INSERT INTO card_drafts
		(id,card_id,base_version,status,note,card,effects,created_by)
		VALUES($1,$2,$3,'draft',$4,$5,$6,$7) RETURNING `+draftColumns,
		draft.ID, draft.CardID, draft.BaseVersion, draft.Note, draft.Card, draft.Effects, draft.CreatedBy)
	saved, err := scanDraft(row)
	if err != nil {
		return domain.CardDraft{}, err
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.CardDraft{}, err
	}
	return saved, tx.Commit()
}

// UpdateDraft grava conteúdo/status/validação de um draft ainda não publicado.
func (p *Postgres) UpdateDraft(ctx context.Context, draft domain.CardDraft,
	audit domain.AuditEntry) (domain.CardDraft, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CardDraft{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `UPDATE card_drafts
		SET card=$2, effects=$3, note=$4, status=$5, last_validation=$6, updated_at=now()
		WHERE id=$1 AND status IN ('draft','validated')
		RETURNING `+draftColumns,
		draft.ID, draft.Card, draft.Effects, draft.Note, draft.Status, nullableJSON(draft.LastValidation))
	saved, err := scanDraft(row)
	if err != nil {
		return domain.CardDraft{}, err
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.CardDraft{}, err
	}
	return saved, tx.Commit()
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func (p *Postgres) Draft(ctx context.Context, id string) (domain.CardDraft, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT `+draftColumns+` FROM card_drafts WHERE id=$1`, id)
	return scanDraft(row)
}

func (p *Postgres) ListDrafts(ctx context.Context, status domain.DraftStatus) ([]domain.CardDraft, error) {
	query := `SELECT ` + draftColumns + ` FROM card_drafts`
	args := []any{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY updated_at DESC LIMIT 200`
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.CardDraft
	for rows.Next() {
		d, err := scanDraft(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// --- Bans emergenciais ---

func (p *Postgres) CreateBan(ctx context.Context, ban domain.CardBan,
	audit domain.AuditEntry) (domain.CardBan, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CardBan{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `INSERT INTO ranked_card_bans(id,card_id,reason,created_by)
		VALUES($1,$2,$3,$4) RETURNING id,card_id,reason,created_by,created_at`,
		ban.ID, ban.CardID, ban.Reason, ban.CreatedBy)
	var saved domain.CardBan
	if err := row.Scan(&saved.ID, &saved.CardID, &saved.Reason, &saved.CreatedBy, &saved.CreatedAt); err != nil {
		return domain.CardBan{}, mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.CardBan{}, err
	}
	return saved, tx.Commit()
}

func (p *Postgres) LiftBan(ctx context.Context, cardID, liftedBy string,
	audit domain.AuditEntry) (domain.CardBan, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CardBan{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `UPDATE ranked_card_bans
		SET lifted_by=$2, lifted_at=now() WHERE card_id=$1 AND lifted_at IS NULL
		RETURNING id,card_id,reason,created_by,created_at,lifted_by,lifted_at`, cardID, liftedBy)
	var saved domain.CardBan
	var liftedByOut sql.NullString
	var liftedAt sql.NullTime
	if err := row.Scan(&saved.ID, &saved.CardID, &saved.Reason, &saved.CreatedBy,
		&saved.CreatedAt, &liftedByOut, &liftedAt); err != nil {
		return domain.CardBan{}, mapError(err)
	}
	saved.LiftedBy = liftedByOut.String
	if liftedAt.Valid {
		t := liftedAt.Time
		saved.LiftedAt = &t
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.CardBan{}, err
	}
	return saved, tx.Commit()
}

func (p *Postgres) ActiveBans(ctx context.Context) ([]domain.CardBan, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,card_id,reason,created_by,created_at
		FROM ranked_card_bans WHERE lifted_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.CardBan
	for rows.Next() {
		var ban domain.CardBan
		if err := rows.Scan(&ban.ID, &ban.CardID, &ban.Reason, &ban.CreatedBy, &ban.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ban)
	}
	return out, rows.Err()
}

// --- Temporadas ---

// CreateSeason encerra a temporada aberta (no início da nova) e abre a nova.
func (p *Postgres) CreateSeason(ctx context.Context, season domain.Season,
	audit domain.AuditEntry) (domain.Season, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Season{}, err
	}
	defer tx.Rollback()
	// Fecha as temporadas abertas e captura QUAIS fecharam nesta transação —
	// as recompensas de fim de temporada (ADR-034) só valem para a transição
	// real, o que torna a concessão idempotente por construção.
	closedRows, err := tx.QueryContext(ctx, `UPDATE seasons SET ends_at=$1
		WHERE ends_at IS NULL AND starts_at < $1 RETURNING id`, season.StartsAt)
	if err != nil {
		return domain.Season{}, mapError(err)
	}
	var closed []string
	for closedRows.Next() {
		var id string
		if err := closedRows.Scan(&id); err != nil {
			closedRows.Close()
			return domain.Season{}, err
		}
		closed = append(closed, id)
	}
	closedRows.Close()
	if err := closedRows.Err(); err != nil {
		return domain.Season{}, err
	}
	for _, seasonID := range closed {
		if err := grantSeasonRewards(ctx, tx, seasonID, audit.Actor); err != nil {
			return domain.Season{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO seasons(id,name,ruleset_version,starts_at)
		VALUES($1,$2,$3,$4)`, season.ID, season.Name, season.RulesetVersion, season.StartsAt); err != nil {
		return domain.Season{}, mapError(err)
	}
	if err := appendAudit(ctx, tx, audit); err != nil {
		return domain.Season{}, err
	}
	return season, tx.Commit()
}

// grantSeasonRewards credita Fragmentos pela patente final de cada ranqueado
// da temporada fechada (bot fora; exige ao menos 1 partida), com trilha em
// economy_transactions e um resumo na auditoria — na MESMA transação do
// fechamento.
func grantSeasonRewards(ctx context.Context, tx *sql.Tx, seasonID, actor string) error {
	rows, err := tx.QueryContext(ctx, `SELECT user_id, rating FROM ranked_ratings
		WHERE season_id=$1 AND games >= 1 AND user_id <> $2 ORDER BY user_id`,
		seasonID, domain.BotUserID)
	if err != nil {
		return mapError(err)
	}
	type grant struct {
		userID    string
		rating    int
		tier      string
		fragments int
	}
	var grants []grant
	for rows.Next() {
		var g grant
		if err := rows.Scan(&g.userID, &g.rating); err != nil {
			rows.Close()
			return err
		}
		tier, fragments := domain.SeasonRewardForRating(g.rating)
		g.tier, g.fragments = tier.Key, fragments
		if g.fragments > 0 {
			grants = append(grants, g)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	total := 0
	for _, g := range grants {
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_wallets(user_id,fragments)
			VALUES($1,$2) ON CONFLICT (user_id) DO UPDATE
			SET fragments = player_wallets.fragments + $2, updated_at = now()`,
			g.userID, g.fragments); err != nil {
			return mapError(err)
		}
		payload := fmt.Sprintf(`{"season_id":%q,"tier":%q,"rating":%d,"fragments":%d}`,
			seasonID, g.tier, g.rating, g.fragments)
		if _, err := tx.ExecContext(ctx, `INSERT INTO economy_transactions(id,user_id,kind,source,payload)
			VALUES(gen_random_uuid(),$1,'fragment_grant','season_reward',$2)`,
			g.userID, payload); err != nil {
			return mapError(err)
		}
		total += g.fragments
	}
	summary := fmt.Sprintf(`{"players":%d,"fragments":%d}`, len(grants), total)
	return appendAudit(ctx, tx, domain.AuditEntry{Actor: actor, Action: "season:rewards",
		Subject: seasonID, Payload: json.RawMessage(summary)})
}

// --- Telemetria e auditoria ---

func (p *Postgres) MatchTelemetry(ctx context.Context) (domain.MatchTelemetry, error) {
	var t domain.MatchTelemetry
	err := p.db.QueryRowContext(ctx, `SELECT count(*),
		count(*) FILTER (WHERE status='finished') FROM matches WHERE mode='pvp'`).
		Scan(&t.TotalMatches, &t.FinishedMatches)
	if err != nil {
		return t, mapError(err)
	}
	err = p.db.QueryRowContext(ctx, `
		WITH samples AS (
			SELECT m.id,
				extract(epoch FROM (m.ended_at - m.started_at))::double precision AS duration_seconds,
				COALESCE(max((e.event->>'round')::int), 0)::double precision AS rounds
			FROM matches m
			LEFT JOIN match_events e ON e.match_id = m.id
			WHERE m.mode = 'pvp' AND m.status = 'finished'
				AND m.started_at IS NOT NULL AND m.ended_at IS NOT NULL
			GROUP BY m.id
		)
		SELECT count(*),
			COALESCE(avg(duration_seconds), 0),
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_seconds), 0),
			COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_seconds), 0),
			COALESCE(avg(rounds), 0),
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY rounds), 0),
			COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY rounds), 0),
			count(*) FILTER (WHERE duration_seconds >= 1800)
		FROM samples`).Scan(&t.Rhythm.SampleMatches, &t.Rhythm.AverageDurationSeconds,
		&t.Rhythm.P50DurationSeconds, &t.Rhythm.P95DurationSeconds,
		&t.Rhythm.AverageRounds, &t.Rhythm.P50Rounds, &t.Rhythm.P95Rounds,
		&t.Rhythm.OverThirtyMinutes)
	if err != nil {
		return t, mapError(err)
	}
	// O campeão de um assento vem do deck (match_players não tem coluna
	// própria) — bug pego pelo primeiro consumidor real (painel LiveOps).
	rows, err := p.db.QueryContext(ctx, `
		SELECT d.champion_id, count(*) AS games,
		       count(*) FILTER (WHERE m.winner_slot = mp.slot) AS wins
		FROM match_players mp
		JOIN decks d ON d.id = mp.deck_id
		JOIN matches m ON m.id = mp.match_id
		WHERE m.status = 'finished' AND m.mode = 'pvp'
		GROUP BY d.champion_id ORDER BY d.champion_id`)
	if err != nil {
		return t, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s domain.ChampionMatchStats
		if err := rows.Scan(&s.ChampionID, &s.Games, &s.Wins); err != nil {
			return t, err
		}
		if s.Games > 0 {
			s.WinRate = float64(s.Wins) / float64(s.Games)
		}
		t.ByChampion = append(t.ByChampion, s)
	}
	return t, rows.Err()
}

// AppendAudit registra uma ação administrativa fora de transação de efeito.
func (p *Postgres) AppendAudit(ctx context.Context, audit domain.AuditEntry) error {
	payload := audit.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO admin_audit(actor,action,subject,payload)
		VALUES($1,$2,$3,$4)`, audit.Actor, audit.Action, audit.Subject, payload)
	return mapError(err)
}

func (p *Postgres) ListAudit(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,actor,action,subject,payload,created_at
		FROM admin_audit ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var out []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Subject, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// RotateToRuleset concede a coleção da nova versão a todos os jogadores e
// clona os decks válidos da versão ativa atual (fecha a lacuna operacional do
// ADR-022: publicar → rotacionar → ativar).
func (p *Postgres) RotateToRuleset(ctx context.Context, version string,
	audit domain.AuditEntry) (int, int, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT true FROM rulesets WHERE version=$1`, version).Scan(&exists); err != nil {
		return 0, 0, mapError(err)
	}
	var current string
	if err := tx.QueryRowContext(ctx,
		`SELECT version FROM rulesets WHERE active`).Scan(&current); err != nil {
		return 0, 0, mapError(err)
	}
	if current == version {
		return 0, 0, fmt.Errorf("%w: %s já é a versão ativa", domain.ErrConflict, version)
	}

	result, err := tx.ExecContext(ctx, `INSERT INTO player_cards(user_id,card_id,ruleset_version,quantity)
		SELECT u.id, cd.id, cd.ruleset_version,
		       CASE WHEN cd.rarity='Lendária' THEN 1 ELSE 2 END
		FROM users u CROSS JOIN card_definitions cd
		WHERE cd.ruleset_version=$1
		ON CONFLICT DO NOTHING`, version)
	if err != nil {
		return 0, 0, mapError(err)
	}
	granted, _ := result.RowsAffected()
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_champions(user_id,champion_id,ruleset_version)
		SELECT u.id, c.id, c.ruleset_version FROM users u CROSS JOIN champions c
		WHERE c.ruleset_version=$1 ON CONFLICT DO NOTHING`, version); err != nil {
		return 0, 0, mapError(err)
	}

	// Clona decks da versão ativa cujos conteúdos seguem legais na nova.
	targetRuleset, err := engine.RulesetByVersion(version)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT d.id, d.user_id, d.name, d.champion_id
		FROM decks d WHERE d.ruleset_version=$1`, current)
	if err != nil {
		return 0, 0, mapError(err)
	}
	type deckRow struct{ id, userID, name, champion string }
	var deckRows []deckRow
	for rows.Next() {
		var d deckRow
		if err := rows.Scan(&d.id, &d.userID, &d.name, &d.champion); err != nil {
			rows.Close()
			return 0, 0, err
		}
		deckRows = append(deckRows, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	cloned := 0
	for _, d := range deckRows {
		cardRows, err := tx.QueryContext(ctx,
			`SELECT card_id, quantity FROM deck_cards WHERE deck_id=$1 ORDER BY card_id`, d.id)
		if err != nil {
			return 0, 0, mapError(err)
		}
		var expanded []string
		type pair struct {
			card string
			qty  int
		}
		var pairs []pair
		for cardRows.Next() {
			var card string
			var qty int
			if err := cardRows.Scan(&card, &qty); err != nil {
				cardRows.Close()
				return 0, 0, err
			}
			pairs = append(pairs, pair{card, qty})
			for i := 0; i < qty; i++ {
				expanded = append(expanded, card)
			}
		}
		cardRows.Close()
		if err := cardRows.Err(); err != nil {
			return 0, 0, err
		}
		if targetRuleset.ValidateDeck(d.champion, expanded) != nil {
			continue // deck ficou ilegal na nova versão; dono ajusta no builder
		}
		// UNIQUE(user_id, name): o clone ganha sufixo da versão (limite 64).
		suffix := " · " + version
		name := d.name
		if len(name)+len(suffix) > 64 {
			name = name[:64-len(suffix)]
		}
		name += suffix
		var newID string
		if err := tx.QueryRowContext(ctx, `INSERT INTO decks(id,user_id,name,champion_id,ruleset_version)
			VALUES(gen_random_uuid(),$1,$2,$3,$4)
			ON CONFLICT (user_id, name) DO NOTHING RETURNING id`,
			d.userID, name, d.champion, version).Scan(&newID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // já rotacionado antes (idempotência)
			}
			return 0, 0, mapError(err)
		}
		for _, pr := range pairs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
				VALUES($1,$2,$3,$4)`, newID, pr.card, version, pr.qty); err != nil {
				return 0, 0, mapError(err)
			}
		}
		cloned++
	}

	if err := appendAudit(ctx, tx, audit); err != nil {
		return 0, 0, err
	}
	return int(granted), cloned, tx.Commit()
}
