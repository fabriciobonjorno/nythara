package storage

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

//go:embed migrations/000001_phase3.up.sql
var migrationUp string

//go:embed migrations/000001_phase3.down.sql
var migrationDown string

//go:embed migrations/000002_battles.up.sql
var migration2Up string

//go:embed migrations/000002_battles.down.sql
var migration2Down string

//go:embed migrations/000003_liveops.up.sql
var migration3Up string

//go:embed migrations/000003_liveops.down.sql
var migration3Down string

type migration struct {
	version int64
	up      string
	down    string
}

var migrations = []migration{
	{version: 1, up: migrationUp, down: migrationDown},
	{version: 2, up: migration2Up, down: migration2Down},
	{version: 3, up: migration3Up, down: migration3Down},
}

type Postgres struct {
	db *sql.DB
}

func Open(ctx context.Context, databaseURL string) (*Postgres, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abrir postgres: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectar postgres: %w", err)
	}
	return &Postgres{db: db}, nil
}

func (p *Postgres) Close() error                   { return p.db.Close() }
func (p *Postgres) Ping(ctx context.Context) error { return p.db.PingContext(ctx) }

func (p *Postgres) MigrateUp(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(882731001)`); err != nil {
		return err
	}
	for _, migration := range migrations {
		var tableExists bool
		if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&tableExists); err != nil {
			return err
		}
		if tableExists {
			var applied bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, migration.version).Scan(&applied); err != nil {
				return err
			}
			if applied {
				continue
			}
		}
		if _, err := tx.ExecContext(ctx, migration.up); err != nil {
			return fmt.Errorf("migration %d up: %w", migration.version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, migration.version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *Postgres) MigrateDown(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(882731001)`); err != nil {
		return err
	}
	var tableExists bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&tableExists); err != nil {
		return err
	}
	if !tableExists {
		return tx.Commit()
	}
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(version),0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	if current == 0 {
		return tx.Commit()
	}
	var selected *migration
	for i := range migrations {
		if migrations[i].version == current {
			selected = &migrations[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("migration %d não é conhecida por este binário", current)
	}
	if _, err := tx.ExecContext(ctx, selected.down); err != nil {
		return fmt.Errorf("migration %d down: %w", selected.version, err)
	}
	if selected.version != 1 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version=$1`, selected.version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *Postgres) SyncCatalog(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE rulesets SET active=false WHERE active`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO rulesets(version,active) VALUES($1,true)
		ON CONFLICT(version) DO UPDATE SET active=true`, engine.RulesetVersion); err != nil {
		return err
	}
	for _, c := range engine.CardList {
		raw, err := json.Marshal(c)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO card_definitions
			(id,ruleset_version,name,faction,card_type,rarity,cost,eclipse_shift,sigil,rules_text,definition)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT(id,ruleset_version) DO UPDATE SET name=EXCLUDED.name,faction=EXCLUDED.faction,
			card_type=EXCLUDED.card_type,rarity=EXCLUDED.rarity,cost=EXCLUDED.cost,
			eclipse_shift=EXCLUDED.eclipse_shift,sigil=EXCLUDED.sigil,rules_text=EXCLUDED.rules_text,
			definition=EXCLUDED.definition WHERE card_definitions.definition=EXCLUDED.definition`,
			c.ID, engine.RulesetVersion, c.Name, c.Faction, string(c.Type), string(c.Rarity), c.Cost,
			c.EclipseShift, string(c.Sigil), c.RulesText, raw)
		if err != nil {
			return fmt.Errorf("sincronizar carta %s: %w", c.ID, err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("carta %s mudou sem novo RulesetVersion", c.ID)
		}
	}
	for id, champion := range engine.Champions {
		raw, err := json.Marshal(champion)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO champions
			(id,ruleset_version,name,faction,vitality,definition) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(id,ruleset_version) DO UPDATE SET name=EXCLUDED.name,faction=EXCLUDED.faction,
			vitality=EXCLUDED.vitality,definition=EXCLUDED.definition
			WHERE champions.definition=EXCLUDED.definition`, id, engine.RulesetVersion,
			champion.Name, champion.Faction, champion.Vitality, raw)
		if err != nil {
			return fmt.Errorf("sincronizar campeão %s: %w", id, err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("campeão %s mudou sem novo RulesetVersion", id)
		}
	}
	cardsPayload, err := json.Marshal(engine.CardList)
	if err != nil {
		return err
	}
	championIDs := make([]string, 0, len(engine.Champions))
	for id := range engine.Champions {
		championIDs = append(championIDs, id)
	}
	sort.Strings(championIDs)
	champList := make([]*engine.ChampionDef, 0, len(championIDs))
	for _, id := range championIDs {
		champList = append(champList, engine.Champions[id])
	}
	champsPayload, err := json.Marshal(champList)
	if err != nil {
		return err
	}
	effectsPayload, err := json.Marshal(engine.Effects)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ruleset_payloads(version,cards,champions,effects)
		VALUES($1,$2,$3,$4) ON CONFLICT(version) DO NOTHING`,
		engine.RulesetVersion, cardsPayload, champsPayload, effectsPayload); err != nil {
		return fmt.Errorf("snapshot do ruleset embutido: %w", err)
	}

	const alphaSeasonStart = "2026-08-10T00:00:00Z"
	if _, err := tx.ExecContext(ctx, `UPDATE seasons SET ends_at=$2
		WHERE ends_at IS NULL AND ruleset_version<>$1 AND starts_at<$2`,
		engine.RulesetVersion, alphaSeasonStart); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO seasons(id,name,ruleset_version,starts_at)
		VALUES('00000000-0000-4000-8000-000000000002','Alpha 0.4', $1, $2)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name
		WHERE seasons.ruleset_version=EXCLUDED.ruleset_version`, engine.RulesetVersion, alphaSeasonStart)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("a temporada Alpha exige um novo ID para o novo RulesetVersion")
	}
	return tx.Commit()
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User, starterRuleset string) (domain.User, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `INSERT INTO users(id,email,password_hash,role) VALUES($1,$2,$3,$4)
		RETURNING created_at`, user.ID, user.Email, user.PasswordHash, user.Role).Scan(&user.CreatedAt)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_profiles(user_id,display_name) VALUES($1,$2)`, user.ID, user.DisplayName); err != nil {
		return domain.User{}, mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_cards(user_id,card_id,ruleset_version,quantity)
		SELECT $1,id,ruleset_version,CASE WHEN rarity='Lendária' THEN 1 ELSE 2 END
		FROM card_definitions WHERE ruleset_version=$2`, user.ID, starterRuleset); err != nil {
		return domain.User{}, mapError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_champions(user_id,champion_id,ruleset_version)
		SELECT $1,id,ruleset_version FROM champions WHERE ruleset_version=$2`, user.ID, starterRuleset); err != nil {
		return domain.User{}, mapError(err)
	}
	payload := []byte(fmt.Sprintf(`{"ruleset_version":%q,"grant":"alpha_complete"}`, starterRuleset))
	if _, err := tx.ExecContext(ctx, `INSERT INTO economy_transactions(id,user_id,kind,source,payload)
		VALUES(gen_random_uuid(),$1,'collection_grant','registration',$2)`, user.ID, payload); err != nil {
		return domain.User{}, mapError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func scanUser(row interface{ Scan(...any) error }) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.PasswordHash, &user.CreatedAt)
	return user, mapError(err)
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(p.db.QueryRowContext(ctx, `SELECT u.id,u.email,p.display_name,u.role,u.password_hash,u.created_at
		FROM users u JOIN player_profiles p ON p.user_id=u.id WHERE u.email=$1`, email))
}

