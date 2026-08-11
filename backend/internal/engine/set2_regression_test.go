package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// Regressão ADR-032: uma cadeia que COMPRA carta depois de abrir uma decisão
// de reordenação de topo (VR-091 em Guarda × retaliação do VR-120) não pode
// corromper zonas. O Apply refresca as opções pendentes; a resolução ignora
// cartas que saíram do baralho.
func TestReorderDecisionSurvivesMidChainDraw(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-090", "VR-115"), deckWith("VR-120", "VR-091"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[1].Essence = 5 // cirurgia: VR-120 custa 4 na rodada 1

	// Rito: p0 deixa um Rito no descarte (condição do VR-115); p1 baixa o
	// Arquivo Ardente.
	h.play(0, h.handInst(0, "VR-090"))
	_ = s
	h.pass(0)
	h.play(1, h.handInst(1, "VR-120"))
	h.pass(1)

	// Confronto: VR-115 ataca; VR-091 defende (previne tudo) e abre a
	// reordenação; o After do Assalto amaldiçoa p1; VR-120 retalia e COMPRA.
	h.play(0, h.handInst(0, "VR-115"))
	h.play(1, h.handInst(1, "VR-091"))

	d := h.g.State().Pending
	if d == nil || d.Kind != engine.DecReorderTop {
		t.Fatalf("esperava reordenação pendente; %+v", d)
	}
	deckSet := map[string]bool{}
	for _, id := range h.g.State().Players[1].Deck {
		deckSet[id] = true
	}
	for _, option := range d.Options {
		if !deckSet[option] {
			t.Fatalf("opção %s não está mais no baralho (refresh falhou)", option)
		}
	}
	if len(d.Options) != d.N {
		t.Fatalf("N=%d difere das opções %d", d.N, len(d.Options))
	}
	// Resolve com a permutação invertida das opções VIVAS.
	picks := make([]string, 0, len(d.Options))
	for i := len(d.Options) - 1; i >= 0; i-- {
		picks = append(picks, d.Options[i])
	}
	h.choose(1, picks...)

	assertNoZoneDuplicates(t, h.g.State())
}

// Regressão encontrada pelo gate variado de 100 mil (jogos 62749/97669):
// Passo Calculado abre uma escolha de descarte, mas um gatilho posterior de
// Matilha à Espreita derruba Kaedor a 0; Recusa da Morte esvazia a mão. A
// escolha já apresentada precisa ser encerrada automaticamente, pois não há
// mais carta legal para selecionar.
func TestPresentedDecisionClosesWhenLaterTriggerRemovesAllOptions(t *testing.T) {
	h := newHarness(t, "CH-VA-01", "CH-VH-02",
		deckWith("VR-108", "VR-105"), deckWith("VR-062"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)

	s := h.g.State()
	s.Players[0].Essence = 6
	guard := h.handInst(1, "VR-062")
	// Deixa Kaedor apenas com a Guarda; ela compra uma carta e abre descarte.
	for _, id := range append([]string{}, s.Players[1].Hand...) {
		if id == guard {
			continue
		}
		s.Cards[id].Zone = engine.ZoneDiscard
		s.Players[1].Discard = append(s.Players[1].Discard, id)
	}
	s.Players[1].Hand = []string{guard}

	// A Manifestação entra no Rito e dispara depois do Assalto de custo 2.
	h.play(0, h.handInst(0, "VR-108"))
	h.pass(0)
	h.pass(1)
	s.Players[1].Vitality = 1
	h.play(0, h.handInst(0, "VR-105"))
	h.play(1, guard)

	if s.Over || s.Players[1].Vitality != 1 || !s.Players[1].UltimateUsed {
		t.Fatalf("Recusa da Morte não resolveu como esperado: over=%v vit=%d ultimate=%v",
			s.Over, s.Players[1].Vitality, s.Players[1].UltimateUsed)
	}
	if s.Pending != nil {
		t.Fatalf("decisão impossível permaneceu aberta: %+v", s.Pending)
	}
	if len(s.Players[1].Hand) != 0 {
		t.Fatalf("Recusa da Morte deveria esvaziar a mão: %v", s.Players[1].Hand)
	}
	if _, ok := engine.RequiredPlayer(h.g); !ok {
		t.Fatal("partida ficou sem ator depois de encerrar a decisão inválida")
	}
}

// assertNoZoneDuplicates confere que cada instância aparece em exatamente uma
// zona (a mesma invariante do verificador do simulador).
func assertNoZoneDuplicates(t *testing.T, s *engine.GameState) {
	t.Helper()
	seen := map[string]string{}
	record := func(where string, ids []string) {
		for _, id := range ids {
			if prev, ok := seen[id]; ok {
				t.Fatalf("carta %s em duas zonas: %s e %s", id, prev, where)
			}
			seen[id] = where
		}
	}
	for player := 0; player < 2; player++ {
		p := s.Players[player]
		record("deck", p.Deck)
		record("mão", p.Hand)
		record("descarte", p.Discard)
		record("exílio", p.Exile)
		record("jogo", p.Relics)
		record("jogo", p.Manifs)
	}
}
