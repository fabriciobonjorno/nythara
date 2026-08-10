package battle

import "veurubro/backend/internal/engine"

type PlayerView struct {
	Champion        string          `json:"champion"`
	Vitality        int             `json:"vitality"`
	MaxVitality     int             `json:"max_vitality"`
	Essence         int             `json:"essence"`
	EssenceCap      int             `json:"essence_cap"`
	TempEssence     int             `json:"temp_essence"`
	DeckCount       int             `json:"deck_count"`
	HandCount       int             `json:"hand_count"`
	Hand            []string        `json:"hand,omitempty"`
	Discard         []string        `json:"discard"`
	Exile           []string        `json:"exile"`
	Relics          []string        `json:"relics"`
	Manifs          []string        `json:"manifs"`
	Stance          engine.Stance   `json:"stance,omitempty"`
	StanceCommitted bool            `json:"stance_committed"`
	MulliganDone    bool            `json:"mulligan_done"`
	Fatigue         int             `json:"fatigue"`
	Trail           []engine.Sigil  `json:"trail"`
	Ward            int             `json:"ward"`
	Exposto         bool            `json:"exposto"`
	Bleeds          []engine.TimedN `json:"bleeds,omitempty"`
	Curses          []engine.TimedN `json:"curses,omitempty"`
	UltimateUsed    bool            `json:"ultimate_used"`
}

type CardView struct {
	ID    string      `json:"id"`
	Def   string      `json:"def"`
	Owner int         `json:"owner"`
	Zone  engine.Zone `json:"zone"`
}

type DecisionView struct {
	ID      int                 `json:"id"`
	Player  int                 `json:"player"`
	Kind    engine.DecisionKind `json:"kind"`
	Options []string            `json:"options,omitempty"`
	N       int                 `json:"n"`
	MinN    int                 `json:"min_n,omitempty"`
	Card    string              `json:"card,omitempty"`
	Source  string              `json:"source,omitempty"`
}

type StateView struct {
	RulesetVersion string              `json:"ruleset_version"`
	Round          int                 `json:"round"`
	Phase          engine.Phase        `json:"phase"`
	Initiative     int                 `json:"initiative"`
	Active         int                 `json:"active"`
	Eclipse        int                 `json:"eclipse"`
	EclipseState   engine.EclipseTotal `json:"eclipse_state,omitempty"`
	Players        [2]PlayerView       `json:"players"`
	Cards          map[string]CardView `json:"cards"`
	Pending        *DecisionView       `json:"pending,omitempty"`
	Guard          *engine.GuardCtx    `json:"guard,omitempty"`
	RiteReact      *engine.RiteReact   `json:"rite_react,omitempty"`
	Extra          *engine.ExtraWindow `json:"extra,omitempty"`
	Over           bool                `json:"over"`
	Winner         int                 `json:"winner"`
	EndReason      string              `json:"end_reason,omitempty"`
}

func ViewFor(state *engine.GameState, viewer int, spectator bool) StateView {
	view := StateView{RulesetVersion: state.RulesetVersion, Round: state.Round, Phase: state.Phase,
		Initiative: state.Initiative, Active: state.Active, Eclipse: state.Eclipse,
		EclipseState: state.EclipseState, Cards: map[string]CardView{}, Over: state.Over, Winner: state.Winner,
		EndReason: state.EndReason}
	if state.Guard != nil {
		guard := *state.Guard
		view.Guard = &guard
	}
	if state.RiteReact != nil {
		reaction := *state.RiteReact
		view.RiteReact = &reaction
	}
	if state.Extra != nil {
		extra := *state.Extra
		view.Extra = &extra
	}
	for slot, player := range state.Players {
		pv := PlayerView{Champion: player.Champion, Vitality: player.Vitality, MaxVitality: player.MaxVitality,
			Essence: player.Essence, EssenceCap: player.EssenceCap, TempEssence: player.TempEssence,
			DeckCount: len(player.Deck), HandCount: len(player.Hand), Discard: append([]string{}, player.Discard...),
			Exile: append([]string{}, player.Exile...), Relics: append([]string{}, player.Relics...),
			Manifs: append([]string{}, player.Manifs...), Stance: player.Stance,
			StanceCommitted: player.StanceCommitted, MulliganDone: player.MulliganDone, Fatigue: player.Fatigue,
			Trail: append([]engine.Sigil{}, player.Trail...), Ward: player.Ward, Exposto: player.Exposto,
			Bleeds: append([]engine.TimedN{}, player.Bleeds...), Curses: append([]engine.TimedN{}, player.Curses...),
			UltimateUsed: player.UltimateUsed}
		if !spectator && slot == viewer {
			pv.Hand = append([]string{}, player.Hand...)
		}
		if state.Phase == engine.PhaseStance && !(state.Players[0].StanceCommitted && state.Players[1].StanceCommitted) &&
			(spectator || slot != viewer) {
			pv.Stance = ""
		}
		view.Players[slot] = pv
	}
	for id, card := range state.Cards {
		public := card.Zone == engine.ZoneDiscard || card.Zone == engine.ZoneExile || card.Zone == engine.ZonePlay
		ownedPrivate := !spectator && card.Owner == viewer && card.Zone == engine.ZoneHand
		if public || ownedPrivate {
			view.Cards[id] = CardView{ID: id, Def: card.Def, Owner: card.Owner, Zone: card.Zone}
		}
	}
	if decision := state.Pending; decision != nil {
		view.Pending = &DecisionView{ID: decision.ID, Player: decision.Player, Kind: decision.Kind,
			N: decision.N, MinN: decision.MinN, Source: decision.Source}
		if !spectator && decision.Player == viewer {
			view.Pending.Options = append([]string{}, decision.Options...)
			view.Pending.Card = decision.Card
			for _, id := range decision.Options {
				if card := state.Cards[id]; card != nil {
					view.Cards[id] = CardView{ID: id, Def: card.Def, Owner: card.Owner, Zone: card.Zone}
				}
			}
		}
	}
	return view
}

func RedactEvents(events []engine.Event, viewer int, spectator bool) []engine.Event {
	result := make([]engine.Event, len(events))
	for i, event := range events {
		result[i] = event
		private := event.Kind == engine.EvCardDrawn || event.Kind == engine.EvHandRevealed ||
			event.Kind == engine.EvDeckTopRevealed
		if private && (spectator || event.P != viewer) {
			result[i].Card = ""
			result[i].Def = ""
		}
		if spectator && event.Kind == engine.EvCardLocked {
			result[i].Card = ""
			result[i].Def = ""
		}
	}
	return result
}
