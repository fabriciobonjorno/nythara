package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"veurubro/backend/internal/domain"
)

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

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rulesets(version,active) VALUES($1,false)`, payload.Version); err != nil {
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
	if _, err := tx.ExecContext(ctx, `UPDATE seasons SET ends_at=$1
		WHERE ends_at IS NULL AND starts_at < $1`, season.StartsAt); err != nil {
		return domain.Season{}, mapError(err)
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

// --- Telemetria e auditoria ---

func (p *Postgres) MatchTelemetry(ctx context.Context) (domain.MatchTelemetry, error) {
	var t domain.MatchTelemetry
	err := p.db.QueryRowContext(ctx, `SELECT count(*),
		count(*) FILTER (WHERE status='finished') FROM matches`).
		Scan(&t.TotalMatches, &t.FinishedMatches)
	if err != nil {
		return t, mapError(err)
	}
	rows, err := p.db.QueryContext(ctx, `
		SELECT mp.champion_id, count(*) AS games,
		       count(*) FILTER (WHERE m.winner_slot = mp.slot) AS wins
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE m.status = 'finished'
		GROUP BY mp.champion_id ORDER BY mp.champion_id`)
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
