package engine_test

import (
	"encoding/json"
	"testing"

	"veurubro/backend/internal/engine"
)

// Decisões no Confronto (ADR-058). O que importa travar: a escolha muda estado,
// a mesa não aceita outra ação enquanto ela está aberta, resposta inválida não
// muta nada, e o replay reproduz a partida com a escolha dentro.

func decisionGame(t *testing.T, first string) *engine.Game {
	t.Helper()
	rs, err := engine.RulesetByVersion(engine.DecisionRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	deck0 := deckForVersion(t, rs, "CH-VH-01", first)
	deck1 := deckForVersion(t, rs, "CH-SO-01")
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.DecisionRulesetVersion,
		Seed: 8101, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestConfrontDiscardDecisionMovesCardAndRunsContinuation(t *testing.T) {
	// VR-049: descarte 1 escolhido, depois compre 2.
	g := decisionGame(t, "VR-049")
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	riteID := instanceByDef(t, g, 0, "VR-049")
	beforeRite := g.State()
	roundBefore := beforeRite.Round
	opponentHandBefore := len(beforeRite.Players[1].Hand)
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: riteID}); err != nil {
		t.Fatal(err)
	}
	pending := g.State().Pending
	if pending == nil {
		t.Fatal("a carta não abriu decisão")
	}
	if pending.Player != 0 || pending.N != 1 || len(pending.Options) == 0 {
		t.Fatalf("decisão malformada: %+v", pending)
	}
	if state := g.State(); state.Active != 0 || state.Phase != engine.PhaseRite || state.Round != roundBefore {
		t.Fatalf("Rito avançou o turno antes da escolha: active=%d phase=%s round=%d",
			state.Active, state.Phase, state.Round)
	} else if len(state.Players[1].Hand) != opponentHandBefore {
		t.Fatalf("rival comprou antes da escolha: mão=%d, esperado %d",
			len(state.Players[1].Hand), opponentHandBefore)
	} else if state.Cards[riteID].Zone != engine.ZoneClash {
		t.Fatalf("Rito saiu da mesa antes da escolha: %s", state.Cards[riteID].Zone)
	} else if state.PendingConfrontRite == nil || state.PendingConfrontRite.Inst != riteID {
		t.Fatalf("continuação do Rito não foi preservada: %+v", state.PendingConfrontRite)
	}

	// Com decisão aberta, a mesa recusa qualquer outra ação.
	before := len(g.Log)
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err == nil {
		t.Fatal("passar deveria ser recusado com decisão pendente")
	}
	if len(g.Log) != before {
		t.Fatal("comando recusado emitiu eventos")
	}

	// Escolha inválida também não muta nada.
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose,
		DecisionID: pending.ID, Cards: []string{"p1-c00"}}); err == nil {
		t.Fatal("carta fora das opções deveria ser recusada")
	}
	if len(g.Log) != before {
		t.Fatal("escolha inválida emitiu eventos")
	}

	chosen := pending.Options[0]
	handBefore := len(g.State().Players[0].Hand)
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose,
		DecisionID: pending.ID, Cards: []string{chosen}}); err != nil {
		t.Fatal(err)
	}
	if g.State().Pending != nil {
		t.Fatal("decisão continuou pendente depois de respondida")
	}
	if g.State().Cards[chosen].Zone != engine.ZoneDiscard {
		t.Fatalf("carta escolhida não foi para o descarte: %s", g.State().Cards[chosen].Zone)
	}
	if g.State().Cards[riteID].Zone != engine.ZoneDiscard {
		t.Fatalf("Rito não foi descartado depois da escolha: %s", g.State().Cards[riteID].Zone)
	}
	// Descartou 1 e comprou 2: a continuação rodou.
	if got := len(g.State().Players[0].Hand); got != handBefore-1+2 {
		t.Fatalf("mão após a escolha: %d, esperado %d", got, handBefore+1)
	}
	if state := g.State(); state.Active != 1 || state.Phase != engine.PhaseAssault || state.Round != roundBefore+1 {
		t.Fatalf("turno não avançou após finalizar o Rito: active=%d phase=%s round=%d",
			state.Active, state.Phase, state.Round)
	} else if len(state.Players[1].Hand) != opponentHandBefore+1 {
		t.Fatalf("rival não comprou ao receber o turno: mão=%d, esperado %d",
			len(state.Players[1].Hand), opponentHandBefore+1)
	}
}

func TestConfrontDecisionReplaysIdentically(t *testing.T) {
	g := decisionGame(t, "VR-049")
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-049")}); err != nil {
		t.Fatal(err)
	}
	pending := g.State().Pending
	if pending == nil {
		t.Fatal("a carta não abriu decisão")
	}
	pendingSnapshot, err := g.SnapshotJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := engine.RestoreSnapshot(g.Cfg, pendingSnapshot, g.Log, g.CommandLog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose,
		DecisionID: pending.ID, Cards: []string{pending.Options[1]}}); err != nil {
		t.Fatal(err)
	}
	restoredPending := restored.State().Pending
	if restoredPending == nil {
		t.Fatal("snapshot restaurado perdeu a decisão pendente")
	}
	if _, err := restored.Apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose,
		DecisionID: restoredPending.ID, Cards: []string{restoredPending.Options[1]}}); err != nil {
		t.Fatal(err)
	}
	wantRestored, _ := g.SnapshotJSON()
	gotRestored, _ := restored.SnapshotJSON()
	if string(wantRestored) != string(gotRestored) {
		t.Fatal("snapshot restaurado divergiu ao finalizar Rito pendente")
	}
	replayed, err := engine.Replay(g.Cfg, g.CommandLog)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(g.Log)
	got, _ := json.Marshal(replayed.Log)
	if string(want) != string(got) {
		t.Fatal("replay divergiu com decisão no meio da partida")
	}
	wantState, err := g.SnapshotJSON()
	if err != nil {
		t.Fatal(err)
	}
	gotState, err := replayed.SnapshotJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(wantState) != string(gotState) {
		t.Fatal("estado final do replay divergiu com decisão no meio da partida")
	}
}

func TestServedRulesetKeepsDecisionCardsOut(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    bool
	}{
		{engine.AvatarRulesetVersion, false},
		{engine.LongDuelRulesetVersion, false},
		{engine.DecisionRulesetVersion, true},
	} {
		rs, err := engine.RulesetByVersion(tc.version)
		if err != nil {
			t.Fatal(err)
		}
		if got := rs.Cards["VR-002"].Confront.Legal; got != tc.want {
			t.Fatalf("%s: VR-002 legal=%v, esperado %v", tc.version, got, tc.want)
		}
	}
}
