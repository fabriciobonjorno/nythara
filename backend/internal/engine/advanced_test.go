package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// Testes dos sistemas da engine completa: cópias, reação, janelas extras,
// ativações, informação oculta e Campeões.

func TestVR035CounterAndPassBranches(t *testing.T) {
	// Counter: VR-053 (custo 2) é anulada; conjurador recupera 1 temporária.
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-053", "VR-015"), deckWith("VR-035"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	vr053 := h.handInst(0, "VR-053")
	h.play(0, vr053)
	s := h.g.State()
	if s.RiteReact == nil {
		t.Fatal("a janela de reação deveria ter aberto")
	}
	h.play(1, h.handInst(1, "VR-035"))
	if len(s.Players[1].Curses) != 0 {
		t.Fatal("o Rito anulado não pode ter aplicado a Maldição")
	}
	if s.Cards[vr053].Zone != engine.ZoneDiscard {
		t.Fatal("o Rito anulado vai ao descarte")
	}
	if s.Players[0].TempEssence != 1 {
		t.Fatalf("reembolso: %d temp; esperado 1 (metade de 2)", s.Players[0].TempEssence)
	}

	// Sem counter na mão, o próximo Rito direcionado resolve direto.
	h.play(0, h.handInst(0, "VR-015"))
	if s.RiteReact != nil {
		t.Fatal("sem counter na mão não há janela de reação")
	}
	if !s.Players[1].Exposto {
		t.Fatal("VR-015 deveria ter resolvido")
	}
}

func TestVR035PassResumesRite(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-053"), deckWith("VR-035"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-053"))
	h.pass(1) // defensor guarda o counter para depois
	s := h.g.State()
	if s.RiteReact != nil || len(s.Players[1].Curses) != 1 {
		t.Fatal("após o passe, o Rito resolve normalmente")
	}
}

func TestVR006PeekTaxAndVR010Block(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-006", "VR-006"), deckWith("VR-010"), 1)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)

	// Sem o Salão: VR-006 olha 2 e taxa 1.
	h.pass(1)
	h.play(0, h.handInst(0, "VR-006"))
	s := h.g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecRevealTax {
		t.Fatalf("esperava escolha de sobretaxa; %+v", s.Pending)
	}
	taxed := s.Pending.Options[0]
	h.choose(0, taxed)
	found := false
	for _, m := range s.Players[1].CostMods {
		if m.Instance == taxed && m.Delta == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a carta revelada deveria estar taxada em +1")
	}
	h.pass(0)
	h.pass(1)
	h.pass(0) // confronto vazio da rodada 1

	// Rodada 2 (iniciativa p0): p1 baixa o Salão sem Espelhos.
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.pass(0)
	h.play(1, h.handInst(1, "VR-010"))
	h.pass(1)
	h.passConfront()

	// Rodada 3 (iniciativa p1): o próximo VR-006 falha e desloca p/ Noite.
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.pass(1)
	before := s.Eclipse
	h.play(0, h.handInst(0, "VR-006"))
	if s.Pending != nil {
		t.Fatal("VR-006 bloqueada não pode gerar escolha")
	}
	if s.Eclipse != before+1 {
		t.Fatalf("o Salão deveria ter deslocado 1 para a Noite (%d → %d)", before, s.Eclipse)
	}
}

func TestVR018RevealAndLock(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-018"), deckWith("VR-013"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.play(0, h.handInst(0, "VR-018"))
	s := h.g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecLockAssault {
		t.Fatalf("esperava escolha de trava; %+v", s.Pending)
	}
	locked := s.Pending.Options[0]
	h.choose(0, locked)
	h.pass(0)
	h.pass(1)
	// p1 tenta jogar o Assalto travado no Confronto.
	h.pass(0)
	h.mustFail(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: locked},
		engine.ErrIllegalTarget)
}

func TestVR029DeclareRevealTop(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-029"), deckWith("VR-013"), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	top := h.g.State().Players[1].Deck[0]
	topType := string(engine.Cards[h.g.State().Cards[top].Def].Type)
	before := len(h.g.State().Players[0].Hand)
	h.play(0, h.handInst(0, "VR-029"))
	h.choose(0, topType) // declara o tipo certo de propósito
	if got := len(h.g.State().Players[0].Hand); got != before { // -1 jogada +1 acerto
		t.Fatalf("acerto deveria comprar 1 (mão %d)", got)
	}
}

