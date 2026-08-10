package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("não encontrado")
	ErrConflict            = errors.New("conflito")
	ErrUnauthorized        = errors.New("não autenticado")
	ErrForbidden           = errors.New("não autorizado")
	ErrInvalidCredentials  = errors.New("credenciais inválidas")
	ErrInvalid             = errors.New("dados inválidos")
	ErrIdempotencyConflict = errors.New("chave de idempotência reutilizada com conteúdo diferente")
)

// BotUserID é a conta reservada do bot de treino (semeada na migração 4;
// jamais autentica).
const BotUserID = "00000000-0000-4000-8000-0000000000b0"

type Role string

const (
	RolePlayer Role = "player"
	RoleAdmin  Role = "admin"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Principal struct {
	UserID      string `json:"user_id"`
	Role        Role   `json:"role"`
	DisplayName string `json:"display_name"`
}

type SessionTokens struct {
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	AccessExpiry  time.Time `json:"access_expires_at"`
	RefreshExpiry time.Time `json:"refresh_expires_at"`
}

type CollectionCard struct {
	CardID   string `json:"card_id"`
	Quantity int    `json:"quantity"`
}

type Collection struct {
	RulesetVersion string           `json:"ruleset_version"`
	Cards          []CollectionCard `json:"cards"`
	Champions      []string         `json:"champions"`
}

type DeckCard struct {
	CardID   string `json:"card_id"`
	Quantity int    `json:"quantity"`
}

type Deck struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Name           string     `json:"name"`
	ChampionID     string     `json:"champion_id"`
	RulesetVersion string     `json:"ruleset_version"`
	Cards          []DeckCard `json:"cards"`
	Version        int64      `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Season struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	RulesetVersion string     `json:"ruleset_version"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
}

type Reward struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Source         string    `json:"source"`
	RulesetVersion string    `json:"ruleset_version"`
	CardID         string    `json:"card_id,omitempty"`
	ChampionID     string    `json:"champion_id,omitempty"`
	Quantity       int       `json:"quantity"`
	CreatedAt      time.Time `json:"created_at"`
}

type TokenRecord struct {
	Principal Principal
	ExpiresAt time.Time
}

type NewSession struct {
	ID           string
	UserID       string
	AccessHash   []byte
	RefreshHash  []byte
	AccessUntil  time.Time
	RefreshUntil time.Time
}

type RotatedSession struct {
	AccessHash   []byte
	RefreshHash  []byte
	AccessUntil  time.Time
	RefreshUntil time.Time
}

type Mutation struct {
	Key         string
	Operation   string
	RequestHash []byte
	ScopeUserID string
}

// --- Admin/LiveOps (Fase 7) ---

