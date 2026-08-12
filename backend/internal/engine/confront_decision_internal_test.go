package engine

import (
	"bytes"
	"testing"
)

// Regressões de borda da máquina de decisão. Este teste fica no pacote
// interno para pedir uma decisão N=2 sem criar uma carta artificial no
// catálogo versionado.
func TestConfrontDecisionRejectsInvalidSelectionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		command func(*Decision) Command
		code    string
	}{
		{name: "menos que N", command: func(d *Decision) Command {
			return Command{Player: 0, Kind: CmdKindChoose, DecisionID: d.ID, Cards: []string{d.Options[0]}}
		}, code: ErrBadCommand},
		{name: "mais que N", command: func(d *Decision) Command {
			return Command{Player: 0, Kind: CmdKindChoose, DecisionID: d.ID,
				Cards: []string{d.Options[0], d.Options[1], d.Options[2]}}
		}, code: ErrBadCommand},
		{name: "carta duplicada", command: func(d *Decision) Command {
			return Command{Player: 0, Kind: CmdKindChoose, DecisionID: d.ID,
				Cards: []string{d.Options[0], d.Options[0]}}
		}, code: ErrBadCommand},
		{name: "id de decisão incorreto", command: func(d *Decision) Command {
			return Command{Player: 0, Kind: CmdKindChoose, DecisionID: d.ID + 1,
				Cards: []string{d.Options[0], d.Options[1]}}
		}, code: ErrBadCommand},
		{name: "opção não oferecida", command: func(d *Decision) Command {
			return Command{Player: 0, Kind: CmdKindChoose, DecisionID: d.ID,
				Cards: []string{d.Options[0], "p1-c00"}}
		}, code: ErrInvalidCard},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := decisionEdgeGame(t)
			g.confrontRequestDiscard(0, 2, nil, "edge-test")
			pending := g.s.Pending
			if pending == nil || pending.N != 2 || len(pending.Options) < 3 {
				t.Fatalf("decisão de teste inválida: %+v", pending)
			}
			beforeState, err := g.SnapshotJSON()
			if err != nil {
				t.Fatal(err)
			}
			beforeLog := len(g.Log)
			_, err = g.Apply(test.command(pending))
			if err == nil {
				t.Fatal("escolha inválida foi aceita")
			}
			commandErr, ok := err.(*CommandError)
			if !ok || commandErr.Code != test.code {
				t.Fatalf("erro=%v; esperado código %s", err, test.code)
			}
			afterState, snapErr := g.SnapshotJSON()
			if snapErr != nil {
				t.Fatal(snapErr)
			}
			if !bytes.Equal(beforeState, afterState) || len(g.Log) != beforeLog {
				t.Fatal("escolha inválida alterou estado ou log")
			}
		})
	}
}

func TestConfrontDecisionContinuationChecksWinAndEmitsDecisionID(t *testing.T) {
	g := decisionEdgeGame(t)
	g.s.NextDecID = 7
	g.s.Players[0].Vitality = 1
	g.s.Players[0].Deck = nil
	g.confrontRequestDiscard(0, 1, []Op{{Op: "draw", N: 1}}, "edge-test")
	pending := g.s.Pending
	if pending == nil || pending.ID != 8 || len(pending.Options) == 0 {
		t.Fatalf("decisão de teste inválida: %+v", pending)
	}

	if _, err := g.Apply(Command{Player: 0, Kind: CmdKindChoose,
		DecisionID: pending.ID, Cards: []string{pending.Options[0]}}); err != nil {
		t.Fatal(err)
	}
	if !g.s.Over || g.s.Winner != 1 || g.s.Phase != PhaseOver {
		t.Fatalf("Fadiga da continuação não encerrou a partida: over=%v winner=%d phase=%s",
			g.s.Over, g.s.Winner, g.s.Phase)
	}

	for i := len(g.Log) - 1; i >= 0; i-- {
		if g.Log[i].Kind == EvDecisionResolved {
			if g.Log[i].N != pending.ID {
				t.Fatalf("decision_resolved.n=%d, esperado ID %d", g.Log[i].N, pending.ID)
			}
			return
		}
	}
	t.Fatal("evento decision_resolved ausente")
}

func decisionEdgeGame(t *testing.T) *Game {
	t.Helper()
	rs, err := RulesetByVersion(DecisionRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	deck0, err := rs.PreconstructedDeck("CH-VH-01")
	if err != nil {
		t.Fatal(err)
	}
	deck1, err := rs.PreconstructedDeck("CH-SO-01")
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGame(Config{RulesetVersion: DecisionRulesetVersion, Seed: 9601,
		SkipShuffle: true, FirstPlayer: 0, Players: [2]PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck0},
			{ChampionID: "CH-SO-01", Deck: deck1},
		}})
	if err != nil {
		t.Fatal(err)
	}
	return g
}
