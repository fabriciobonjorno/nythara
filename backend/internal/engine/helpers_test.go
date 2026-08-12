package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// deckWith preenche um deck de teste: prefixo na ordem dada (topo primeiro)
// e completa com VR-006 (não suportada na DSL — nunca jogável por engano).
func deckWith(prefix ...string) []string {
	d := append([]string{}, prefix...)
	for len(d) < engine.DeckSize {
		d = append(d, "VR-006")
	}
	return d
}

// harness encurta scripts de partida nos testes.
type harness struct {
	t *testing.T
	g *engine.Game
}

func newHarness(t *testing.T, champ0, champ1 string, deck0, deck1 []string, first int) *harness {
	t.Helper()
	g, err := engine.NewGame(engine.Config{
		Seed:        42,
		Players:     [2]engine.PlayerSetup{{ChampionID: champ0, Deck: deck0}, {ChampionID: champ1, Deck: deck1}},
		SkipShuffle: true,
		FirstPlayer: first,
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return &harness{t: t, g: g}
}

func (h *harness) must(cmd engine.Command) {
	h.t.Helper()
	if _, err := h.g.Apply(cmd); err != nil {
		h.t.Fatalf("comando %+v rejeitado: %v", cmd, err)
	}
}

func (h *harness) mustFail(cmd engine.Command, code string) {
	h.t.Helper()
	before := len(h.g.Log)
	_, err := h.g.Apply(cmd)
	if err == nil {
		h.t.Fatalf("comando %+v deveria ter sido rejeitado (%s)", cmd, code)
	}
	ce, ok := err.(*engine.CommandError)
	if !ok || ce.Code != code {
		h.t.Fatalf("comando %+v: erro %v; esperado código %s", cmd, err, code)
	}
	if len(h.g.Log) != before {
		h.t.Fatalf("comando rejeitado emitiu eventos")
	}
}

func (h *harness) keepAll() {
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindMulligan})
	h.must(engine.Command{Player: 1, Kind: engine.CmdKindMulligan})
}
func (h *harness) stances(s0, s1 engine.Stance) {
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindStance, Stance: s0})
	h.must(engine.Command{Player: 1, Kind: engine.CmdKindStance, Stance: s1})
}
func (h *harness) pass(p int) { h.must(engine.Command{Player: p, Kind: engine.CmdKindPass}) }
func (h *harness) play(p int, inst string) {
	h.must(engine.Command{Player: p, Kind: engine.CmdKindPlay, Card: inst})
}
func (h *harness) choose(p int, picks ...string) {
	h.t.Helper()
	d := h.g.State().Pending
	if d == nil {
		h.t.Fatalf("não há decisão pendente")
	}
	h.must(engine.Command{Player: p, Kind: engine.CmdKindChoose, DecisionID: d.ID, Cards: picks})
}

// bothPassRite passa a janela de Rito para os dois jogadores, na ordem da
// iniciativa.
func (h *harness) bothPassRite() {
	s := h.g.State()
	h.pass(s.Active)
	h.pass(h.g.State().Active)
}

// handInst devolve a instância na mão do jogador com a definição dada.
func (h *harness) handInst(p int, def string) string {
	h.t.Helper()
	s := h.g.State()
	for _, id := range s.Players[p].Hand {
		if s.Cards[id].Def == def {
			return id
		}
	}
	h.t.Fatalf("jogador %d não tem %s na mão (mão: %v)", p, def, s.Players[p].Hand)
	return ""
}

func (h *harness) vitality(p int) int { return h.g.State().Players[p].Vitality }

func (h *harness) assertVit(p, want int) {
	h.t.Helper()
	if got := h.vitality(p); got != want {
		h.t.Fatalf("vitalidade do jogador %d: %d; esperado %d", p, got, want)
	}
}

func (h *harness) assertEclipse(want int) {
	h.t.Helper()
	if got := h.g.State().Eclipse; got != want {
		h.t.Fatalf("eclipse: %d; esperado %d", got, want)
	}
}