func (p *Postgres) UserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(p.db.QueryRowContext(ctx, `SELECT u.id,u.email,p.display_name,u.role,u.password_hash,u.created_at
		FROM users u JOIN player_profiles p ON p.user_id=u.id WHERE u.id=$1`, id))
}

func (p *Postgres) CreateSession(ctx context.Context, s domain.NewSession) error {
	_, err := p.db.ExecContext(ctx, `INSERT INTO auth_sessions
		(id,user_id,access_hash,refresh_hash,access_expires_at,refresh_expires_at)
		VALUES($1,$2,$3,$4,$5,$6)`, s.ID, s.UserID, s.AccessHash, s.RefreshHash, s.AccessUntil, s.RefreshUntil)
	return mapError(err)
}

func (p *Postgres) AccessToken(ctx context.Context, hash []byte, now time.Time) (domain.TokenRecord, error) {
	var token domain.TokenRecord
	err := p.db.QueryRowContext(ctx, `SELECT u.id,u.role,p.display_name,s.access_expires_at
		FROM auth_sessions s JOIN users u ON u.id=s.user_id JOIN player_profiles p ON p.user_id=u.id
		WHERE s.access_hash=$1 AND s.revoked_at IS NULL AND s.access_expires_at>$2`, hash, now).
		Scan(&token.Principal.UserID, &token.Principal.Role, &token.Principal.DisplayName, &token.ExpiresAt)
	return token, mapError(err)
}

