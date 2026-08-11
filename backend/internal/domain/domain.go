package domain

import (
	"context"
	"encoding/json"
	"errors"
	"math"
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
	Active         bool       `json:"active"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
	SystemProvided bool       `json:"system_provided,omitempty"`
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

// --- Progressão (P1) ---

// EloDelta calcula a variação de rating (K=32) de A contra B; nunca zero.
func EloDelta(ratingA, ratingB int, wonA bool) int {
	expected := 1.0 / (1.0 + math.Pow(10, (float64(ratingB)-float64(ratingA))/400.0))
	score := 0.0
	if wonA {
		score = 1.0
	}
	delta := int(math.Round(32.0 * (score - expected)))
	if delta == 0 {
		if wonA {
			return 1
		}
		return -1
	}
	return delta
}

// RitualIncrement é o progresso de um ritual creditado por uma partida.
type RitualIncrement struct {
	RitualID string `json:"ritual_id"`
	Delta    int    `json:"delta"`
	Target   int    `json:"target"`
	Reward   int    `json:"reward"`
}

// PlayerMatchProgress é a fatia de progresso de um jogador numa partida.
type PlayerMatchProgress struct {
	UserID     string            `json:"user_id"`
	ChampionID string            `json:"champion_id"`
	Won        bool              `json:"won"`
	MasteryXP  int               `json:"mastery_xp"`
	RitualDay  string            `json:"ritual_day"`
	Rituals    []RitualIncrement `json:"rituals"`
}

// MatchProgress é a gravação idempotente de progresso de uma partida.
type MatchProgress struct {
	MatchID  string                `json:"match_id"`
	Ranked   bool                  `json:"ranked"` // PvP atualiza rating
	SeasonID string                `json:"season_id,omitempty"`
	Players  []PlayerMatchProgress `json:"players"`
}

// RitualState é um ritual do dia com progresso corrente.
type RitualState struct {
	RitualID    string     `json:"ritual_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Progress    int        `json:"progress"`
	Target      int        `json:"target"`
	Reward      int        `json:"reward"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ChampionMastery é a maestria acumulada com um Campeão.
type ChampionMastery struct {
	ChampionID string `json:"champion_id"`
	XP         int    `json:"xp"`
	Level      int    `json:"level"`
	Title      string `json:"title,omitempty"` // derivado de Level (ranks.go)
	Games      int    `json:"games"`
	Wins       int    `json:"wins"`
}

// RankedStanding é o rating do jogador na temporada.
type RankedStanding struct {
	SeasonID string    `json:"season_id"`
	Rating   int       `json:"rating"`
	Games    int       `json:"games"`
	Wins     int       `json:"wins"`
	Position int       `json:"position,omitempty"`
	Tier     *RankTier `json:"tier,omitempty"` // derivado de Rating (ranks.go)
}

// LeaderboardEntry é uma linha do ranking da temporada.
type LeaderboardEntry struct {
	Position    int    `json:"position"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Rating      int    `json:"rating"`
	Games       int    `json:"games"`
	Wins        int    `json:"wins"`
	Tier        string `json:"tier,omitempty"` // nome da patente (ranks.go)
}

// MatchSummary é uma linha do histórico pessoal de partidas.
type MatchSummary struct {
	MatchID          string     `json:"match_id"`
	Mode             string     `json:"mode"`
	RulesetVersion   string     `json:"ruleset_version"`
	MySlot           int        `json:"my_slot"`
	MyChampion       string     `json:"my_champion"`
	Opponent         string     `json:"opponent"`
	OpponentChampion string     `json:"opponent_champion"`
	Won              bool       `json:"won"`
	Draw             bool       `json:"draw,omitempty"`
	EndReason        string     `json:"end_reason,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

// ReplayPlayer identifica um assento na crônica de uma partida encerrada.
type ReplayPlayer struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	ChampionID  string `json:"champion_id"`
}

// MatchReplayData é a crônica pós-partida: cabeçalho + log de eventos
// authoritative (a engine é determinística; o log É a partida).
type MatchReplayData struct {
	MatchID        string            `json:"match_id"`
	Mode           string            `json:"mode"`
	RulesetVersion string            `json:"ruleset_version"`
	Status         string            `json:"status"`
	Players        [2]ReplayPlayer   `json:"players"`
	Winner         *int              `json:"winner,omitempty"`
	EndReason      string            `json:"end_reason,omitempty"`
	Events         []json.RawMessage `json:"events"`
	// StartingVitality é a base do ruleset em que a partida foi jogada. Sem
	// ela o cliente teria de adivinhar, e a Vitalidade do Campeão é um valor
	// legado que o Modo Confronto sobrescreve: replay antigo e replay novo
	// desenham escalas diferentes.
	StartingVitality int `json:"starting_vitality,omitempty"`
}

// ProgressSummary alimenta a Home: rituais do dia, carteira, maestria, rating.
type ProgressSummary struct {
	Day       string            `json:"day"`
	Rituals   []RitualState     `json:"rituals"`
	Fragments int               `json:"fragments"`
	Mastery   []ChampionMastery `json:"mastery"`
	Ranked    *RankedStanding   `json:"ranked,omitempty"`
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

	// Progressão (P1). RecordMatchProgress é idempotente por partida (retorna
	// false quando a partida já havia sido creditada).
	RecordMatchProgress(ctx context.Context, progress MatchProgress) (bool, error)
	RitualsFor(ctx context.Context, userID, day string) ([]RitualState, error)
	SaveRitualStates(ctx context.Context, userID, day string, states []RitualState) error
	Fragments(ctx context.Context, userID string) (int, error)
	MasteryFor(ctx context.Context, userID string) ([]ChampionMastery, error)
	RankedStandingFor(ctx context.Context, userID, seasonID string) (RankedStanding, error)
	Leaderboard(ctx context.Context, seasonID string, limit int) ([]LeaderboardEntry, error)

	// Arena: histórico pessoal e crônica de partidas encerradas.
	MatchHistory(ctx context.Context, userID string, limit int) ([]MatchSummary, error)
	MatchReplay(ctx context.Context, matchID string) (MatchReplayData, error)

	// Recado do Alpha: opcional, um por partida.
	SaveFeedback(ctx context.Context, entry Feedback) error
}

// Feedback é o recado que o jogador decide deixar depois de um duelo. Nada no
// produto depende dele: é convite, não requisito.
type Feedback struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	MatchID        string    `json:"match_id,omitempty"`
	RulesetVersion string    `json:"ruleset_version"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// FeedbackMaxLength mantém o recado dentro do que a coluna aceita e evita que
// um envio acidental de arquivo vire uma linha gigante no banco.
const FeedbackMaxLength = 2000
