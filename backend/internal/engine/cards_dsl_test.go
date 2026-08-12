package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// Testes das cartas novas da Fase 2 (definidas em data/effects_alpha.json).

func TestVR004HealShiftsNight(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-004", "VR-003"), deckWith("VR-013", "VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-004"))
	h.pass(0)

	// A Taça (shift +1) deixou o Eclipse em +1: VR-013 causa só 2.
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	h.assertVit(0, 25)
	h.play(1, h.handInst(1, "VR-013"))
	h.play(0, h.handInst(0, "VR-003")) // previne tudo → cura 1 → Taça move p/ Noite
	h.assertVit(0, 26)
	found := false
	for _, e := range h.g.Log {
		if e.Kind == engine.EvEclipseShifted && e.S == "VR-004" {
			found = true
		}
	}
	if !found {
		t.Fatal("VR-004 deveria ter deslocado o Eclipse ao curar")
	}
}

func TestVR007ServoReducesSacrifice(t *testing.T) {
	h := newHarness(t, "CH-VH-01", "CH-CI-01",
		deckWith("VR-007", "VR-002"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-007"))
	h.play(0, h.handInst(0, "VR-002"))
	// Sacrifício 2-1=1; passiva de Seris ainda conta o sacrifício.
	h.assertVit(0, 26)
	if got := h.g.State().Players[0].TempEssence; got != 1 {
		t.Fatalf("temp essence: %d; esperado 1", got)
	}
	h.choose(0, h.g.State().Players[0].Hand[0])
}

func TestVR008RecoverAssaultCostsVitality(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-008", "VR-013"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	// Sem Assalto no descarte: requisito falha.
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-008")},
		engine.ErrRequirement)
	h.bothPassRite()
	vr013 := h.handInst(0, "VR-013")
	h.play(0, vr013)
	h.pass(1)
	h.pass(0)
	h.pass(1)

	// Rodada 2 (iniciativa p1).
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-008"))
	h.choose(0, vr013)
	s := h.g.State()
	if s.Cards[vr013].Zone != engine.ZoneHand {
		t.Fatalf("VR-013 deveria ter voltado à mão; está em %s", s.Cards[vr013].Zone)
	}
	h.assertVit(0, 26) // perdeu Vitalidade igual ao custo (1)
}

func TestVR011RequiresHighEclipse(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-011", "VR-068", "VR-068"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-011")},
		engine.ErrRequirement)
	h.play(0, h.handInst(0, "VR-068"))
	h.play(0, h.handInst(0, "VR-068"))
	h.assertEclipse(2)
	h.pass(0)
	h.pass(1)
	h.passConfront()

	// Rodada 2: eclipse persistiu em +2; VR-011 agora é legal.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	before := len(h.g.State().Players[0].Hand)
	h.play(0, h.handInst(0, "VR-011"))
	if got := len(h.g.State().Players[0].Hand); got != before-1+3 {
		t.Fatalf("mão: %d; esperado %d (compra 3)", got, before-1+3)
	}
	// Deslocamento intrínseco +2 leva a +3 → Eclipse Noturno.
	if h.g.State().EclipseState != engine.EclipseNight {
		t.Fatalf("esperava Eclipse Noturno; %q", h.g.State().EclipseState)
	}
}

func TestVR012NightFinisher(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-012"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	s := h.g.State()
	s.Players[0].Essence = 5
	s.Players[0].Vitality = 25
	s.EclipseState = engine.EclipseNight
	h.play(0, h.handInst(0, "VR-012"))
	h.pass(1)
	h.assertVit(1, 20) // 7 no Eclipse Noturno
	h.assertVit(0, 27) // cura limitada à Vitalidade máxima
}

