package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// passConfront passa a janela de Confronto dos dois jogadores.
func (h *harness) passConfront() {
	h.pass(h.g.State().Active)
	h.pass(h.g.State().Active)
}

// passRound atravessa uma rodada inteira sem jogar nada.
func (h *harness) passRound() {
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.passConfront()
}

func TestSecondAssaultBonusVR037(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-037", "VR-037"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1) // sem guarda
	h.assertVit(1, 25)

	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.assertVit(1, 21) // 2 + (2+2)
}

func TestRaukSecondAssaultChampionBonus(t *testing.T) {
	h := newHarness(t, "CH-VA-01", "CH-CI-01",
		deckWith("VR-037", "VR-037"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.assertVit(1, 25)
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.assertVit(1, 20) // 2 + (2+2+1 Rauk)
}

func TestMultiHitVsWardVR044(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-044"), deckWith("VR-069"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)

	// p1 tem iniciativa: Sino do Primeiro Quarto cruza o 0 → Ward 1.
	h.play(1, h.handInst(1, "VR-069"))
	if got := h.g.State().Players[1].Ward; got != 1 {
		t.Fatalf("ward: %d; esperado 1", got)
	}
	h.assertEclipse(-1)
	h.pass(1)
	h.pass(0)

	h.pass(1)                          // confronto de p1
	h.g.State().Players[0].Essence = 4 // VR-044 custa 4
	h.play(0, h.handInst(0, "VR-044"))
	h.pass(1)
	// 3 instâncias de 2 (alpha-0.6.0): a 1ª perde 1 no Ward; 5 atravessam.
	h.assertVit(1, 22)
	if got := h.g.State().Players[1].Ward; got != 0 {
		t.Fatalf("ward final: %d; esperado 0", got)
	}
}

func TestPierceWardVR020(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-020"), deckWith("VR-069"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-069")) // Ward 1
	h.pass(1)
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-020"))
	h.pass(1)
	// Ignora 1 de Ward (alpha-0.6.0): o Ward 1 é contornado; 3 de dano cheio.
	h.assertVit(1, 24)
	if got := h.g.State().Players[1].Ward; got != 1 {
		t.Fatalf("ward deveria permanecer 1 (contornado), tem %d", got)
	}
}

func TestBleedTimingVR009(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-009"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-009"))
	h.pass(1)
	h.assertVit(1, 25) // dano imediato 2
	h.pass(0)
	h.pass(1)
	// Crepúsculo da rodada 1: Sangramento ainda não dispara.
	h.assertVit(1, 25)
	h.passRound()
	// Crepúsculo da rodada 2: dispara Sangramento 2.
	h.assertVit(1, 23)
	if got := len(h.g.State().Players[1].Bleeds); got != 0 {
		t.Fatalf("sangramentos restantes: %d; esperado 0", got)
	}
}

func TestCurseVR053(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-053"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-053"))
	h.pass(0)
	h.pass(1)
	h.passConfront()
	h.assertVit(1, 27) // rodada 1: nada
	h.passRound()
	// Crepúsculo da rodada 2: p1 com 7 cartas (≥5) sofre 3 (alpha-0.5.0).
	h.assertVit(1, 24)
}

func TestGuardHealVR003(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-013"), deckWith("VR-003"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.assertVit(1, 25) // eclipse 0: sem bônus → 2
	h.assertEclipse(-1)

	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-003"))
	// Eclipse -1 → 3; preveniu os 3 → cura 1.
	h.assertVit(1, 26)
}

func TestPeleGrossaVR042(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-020", "VR-013", "VR-020"), deckWith("VR-042", "VR-042"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	// Sem dano prévio na rodada: previne 3 de 3 (alpha-0.6.0) → nada passa.
	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-042"))
	h.assertVit(1, 27)
	h.pass(0)
	h.pass(1)

	h.passRound() // rodada 2 inteira vazia

	// Rodada 3 (iniciativa de volta a p0, essência 5).
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.assertVit(1, 25) // 2 de dano; o bônus exige Aurora funda
	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-042"))
	h.assertVit(1, 25) // previne 5 de 3 → nada passa
}

func TestResonanceComboVR005(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-001", "VR-061", "VR-005"), deckWith("VR-013"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.bothPassRite()
	h.pass(0) // p0 não ataca na rodada 1
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	h.assertVit(0, 25) // VR-013 sem bônus a 0
	h.pass(1)

	// Rodada 2: iniciativa de p1; p0 monta Presa→Coroa e fecha com VR-005.
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.bothPassRite()
	h.pass(1)
	h.play(0, h.handInst(0, "VR-001"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-061"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-005"))
	h.pass(1)
	// Trilha própria: Presa, Coroa → bônus: cura 2 (teto 27).
	h.assertVit(0, 27)
	h.assertVit(1, 27-2-2-3)
}

