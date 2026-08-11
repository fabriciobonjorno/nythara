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

//go:embed migrations/000004_practice.up.sql
var migration4Up string

//go:embed migrations/000004_practice.down.sql
var migration4Down string

//go:embed migrations/000005_progression.up.sql
var migration5Up string

//go:embed migrations/000005_progression.down.sql
var migration5Down string

//go:embed migrations/000006_identity.up.sql
var migration6Up string

//go:embed migrations/000006_identity.down.sql
var migration6Down string

//go:embed migrations/000007_confront.up.sql
var migration7Up string

//go:embed migrations/000007_confront.down.sql
var migration7Down string

//go:embed migrations/000008_ruleset_mode.up.sql
var migration8Up string

//go:embed migrations/000008_ruleset_mode.down.sql
var migration8Down string

//go:embed migrations/000009_confront_composition.up.sql
var migration9Up string

//go:embed migrations/000009_confront_composition.down.sql
var migration9Down string

//go:embed migrations/000010_confront_single_deck.up.sql
var migration10Up string

//go:embed migrations/000010_confront_single_deck.down.sql
var migration10Down string

//go:embed migrations/000011_feedback.up.sql
var migration11Up string

//go:embed migrations/000011_feedback.down.sql
var migration11Down string

type migration struct {
	version int64
	up      string
	down    string
}

