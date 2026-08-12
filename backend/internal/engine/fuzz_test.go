package engine_test

import (
	"bytes"
	"testing"

	"veurubro/backend/internal/engine"
)

// FuzzRejectedCommandIsAtomic prova a fronteira autoritativa: qualquer
// intenção rejeitada preserva integralmente estado, eventos e histórico.
func FuzzRejectedCommandIsAtomic(f *testing.F) {
	for _, seed := range []uint64{0, 1, 2, 42, 1<<63 - 1} {
		f.Add(seed, []byte{0, 1, 2, 3, 4, 5, 6, 7})
	}
	f.Fuzz(func(t *testing.T, seed uint64, raw []byte) {
		if len(raw) < 8 {
			t.Skip()
		}
		g, err := engine.NewGame(engine.Config{Seed: seed, FirstPlayer: int(raw[0] % 2),
			Players: [2]engine.PlayerSetup{{ChampionID: "CH-CI-01", Deck: deckWith()},
				{ChampionID: "CH-CI-02", Deck: deckWith()}}})
		if err != nil {
			t.Fatal(err)
		}
		kinds := []engine.CommandKind{engine.CmdKindPlay, engine.CmdKindChoose, engine.CmdKindPass,
			engine.CmdKindUltimate, engine.CmdKindActivate, engine.CmdKindStance, engine.CmdKindMulligan}
		stances := []engine.Stance{engine.StanceArcano, engine.StancePredacao, engine.StanceVigilia, engine.Stance("invalida")}
		cmd := engine.Command{Player: int(raw[1]%4) - 1, Kind: kinds[int(raw[2])%len(kinds)],
			Card: "inst-inexistente", Cards: []string{"x", "x", "y"}, DecisionID: int(raw[3]),
			Stance: stances[int(raw[4])%len(stances)]}
		before, err := g.SnapshotJSON()
		if err != nil {
			t.Fatal(err)
		}
		logLen, commandLen := len(g.Log), len(g.CommandLog)
		if _, err = g.Apply(cmd); err == nil {
			return // comandos válidos têm suas próprias propriedades/simulações
		}
		after, snapErr := g.SnapshotJSON()
		if snapErr != nil {
			t.Fatal(snapErr)
		}
		if !bytes.Equal(before, after) || len(g.Log) != logLen || len(g.CommandLog) != commandLen {
			t.Fatalf("comando rejeitado mutou a partida: %+v err=%v", cmd, err)
		}
	})
}