func TestAntiResonanceGuardVR017(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-001", "VR-017", "VR-017"), deckWith("VR-025", "VR-020"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-001")) // Sigilo Presa fica por último na linha global
	h.pass(1)
	h.assertVit(1, 25)
	h.pass(0)

	// VR-025 chega com bônus de Ressonância +2 (alpha-0.6.0): 5 de dano
	// contra prevenção 2+2 do VR-017 → 1 atravessa.
	h.play(1, h.handInst(1, "VR-025"))
	h.play(0, h.handInst(0, "VR-017"))
	h.assertVit(0, 26)
	h.pass(1)

	// Rodada 2: trilhas zeradas; VR-020 (sem bônus) contra VR-017 → previne só 2.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(1, h.handInst(1, "VR-020"))
	h.play(0, h.handInst(0, "VR-017"))
	h.assertVit(0, 25)
}

func TestVeilBlocksTargetedRite(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-015"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	// Um Véu concedido na rodada anterior continua ativo nesta rodada.
	h.g.State().Players[1].VeilRound = h.g.State().Round
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-015")},
		engine.ErrIllegalTarget)
	if got := h.g.State().Players[0].Essence; got != 3 {
		t.Fatalf("essência não deveria ter sido gasta: %d", got)
	}
}

// Com o set completo (130/130), não existe mais carta não implementada; o
// teste vira uma trava de cobertura total + a rejeição de jogadas fora de
// janela (counter fora da reação).
func TestFullCoverageAndCounterOnlyInReaction(t *testing.T) {
	r := engine.ImplementationReport()
	if len(r.MissingCards) != 0 {
		t.Fatalf("cartas sem implementação: %v", r.MissingCards)
	}
	if len(r.MissingChampions) != 0 {
		t.Fatalf("campeões sem implementação: %v", r.MissingChampions)
	}

	// VR-035 não é jogável como Guarda comum numa janela de Guarda.
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013"), deckWith("VR-035"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-013"))
	h.mustFail(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: h.handInst(1, "VR-035")},
		engine.ErrWrongPhase)
}

func TestFatigueOnEmptyDeck(t *testing.T) {
	h := newHarness(t, "CH-VH-01", "CH-CI-01",
		deckWith("VR-002"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)

	// Cirurgia de estado: esvazia o deck de p0 para o descarte.
	s := h.g.State()
	p := s.Players[0]
	for _, id := range p.Deck {
		s.Cards[id].Zone = engine.ZoneDiscard
		p.Discard = append(p.Discard, id)
	}
	p.Deck = nil

	// Dívida de Sangue: sacrifica 2 e compra 2 → 1ª compra: Fadiga 6 +
	// reembaralha o descarte; 2ª compra normal.
	h.play(0, h.handInst(0, "VR-002"))
	// Seris: 27 - 2 (sacrifício) - 6 (fadiga alpha-0.8.0) = 19.
	h.assertVit(0, 19)
	if p.Fatigue != 6 {
		t.Fatalf("fadiga: %d; esperado 6", p.Fatigue)
	}
	if len(p.Deck) == 0 {
		t.Fatal("deck deveria ter sido reembaralhado do descarte")
	}
}

func TestSigilEnginesVR040AndVR043(t *testing.T) {
	// VR-040: o 2º Sigilo Garra da rodada gera um extra.
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-040", "VR-037", "VR-037"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-040"))
	h.pass(0)
	h.pass(1)
	h.passConfront()

	// Rodada 2 (essência 4): dois Assaltos Garra disparam o Totem.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.pass(1) // iniciativa de p1 no Confronto
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	trail := h.g.State().Players[0].Trail
	if len(trail) != 1 {
		t.Fatalf("trilha após 1º Garra: %v", trail)
	}
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	trail = h.g.State().Players[0].Trail
	if len(trail) != 3 { // Garra, Garra, Garra extra do Totem
		t.Fatalf("trilha após 2º Garra: %v; esperado 3 sigilos", trail)
	}

	// VR-043: Assalto de custo 1 deixa Sigilo Garra extra (1x por rodada).
	h2 := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-043", "VR-013"), deckWith(), 0)
	h2.keepAll()
	h2.stances(engine.StanceArcano, engine.StanceArcano)
	h2.play(0, h2.handInst(0, "VR-043"))
	h2.pass(0)
	h2.pass(1)
	h2.play(0, h2.handInst(0, "VR-013")) // custo 1, sigilo Sol
	h2.pass(1)
	// A própria Manifestação emitiu Garra ao entrar; o gatilho adiciona outro.
	trail = h2.g.State().Players[0].Trail
	if len(trail) != 3 || trail[1] != engine.SigilSol || trail[2] != engine.SigilGarra {
		t.Fatalf("trilha: %v; esperado [Garra Sol Garra]", trail)
	}
}

