package engine_test

import (
	"encoding/json"
	"testing"

	"veurubro/backend/internal/engine"
)

// Decks de simulação: apenas cartas implementadas (18 distintas ×2).
func simDecks() ([]string, []string) {
	dup := func(ids ...string) []string {
		var d []string
		for _, id := range ids {
			d = append(d, id, id)
		}
		return d
	}
	d0 := dup("VR-001", "VR-002", "VR-006", "VR-018", "VR-029", "VR-030",
		"VR-034", "VR-036", "VR-037", "VR-040", "VR-043", "VR-047",
		"VR-049", "VR-055", "VR-058", "VR-063", "VR-068", "VR-076")
	d1 := dup("VR-013", "VR-014", "VR-015", "VR-017", "VR-020", "VR-025",
		"VR-026", "VR-031", "VR-032", "VR-035", "VR-039", "VR-042",
		"VR-053", "VR-060", "VR-066", "VR-069", "VR-072", "VR-074")
	return d0, d1
}

// TestRandomGamesEndAndReplayDeterministically roda partidas completas com o
// bot random-legal e verifica, a cada comando: nenhuma rejeição, invariantes
// de estado; ao final: a partida termina e o replay (config + command log)
// reproduz o log de eventos e o snapshot final bit a bit.
func TestRandomGamesEndAndReplayDeterministically(t *testing.T) {
	d0, d1 := simDecks()
	const games = 40
	const maxCommands = 2000

	totalCommands, totalRounds := 0, 0
	winners := [2]int{}

	for seed := uint64(1); seed <= games; seed++ {
		cfg := engine.Config{
			Seed:        seed,
			FirstPlayer: -1,
			Players: [2]engine.PlayerSetup{
				{ChampionID: "CH-MI-02", Deck: d0}, // Oren: scry pré-mulligan + ultimate com decisão
				{ChampionID: "CH-CI-01", Deck: d1}, // Voren: recuperação + Destilação Negra
			},
		}
		g, err := engine.NewGame(cfg)
		if err != nil {
			t.Fatalf("seed %d: NewGame: %v", seed, err)
		}
		bot := &engine.RandomBot{RNG: engine.NewRNG(seed*7919 + 13)}

		steps := 0
		for ; steps < maxCommands; steps++ {
			cmd, ok := bot.Next(g)
			if !ok {
				break
			}
			if _, err := g.Apply(cmd); err != nil {
				t.Fatalf("seed %d passo %d: bot gerou comando ilegal %+v: %v", seed, steps, cmd, err)
			}
			checkInvariants(t, g, seed, steps)
		}
		s := g.State()
		if !s.Over {
			t.Fatalf("seed %d: partida não terminou em %d comandos (rodada %d)", seed, maxCommands, s.Round)
		}
		winners[s.Winner]++
		totalCommands += steps
		totalRounds += s.Round

		// Replay integral.
		g2, err := engine.Replay(cfg, g.CommandLog)
		if err != nil {
			t.Fatalf("seed %d: replay: %v", seed, err)
		}
		ja, _ := json.Marshal(g.Log)
		jb, _ := json.Marshal(g2.Log)
		if string(ja) != string(jb) {
			t.Fatalf("seed %d: log de eventos do replay diverge", seed)
		}
		sa, _ := g.SnapshotJSON()
		sb, _ := g2.SnapshotJSON()
		if string(sa) != string(sb) {
			t.Fatalf("seed %d: snapshot final do replay diverge", seed)
		}
	}
	t.Logf("%d partidas: média %d comandos, %.1f rodadas; vitórias p0=%d p1=%d",
		games, totalCommands/games, float64(totalRounds)/games, winners[0], winners[1])
}

func checkInvariants(t *testing.T, g *engine.Game, seed uint64, step int) {
	t.Helper()
	s := g.State()
	fail := func(format string, args ...any) {
		t.Fatalf("seed %d passo %d: invariante violada: "+format, append([]any{seed, step}, args...)...)
	}

	if s.Eclipse < engine.EclipseMin || s.Eclipse > engine.EclipseMax {
		fail("eclipse fora do intervalo: %d", s.Eclipse)
	}
	for i := 0; i < 2; i++ {
		p := s.Players[i]
		if p.Essence < 0 || p.TempEssence < 0 || p.Essence > engine.EssenceMax {
			fail("essência inválida p%d: %d (+%d)", i, p.Essence, p.TempEssence)
		}
		if p.Vitality > p.MaxVitality {
			fail("vitalidade acima do máximo p%d: %d > %d", i, p.Vitality, p.MaxVitality)
		}
		if len(p.Trail) > engine.ResonanceCap+1 {
			fail("trilha de p%d estourou: %d", i, len(p.Trail))
		}
		// O limite de mão vale no Crepúsculo; a compra da rodada seguinte
		// pode elevar a mão a 8 até o próximo Crepúsculo.
		if s.Phase == engine.PhaseStance && len(p.Hand) > engine.HandLimit+1 {
			fail("mão de p%d acima do limite pós-compra: %d", i, len(p.Hand))
		}

		// Conservação de zonas: as 36 cartas do jogador existem em exatamente
		// uma zona (um Assalto pode estar transitório durante a janela de Guarda).
		seen := map[string]bool{}
		total := 0
		count := func(ids []string, zone engine.Zone) {
			for _, id := range ids {
				inst := s.Cards[id]
				if inst == nil || inst.Owner != i {
					fail("carta %q em zona de p%d com dono errado", id, i)
				}
				if seen[id] {
					fail("carta %q em duas zonas", id)
				}
				seen[id] = true
				total++
			}
		}
		count(p.Deck, engine.ZoneDeck)
		count(p.Hand, engine.ZoneHand)
		count(p.Discard, engine.ZoneDiscard)
		count(p.Exile, engine.ZoneExile)
		count(p.Relics, engine.ZonePlay)
		count(p.Manifs, engine.ZonePlay)
		want := engine.DeckSize
		// Assalto em resolução (cópias não têm carta física).
		if s.Guard != nil {
			if inst := s.Cards[s.Guard.AssaultInst]; inst != nil && inst.Owner == i {
				want--
			}
		}
		// Rito suspenso na janela de reação (VR-035).
		if s.RiteReact != nil && s.RiteReact.Caster == i {
			want--
		}
		// Rito cujo texto aguarda uma decisão antes da finalização (ADR-021).
		for _, pending := range s.PendingRites {
			if pending.Player == i && pending.Inst != "" {
				want--
			}
		}
		if total != want {
			fail("conservação de zonas p%d: %d cartas; esperado %d", i, total, want)
		}
	}
	if s.Over && s.Winner != 0 && s.Winner != 1 {
		fail("partida encerrada sem vencedor válido")
	}
}