func (p *Postgres) RotateSession(ctx context.Context, old []byte, next domain.RotatedSession, now time.Time) (string, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var id, userID string
	var expires time.Time
	var revoked sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id,user_id,refresh_expires_at,revoked_at FROM auth_sessions
		WHERE refresh_hash=$1 FOR UPDATE`, old).Scan(&id, &userID, &expires, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		var reusedSession string
		reuseErr := tx.QueryRowContext(ctx, `SELECT session_id FROM auth_refresh_history
			WHERE refresh_hash=$1`, old).Scan(&reusedSession)
		if reuseErr == nil {
			if _, updateErr := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1
				WHERE id=$2 AND revoked_at IS NULL`, now, reusedSession); updateErr != nil {
				return "", updateErr
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return "", commitErr
			}
			return "", domain.ErrInvalidCredentials
		}
		return "", domain.ErrInvalidCredentials
	}
	if err != nil || revoked.Valid || !expires.After(now) {
		return "", domain.ErrInvalidCredentials
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO auth_refresh_history(refresh_hash,session_id,consumed_at)
		VALUES($1,$2,$3)`, old, id, now); err != nil {
		return "", mapError(err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET access_hash=$1,refresh_hash=$2,
		access_expires_at=$3,refresh_expires_at=$4,rotated_at=$5 WHERE id=$6`, next.AccessHash,
		next.RefreshHash, next.AccessUntil, next.RefreshUntil, now, id)
	if err != nil {
		return "", mapError(err)
	}
	return userID, tx.Commit()
}