var migrations = []migration{
	{version: 1, up: migrationUp, down: migrationDown},
	{version: 2, up: migration2Up, down: migration2Down},
	{version: 3, up: migration3Up, down: migration3Down},
	{version: 4, up: migration4Up, down: migration4Down},
	{version: 5, up: migration5Up, down: migration5Down},
	{version: 6, up: migration6Up, down: migration6Down},
	{version: 7, up: migration7Up, down: migration7Down},
	{version: 8, up: migration8Up, down: migration8Down},
	{version: 9, up: migration9Up, down: migration9Down},
	{version: 10, up: migration10Up, down: migration10Down},
	{version: 11, up: migration11Up, down: migration11Down},
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
	activeRS := engine.CompetitiveRuleset()
	activeVersion := engine.CompetitiveRulesetVersion
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE rulesets SET active=false WHERE active`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO rulesets(version,active,mode) VALUES($1,true,$2)
		ON CONFLICT(version) DO UPDATE SET active=true,mode=EXCLUDED.mode`, activeVersion, activeRS.Mode); err != nil {
		return err
	}
	for _, c := range activeRS.CardList {
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
			c.ID, activeVersion, c.Name, c.Faction, string(c.Type), string(c.Rarity), c.Cost,
			c.EclipseShift, string(c.Sigil), c.RulesText, raw)
		if err != nil {
			return fmt.Errorf("sincronizar carta %s: %w", c.ID, err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("carta %s mudou sem novo RulesetVersion", c.ID)
		}
	}
	for id, champion := range activeRS.Champions {
		raw, err := json.Marshal(champion)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO champions
			(id,ruleset_version,name,faction,vitality,definition) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(id,ruleset_version) DO UPDATE SET name=EXCLUDED.name,faction=EXCLUDED.faction,
			vitality=EXCLUDED.vitality,definition=EXCLUDED.definition
			WHERE champions.definition=EXCLUDED.definition`, id, activeVersion,
			champion.Name, champion.Faction, champion.Vitality, raw)
		if err != nil {
			return fmt.Errorf("sincronizar campeão %s: %w", id, err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("campeão %s mudou sem novo RulesetVersion", id)
		}
	}
	cardsPayload, err := json.Marshal(activeRS.CardList)
	if err != nil {
		return err
	}
	championIDs := make([]string, 0, len(activeRS.Champions))
	for id := range activeRS.Champions {
		championIDs = append(championIDs, id)
	}
	sort.Strings(championIDs)
	champList := make([]*engine.ChampionDef, 0, len(championIDs))
	for _, id := range championIDs {
		champList = append(champList, activeRS.Champions[id])
	}
	champsPayload, err := json.Marshal(champList)
	if err != nil {
		return err
	}
	effectsPayload, err := json.Marshal(activeRS.Effects)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ruleset_payloads(version,cards,champions,effects)
		VALUES($1,$2,$3,$4) ON CONFLICT(version) DO NOTHING`,
		activeVersion, cardsPayload, champsPayload, effectsPayload); err != nil {
		return fmt.Errorf("snapshot do ruleset embutido: %w", err)
	}
	// Migração de metadado: snapshots alpha-0.9 gravados antes de o modo se
	// tornar explícito já têm o mesmo conteúdo executável. Preenche somente a
	// chave ausente, sem substituir cartas/efeitos de uma versão imutável.
	if _, err := tx.ExecContext(ctx, `UPDATE ruleset_payloads
		SET effects=jsonb_set(effects,'{mode}',to_jsonb($2::text),true)
		WHERE version=$1 AND COALESCE(effects->>'mode','')=''`, activeVersion, activeRS.Mode); err != nil {
		return fmt.Errorf("metadado do snapshot embutido: %w", err)
	}

	// O grant de registro "alpha_complete" é uma autorização contínua durante
	// o Alpha, não um snapshot de um ruleset antigo. Propaga apenas para contas
	// que já receberam esse grant auditado e só insere os itens ausentes.
	if _, err := tx.ExecContext(ctx, `WITH entitled AS (
			SELECT DISTINCT user_id FROM economy_transactions
			WHERE kind='collection_grant' AND payload->>'grant'='alpha_complete'
		)
		INSERT INTO player_champions(user_id,champion_id,ruleset_version)
		SELECT entitled.user_id,champions.id,champions.ruleset_version
		FROM entitled CROSS JOIN champions
		WHERE champions.ruleset_version=$1
		ON CONFLICT DO NOTHING`, activeVersion); err != nil {
		return fmt.Errorf("propagar campeões do grant alpha completo: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `WITH entitled AS (
			SELECT DISTINCT user_id FROM economy_transactions
			WHERE kind='collection_grant' AND payload->>'grant'='alpha_complete'
		)
		INSERT INTO player_cards(user_id,card_id,ruleset_version,quantity)
		SELECT entitled.user_id,card_definitions.id,card_definitions.ruleset_version,
			CASE WHEN card_definitions.rarity='Lendária' THEN 1 ELSE 2 END
		FROM entitled CROSS JOIN card_definitions
		WHERE card_definitions.ruleset_version=$1
		ON CONFLICT DO NOTHING`, activeVersion); err != nil {
		return fmt.Errorf("propagar cartas do grant alpha completo: %w", err)
	}

	// O bot precisa possuir a coleção para os decks dele passarem nas
	// constraints de posse (mesma regra dos jogadores).
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_champions(user_id,champion_id,ruleset_version)
		SELECT $1,id,ruleset_version FROM champions WHERE ruleset_version=$2
		ON CONFLICT DO NOTHING`, domain.BotUserID, activeVersion); err != nil {
		return fmt.Errorf("campeões do bot: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_cards(user_id,card_id,ruleset_version,quantity)
		SELECT $1,id,ruleset_version,CASE WHEN rarity='Lendária' THEN 1 ELSE 2 END
		FROM card_definitions WHERE ruleset_version=$2
		ON CONFLICT DO NOTHING`, domain.BotUserID, activeVersion); err != nil {
		return fmt.Errorf("cartas do bot: %w", err)
	}
	// Toda conta com coleção do Confronto recebe um único baralho pronto. É
	// editável imediatamente; a trava começa apenas depois do primeiro save do
	// jogador. O mesmo caminho semeia o bot, evitando uma exceção de regras.
	if _, err := tx.ExecContext(ctx, `WITH owners AS (
			SELECT user_id,min(champion_id) AS avatar_id
			FROM player_champions WHERE ruleset_version=$1 GROUP BY user_id
		)
		INSERT INTO decks(id,user_id,name,champion_id,ruleset_version,active,system_provided)
		SELECT gen_random_uuid(),owners.user_id,
			CASE WHEN owners.user_id=$2 THEN 'Bot Confronto · '||$1 ELSE 'Meu Baralho · '||$1 END,
			owners.avatar_id,$1,true,true FROM owners
		WHERE NOT EXISTS (
			SELECT 1 FROM decks existing
			WHERE existing.user_id=owners.user_id AND existing.ruleset_version=$1
		)
		ON CONFLICT DO NOTHING`, activeVersion, domain.BotUserID); err != nil {
		return fmt.Errorf("baralho inicial do Confronto: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `WITH ranked AS (
			SELECT id,card_type,row_number() OVER (PARTITION BY card_type ORDER BY cost,id) AS pos
			FROM card_definitions
			WHERE ruleset_version=$1 AND card_type IN ('Assalto','Guarda','Rito')
			  AND rarity<>'Lendária'
			  AND COALESCE((definition->'confront'->>'legal')::boolean,false)
		), chosen AS (SELECT id FROM ranked WHERE pos<=5)
		INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
		SELECT d.id,chosen.id,$1,2 FROM decks d CROSS JOIN chosen
		WHERE d.ruleset_version=$1 AND d.system_provided
		  AND NOT EXISTS (SELECT 1 FROM deck_cards dc WHERE dc.deck_id=d.id)
		ON CONFLICT DO NOTHING`, activeVersion); err != nil {
		return fmt.Errorf("cartas do baralho inicial do Confronto: %w", err)
	}

	const alphaSeasonStart = "2026-08-11T00:00:00Z"
	// Fecha no relógio real (ends_at > starts_at garantido para temporadas já
	// iniciadas); a constante seguia igual ao starts_at e violava o check.
	if _, err := tx.ExecContext(ctx, `UPDATE seasons SET ends_at=now()
		WHERE ends_at IS NULL AND ruleset_version<>$1 AND starts_at<now()`,
		activeVersion); err != nil {
		return err
	}
	// Garante uma temporada aberta sem reabrir uma temporada já encerrada.
	// Isso importa depois de uma virada operada pelo admin: o boot seguinte
	// fecha uma temporada de outro ruleset, mas nunca deve ressuscitar o ID
	// histórico (nem permitir uma segunda concessão de recompensa).
	if _, err := tx.ExecContext(ctx, `INSERT INTO seasons(id,name,ruleset_version,starts_at)
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM seasons WHERE id='00000000-0000-4000-8000-000000000012')
			THEN gen_random_uuid()
			ELSE '00000000-0000-4000-8000-000000000012'::uuid
		END,'Alpha 0.10.2 — Selos Táticos',$1,$2::timestamptz
		WHERE NOT EXISTS (SELECT 1 FROM seasons WHERE ends_at IS NULL)`, activeVersion, alphaSeasonStart); err != nil {
		return err
	}
	var openRuleset string
	if err := tx.QueryRowContext(ctx, `SELECT ruleset_version FROM seasons
		WHERE ends_at IS NULL LIMIT 1`).Scan(&openRuleset); err != nil {
		return fmt.Errorf("temporada Alpha ausente após sincronização: %w", err)
	}
	if openRuleset != activeVersion {
		return fmt.Errorf("temporada aberta usa %s; esperado %s", openRuleset, activeVersion)
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
	if engine.IsConfrontVersion(starterRuleset) {
		rs, err := engine.RulesetByVersion(starterRuleset)
		if err != nil {
			return domain.User{}, err
		}
		avatarIDs := make([]string, 0, len(rs.Champions))
		for id := range rs.Champions {
			avatarIDs = append(avatarIDs, id)
		}
		sort.Strings(avatarIDs)
		if len(avatarIDs) == 0 {
			return domain.User{}, fmt.Errorf("ruleset Confronto sem avatares")
		}
		precon, err := rs.PreconstructedDeck(avatarIDs[0])
		if err != nil {
			return domain.User{}, err
		}
		var deckID string
		if err := tx.QueryRowContext(ctx, `INSERT INTO decks
			(id,user_id,name,champion_id,ruleset_version,active,system_provided)
			VALUES(gen_random_uuid(),$1,'Meu Baralho',$2,$3,true,true) RETURNING id`,
			user.ID, avatarIDs[0], starterRuleset).Scan(&deckID); err != nil {
			return domain.User{}, mapError(err)
		}
		counts := map[string]int{}
		for _, card := range precon {
			counts[card]++
		}
		ids := make([]string, 0, len(counts))
		for id := range counts {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
				VALUES($1,$2,$3,$4)`, deckID, id, starterRuleset, counts[id]); err != nil {
				return domain.User{}, mapError(err)
			}
		}
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
	var locked sql.NullTime
	err := row.Scan(&d.ID, &d.UserID, &d.Name, &d.ChampionID, &d.RulesetVersion,
		&d.Version, &d.Active, &locked, &d.SystemProvided, &d.CreatedAt, &d.UpdatedAt)
	if locked.Valid {
		d.LockedUntil = &locked.Time
	}
	return d, mapError(err)
}