func TestResonanceCapSuppressesSigils(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	p := h.g.State().Players[0]
	p.Trail = []engine.Sigil{engine.SigilSol, engine.SigilSol, engine.SigilSol, engine.SigilSol, engine.SigilSol}
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	if got := len(p.Trail); got != 5 {
		t.Fatalf("trilha estourou o limite: %d sigilos", got)
	}
}

func TestPickTop2VR026(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-026"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)

	s := h.g.State()
	p := s.Players[0]
	top0, top1 := p.Deck[0], p.Deck[1]
	deckBefore := len(p.Deck)

	h.play(0, h.handInst(0, "VR-026"))
	d := s.Pending
	if d == nil || d.Kind != engine.DecPickTop2 {
		t.Fatalf("decisão pendente inesperada: %+v", d)
	}
	h.choose(0, top1)
	if s.Cards[top1].Zone != engine.ZoneHand {
		t.Fatalf("escolhida deveria estar na mão")
	}
	if s.Cards[top0].Zone != engine.ZoneDeck || p.Deck[len(p.Deck)-1] != top0 {
		t.Fatalf("preterida deveria estar no fundo do deck")
	}
	if len(p.Deck) != deckBefore-1 {
		t.Fatalf("deck: %d; esperado %d", len(p.Deck), deckBefore-1)
	}
}

func TestRiteDecisionFinishesBeforeSigilTriggers(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-028", "VR-026"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-028"))

	s := h.g.State()
	p := s.Players[0]
	// O próximo Espelho será o terceiro Sigilo distinto e acionará VR-028.
	p.Trail = []engine.Sigil{engine.SigilPresa, engine.SigilSol}
	top0, top1 := p.Deck[0], p.Deck[1]
	h.play(0, h.handInst(0, "VR-026"))

	if s.Pending == nil || s.Pending.Kind != engine.DecPickTop2 || len(s.PendingRites) != 1 {
		t.Fatalf("Rito deveria aguardar pick_top2 antes de finalizar: pending=%+v rites=%+v", s.Pending, s.PendingRites)
	}
	if s.Cards[top0].Zone != engine.ZoneDeck || s.Cards[top1].Zone != engine.ZoneDeck {
		t.Fatalf("opções devem permanecer no deck durante a decisão: %s/%s", s.Cards[top0].Zone, s.Cards[top1].Zone)
	}
	if len(p.Trail) != 2 {
		t.Fatalf("Sigilo do Rito foi emitido cedo demais: %v", p.Trail)
	}

	h.choose(0, top1)
	if s.Cards[top1].Zone != engine.ZoneHand || s.Cards[top0].Zone != engine.ZoneDeck {
		t.Fatalf("pick_top2 deixou zonas inconsistentes: escolhida=%s preterida=%s", s.Cards[top1].Zone, s.Cards[top0].Zone)
	}
	if p.Deck[len(p.Deck)-1] != top0 {
		t.Fatalf("carta preterida não foi ao fundo: %v", p.Deck)
	}
	if len(s.PendingRites) != 0 {
		t.Fatalf("continuação do Rito não foi consumida: %+v", s.PendingRites)
	}
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("VR-028 deveria comprar e só então pedir descarte: %+v", s.Pending)
	}
	for _, id := range p.Hand {
		if s.Cards[id].Zone != engine.ZoneHand {
			t.Fatalf("carta %s está na mão com zona %s", id, s.Cards[id].Zone)
		}
	}
}

func TestQueuedDiscardDecisionRefreshesHandOptions(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-028", "VR-028", "VR-026"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 8
	h.play(0, h.handInst(0, "VR-028"))
	h.play(0, h.handInst(0, "VR-028"))
	for _, id := range s.Players[0].Relics {
		s.Cards[id].UsedRound = 0
	}
	s.Players[0].Trail = []engine.Sigil{engine.SigilPresa, engine.SigilSol}

	h.play(0, h.handInst(0, "VR-026"))
	h.choose(0, s.Pending.Options[0]) // resolve o pick_top2 e aciona as 2 Relíquias
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN || len(s.DecQueue) != 1 {
		t.Fatalf("esperava dois descartes serializados: pending=%+v queue=%+v", s.Pending, s.DecQueue)
	}
	discarded := s.Pending.Options[0]
	h.choose(0, discarded)
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("segundo descarte não foi promovido: %+v", s.Pending)
	}
	if contains(s.Pending.Options, discarded) {
		t.Fatalf("segunda decisão reteve opção já descartada %s: %v", discarded, s.Pending.Options)
	}
	h.choose(0, s.Pending.Options[0])
}