func (p *Postgres) RevokeSession(ctx context.Context, refreshHash []byte) error {
	_, err := p.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE refresh_hash=$1`, refreshHash)
	return err
}

func (p *Postgres) Collection(ctx context.Context, userID, ruleset string) (domain.Collection, error) {
	result := domain.Collection{RulesetVersion: ruleset, Cards: []domain.CollectionCard{}, Champions: []string{}}
	rows, err := p.db.QueryContext(ctx, `SELECT card_id,quantity FROM player_cards
		WHERE user_id=$1 AND ruleset_version=$2 AND quantity>0 ORDER BY card_id`, userID, ruleset)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var card domain.CollectionCard
		if err := rows.Scan(&card.CardID, &card.Quantity); err != nil {
			return result, err
		}
		result.Cards = append(result.Cards, card)
	}
	rows.Close()
	rows, err = p.db.QueryContext(ctx, `SELECT champion_id FROM player_champions
		WHERE user_id=$1 AND ruleset_version=$2 ORDER BY champion_id`, userID, ruleset)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return result, err
		}
		result.Champions = append(result.Champions, id)
	}
	return result, rows.Err()
}

func scanDeck(row interface{ Scan(...any) error }) (domain.Deck, error) {
	var d domain.Deck
	err := row.Scan(&d.ID, &d.UserID, &d.Name, &d.ChampionID, &d.RulesetVersion,
		&d.Version, &d.CreatedAt, &d.UpdatedAt)
	return d, mapError(err)
}

func (p *Postgres) ListDecks(ctx context.Context, userID string) ([]domain.Deck, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,user_id,name,champion_id,ruleset_version,version,created_at,updated_at
		FROM decks WHERE user_id=$1 ORDER BY updated_at DESC,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	decks := []domain.Deck{}
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		if err := p.loadDeckCards(ctx, &d); err != nil {
			return nil, err
		}
		decks = append(decks, d)
	}
	return decks, rows.Err()
}

func (p *Postgres) Deck(ctx context.Context, userID, deckID string) (domain.Deck, error) {
	d, err := scanDeck(p.db.QueryRowContext(ctx, `SELECT id,user_id,name,champion_id,ruleset_version,version,created_at,updated_at
		FROM decks WHERE user_id=$1 AND id=$2`, userID, deckID))
	if err != nil {
		return d, err
	}
	return d, p.loadDeckCards(ctx, &d)
}

func (p *Postgres) loadDeckCards(ctx context.Context, d *domain.Deck) error {
	rows, err := p.db.QueryContext(ctx, `SELECT card_id,quantity FROM deck_cards WHERE deck_id=$1 ORDER BY card_id`, d.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	d.Cards = []domain.DeckCard{}
	for rows.Next() {
		var c domain.DeckCard
		if err := rows.Scan(&c.CardID, &c.Quantity); err != nil {
			return err
		}
		d.Cards = append(d.Cards, c)
	}
	return rows.Err()
}

func (p *Postgres) SaveDeck(ctx context.Context, d domain.Deck, expected *int64, m domain.Mutation) (domain.Deck, bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return d, false, err
	}
	defer tx.Rollback()
	replayed, body, err := claimIdempotency(ctx, tx, d.UserID, m)
	if err != nil {
		return d, false, err
	}
	if replayed {
		if err := json.Unmarshal(body, &d); err != nil {
			return d, false, err
		}
		return d, true, tx.Commit()
	}
	if expected == nil {
		err = tx.QueryRowContext(ctx, `INSERT INTO decks(id,user_id,name,champion_id,ruleset_version)
			VALUES($1,$2,$3,$4,$5) RETURNING version,created_at,updated_at`, d.ID, d.UserID,
			d.Name, d.ChampionID, d.RulesetVersion).Scan(&d.Version, &d.CreatedAt, &d.UpdatedAt)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE decks SET name=$1,champion_id=$2,ruleset_version=$3,
			version=version+1,updated_at=now() WHERE id=$4 AND user_id=$5 AND version=$6
			RETURNING version,created_at,updated_at`, d.Name, d.ChampionID, d.RulesetVersion,
			d.ID, d.UserID, *expected).Scan(&d.Version, &d.CreatedAt, &d.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return d, false, domain.ErrConflict
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `DELETE FROM deck_cards WHERE deck_id=$1`, d.ID)
		}
	}
	if err != nil {
		return d, false, mapError(err)
	}
	for _, c := range d.Cards {
		if _, err := tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
			VALUES($1,$2,$3,$4)`, d.ID, c.CardID, d.RulesetVersion, c.Quantity); err != nil {
			return d, false, mapError(err)
		}
	}
	body, _ = json.Marshal(d)
	if err := finishIdempotency(ctx, tx, d.UserID, m, body); err != nil {
		return d, false, err
	}
	if err := tx.Commit(); err != nil {
		return d, false, mapError(err)
	}
	return d, false, nil
}

func (p *Postgres) DeleteDeck(ctx context.Context, userID, deckID string, expected int64, m domain.Mutation) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	replayed, _, err := claimIdempotency(ctx, tx, userID, m)
	if err != nil || replayed {
		if replayed {
			return true, tx.Commit()
		}
		return false, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM decks WHERE id=$1 AND user_id=$2 AND version=$3`, deckID, userID, expected)
	if err != nil {
		return false, mapError(err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return false, domain.ErrConflict
	}
	if err := finishIdempotency(ctx, tx, userID, m, []byte(`{"deleted":true}`)); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func (p *Postgres) ActiveSeason(ctx context.Context) (domain.Season, error) {
	var s domain.Season
	var end sql.NullTime
	err := p.db.QueryRowContext(ctx, `SELECT id,name,ruleset_version,starts_at,ends_at FROM seasons
		WHERE starts_at<=now() AND (ends_at IS NULL OR ends_at>now()) ORDER BY starts_at DESC LIMIT 1`).
		Scan(&s.ID, &s.Name, &s.RulesetVersion, &s.StartsAt, &end)
	if end.Valid {
		s.EndsAt = &end.Time
	}
	return s, mapError(err)
}

func (p *Postgres) ListRewards(ctx context.Context, userID string) ([]domain.Reward, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,user_id,source,ruleset_version,COALESCE(card_id,''),
		COALESCE(champion_id,''),quantity,created_at FROM rewards WHERE user_id=$1 ORDER BY created_at DESC,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Reward{}
	for rows.Next() {
		var reward domain.Reward
		if err := rows.Scan(&reward.ID, &reward.UserID, &reward.Source, &reward.RulesetVersion,
			&reward.CardID, &reward.ChampionID, &reward.Quantity, &reward.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, reward)
	}
	return result, rows.Err()
}

func (p *Postgres) GrantReward(ctx context.Context, reward domain.Reward, m domain.Mutation) (domain.Reward, bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return reward, false, err
	}
	defer tx.Rollback()
	scopeUserID := mutationScope(reward.UserID, m)
	replayed, body, err := claimIdempotency(ctx, tx, scopeUserID, m)
	if err != nil {
		return reward, false, err
	}
	if replayed {
		if err := json.Unmarshal(body, &reward); err != nil {
			return reward, false, err
		}
		return reward, true, tx.Commit()
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO rewards(id,user_id,source,ruleset_version,card_id,champion_id,quantity)
		VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7) RETURNING created_at`, reward.ID, reward.UserID,
		reward.Source, reward.RulesetVersion, reward.CardID, reward.ChampionID, reward.Quantity).Scan(&reward.CreatedAt)
	if err != nil {
		return reward, false, mapError(err)
	}
	if reward.CardID != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO player_cards(user_id,card_id,ruleset_version,quantity)
			VALUES($1,$2,$3,$4) ON CONFLICT(user_id,card_id,ruleset_version)
			DO UPDATE SET quantity=player_cards.quantity+EXCLUDED.quantity,updated_at=now()`, reward.UserID,
			reward.CardID, reward.RulesetVersion, reward.Quantity)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO player_champions(user_id,champion_id,ruleset_version)
			VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, reward.UserID, reward.ChampionID, reward.RulesetVersion)
	}
	if err != nil {
		return reward, false, mapError(err)
	}
	payload, _ := json.Marshal(reward)
	if _, err := tx.ExecContext(ctx, `INSERT INTO economy_transactions(id,user_id,kind,source,payload)
		VALUES(gen_random_uuid(),$1,'reward',$2,$3)`, reward.UserID, reward.Source, payload); err != nil {
		return reward, false, mapError(err)
	}
	if err := finishIdempotency(ctx, tx, scopeUserID, m, payload); err != nil {
		return reward, false, err
	}
	return reward, false, tx.Commit()
}