func TestVR030CopiesLastSigil(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-013", "VR-030"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	h.play(0, h.handInst(0, "VR-013")) // Sol
	h.pass(1)
	h.pass(0)
	h.pass(1)

	// Rodada 2: o Duplo copia o Sigilo do último jogado (Sol).
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-030"))
	trail := h.g.State().Players[0].Trail
	if len(trail) != 2 || trail[0] != engine.SigilEspelho || trail[1] != engine.SigilSol {
		t.Fatalf("trilha: %v; esperado [Espelho Sol]", trail)
	}
}

func TestVR032MirrorsRelicTriggers(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-032", "VR-001"), deckWith("VR-070"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.play(1, h.handInst(1, "VR-070"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-032"))
	h.choose(0, h.g.State().Players[1].Relics[0])
	h.pass(0)
	// Confronto: p1 passa; p0 causa exatamente 2 → a Caixa espelhada dá +1.
	h.pass(1)
	h.play(0, h.handInst(0, "VR-001"))
	h.pass(1)
	h.assertVit(1, 27) // 2 + 1 do espelho
}

func TestVR034ReEmitsSigils(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-063", "VR-034"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 5
	h.play(0, h.handInst(0, "VR-063")) // Espelho na trilha
	h.choose(0, s.Players[0].Deck[0], s.Players[0].Deck[1], s.Players[0].Deck[2])
	h.play(0, h.handInst(0, "VR-034"))
	// O Sigilo do próprio VR-034 entra antes de a escolha ser respondida.
	h.choose(0, "Espelho")
	trail := s.Players[0].Trail
	if len(trail) != 3 {
		t.Fatalf("trilha: %v; esperado 3 Espelhos", trail)
	}
}

func TestVR036CopiesPlayedCard(t *testing.T) {
	// p1 (iniciativa) resolve um Rito; p0 copia com o Oitavo Reflexo.
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-036"), deckWith("VR-049"), 1)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	h.play(1, h.handInst(1, "VR-049"))
	h.choose(1, s.Players[1].Hand[0])
	h.pass(1)

	s.Players[0].Essence = 5
	handBefore := len(s.Players[0].Hand)
	h.play(0, h.handInst(0, "VR-036"))
	if s.Pending == nil || s.Pending.Kind != engine.DecCopyPlayed {
		t.Fatalf("esperava escolha de cópia; %+v", s.Pending)
	}
	h.choose(0, "VR-049")
	// A cópia do VR-049 pede o descarte e compra 2.
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("a cópia deveria pedir descarte; %+v", s.Pending)
	}
	h.choose(0, s.Players[0].Hand[0])
	if got := len(s.Players[0].Hand); got != handBefore { // -1 jogada -1 descarte +2 compras
		t.Fatalf("mão: %d; esperado %d", got, handBefore)
	}
}

func TestVR047ExtraWindowWithCostCap(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-037", "VR-037", "VR-047", "VR-001", "VR-020"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	s := h.g.State()
	s.Players[0].Essence = 8
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.play(0, h.handInst(0, "VR-047")) // trilha: Garra, Garra, Garra → janela extra
	h.pass(1)
	h.pass(0)
	h.pass(1)
	if s.Extra == nil || s.Extra.Player != 0 || s.Extra.MaxCost != 2 {
		t.Fatalf("janela extra esperada para p0 (custo ≤2); %+v", s.Extra)
	}
	// Custo 3 é rejeitado; custo 1 entra.
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: h.handInst(0, "VR-020")},
		engine.ErrWrongPhase)
	h.play(0, h.handInst(0, "VR-001"))
	h.pass(1)
	h.pass(0)
	if s.Round != 2 {
		t.Fatalf("a rodada deveria ter avançado após a janela extra; round=%d", s.Round)
	}
}

func TestVR048MoonHeartReturns(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-048", "VR-001"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	s := h.g.State()
	s.Players[0].Essence = 8
	h.play(0, h.handInst(0, "VR-048"))
	h.pass(0)
	h.pass(1)
	s.EclipseState = engine.EclipseNight // cirurgia: Noite ativa
	vr001 := h.handInst(0, "VR-001")
	h.play(0, vr001)
	h.pass(1)
	h.pass(0)
	h.pass(1)
	if s.Cards[vr001].Zone != engine.ZoneHand {
		t.Fatalf("o Assalto deveria ter voltado à mão; está em %s", s.Cards[vr001].Zone)
	}
}

