package engine

// Event é um evento de domínio, achatado para serialização estável (replay,
// golden tests, telemetria). O servidor redige campos ocultos por espectador
// em camada superior; a engine registra a verdade completa.
type Event struct {
	Seq   int    `json:"seq"`
	Round int    `json:"round"`
	Kind  string `json:"kind"`
	P     int    `json:"p"` // jogador afetado/agente; -1 quando não se aplica
	Card  string `json:"card,omitempty"`
	Def   string `json:"def,omitempty"`
	N     int    `json:"n"`
	From  int    `json:"from"`
	To    int    `json:"to"`
	S     string `json:"s,omitempty"`
}

// Kinds de evento.
const (
	EvMatchStarted       = "match_started"
	EvMulliganDone       = "mulligan_done"
	EvRoundStarted       = "round_started"
	EvEssenceRecharged   = "essence_recharged"
	EvTempEssence        = "temp_essence_gained"
	EvCardDrawn          = "card_drawn"
	EvDeckReshuffled     = "deck_reshuffled"
	EvFatigue            = "fatigue"
	EvStanceCommitted    = "stance_committed"
	EvStancesRevealed    = "stances_revealed"
	EvWindowOpened       = "window_opened"
	EvCardPlayed         = "card_played"
	EvEssenceSpent       = "essence_spent"
	EvSacrifice          = "vitality_sacrificed"
	EvDamage             = "damage_dealt"
	EvPrevented          = "damage_prevented"
	EvWardGained         = "ward_gained"
	EvWardConsumed       = "ward_consumed"
	EvHealed             = "healed"
	EvStatusApplied      = "status_applied"
	EvStatusExpired      = "status_expired"
	EvBleedTriggered     = "bleed_triggered"
	EvCurseTriggered     = "curse_triggered"
	EvSigilAdded         = "sigil_added"
	EvEclipseShifted     = "eclipse_shifted"
	EvEclipseTriggered   = "eclipse_triggered"
	EvEclipseReset       = "eclipse_reset"
	EvCardDiscarded      = "card_discarded"
	EvCardExiled         = "card_exiled"
	EvCardToHand         = "card_to_hand"
	EvCardToBottom       = "card_to_bottom"
	EvPermanentEntered   = "permanent_entered"
	EvDecisionRequested  = "decision_requested"
	EvDecisionResolved   = "decision_resolved"
	EvCostModAdded       = "cost_mod_added"
	EvPass               = "pass"
	EvTwilight           = "twilight"
	EvMatchEnded         = "match_ended"
	EvChainTruncated     = "chain_truncated"
	EvStatusFizzled      = "status_fizzled"
	EvDeckReordered      = "deck_reordered"
	EvPermanentDestroyed = "permanent_destroyed"
	EvActivated          = "activated"
	EvUltimate           = "ultimate"
	EvHandRevealed       = "hand_revealed"
	EvDeckTopRevealed    = "deck_top_revealed"
	EvCardLocked         = "card_locked"
	EvRiteReactWindow    = "rite_react_window"
	EvRiteCountered      = "rite_countered"
	EvRiteFizzled        = "rite_fizzled"
	EvCopyResolved       = "copy_resolved"
	EvShiftNullified     = "shift_nullified"
	EvMarkerGained       = "marker_gained"
	EvCardReturned       = "card_returned"
	EvExtraWindow        = "extra_window"

	// Modo Confronto (alpha-0.9.x). Estes eventos são deliberadamente
	// semânticos para o cliente animar voo, resposta, impacto e estilhaço sem
	// tentar deduzir regras a partir de mudanças de estado.
	EvTurnStarted           = "turn_started"
	EvVitalitySpent         = "vitality_spent"
	EvCardBurned            = "card_burned"
	EvConfrontationOpened   = "confrontation_opened"
	EvGuardCommitted        = "guard_committed"
	EvConfrontationResolved = "confrontation_resolved"
	EvCardShattered         = "card_shattered"
)

func (g *Game) emit(e Event) {
	e.Seq = len(g.Log)
	e.Round = g.s.Round
	g.Log = append(g.Log, e)
}
