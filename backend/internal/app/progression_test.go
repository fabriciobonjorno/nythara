package app

import (
	"context"
	"strings"
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

func TestRitualPoolOnlyTeachesConfrontMode(t *testing.T) {
	for _, ritual := range RitualPool {
		copy := strings.ToLower(ritual.Title + " " + ritual.Description)
		for _, removed := range []string{"eclipse", "sigilo", "trilha", "relíquia", "manifestação", "postura", "ultimate"} {
			if strings.Contains(copy, removed) {
				t.Fatalf("ritual %s ainda ensina mecânica removida %q: %s", ritual.ID, removed, copy)
			}
		}
	}
}

func TestStatsFromConfrontEventsPowerCurrentRituals(t *testing.T) {
	rs := engine.CompetitiveRuleset()
	var assault, guard, rite string
	for _, card := range rs.CardList {
		if card.Confront == nil || !card.Confront.Legal {
			continue
		}
		switch card.Type {
		case engine.TypeAssalto:
			if assault == "" {
				assault = card.ID
			}
		case engine.TypeGuarda:
			if guard == "" {
				guard = card.ID
			}
		case engine.TypeRito:
			if rite == "" {
				rite = card.ID
			}
		}
	}
	if assault == "" || guard == "" || rite == "" {
		t.Fatalf("pool competitivo incompleto: assalto=%q guarda=%q rito=%q", assault, guard, rite)
	}
	events := []engine.Event{
		{Kind: engine.EvCardPlayed, P: 0, Def: assault, Round: 1},
		{Kind: engine.EvCardPlayed, P: 1, Def: guard, Round: 1},
		{Kind: engine.EvCardPlayed, P: 0, Def: rite, Round: 1},
		{Kind: engine.EvConfrontationResolved, P: 0, S: "assault", N: 4, Round: 1},
		{Kind: engine.EvCardShattered, P: 1, Def: guard, Round: 1},
		{Kind: engine.EvConfrontationResolved, P: 0, S: "guard", Round: 2},
		{Kind: engine.EvCardShattered, P: 0, Def: assault, Round: 2},
		{Kind: engine.EvDamage, P: 1, N: 4, S: assault, Round: 1},
		{Kind: engine.EvDamage, P: 0, N: 2, S: "Pressão de Nythara", Round: 25},
		{Kind: engine.EvDamage, P: 1, N: 2, S: "Pressão de Nythara", Round: 25},
	}
	stats := StatsFromEvents(rs, events)
	if stats[0].AssaultsPlayed != 1 || stats[0].RitosResolved != 1 || stats[1].GuardsPlayed != 1 {
		t.Fatalf("tipos jogados incorretos: %+v", stats)
	}
	if stats[0].ConfrontsWon != 1 || stats[1].ConfrontsWon != 1 {
		t.Fatalf("vencedores de confronto incorretos: %+v", stats)
	}
	if stats[0].RivalCardsShattered != 1 || stats[1].RivalCardsShattered != 1 {
		t.Fatalf("estilhaços incorretos: %+v", stats)
	}
	if stats[0].DamageDealt != 4 || stats[1].DamageDealt != 0 {
		t.Fatalf("Pressão não pode contar como dano causado: %+v", stats)
	}
	if got := ritualProgressFor(RitualDef{ID: "win_confronts_5"}, stats[0], false, true); got != 1 {
		t.Fatalf("progresso de confrontos: %d", got)
	}
	if got := ritualProgressFor(RitualDef{ID: "shatter_rival_4"}, stats[0], false, true); got != 1 {
		t.Fatalf("progresso de estilhaços: %d", got)
	}
	if got := ritualProgressFor(RitualDef{ID: "play_assaults_6"}, stats[0], false, true); got != 1 {
		t.Fatalf("progresso de Assaltos: %d", got)
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

func TestMasteryXPOnlyComesFromHumanPvP(t *testing.T) {
	if got := MasteryXPFor(true, false); got != 0 {
		t.Fatalf("treino venceu e recebeu %d XP de maestria; esperado 0", got)
	}
	if got := MasteryXPFor(false, false); got != 0 {
		t.Fatalf("treino perdeu e recebeu %d XP de maestria; esperado 0", got)
	}
	if got := MasteryXPFor(false, true); got != 15 {
		t.Fatalf("derrota PvP recebeu %d XP de maestria; esperado 15", got)
	}
	if got := MasteryXPFor(true, true); got != 30 {
		t.Fatalf("vitória PvP recebeu %d XP de maestria; esperado 30", got)
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
	if entry.UserID != "humano-1" || !entry.Won || entry.MasteryXP != 0 || entry.AccountXP != 0 {
		t.Fatalf("entrada inesperada: %+v", entry)
	}
	if entry.RitualDay != "2026-08-10" {
		t.Fatalf("dia do ritual: %s", entry.RitualDay)
	}
}

func TestRecordFinishedPvPIsTheOnlySourceOfAccountXP(t *testing.T) {
	fake := &progressFake{}
	service := New(fake)
	winner := 0
	players := [2]FinishedPlayer{
		{UserID: "humano-1", ChampionID: "CH-VH-01"},
		{UserID: "humano-2", ChampionID: "CH-SO-01"},
	}
	if err := service.RecordFinishedMatch(context.Background(), "match-pvp", "pvp",
		engine.CompetitiveRulesetVersion, players, &winner, nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.captured; got == nil || len(got.Players) != 2 ||
		got.Players[0].AccountXP != 30 || got.Players[1].AccountXP != 15 {
		t.Fatalf("XP PvP inesperado: %+v", got)
	}

	if err := service.RecordFinishedMatch(context.Background(), "match-bot", "bot_custom",
		engine.CompetitiveRulesetVersion, players, &winner, nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.captured; got.Players[0].MasteryXP != 0 || got.Players[1].MasteryXP != 0 ||
		got.Players[0].AccountXP != 0 || got.Players[1].AccountXP != 0 || got.Ranked {
		t.Fatalf("modo não-PvP concedeu XP/rating: %+v", got)
	}
}

// Auditoria pós-0.8.1: perdas de sistema (Fadiga, Ruptura do Véu) e dano de
// carta própria não são "dano causado" — rituais de dano não podem progredir
// com eles.
func TestStatsFromEventsDamageAuthorship(t *testing.T) {
	rs := engine.Builtin()
	events := []engine.Event{
		// Assalto real do p0 contra p1: credita p0.
		{Seq: 0, Round: 3, Kind: "damage_dealt", P: 1, N: 4, Card: "p0-c05", Def: "VR-001"},
		// Fadiga do p1: ninguém causou.
		{Seq: 1, Round: 20, Kind: "damage_dealt", P: 1, N: 6, S: "Fadiga"},
		// Ruptura atinge os dois: ninguém causou.
		{Seq: 2, Round: 25, Kind: "damage_dealt", P: 0, N: 1, S: "Ruptura do Véu"},
		{Seq: 3, Round: 25, Kind: "damage_dealt", P: 1, N: 1, S: "Ruptura do Véu"},
		// Trono de Espinhos do p1 machuca o próprio p1: não credita p0.
		{Seq: 4, Round: 5, Kind: "damage_dealt", P: 1, N: 1, Card: "p1-c12", Def: "VR-088"},
		// Sangramento aplicado pelo p0 dispara em p1: autoria real, credita p0.
		{Seq: 5, Round: 6, Kind: "damage_dealt", P: 1, N: 2, S: "Sangramento"},
		// A Pressão do modo Confronto é perda de sistema, como Fadiga.
		{Seq: 6, Round: 25, Kind: "damage_dealt", P: 0, N: 2, S: "Pressão de Nythara"},
		{Seq: 7, Round: 25, Kind: "damage_dealt", P: 1, N: 2, S: "Pressão de Nythara"},
	}
	stats := StatsFromEvents(rs, events)
	if stats[0].DamageDealt != 6 {
		t.Fatalf("p0 causou %d; esperado 6 (4 do Assalto + 2 do Sangramento)", stats[0].DamageDealt)
	}
	if stats[1].DamageDealt != 0 {
		t.Fatalf("p1 causou %d; esperado 0 (Fadiga/Ruptura/auto-dano não contam)", stats[1].DamageDealt)
	}
}