func TestVR055PreExileAndDirection(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049", "VR-055"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	vr049 := h.handInst(0, "VR-049")
	h.play(0, vr049)
	h.choose(0, h.handInst(0, "VR-006"))
	h.pass(0)
	h.pass(1)
	s := h.g.State()
	s.Players[0].Essence = 5
	h.play(0, h.handInst(0, "VR-055"))
	if s.Pending == nil || s.Pending.Kind != engine.DecExilePick {
		t.Fatalf("esperava exílio pré-resolução; %+v", s.Pending)
	}
	h.choose(0, vr049) // exila o Rito
	h.pass(1)          // janela de Guarda anunciada após a escolha
	h.assertVit(1, 25) // 5 de dano (alpha-0.5.0)
	if s.Pending == nil || s.Pending.Kind != engine.DecDirection {
		t.Fatalf("Rito exilado deveria abrir escolha de direção; %+v", s.Pending)
	}
	h.choose(0, "noite")
	h.assertEclipse(1)
	if s.Cards[vr049].Zone != engine.ZoneExile {
		t.Fatal("a carta escolhida deveria estar exilada")
	}
}

func TestActivatedAbilitiesVR058AndVR072(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049", "VR-058", "VR-072", "VR-063"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 8
	vr049 := h.handInst(0, "VR-049")
	h.play(0, vr049)
	h.choose(0, h.handInst(0, "VR-006")) // descarte: agora há 2 no descarte
	h.play(0, h.handInst(0, "VR-058"))

	forno := s.Players[0].Relics[0]
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindActivate, Card: forno})
	h.choose(0, s.Players[0].Discard[0], s.Players[0].Discard[1])
	if got := s.Players[0].TempEssence; got != 2 {
		t.Fatalf("Forno: %d temp; esperado 2", got)
	}
	// Uma vez por rodada.
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindActivate, Card: forno},
		engine.ErrBadCommand)

	// Mercador: descarta 1 → próxima carta neutra custa 1 a menos.
	h.play(0, h.handInst(0, "VR-072"))
	mercador := s.Players[0].Manifs[0]
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindActivate, Card: mercador})
	h.choose(0, h.handInst(0, "VR-006")) // descarta um filler, preservando o VR-063
	before := s.Players[0].Essence + s.Players[0].TempEssence
	h.play(0, h.handInst(0, "VR-063")) // neutra, custo 1 → 0
	if got := s.Players[0].Essence + s.Players[0].TempEssence; got != before {
		t.Fatalf("carta neutra deveria custar 0 (%d → %d)", before, got)
	}
}

func TestVR060FinalFormula(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049", "VR-002", "VR-060"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 8
	h.play(0, h.handInst(0, "VR-049"))
	h.choose(0, h.handInst(0, "VR-006"))
	h.play(0, h.handInst(0, "VR-002"))
	h.choose(0, h.handInst(0, "VR-006"))
	if got := len(s.Players[0].Discard); got != 4 {
		t.Fatalf("descarte com %d cartas; esperado 4", got)
	}
	h.play(0, h.handInst(0, "VR-060"))
	if s.Pending == nil || s.Pending.Kind != engine.DecFormulaChoice {
		t.Fatalf("esperava 1 escolha da Fórmula; %+v", s.Pending)
	}
	h.choose(0, "dano_2")
	h.assertVit(1, 28)
	if got := len(s.Players[0].Exile); got != 4 {
		t.Fatalf("exílio com %d cartas; esperado 4", got)
	}
}

func TestVR076ClockMovesWhenStill(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-076"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	s.Players[0].Essence = 4
	h.play(0, h.handInst(0, "VR-076"))
	h.pass(0)
	h.pass(1)
	h.passConfront()
	// Ninguém moveu o Eclipse: o Relógio pergunta a direção (empate → dono).
	if s.Pending == nil || s.Pending.Kind != engine.DecDirection || s.Pending.Player != 0 {
		t.Fatalf("esperava escolha de direção do dono; %+v", s.Pending)
	}
	h.choose(0, "aurora")
	h.assertEclipse(-1)
	if s.Round != 2 {
		t.Fatalf("round=%d; esperado 2", s.Round)
	}
}

// --- Campeões ---