func TestCopiedRiteDecisionFinishesBeforeCopiedSigil(t *testing.T) {
	h := newHarness(t, "CH-MI-01", "CH-CI-01",
		deckWith("VR-028"), deckWith("VR-063"), 1)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(1, h.handInst(1, "VR-063"))
	h.choose(1, h.g.State().Pending.Options...)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-028"))

	s := h.g.State()
	s.Players[0].Trail = []engine.Sigil{engine.SigilPresa, engine.SigilSol}
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	if s.Pending == nil || s.Pending.Kind != engine.DecReorderTop || len(s.PendingRites) != 1 || s.PendingRites[0].CopyDepth == 0 {
		t.Fatalf("cópia deveria aguardar reorder_top: pending=%+v rites=%+v", s.Pending, s.PendingRites)
	}
	for _, id := range s.Pending.Options {
		if s.Cards[id].Zone != engine.ZoneDeck {
			t.Fatalf("opção %s saiu do deck antes da escolha: %s", id, s.Cards[id].Zone)
		}
	}
	h.choose(0, s.Pending.Options...)
	for _, id := range s.Players[0].Deck {
		if s.Cards[id].Zone != engine.ZoneDeck {
			t.Fatalf("carta %s está no deck com zone=%s", id, s.Cards[id].Zone)
		}
	}
	for _, id := range s.Players[0].Hand {
		if s.Cards[id].Zone != engine.ZoneHand {
			t.Fatalf("carta %s está na mão com zone=%s", id, s.Cards[id].Zone)
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestLootVR049(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	p := h.g.State().Players[0]
	h.play(0, h.handInst(0, "VR-049"))
	h.choose(0, h.handInst(0, "VR-006"))
	if got := len(p.Hand); got != 6 { // 6 -1 jogada -1 descarte +2 compras
		t.Fatalf("mão: %d; esperado 6", got)
	}
	if got := p.Essence; got != 2 {
		t.Fatalf("essência: %d; esperado 2", got)
	}
}

func TestExileHealVR051AndVR050(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-050", "VR-051"), deckWith("VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	// p1 (iniciativa) ataca; p0 defende com Costura de Cinzas e a exila.
	h.play(1, h.handInst(1, "VR-013"))
	vr051 := h.handInst(0, "VR-051")
	h.play(0, vr051)
	h.choose(0, "yes")
	h.assertVit(0, 27) // preveniu 3 de 3; cura 1 sem efeito no máximo
	s := h.g.State()
	if s.Cards[vr051].Zone != engine.ZoneExile {
		t.Fatalf("VR-051 deveria estar exilada; está em %s", s.Cards[vr051].Zone)
	}
	if !s.Players[0].ExiledRound {
		t.Fatal("ExiledRound de p0 deveria estar ativo")
	}
	h.pass(1)

	// Janela de p0: Lâmina Carbonizada com bônus de exílio → 4 (alpha-0.5.0).
	h.play(0, h.handInst(0, "VR-050"))
	h.pass(1)
	h.assertVit(1, 23)
}

func TestKaedorCostReduction(t *testing.T) {
	h := newHarness(t, "CH-VH-02", "CH-CI-01",
		deckWith("VR-013", "VR-013"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	p := h.g.State().Players[0]
	p.Vitality = 15 // cirurgia: zona de desespero de Kaedor

	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	if got := p.Essence; got != 3 {
		t.Fatalf("1º Assalto deveria custar 0 (essência %d)", got)
	}
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	if got := p.Essence; got != 2 {
		t.Fatalf("2º Assalto deveria custar 1 (essência %d)", got)
	}
}

func TestSaelaGuardThenCheapAssault(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-VA-02",
		deckWith("VR-013"), deckWith("VR-014", "VR-013"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-014")) // Saela: próximo Assalto -1
	h.pass(0)

	p1 := h.g.State().Players[1]
	before := p1.Essence
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	if p1.Essence != before {
		t.Fatalf("Assalto pós-Guarda deveria custar 0 (essência %d → %d)", before, p1.Essence)
	}
}

func TestVigiliaOnlyFirstGuard(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-013"), deckWith("VR-014", "VR-014"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceVigilia)
	h.bothPassRite()

	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-014"))
	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-014"))
	if got := h.g.State().Players[1].Essence; got != 0 {
		t.Fatalf("essência de p1: %d; esperado 0 (custo 2: 1 com Vigília + 2)", got)
	}
}

func TestHandLimitAtTwilight(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01", deckWith(), deckWith(), 0)
	h.keepAll()
	h.passRound() // r1: mãos 6
	h.passRound() // r2: mãos 7
	// r3: mãos 8 → Crepúsculo exige descarte dos dois (iniciativa primeiro).
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.passConfront()

	s := h.g.State()
	if s.Phase != engine.PhaseTwilight || s.Pending == nil {
		t.Fatalf("esperava decisão de limite de mão no Crepúsculo; fase=%s", s.Phase)
	}
	first := s.Pending.Player
	if first != 0 { // iniciativa da rodada 3 é p0
		t.Fatalf("primeira decisão deveria ser da iniciativa (p0); veio de p%d", first)
	}
	h.choose(0, s.Players[0].Hand[0])
	if s.Pending == nil || s.Pending.Player != 1 {
		t.Fatalf("segunda decisão deveria ser de p1")
	}
	h.choose(1, s.Players[1].Hand[0])
	if s.Round != 4 || len(s.Players[0].Hand) != 8 {
		// rodada 4 já comprou: 7 + 1
		t.Fatalf("round=%d mão=%d; esperado round 4 e mão 8", s.Round, len(s.Players[0].Hand))
	}
}

func TestEclipseTotalTriggersAndResets(t *testing.T) {
	h := newHarness(t, "CH-VH-01", "CH-CI-01",
		deckWith("VR-068", "VR-068", "VR-002"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-068")) // 0 → +1 (cruza: compra 1)
	h.play(0, h.handInst(0, "VR-068")) // +1 → +2
	h.assertEclipse(2)
	h.pass(0)
	h.pass(1)
	h.passConfront()

	// Rodada 2: sem estado total, o medidor persiste. Iniciativa é de p1.
	h.assertEclipse(2)
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-002")) // deslocamento intrínseco +1 → +3
	s := h.g.State()
	if s.Pending != nil { // descarte da Dívida de Sangue
		h.choose(0, s.Players[0].Hand[0])
	}
	if s.EclipseState != engine.EclipseNight {
		t.Fatalf("estado do eclipse: %q; esperado Eclipse Noturno", s.EclipseState)
	}
	h.assertEclipse(3)
	h.pass(0)
	h.passConfront()

	// Fim da rodada com estado total: medidor volta a 0.
	h.assertEclipse(0)
	if s.EclipseState != engine.EclipseNone {
		t.Fatalf("estado do eclipse deveria ter encerrado; %q", s.EclipseState)
	}
}

func TestVR062DrawFilterWhenBehind(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013"), deckWith("VR-015", "VR-062"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(0)
	h.play(1, h.handInst(1, "VR-015")) // p1 fica com 5 cartas
	h.pass(1)

	h.play(0, h.handInst(0, "VR-013")) // p0 fica com 5
	h.play(1, h.handInst(1, "VR-062")) // p1 fica com 4 < 5 → compra e filtra
	s := h.g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("esperava decisão de descarte do filtro; %+v", s.Pending)
	}
	h.choose(1, s.Players[1].Hand[0])
	if got := len(s.Players[1].Hand); got != 4 {
		t.Fatalf("mão de p1: %d; esperado 4", got)
	}
}

func TestVeilGuardVR033(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-015", "VR-020"), deckWith("VR-033"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-033"))
	h.assertVit(1, 27) // 3 - 4
	s := h.g.State()
	if s.Players[1].VeilRound != s.Round+1 {
		t.Fatalf("Véu deveria durar até a próxima rodada; fim=%d rodada=%d", s.Players[1].VeilRound, s.Round)
	}
	h.pass(s.Active)
	h.pass(h.g.State().Active)
	if s.Round != 2 || s.Players[1].VeilRound != s.Round {
		t.Fatalf("Véu deveria seguir ativo na rodada 2; fim=%d rodada=%d", s.Players[1].VeilRound, s.Round)
	}
	h.stances(engine.StanceArcano, engine.StanceArcano)
	if s.Active == 1 {
		h.pass(1)
	}
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-015")},
		engine.ErrIllegalTarget)
	h.pass(0) // encerra Ritos
	h.pass(1)
	h.pass(0) // encerra Confronto e a rodada 2
	if s.Players[1].VeilRound != 0 {
		t.Fatal("Véu deveria expirar no Crepúsculo da rodada seguinte")
	}
}
