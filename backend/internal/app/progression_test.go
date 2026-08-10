package app

import (
	"context"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

func TestRitualsForDayAreDeterministicAndDistinct(t *testing.T) {
	a := ritualsForDay("2026-08-10", "user-1")
	b := ritualsForDay("2026-08-10", "user-1")
	if len(a) != DailyRituals {
		t.Fatalf("rituais: %d", len(a))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatal("sorteio deveria ser determinístico por (dia, usuário)")
		}
	}
	seen := map[string]bool{}
	for _, def := range a {
		if seen[def.ID] {
			t.Fatal("rituais do dia devem ser distintos")
		}
		seen[def.ID] = true
	}
	other := ritualsForDay("2026-08-11", "user-1")
	same := true
	for i := range a {
		if a[i].ID != other[i].ID {
			same = false
		}
	}
	if same {
		t.Fatal("dias diferentes deveriam variar o sorteio")
	}
}

func TestStatsFromRealMatchEvents(t *testing.T) {
	// Uma partida real curta: p0 joga um Rito e um Assalto que causa dano.
	cfg := engine.Config{Seed: 7, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-CI-01", Deck: testDeck36("VR-049", "VR-001")},
			{ChampionID: "CH-CI-01", Deck: testDeck36()},
		}}
	g, err := engine.NewGame(cfg)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(cmd engine.Command) {
		t.Helper()
		if _, err := g.Apply(cmd); err != nil {
			t.Fatalf("%+v: %v", cmd, err)
		}
	}
	apply(engine.Command{Player: 0, Kind: engine.CmdKindMulligan})
	apply(engine.Command{Player: 1, Kind: engine.CmdKindMulligan})
	apply(engine.Command{Player: 0, Kind: engine.CmdKindStance, Stance: engine.StanceVigilia})
	apply(engine.Command{Player: 1, Kind: engine.CmdKindStance, Stance: engine.StanceVigilia})
	var vr049 string
	for _, id := range g.State().Players[0].Hand {
		if g.State().Cards[id].Def == "VR-049" {
			vr049 = id
		}
	}
	apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: vr049})
	d := g.State().Pending
	apply(engine.Command{Player: 0, Kind: engine.CmdKindChoose, DecisionID: d.ID,
		Cards: []string{g.State().Players[0].Hand[len(g.State().Players[0].Hand)-1]}})
	apply(engine.Command{Player: 0, Kind: engine.CmdKindPass})
	apply(engine.Command{Player: 1, Kind: engine.CmdKindPass})
	var vr001 string
	for _, id := range g.State().Players[0].Hand {
		if g.State().Cards[id].Def == "VR-001" {
			vr001 = id
		}
	}
	apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: vr001})
	apply(engine.Command{Player: 1, Kind: engine.CmdKindPass})

	stats := StatsFromEvents(engine.Builtin(), g.Log)
	if stats[0].RitosResolved != 1 {
		t.Fatalf("ritos de p0: %d; esperado 1", stats[0].RitosResolved)
	}
	if stats[0].DamageDealt != 2 {
		t.Fatalf("dano de p0: %d; esperado 2", stats[0].DamageDealt)
	}
	if stats[1].DamageDealt != 0 {
		t.Fatalf("dano de p1: %d; esperado 0", stats[1].DamageDealt)
	}
}

func testDeck36(prefix ...string) []string {
	deck := append([]string{}, prefix...)
	for len(deck) < engine.DeckSize {
		deck = append(deck, "VR-006")
	}
	return deck
}

func TestEloDeltaProperties(t *testing.T) {
	if d := domain.EloDelta(1000, 1000, true); d != 16 {
		t.Fatalf("iguais, vitória: %d; esperado 16", d)
	}
	if d := domain.EloDelta(1000, 1000, false); d != -16 {
		t.Fatalf("iguais, derrota: %d; esperado -16", d)
	}
	if d := domain.EloDelta(1400, 1000, true); d <= 0 || d > 6 {
		t.Fatalf("favorito vencendo ganha pouco: %d", d)
	}
	if d := domain.EloDelta(1000, 1400, true); d < 26 {
		t.Fatalf("azarão vencendo ganha muito: %d", d)
	}
	if d := domain.EloDelta(2400, 1000, true); d != 1 {
		t.Fatalf("vitória nunca rende zero: %d", d)
	}
}

func TestMasteryLevelCurve(t *testing.T) {
	if MasteryLevel(0) != 1 || MasteryLevel(99) != 1 {
		t.Fatal("nível 1 até 99 XP")
	}
	if MasteryLevel(100) != 2 {
		t.Fatal("100 XP = nível 2")
	}
	if MasteryLevel(1_000_000) != 50 {
		t.Fatal("teto no nível 50")
	}
}

// fake mínimo: captura a gravação de progresso.
type progressFake struct {
	domain.Store
	captured *domain.MatchProgress
}

func (f *progressFake) ActiveSeason(context.Context) (domain.Season, error) {
	return domain.Season{ID: "season-1", Name: "Alpha 0.5"}, nil
}

func (f *progressFake) RecordMatchProgress(_ context.Context, progress domain.MatchProgress) (bool, error) {
	f.captured = &progress
	return true, nil
}

func TestRecordFinishedMatchBuildsProgress(t *testing.T) {
	fake := &progressFake{}
	service := NewWithClock(fake, func() time.Time {
		return time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	})
	winner := 0
	events := []engine.Event{
		{Kind: engine.EvCardPlayed, P: 0, Def: "VR-049", Round: 1},
		{Kind: engine.EvDamage, P: 1, N: 30, Round: 2},
	}
	players := [2]FinishedPlayer{
		{UserID: "humano-1", ChampionID: "CH-VH-01"},
		{UserID: domain.BotUserID, ChampionID: "CH-SO-01"},
	}
	err := service.RecordFinishedMatch(context.Background(), "match-1", "practice",
		engine.RulesetVersion, players, &winner, events)
	if err != nil {
		t.Fatal(err)
	}
	got := fake.captured
	if got == nil || len(got.Players) != 1 {
		t.Fatalf("bot não pode progredir; gravado: %+v", got)
	}
	if got.Ranked {
		t.Fatal("treino jamais é ranqueado")
	}
	entry := got.Players[0]
	if entry.UserID != "humano-1" || !entry.Won || entry.MasteryXP != MasteryXPFor(true, false) {
		t.Fatalf("entrada inesperada: %+v", entry)
	}
	if entry.RitualDay != "2026-08-10" {
		t.Fatalf("dia do ritual: %s", entry.RitualDay)
	}
}