func TestVR016ShortensCurse(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-053"), deckWith("VR-016"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-016"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-053"))
	h.pass(0)
	h.passConfront()
	// Sem o Rosário a Maldição dispararia só no fim da rodada 2; com ele,
	// dispara já no Crepúsculo da rodada 1 (mão de p1 ≥ 5). Maldição vale 3.
	h.assertVit(1, 24)
	if got := len(h.g.State().Players[1].Curses); got != 0 {
		t.Fatalf("maldições restantes: %d", got)
	}
}

func TestVR019WardOnFirstGuard(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013"), deckWith("VR-019", "VR-014"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceVigilia)
	h.pass(0)
	h.g.State().Players[1].Essence = 6 // VR-019 custa 4 no alpha-0.5.0
	h.play(1, h.handInst(1, "VR-019"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-014")) // custo 1 pela Vigília; Sentinela → Ward 1
	if got := h.g.State().Players[1].Ward; got != 1 {
		t.Fatalf("ward: %d; esperado 1", got)
	}
	h.assertVit(1, 27)
}

func TestVR021SuppressesManifs(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-021"), deckWith("VR-043", "VR-013", "VR-013", "VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-043"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-021")) // silencia Manifestações de p1 até a rodada 2
	h.pass(0)
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	// Corredora suprimida: sem Sigilo Garra extra (só Garra da entrada + Sol).
	trail := h.g.State().Players[1].Trail
	if len(trail) != 2 {
		t.Fatalf("trilha r1: %v; esperado 2 sigilos (supressão ativa)", trail)
	}
	h.pass(1)
	h.pass(0)

	h.passRound() // rodada 2: ainda suprimida

	// Rodada 3: supressão expirou; o gatilho volta a funcionar.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	trail = h.g.State().Players[1].Trail
	if len(trail) != 2 || trail[1] != engine.SigilGarra {
		t.Fatalf("trilha r3: %v; esperado [Sol Garra]", trail)
	}
}

func TestVR022PreventsAllOpponentDraws(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-020"), deckWith("VR-022"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	before := len(h.g.State().Players[0].Hand)
	h.g.State().Players[1].Essence = 4 // VR-022 custa 4 no alpha-0.5.0
	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-022"))
	h.assertVit(1, 27)
	if got := len(h.g.State().Players[0].Hand); got != before-1+1 {
		t.Fatalf("p0 deveria ter comprado 1 (mão %d)", got)
	}
}

func TestVR023ResetToAurora(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-023"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Eclipse = 1
	s.Players[0].Essence = 4
	s.Players[0].Bleeds = []engine.TimedN{{N: 2, Round: 2}}
	s.Players[0].Curses = []engine.TimedN{{N: 2, Round: 2, Kind: "VR-053"}}
	h.play(0, h.handInst(0, "VR-023"))
	h.assertEclipse(0) // alpha-0.6.0: move 1 (ADR-029); +1 → 0
	if len(s.Players[0].Bleeds) != 0 || len(s.Players[0].Curses) != 0 {
		t.Fatal("Sangramento e Maldição deveriam ter sido removidos")
	}
}

func TestVR024UndefendableInAurora(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-024"), deckWith("VR-014"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	s := h.g.State()
	s.Players[0].Essence = 5
	s.EclipseState = engine.EclipseAurora
	h.play(0, h.handInst(0, "VR-024"))
	// Sem janela de Guarda: dano direto de 7.
	if s.Guard != nil {
		t.Fatal("não deveria haver janela de Guarda na Aurora Total")
	}
	h.assertVit(1, 20)
}

func TestVR028DrawFilterOnThirdSigil(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-028", "VR-068", "VR-013", "VR-025"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-028"))
	h.pass(0)
	h.pass(1)
	h.passConfront()

	// Rodada 2: Coroa (VR-068) → Sol (VR-013) → Espelho (VR-025) = 3 distintos.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-068"))
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-025"))
	h.pass(1)
	s := h.g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("Sala dos Ângulos Falsos deveria pedir descarte; pendente: %+v", s.Pending)
	}
	h.choose(0, s.Players[0].Hand[0])
}

