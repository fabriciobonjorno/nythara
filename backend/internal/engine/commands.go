package engine

import "fmt"

// CommandKind enumera as intenções aceitas pela engine. O cliente só envia
// intenção; a engine é a única autoridade de resultado.
type CommandKind string

const (
	CmdKindMulligan CommandKind = "mulligan"
	CmdKindStance   CommandKind = "stance"
	CmdKindPlay     CommandKind = "play"
	CmdKindChoose   CommandKind = "choose"
	CmdKindPass     CommandKind = "pass"
	CmdKindConcede  CommandKind = "concede"
	CmdKindActivate CommandKind = "activate" // habilidade ativada de permanente
	CmdKindUltimate CommandKind = "ultimate" // habilidade única do Campeão
)

// Command é uma intenção de jogador.
type Command struct {
	Player     int         `json:"player"`
	Kind       CommandKind `json:"kind"`
	Card       string      `json:"card,omitempty"`  // play: instância a jogar
	Cards      []string    `json:"cards,omitempty"` // mulligan: devolvidas; choose: escolhas
	Stance     Stance      `json:"stance,omitempty"`
	DecisionID int         `json:"decision_id,omitempty"`
	Reason     string      `json:"reason,omitempty"` // somente servidor: timeout
}

// Erros de validação de comando (a engine nunca aplica comando ilegal).
type CommandError struct {
	Code string
	Msg  string
}

func (e *CommandError) Error() string { return e.Code + ": " + e.Msg }

func errCmd(code, format string, args ...any) error {
	return &CommandError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

const (
	ErrMatchOver      = "match_over"
	ErrWrongPlayer    = "wrong_player"
	ErrWrongPhase     = "wrong_phase"
	ErrPendingChoice  = "pending_choice"
	ErrInvalidCard    = "invalid_card"
	ErrNotImplemented = "card_not_implemented"
	ErrCantAfford     = "cant_afford"
	ErrIllegalTarget  = "illegal_target"
	ErrZoneFull       = "zone_full"
	ErrBadCommand     = "bad_command"
	ErrRequirement    = "requirement_not_met"
)
