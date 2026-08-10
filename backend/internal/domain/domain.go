package domain

import (
	"context"
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
}