func TestVR039BothBranches(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-039", "VR-039"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-039"))
	if !h.g.State().Players[1].Exposto {
		t.Fatal("com 30 de Vitalidade, VR-039 aplica Exposto")
	}
	h.pass(0)
	h.pass(1)
	h.passConfront()

	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.pass(1)
	h.g.State().Players[1].Vitality = 15
	before := len(h.g.State().Players[0].Hand)
	h.play(0, h.handInst(0, "VR-039"))
	if got := len(h.g.State().Players[0].Hand); got != before { // -1 jogada +1 compra
		t.Fatalf("mão: %d; esperado %d (comprou 1)", got, before)
	}
}

func TestVR045BloodMoonBurst(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-045", "VR-013", "VR-013"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-045"))
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.assertVit(0, 25) // 1 por Assalto
	// Lua no Sangue (shift +2) tirou o bônus de eclipse dos Golpes: 2+2.
	h.assertVit(1, 23)
	if got := h.g.State().Players[0].Essence; got != 1 {
		t.Fatalf("essência: %d; esperado 1 (rito 2 com Arcano; assaltos grátis)", got)
	}
}

func TestVR046DestroyRelicAndVR071(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-046", "VR-013"), deckWith("VR-065", "VR-071"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-065"))
	h.pass(1)
	h.pass(0)
	h.passConfront()
	h.passRound()

	// Rodada 3 (iniciativa p1): Uivo destrói a Moeda; Escudo previne 6.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	vr065 := h.g.State().Players[1].Relics[0]
	h.play(0, h.handInst(0, "VR-046"))
	h.choose(0, vr065)
	s := h.g.State()
	if s.Cards[vr065].Zone != engine.ZoneDiscard || len(s.Players[1].Relics) != 0 {
		t.Fatal("a Relíquia deveria ter sido destruída para o descarte")
	}
	h.assertVit(1, 27) // custo 2 < 3: sem dano ao controlador
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-071")) // Relíquia destruída nesta rodada → previne 6
	h.assertVit(1, 27)
}

func TestVR052MillOnExile(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-052", "VR-051"), deckWith("VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-052"))
	h.pass(0)
	h.play(1, h.handInst(1, "VR-013"))
	h.play(0, h.handInst(0, "VR-051"))
	h.choose(0, "yes") // exila a Costura → Arquivo Morto pergunta pelo topo
	s := h.g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecMillTop {
		t.Fatalf("esperava decisão de mill; pendente: %+v", s.Pending)
	}
	top := s.Players[0].Deck[0]
	h.choose(0, "yes")
	if s.Cards[top].Zone != engine.ZoneDiscard {
		t.Fatalf("o topo deveria ter ido ao descarte; está em %s", s.Cards[top].Zone)
	}
}

func TestVR054RecoversCheapCard(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-054"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	vr013 := h.handInst(0, "VR-013")
	h.play(0, vr013)
	h.pass(1)
	h.pass(0)
	h.pass(1)

	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-054"))
	h.choose(0, vr013)
	if h.g.State().Cards[vr013].Zone != engine.ZoneHand {
		t.Fatal("Coletor de Restos deveria recuperar a carta de custo 1")
	}
}

func TestVR056ReturnsToHandWithSurcharge(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-013"), deckWith("VR-056"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-013"))
	vr056 := h.handInst(1, "VR-056")
	h.play(1, vr056)
	s := h.g.State()
	if s.Cards[vr056].Zone != engine.ZoneHand {
		t.Fatalf("Selo deveria voltar à mão; está em %s", s.Cards[vr056].Zone)
	}
	// Segunda Guarda na mesma rodada custaria 2+1=3; p1 tem 3-2=1.
	h.play(0, h.handInst(0, "VR-013"))
	h.mustFail(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: vr056}, engine.ErrCantAfford)
}