func TestKaedorDeathRefusal(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-VH-02",
		deckWith("VR-020"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	s := h.g.State()
	s.Players[1].Vitality = 3
	h.play(0, h.handInst(0, "VR-020")) // 4 de dano letal
	h.pass(1)
	if s.Over {
		t.Fatal("Recusa da Morte deveria ter salvado Kaedor")
	}
	if s.Players[1].Vitality != 1 || len(s.Players[1].Hand) != 0 {
		t.Fatalf("vit=%d mão=%d; esperado 1 e mão vazia", s.Players[1].Vitality, len(s.Players[1].Hand))
	}
	if !s.Players[1].UltimateUsed {
		t.Fatal("a ultimate deveria constar como usada")
	}
}

func TestSaelaUltimateShieldAndBonus(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-VA-02",
		deckWith("VR-020"), deckWith("VR-013"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.must(engine.Command{Player: 1, Kind: engine.CmdKindUltimate})
	h.pass(1)
	h.pass(0)
	// Saela ataca com +2; depois o dano de p0 é reduzido a 0.
	h.play(1, h.handInst(1, "VR-013"))
	h.pass(0)
	h.assertVit(0, 26) // 2 + 2 (VR-013 sem bônus a 0 no alpha-0.5.0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-020"))
	h.pass(1)
	h.assertVit(1, 30) // escudo: primeiro dano → 0
}

func TestRaukUltimateExtraWindow(t *testing.T) {
	h := newHarness(t, "CH-VA-01", "CH-CI-01",
		deckWith("VR-037", "VR-037"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	h.pass(0)
	h.pass(1)
	h.play(0, h.handInst(0, "VR-037"))
	h.pass(1)
	h.pass(0)
	h.pass(1)
	s := h.g.State()
	if s.Extra == nil || s.Extra.Player != 0 {
		t.Fatalf("Caçada Sem Lua deveria abrir janela extra; %+v", s.Extra)
	}
	h.play(0, h.handInst(0, "VR-037")) // 2º Assalto: +2 da carta +1 do Rauk
	h.pass(1)
	h.assertVit(1, 30-2-5)
	h.pass(0)
}

func TestIlyanPassiveAndNullify(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-SO-02",
		deckWith("VR-068", "VR-068", "VR-002"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	s := h.g.State()
	h.play(0, h.handInst(0, "VR-068"))
	h.play(0, h.handInst(0, "VR-068"))
	// Segundo empurrão à Noite na rodada: Ilyan ganha Ward 1 (alpha-0.5.0).
	if got := s.Players[1].Ward; got != 1 {
		t.Fatalf("ward de Ilyan: %d; esperado 1", got)
	}
	h.pass(0)
	// Ilyan arma a anulação; o próximo deslocamento de p0 é anulado.
	h.must(engine.Command{Player: 1, Kind: engine.CmdKindUltimate})
	h.pass(1)
	h.pass(0)
	h.pass(1)
	// Rodada 2: VR-002 de p0 (shift +1 intrínseco) é anulada.
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.pass(1)
	before := s.Eclipse
	h.play(0, h.handInst(0, "VR-002"))
	h.choose(0, s.Players[0].Hand[0])
	if s.Eclipse != before {
		t.Fatalf("o deslocamento deveria ter sido anulado (%d → %d)", before, s.Eclipse)
	}
}

func TestNyraScryAndCopyUltimate(t *testing.T) {
	h := newHarness(t, "CH-MI-01", "CH-CI-01",
		deckWith("VR-013", "VR-025", "VR-063"), deckWith("VR-049"), 1)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	// p1 resolve um Rito (VR-049) para a ultimate de Nyra ter alvo.
	h.play(1, h.handInst(1, "VR-049"))
	s := h.g.State()
	h.choose(1, s.Players[1].Hand[0])
	h.pass(1)
	// Nyra copia o VR-049: descarta 1 e compra 2 sem possuir a carta.
	handBefore := len(s.Players[0].Hand)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	if s.Pending == nil || s.Pending.Kind != engine.DecDiscardN {
		t.Fatalf("cópia do VR-049 deveria pedir descarte; %+v", s.Pending)
	}
	h.choose(0, s.Players[0].Hand[0])
	if got := len(s.Players[0].Hand); got != handBefore+1 { // -1 descarte +2 compras
		t.Fatalf("mão: %d; esperado %d", got, handBefore+1)
	}
}

func TestOrenPreMulliganAndUltimate(t *testing.T) {
	g, err := engine.NewGame(engine.Config{
		Seed: 9, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-MI-02", Deck: deckWith("VR-013")},
			{ChampionID: "CH-CI-01", Deck: deckWith()},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := g.State()
	if s.Pending == nil || s.Pending.Kind != engine.DecReorderTop || s.Pending.Player != 0 {
		t.Fatalf("Oren deveria reordenar o topo antes do mulligan; %+v", s.Pending)
	}
	d := s.Pending
	// Mantém a ordem atual.
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose, DecisionID: d.ID, Cards: d.Options}); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, g: g}
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	before := len(s.Players[0].Hand)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	h.choose(0, "comprar_2")
	if got := len(s.Players[0].Hand); got != before+2 {
		t.Fatalf("Fenda de Probabilidade: mão %d; esperado %d", got, before+2)
	}
}

func TestSerisUltimateHealsHalf(t *testing.T) {
	h := newHarness(t, "CH-VH-01", "CH-CI-01",
		deckWith("VR-020"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	// Sem dano causado, a ultimate é ilegal.
	h.mustFail(engine.Command{Player: 0, Kind: engine.CmdKindUltimate}, engine.ErrRequirement)
	h.bothPassRite()
	s := h.g.State()
	s.Players[0].Vitality = 20
	h.play(0, h.handInst(0, "VR-020"))
	h.pass(1)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	h.assertVit(0, 21) // curou metade de 3 (alpha-0.6.0: VR-020 causa 3)
}

func TestVorenUltimateDistillation(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-CI-01",
		deckWith("VR-049"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	h.play(0, h.handInst(0, "VR-049"))
	h.choose(0, h.handInst(0, "VR-006"))
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	if s.Pending == nil || s.Pending.Kind != engine.DecExilePick {
		t.Fatalf("Destilação Negra deveria pedir exílios; %+v", s.Pending)
	}
	h.choose(0, s.Players[0].Discard[0], s.Players[0].Discard[1])
	// Duas escolhas cura/dano na fila.
	h.choose(0, "dano_1")
	h.choose(0, "dano_1")
	h.assertVit(1, 28)
}

func TestEddaMarkersAndRewrite(t *testing.T) {
	h := newHarness(t, "CH-CI-02", "CH-CI-01",
		deckWith("VR-049", "VR-013"), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	s := h.g.State()
	h.play(0, h.handInst(0, "VR-049"))
	h.choose(0, h.handInst(0, "VR-006"))
	if got := s.Players[0].CinzaMarkers; got != 1 {
		t.Fatalf("marcadores: %d; esperado 1", got)
	}
	s.Players[0].CinzaMarkers = 3 // cirurgia: acelera o teste
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	if s.Pending == nil || s.Pending.Kind != engine.DecEddaReturn {
		t.Fatalf("Página Reescrita deveria pedir a carta; %+v", s.Pending)
	}
	pick := s.Pending.Options[0]
	h.choose(0, pick)
	if s.Cards[pick].Zone != engine.ZoneHand || s.Players[0].CinzaMarkers != 0 {
		t.Fatal("carta na mão e marcadores gastos eram esperados")
	}
}

func TestMaraUltimateAurora(t *testing.T) {
	h := newHarness(t, "CH-SO-01", "CH-CI-01", deckWith(), deckWith(), 0)
	h.keepAll()
	h.stances(engine.StanceVigilia, engine.StanceVigilia)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindUltimate})
	h.assertEclipse(-2)
	h.assertVit(1, 29) // alpha-0.6.0: a ultimate drena 1 (era 2)
}

// Regressão: morte por Fadiga no meio da emissão de Sigilo de uma Guarda
// (compra da passiva de Nyra) não pode corromper a resolução do Assalto.
func TestDeathDuringGuardSigilEmissionEndsCleanly(t *testing.T) {
	h := newHarness(t, "CH-CI-01", "CH-MI-01",
		deckWith("VR-020"), deckWith("VR-014"), 0)
	h.keepAll()
	h.stances(engine.StanceArcano, engine.StanceArcano)
	h.bothPassRite()
	s := h.g.State()
	// Cirurgia: Nyra à beira da morte, sem deck nem descarte, trilha com 2
	// Sigilos — a Guarda emite o 3º, a passiva compra, a Fadiga (2) mata.
	p1 := s.Players[1]
	p1.Vitality = 2
	p1.Trail = []engine.Sigil{engine.SigilSol, engine.SigilCoroa}
	for _, id := range p1.Deck {
		s.Cards[id].Zone = engine.ZoneExile
		p1.Exile = append(p1.Exile, id)
	}
	p1.Deck = nil

	h.play(0, h.handInst(0, "VR-020"))
	h.play(1, h.handInst(1, "VR-014"))
	if !s.Over || s.Winner != 0 {
		t.Fatalf("a partida deveria ter terminado com vitória de p0; over=%v winner=%d", s.Over, s.Winner)
	}
	// Zonas íntegras: Assalto e Guarda finalizaram no descarte.
	if s.Cards[h.g.CommandLog[len(h.g.CommandLog)-1].Card].Zone != engine.ZoneDiscard {
		t.Fatal("a Guarda deveria ter ido ao descarte mesmo com a morte no meio")
	}
}