func (p *Postgres) ListDecks(ctx context.Context, userID string) ([]domain.Deck, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,user_id,name,champion_id,ruleset_version,version,active,locked_until,system_provided,created_at,updated_at
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
	d, err := scanDeck(p.db.QueryRowContext(ctx, `SELECT id,user_id,name,champion_id,ruleset_version,version,active,locked_until,system_provided,created_at,updated_at
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
		err = tx.QueryRowContext(ctx, `INSERT INTO decks(id,user_id,name,champion_id,ruleset_version,active,locked_until,system_provided)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING version,created_at,updated_at`, d.ID, d.UserID,
			d.Name, d.ChampionID, d.RulesetVersion, d.Active, d.LockedUntil, d.SystemProvided).
			Scan(&d.Version, &d.CreatedAt, &d.UpdatedAt)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE decks SET name=$1,champion_id=$2,ruleset_version=$3,
			active=$4,locked_until=$5,system_provided=$6,version=version+1,updated_at=now()
			WHERE id=$7 AND user_id=$8 AND version=$9
			  AND (system_provided OR locked_until IS NULL OR locked_until<=now()
			       OR NOT EXISTS (SELECT 1 FROM rulesets r WHERE r.version=decks.ruleset_version AND r.mode='confront'))
			RETURNING version,created_at,updated_at`, d.Name, d.ChampionID, d.RulesetVersion,
			d.Active, d.LockedUntil, d.SystemProvided, d.ID, d.UserID, *expected).
			Scan(&d.Version, &d.CreatedAt, &d.UpdatedAt)
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
			switch pgErr.ConstraintName {
			case "users_email_key":
				return fmt.Errorf("%w: e-mail já está em uso", domain.ErrConflict)
			case "player_profiles_username_ci_unique":
				return fmt.Errorf("%w: nome de usuário já está em uso", domain.ErrConflict)
			}
			return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
		case "23503", "23514", "22P02":
			return fmt.Errorf("%w: %s", domain.ErrConflict, strings.TrimSpace(pgErr.Message))
		}
	}
	return err
}