func TestVR057TaxesType(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-057"), deckWith("VR-013", "VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	h.pass(1)
	h.pass(0)

	// Rodada 2 (iniciativa de p0): Anatomia taxa Assaltos de p1 em +1.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	vr013Discarded := h.g.State().Players[1].Discard[0]
	h.play(0, h.handInst(0, "VR-057"))
	h.choose(0, vr013Discarded)
	h.pass(0)
	h.pass(1)
	h.pass(0) // confronto de p0
	p1 := h.g.State().Players[1]
	before := p1.Essence
	h.play(1, h.handInst(1, "VR-013"))
	if p1.Essence != before-2 {
		t.Fatalf("Assalto taxado deveria custar 2 (essência %d → %d)", before, p1.Essence)
	}
}

func TestVR059RecoverTwoDiscardTwo(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049", "VR-013", "VR-059"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	vr049 := h.handInst(0, "VR-049")
	h.play(0, vr049)
	h.choose(0, h.handInst(0, "VR-006"))
	h.pass(0)
	h.pass(1)
	vr013 := h.handInst(0, "VR-013")
	h.play(0, vr013)
	h.pass(1)
	h.pass(0)
	h.pass(1)

	// Rodada 2: descarte tem 3 cartas; recupera 2 e descarta 2.
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-059"))
	h.choose(0, vr049, vr013)
	s := h.g.State()
	if s.Cards[vr049].Zone != engine.ZoneHand || s.Cards[vr013].Zone != engine.ZoneHand {
		t.Fatal("as duas cartas deveriam ter voltado à mão")
	}
	if s.Pending == nil || s.Pending.N != 2 {
		t.Fatalf("esperava descarte de 2; pendente: %+v", s.Pending)
	}
	h.choose(0, s.Players[0].Hand[0], s.Players[0].Hand[1])
}

func TestVR063ReorderTop(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-063"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	p := h.g.State().Players[0]
	a, b, c := p.Deck[0], p.Deck[1], p.Deck[2]
	h.play(0, h.handInst(0, "VR-063"))
	h.choose(0, c, a, b)
	if p.Deck[0] != c || p.Deck[1] != a || p.Deck[2] != b {
		t.Fatalf("topo reordenado incorretamente: %v", p.Deck[:3])
	}
}

func TestVR066OpponentPicksDiscard(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-066"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-066"))
	s := h.g.State()
	d := s.Pending
	if d == nil || d.Kind != engine.DecOppDiscardPick || d.Player != 1 {
		t.Fatalf("a escolha deveria ser do oponente; pendente: %+v", d)
	}
	pick := d.Options[0]
	other := d.Options[1]
	h.choose(1, pick)
	if s.Cards[pick].Zone != engine.ZoneDiscard || s.Cards[other].Zone != engine.ZoneHand {
		t.Fatal("a escolhida vai ao descarte; a outra fica na mão")
	}
}

func TestVR067StripsVeil(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-067", "VR-020"), deckWith("VR-033"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-067")},
		engine.ErrRequirement)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-033"))
	s := h.g.State()
	h.pass(s.Active)
	h.pass(h.g.State().Active)
	h.stances(engine.StanceArcano, engine.StanceArcano)
	if s.Active == 1 {
		h.pass(1)
	}
	h.play(0, h.handInst(0, "VR-067"))
	if s.Players[1].VeilRound != 0 || !s.Players[1].Exposto {
		t.Fatal("Véu removido e Exposto aplicado eram esperados")
	}
}

func TestVR070ExactTwoBonus(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-070", "VR-001"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-070"))
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-001")) // 2 exatos → Caixa causa +1
	h.pass(1)
	h.assertVit(1, 24)
}

func TestVR073PunishesSecondDraw(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049"), deckWith("VR-073"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-073"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-049"))
	h.choose(0, h.handInst(0, "VR-006"))
	// A 2ª compra da rodada de p0 dispara o Cão sem Sombra.
	h.assertVit(0, 26)
}

