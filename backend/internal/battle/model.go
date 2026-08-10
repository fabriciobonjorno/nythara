package battle

import (
	"context"
	"errors"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

var (
	ErrNotParticipant = errors.New("usuário não participa da partida")
	ErrNotReady       = errors.New("partida ainda não está pronta")
	ErrSequenceGap    = errors.New("client_sequence fora de ordem")
	ErrSequenceReuse  = errors.New("client_sequence reutilizado com outro payload")
	ErrSpectatorWrite = errors.New("espectador é somente leitura")
	ErrTicketInvalid  = errors.New("ticket inválido ou expirado")
	ErrAlreadyQueued  = errors.New("jogador já está na fila")
	ErrInvalidIntent  = errors.New("intenção inválida")
)

type Status string

const (
	StatusWaitingReady Status = "waiting_ready"
	StatusActive       Status = "active"
	StatusFinished     Status = "finished"
	StatusCancelled    Status = "cancelled"
)

type Participant struct {
	UserID       string `json:"user_id"`
	DeckID       string `json:"deck_id"`
	Slot         int    `json:"slot"`
	Ready        bool   `json:"ready"`
	LastSequence int64  `json:"last_client_sequence"`
}

type Match struct {
	ID        string         `json:"id"`
	Config    engine.Config  `json:"config"`
	Players   [2]Participant `json:"players"`
	Status    Status         `json:"status"`
	Winner    *int           `json:"winner,omitempty"`
	EndReason string         `json:"end_reason,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	StartedAt *time.Time     `json:"started_at,omitempty"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
}

type StoredCommand struct {
	Index          int64
	PlayerSlot     int
	ClientSequence *int64
	Origin         string
	Command        engine.Command
	FirstEventSeq  int
	LastEventSeq   int
}

type Snapshot struct {
	CommandIndex int64
	EventSeq     int
	Data         []byte
}

type LoadedMatch struct {
	Match    Match
	Commands []StoredCommand
	Events   []engine.Event
	Snapshot *Snapshot
}

type Step struct {
	MatchID        string
	CommandIndex   int64
	PlayerSlot     int
	ClientSequence *int64
	Origin         string
	Command        engine.Command
	Events         []engine.Event
	Snapshot       *Snapshot
	Finished       bool
	Winner         *int
	EndReason      string
}

type Store interface {
	CreateMatch(ctx context.Context, match Match) error
	MarkReady(ctx context.Context, matchID string, slot int) error
	StartMatch(ctx context.Context, matchID string, events []engine.Event, snapshot Snapshot) error
	PersistStep(ctx context.Context, step Step) error
	CancelMatch(ctx context.Context, matchID, reason string) error
	LoadMatch(ctx context.Context, matchID string) (LoadedMatch, error)
	ActiveMatchForUser(ctx context.Context, userID string) (Match, int, error)
}

type QueueResult struct {
	Status  string `json:"status"`
	MatchID string `json:"match_id,omitempty"`
	Slot    *int   `json:"slot,omitempty"`
}

type Intent struct {
	Kind       engine.CommandKind `json:"kind"`
	Card       string             `json:"card,omitempty"`
	Cards      []string           `json:"cards,omitempty"`
	Stance     engine.Stance      `json:"stance,omitempty"`
	DecisionID int                `json:"decision_id,omitempty"`
}

func (i Intent) Command(player int) engine.Command {
	return engine.Command{Player: player, Kind: i.Kind, Card: i.Card,
		Cards: append([]string{}, i.Cards...), Stance: i.Stance, DecisionID: i.DecisionID}
}

func (i Intent) Validate() error {
	if len(i.Card) > 64 || len(i.Cards) > engine.DeckSize {
		return ErrInvalidIntent
	}
	for _, card := range i.Cards {
		if len(card) > 64 {
			return ErrInvalidIntent
		}
	}
	return nil
}

type TicketMode string

const (
	TicketPlayer    TicketMode = "player"
	TicketSpectator TicketMode = "spectator"
)

type Ticket struct {
	Token     string     `json:"ticket"`
	MatchID   string     `json:"match_id"`
	UserID    string     `json:"-"`
	Mode      TicketMode `json:"mode"`
	Slot      int        `json:"slot,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
}

type CommandResult struct {
	ClientSequence int64          `json:"client_sequence"`
	Events         []engine.Event `json:"events"`
	Duplicate      bool           `json:"duplicate,omitempty"`
}

type ServerMessage struct {
	Type           string         `json:"type"`
	MatchID        string         `json:"match_id,omitempty"`
	Status         Status         `json:"status,omitempty"`
	Slot           *int           `json:"slot,omitempty"`
	Ready          [2]bool        `json:"ready,omitempty"`
	State          *StateView     `json:"state,omitempty"`
	Events         []engine.Event `json:"events,omitempty"`
	ClientSequence int64          `json:"client_sequence,omitempty"`
	Duplicate      bool           `json:"duplicate,omitempty"`
	Deadline       *time.Time     `json:"deadline,omitempty"`
	Error          *ProtocolError `json:"error,omitempty"`
}

type ProtocolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type QueuedPlayer struct {
	Principal domain.Principal
	Deck      domain.Deck
}
