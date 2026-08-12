package engine_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"veurubro/backend/internal/engine"
)

var update = flag.Bool("update", false, "atualiza arquivos golden")

// TestScriptedMatchGolden joga uma rodada completa roteirizada cobrindo:
// mulligan, posturas, sacrifício + passiva de Seris, decisão de descarte,
// Exposto, Predação, Vigília, passiva de Mara, prevenção, Ressonância global
// (VR-025), deslocamentos de Eclipse, virada de iniciativa e rampa de
// Essência — e compara o log integral de eventos com o golden.
func TestScriptedMatchGolden(t *testing.T) {
	h := newHarness(t,
		"CH-VH-01", "CH-SO-01",
		deckWith("VR-001", "VR-002", "VR-005", "VR-009", "VR-013", "VR-049", "VR-068"),
		deckWith("VR-014", "VR-003", "VR-015", "VR-042", "VR-025", "VR-026"),
		0,
	)
	g := h.g
	h.keepAll()

	// Rodada 1: mãos = 5 iniciais + 1 compra.
	if got := g.State().Players[0].Essence; got != 3 {
		t.Fatalf("essência inicial: %d; esperado 3", got)
	}
	h.stances(engine.StancePredacao, engine.StanceVigilia)

	// Rito p0: Dívida de Sangue (sacrifica 2, compra 2, descarta 1).
	vr002 := h.handInst(0, "VR-002")
	h.play(0, vr002)
	h.assertVit(0, 25)
	if got := g.State().Players[0].TempEssence; got != 1 {
		t.Fatalf("passiva de Seris: temp essence = %d; esperado 1", got)
	}
	h.choose(0, h.handInst(0, "VR-006"))
	h.pass(0)

	// Rito p1: Marca do Caçador (Exposto em p0).
	h.play(1, h.handInst(1, "VR-015"))
	if !g.State().Players[0].Exposto {
		t.Fatal("p0 deveria estar Exposto")
	}
	h.pass(1)

	// Confronto p0: Corte Rubro = 2 +1 (sacrifício) +1 (Predação) = 4.
	// p1 responde com Égide de Lumen (custo 0: Vigília + Mara): previne 3 → 1 passa.
	h.play(0, h.handInst(0, "VR-001"))
	h.play(1, h.handInst(1, "VR-014"))
	h.assertVit(1, 27)

	// Golpe da Alvorada: eclipse em 0 → 2 (bônus exige Aurora funda).
	h.play(0, h.handInst(0, "VR-013"))
	h.pass(1)
	h.assertVit(1, 25)
	h.pass(0)

	// Confronto p1: Refração Dolorosa: 3 base +2 (último sigilo global Presa,
	// alpha-0.6.0); Exposto em p0 → +2 = 7.
	h.play(1, h.handInst(1, "VR-025"))
	h.pass(0)
	h.assertVit(0, 18)
	if g.State().Players[0].Exposto {
		t.Fatal("Exposto deveria ter sido consumido")
	}
	h.pass(1)

	// Rodada 2 começou: iniciativa virou, essência 4, eclipse persistiu em 0
	// (alpha-0.6.0: VR-015 não desloca mais o céu).
	s := g.State()
	if s.Round != 2 || s.Initiative != 1 || s.Phase != engine.PhaseStance {
		t.Fatalf("estado da rodada 2 inesperado: round=%d initiative=%d phase=%s", s.Round, s.Initiative, s.Phase)
	}
	if s.Players[0].Essence != 4 || s.Players[0].TempEssence != 0 {
		t.Fatalf("essência p0 na rodada 2: %d (+%d temp); esperado 4 (+0)", s.Players[0].Essence, s.Players[0].TempEssence)
	}
	h.assertEclipse(0)
	if len(s.Players[0].Trail) != 0 || len(s.RoundSigils) != 0 {
		t.Fatal("trilhas de Ressonância deveriam zerar na nova rodada")
	}

	// Concessão encerra.
	h.must(engine.Command{Player: 1, Kind: engine.CmdKindConcede})
	if !s.Over || s.Winner != 0 {
		t.Fatalf("fim de partida inesperado: over=%v winner=%d", s.Over, s.Winner)
	}

	compareGolden(t, "golden_match_01.json", g.Log)

	// Replay integral: mesmo config + mesmos comandos ⇒ mesmos eventos.
	g2, err := engine.Replay(g.Cfg, g.CommandLog)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	assertSameEvents(t, g.Log, g2.Log)
}

func compareGolden(t *testing.T, name string, events []engine.Event) {
	t.Helper()
	got, err := json.MarshalIndent(events, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden ausente (rode com -update): %v", err)
	}
	if string(want) != string(got) {
		t.Fatalf("log de eventos divergiu do golden %s (rode com -update para regenerar após revisar)", name)
	}
}

func assertSameEvents(t *testing.T, a, b []engine.Event) {
	t.Helper()
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatalf("logs de eventos divergem entre execução e replay")
	}
}