func TestVR074RemovesCurseSmart(t *testing.T) {
	// Com Maldição própria: remove a sua, sem compra.
	h := newHarness(t, "CH-CI-01", "CH-CI-01", deckWith("VR-074"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Curses = []engine.TimedN{{N: 2, Round: 5, Kind: "VR-053"}}
	before := len(s.Players[0].Hand)
	h.play(0, h.handInst(0, "VR-074"))
	if len(s.Players[0].Curses) != 0 {
		t.Fatal("deveria remover a própria Maldição")
	}
	if len(s.Players[0].Hand) != before-1 {
		t.Fatal("não deveria comprar ao remover a própria")
	}

	// Sem Maldição própria: remove a do oponente e compra 1.
	h2 := newHarness(t, "CH-CI-01", "CH-CI-01", deckWith("VR-074"), deckWith(), 0)
	h2.keepAll()
	h2.stances(engine.StanceVigilia, engine.StanceVigilia)
	s2 := h2.g.State()
	s2.Players[1].Curses = []engine.TimedN{{N: 2, Round: 5, Kind: "VR-053"}}
	before2 := len(s2.Players[0].Hand)
	h2.play(0, h2.handInst(0, "VR-074"))
	if len(s2.Players[1].Curses) != 0 {
		t.Fatal("deveria remover a Maldição do oponente")
	}
	if len(s2.Players[0].Hand) != before2 { // -1 jogada +1 compra
		t.Fatal("deveria comprar 1 ao remover do oponente")
	}
}

func TestVR075SwapsEclipse(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013"), deckWith("VR-075"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.g.State().Eclipse = 2
	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-075"))
	// Após o Assalto (shift -1 → +1), p1 confirma a troca: +1 vira -1.
	h.choose(1, "yes")
	h.assertEclipse(-1)
}

func TestVR077NeutralSigilOath(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-077", "VR-061"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(0, h.handInst(0, "VR-077"))
	h.choose(0, "Garra")
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-061"))
	h.pass(1)
	trail := h.g.State().Players[0].Trail
	n := len(trail)
	if n < 2 || trail[n-2] != engine.SigilCoroa || trail[n-1] != engine.SigilGarra {
		t.Fatalf("trilha: %v; esperado [... Coroa Garra]", trail)
	}
}

func TestVR078EndsAssaultWindow(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-013", "VR-013"), deckWith("VR-078"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-013"))
	h.play(1, h.handInst(1, "VR-078")) // 2º Assalto → encerra a janela de p0
	third := h.handInst(0, "VR-013")
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: third},
		engine.ErrWrongPlayer)
	h.pass(1)
	if h.g.State().Round != 2 {
		t.Fatalf("a rodada deveria ter avançado; round=%d", h.g.State().Round)
	}
}

func TestVR079ResetsExtremes(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith(), deckWith("VR-079"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	s := h.g.State()
	s.Players[1].Essence = 5
	h.play(1, h.handInst(1, "VR-079"))
	h.pass(1)
	h.pass(0)
	h.passConfront()
	s.Eclipse = 2 // fim da rodada 1: força o extremo antes da preparação... tarde demais

	// O gatilho roda no início da rodada 3.
	h.passRound()
	h.assertEclipse(0)
	found := false
	for _, e := range h.g.Log {
		if e.Kind == engine.EvEclipseShifted && e.S == "VR-079" {
			found = true
		}
	}
	if !found {
		t.Fatal("O Homem do Campanário deveria ter movido o Eclipse para 0")
	}
}

func TestVR080ForcesTotalEclipse(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-080"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 6
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-080")},
		engine.ErrRequirement)
	s.Eclipse = 2
	before := len(s.Players[1].Hand)
	h.play(0, h.handInst(0, "VR-080"))
	if s.EclipseState != engine.EclipseNight {
		t.Fatalf("esperava Eclipse Noturno; %q", s.EclipseState)
	}
	h.assertEclipse(3)
	if got := len(s.Players[1].Hand); got != before+2 {
		t.Fatalf("oponente deveria ter comprado 2 (mão %d)", got)
	}
}