func mutationScope(defaultUserID string, m domain.Mutation) string {
	if m.ScopeUserID != "" {
		return m.ScopeUserID
	}
	return defaultUserID
}

func claimIdempotency(ctx context.Context, tx *sql.Tx, userID string, m domain.Mutation) (bool, []byte, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO idempotency_keys(user_id,operation,key,request_hash)
		VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, userID, m.Operation, m.Key, m.RequestHash)
	if err != nil {
		return false, nil, mapError(err)
	}
	if n, _ := result.RowsAffected(); n == 1 {
		return false, nil, nil
	}
	var requestHash, body []byte
	err = tx.QueryRowContext(ctx, `SELECT request_hash,response_body FROM idempotency_keys
		WHERE user_id=$1 AND operation=$2 AND key=$3 FOR UPDATE`, userID, m.Operation, m.Key).Scan(&requestHash, &body)
	if err != nil {
		return false, nil, err
	}
	if !bytes.Equal(requestHash, m.RequestHash) {
		return false, nil, domain.ErrIdempotencyConflict
	}
	if len(body) == 0 {
		return false, nil, domain.ErrConflict
	}
	return true, body, nil
}

func finishIdempotency(ctx context.Context, tx *sql.Tx, userID string, m domain.Mutation, body []byte) error {
	_, err := tx.ExecContext(ctx, `UPDATE idempotency_keys SET response_status=200,response_body=$1
		WHERE user_id=$2 AND operation=$3 AND key=$4`, body, userID, m.Operation, m.Key)
	return err
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
		case "23503", "23514", "22P02":
			return fmt.Errorf("%w: %s", domain.ErrConflict, strings.TrimSpace(pgErr.Message))
		}
	}
	return err
}