type RulesetInfo struct {
	Version   string    `json:"version"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// RulesetPayload é o snapshot compilável de uma versão (os três documentos).
type RulesetPayload struct {
	Version   string          `json:"version"`
	Cards     json.RawMessage `json:"cards"`
	Champions json.RawMessage `json:"champions"`
	Effects   json.RawMessage `json:"effects"`
}

type DraftStatus string

const (
	DraftOpen      DraftStatus = "draft"
	DraftValidated DraftStatus = "validated"
	DraftPublished DraftStatus = "published"
	DraftDiscarded DraftStatus = "discarded"
)

// CardDraft é uma proposta de carta (nova ou alteração) sobre uma versão base.
type CardDraft struct {
	ID               string          `json:"id"`
	CardID           string          `json:"card_id"`
	BaseVersion      string          `json:"base_version"`
	Status           DraftStatus     `json:"status"`
	Note             string          `json:"note"`
	Card             json.RawMessage `json:"card"`
	Effects          json.RawMessage `json:"effects"`
	LastValidation   json.RawMessage `json:"last_validation,omitempty"`
	PublishedVersion string          `json:"published_version,omitempty"`
	CreatedBy        string          `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CardBan é a desativação emergencial de uma carta no competitivo.
type CardBan struct {
	ID        string     `json:"id"`
	CardID    string     `json:"card_id"`
	Reason    string     `json:"reason"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	LiftedBy  string     `json:"lifted_by,omitempty"`
	LiftedAt  *time.Time `json:"lifted_at,omitempty"`
}

// AuditEntry registra toda mutação administrativa.
type AuditEntry struct {
	ID        int64           `json:"id"`
	Actor     string          `json:"actor"`
	Action    string          `json:"action"`
	Subject   string          `json:"subject"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// ChampionMatchStats agrega desempenho de um Campeão em partidas reais.
type ChampionMatchStats struct {
	ChampionID string  `json:"champion_id"`
	Games      int     `json:"games"`
	Wins       int     `json:"wins"`
	WinRate    float64 `json:"win_rate"`
}

// MatchTelemetry agrega as partidas persistidas pelo battle server.
type MatchTelemetry struct {
	TotalMatches    int                  `json:"total_matches"`
	FinishedMatches int                  `json:"finished_matches"`
	ByChampion      []ChampionMatchStats `json:"by_champion"`
}

type Store interface {
	CreateUser(ctx context.Context, user User, starterRuleset string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	UserByID(ctx context.Context, id string) (User, error)

	CreateSession(ctx context.Context, session NewSession) error
	AccessToken(ctx context.Context, hash []byte, now time.Time) (TokenRecord, error)
	RotateSession(ctx context.Context, oldRefreshHash []byte, next RotatedSession, now time.Time) (string, error)
	RevokeSession(ctx context.Context, refreshHash []byte) error

	Collection(ctx context.Context, userID, ruleset string) (Collection, error)
	ListDecks(ctx context.Context, userID string) ([]Deck, error)
	Deck(ctx context.Context, userID, deckID string) (Deck, error)
	SaveDeck(ctx context.Context, deck Deck, expectedVersion *int64, mutation Mutation) (Deck, bool, error)
	DeleteDeck(ctx context.Context, userID, deckID string, expectedVersion int64, mutation Mutation) (bool, error)

	ActiveSeason(ctx context.Context) (Season, error)
	ListRewards(ctx context.Context, userID string) ([]Reward, error)
	GrantReward(ctx context.Context, reward Reward, mutation Mutation) (Reward, bool, error)

	// Admin/LiveOps (Fase 7). Toda mutação recebe também uma entrada de
	// auditoria gravada na mesma transação.
	ListRulesets(ctx context.Context) ([]RulesetInfo, error)
	RulesetPayload(ctx context.Context, version string) (RulesetPayload, error)
	ListRulesetPayloads(ctx context.Context) ([]RulesetPayload, error)
	PublishRuleset(ctx context.Context, payload RulesetPayload, draftID string, audit AuditEntry) error
	ActivateRuleset(ctx context.Context, version string, audit AuditEntry) error

	CreateDraft(ctx context.Context, draft CardDraft, audit AuditEntry) (CardDraft, error)
	UpdateDraft(ctx context.Context, draft CardDraft, audit AuditEntry) (CardDraft, error)
	Draft(ctx context.Context, id string) (CardDraft, error)
	ListDrafts(ctx context.Context, status DraftStatus) ([]CardDraft, error)

	CreateBan(ctx context.Context, ban CardBan, audit AuditEntry) (CardBan, error)
	LiftBan(ctx context.Context, cardID, liftedBy string, audit AuditEntry) (CardBan, error)
	ActiveBans(ctx context.Context) ([]CardBan, error)

	CreateSeason(ctx context.Context, season Season, audit AuditEntry) (Season, error)
	// RotateToRuleset concede a coleção da versão e clona decks válidos da
	// ativa (granted, decksCloned).
	RotateToRuleset(ctx context.Context, version string, audit AuditEntry) (int, int, error)

	MatchTelemetry(ctx context.Context) (MatchTelemetry, error)
	ListAudit(ctx context.Context, limit int) ([]AuditEntry, error)
}
