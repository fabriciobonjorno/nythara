package engine_test

import (
	"encoding/json"
	"strings"
	"testing"

	"veurubro/backend/internal/engine"
)

func confrontSetup(t *testing.T) (*engine.Ruleset, []string) {
	t.Helper()
	rs, err := engine.RulesetByVersion(engine.ConfrontRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	deck, err := rs.PreconstructedDeck("CH-VH-01")
	if err != nil {
		t.Fatal(err)
	}
	return rs, deck
}

func TestConfrontRejectsDeckWithoutMinimumTypeComposition(t *testing.T) {
	rs, _ := confrontSetup(t)
	deck := make([]string, 0, engine.ConfrontDeckSize)
	for _, card := range rs.CardList {
		if card.Type != engine.TypeAssalto || card.Confront == nil || !card.Confront.Legal {
			continue
		}
		copies := engine.MaxCopies
		if card.Rarity == engine.RarityLendaria {
			copies = engine.MaxLegendary
		}
		for range copies {
			if len(deck) < engine.ConfrontDeckSize {
				deck = append(deck, card.ID)
			}
		}
	}
	if len(deck) != engine.ConfrontDeckSize {
		t.Fatalf("pool de Assaltos insuficiente para o teste: %d", len(deck))
	}
	if err := rs.ValidateDeck("CH-VH-01", deck); err == nil || !strings.Contains(err.Error(), "composição mínima") {
		t.Fatalf("baralho mono-Assalto deveria falhar pela composição, recebeu %v", err)
	}
}

func TestConfrontImplementationReportAndStarterDeck(t *testing.T) {
	rs, deck := confrontSetup(t)
	legal, excluded := 0, 0
	byType := map[engine.CardType]int{}
	for _, entry := range rs.ConfrontImplementationReport() {
		if entry.Legal {
			legal++
			byType[rs.Cards[entry.CardID].Type]++
		} else {
			excluded++
			if entry.Reason == "" {
				t.Fatalf("%s foi excluída sem motivo", entry.CardID)
			}
		}
	}
	t.Logf("legais=%d excluídas=%d por tipo=%v", legal, excluded, byType)
	if legal < 60 {
		t.Fatalf("pool Confronto pequeno: %d legais, %d excluídas; esperado >=60", legal, excluded)
	}
	minimum := map[engine.CardType]int{engine.TypeAssalto: 10, engine.TypeGuarda: 10, engine.TypeRito: 8}
	for cardType, want := range minimum {
		if byType[cardType] < want {
			t.Fatalf("pool %s insuficiente: %d; esperado >=%d", cardType, byType[cardType], want)
		}
	}
	if len(deck) != engine.ConfrontDeckSize {
		t.Fatalf("baralho inicial: %d", len(deck))
	}
	if err := rs.ValidateDeck("CH-VH-01", deck); err != nil {
		t.Fatalf("baralho inicial ilegal: %v", err)
	}
}

func TestConfrontLegalPlayIDsExcludeRiteWithUnmetRequirement(t *testing.T) {
	_, deck := confrontSetup(t)
	for i, id := range deck {
		if id == "VR-067" {
			deck[0], deck[i] = deck[i], deck[0]
			break
		}
	}
	if deck[0] != "VR-067" {
		t.Fatal("baralho oficial não contém VR-067 para o teste")
	}
	game, err := engine.NewGame(engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion,
		Seed: 901, SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck}, {ChampionID: "CH-SO-01", Deck: append([]string{}, deck...)},
		}})
	if err != nil {
		t.Fatal(err)
	}
	var powder string
	for _, instanceID := range game.State().Players[0].Hand {
		if game.State().Cards[instanceID].Def == "VR-067" {
			powder = instanceID
			break
		}
	}
	if powder == "" {
		t.Fatal("VR-067 não veio na mão inicial determinística")
	}
	if _, err := game.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	for _, instanceID := range game.LegalPlayIDs(0) {
		if instanceID == powder {
			t.Fatal("VR-067 sem Véu rival foi anunciada como jogável")
		}
	}
}

func TestConfront091DoesNotReinterpret090(t *testing.T) {
	initial, err := engine.RulesetByVersion(engine.ConfrontInitialRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := engine.RulesetByVersion(engine.ConfrontRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	tacticalInitial, err := engine.RulesetByVersion(engine.TacticalInitialRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	current := engine.CompetitiveRuleset()
	if !initial.IsConfront() || !historical.IsConfront() || !current.IsConfront() {
		t.Fatal("as duas versões deveriam usar o fluxo Confronto")
	}
	if initial.ConfrontRules.PowerBonus != 0 || initial.ConfrontRules.DrawOnFirstTurn || initial.ConfrontRules.PressureStartTurn != 0 {
		t.Fatalf("alpha-0.9.0 foi reinterpretado: %+v", initial.ConfrontRules)
	}
	if historical.ConfrontRules.PowerBonus != engine.ConfrontPowerBonus || !historical.ConfrontRules.DrawOnFirstTurn || historical.ConfrontRules.PressureStartTurn != engine.ConfrontPressureStartTurn || historical.ConfrontRules.TacticalSeals {
		t.Fatalf("alpha-0.9.1 foi reinterpretado: %+v", historical.ConfrontRules)
	}
	if initial.Cards["VR-001"].Confront.Power+engine.ConfrontPowerBonus != historical.Cards["VR-001"].Confront.Power {
		t.Fatalf("perfil de Poder histórico/atual inesperado: %d/%d",
			initial.Cards["VR-001"].Confront.Power, historical.Cards["VR-001"].Confront.Power)
	}
	if !current.ConfrontRules.TacticalSeals || current.Version != engine.DecisionRulesetVersion {
		t.Fatalf("ruleset de decisões não publicado: %s %+v", current.Version, current.ConfrontRules)
	}
	if !tacticalInitial.ConfrontRules.TacticalSeals || tacticalInitial.ConfrontRules.ExposedPowerBonus != 0 || current.ConfrontRules.ExposedPowerBonus != 2 {
		t.Fatalf("snapshots táticos reinterpretados: inicial=%+v atual=%+v", tacticalInitial.ConfrontRules, current.ConfrontRules)
	}
	// O duelo longo (0.11.0) não pode reescrever o tático (0.10.2): a Guarda só
	// ganha bônus, e a Vitalidade só sobe, na versão nova.
	tactical, err := engine.RulesetByVersion(engine.TacticalRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	if tactical.ConfrontRules.GuardBonus != 0 || tactical.ConfrontRules.StartingVitality != engine.ConfrontStartingVitality ||
		tactical.ConfrontRules.PressureStartTurn != engine.ConfrontPressureStartTurn {
		t.Fatalf("alpha-0.10.2 foi reinterpretado: %+v", tactical.ConfrontRules)
	}
	longDuel, err := engine.RulesetByVersion(engine.LongDuelRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	if longDuel.ConfrontRules.GuardBonus != engine.LongDuelGuardBonus ||
		longDuel.ConfrontRules.StartingVitality != engine.LongDuelStartingVitality ||
		longDuel.ConfrontRules.PressureStartTurn != engine.LongDuelPressureStartTurn {
		t.Fatalf("duelo longo sem seus próprios números: %+v", longDuel.ConfrontRules)
	}
	// O ruleset de Avatares existe registrado e completo, mas não pode vazar
	// poder nem ajuste de carta para a versão que está no ar.
	avatar := current
	if len(avatar.ConfrontRules.ChampionPowers) != len(avatar.Champions) {
		t.Fatalf("nem todo Avatar tem poder: %d de %d",
			len(avatar.ConfrontRules.ChampionPowers), len(avatar.Champions))
	}
	if len(longDuel.ConfrontRules.ChampionPowers) != 0 || len(longDuel.ConfrontRules.CardAdjustments) != 0 {
		t.Fatalf("alpha-0.11.0 foi reinterpretado: %+v", longDuel.ConfrontRules)
	}
	if avatar.Cards["VR-062"].Cost != 0 || longDuel.Cards["VR-062"].Cost == 0 {
		t.Fatalf("ajuste de carta vazou entre versões: %d / %d",
			avatar.Cards["VR-062"].Cost, longDuel.Cards["VR-062"].Cost)
	}
	// A Guarda recebe o bônus; a Vitalidade inicial acompanha o formato.
	if got, want := longDuel.Cards["VR-003"].Confront.Prevention,
		tactical.Cards["VR-003"].Confront.Prevention+engine.LongDuelGuardBonus; got != want {
		t.Fatalf("bônus de Guarda não aplicado ao perfil: %d, esperado %d", got, want)
	}
	if historical.Cards["VR-018"].Confront.Legal || !current.Cards["VR-018"].Confront.Legal {
		t.Fatal("VR-018 deveria permanecer fora de 0.9.1 e entrar somente em 0.10.0")
	}
	if historical.Cards["VR-035"].Confront.Prevention != 0 || current.Cards["VR-035"].Confront.Prevention != 2 {
		t.Fatalf("contra-Rito histórico/tático inesperado: %d/%d",
			historical.Cards["VR-035"].Confront.Prevention, current.Cards["VR-035"].Confront.Prevention)
	}
}

func TestTacticalCatalogTextComesFromExecutableProfile(t *testing.T) {
	rs := engine.CompetitiveRuleset()
	for _, card := range rs.CardList {
		profile := card.Confront
		if profile == nil || !profile.Legal {
			continue
		}
		if profile.TacticalText == "" || profile.Role == "" || len(profile.Keywords) == 0 {
			t.Fatalf("%s sem apresentação tática completa: %+v", card.ID, profile)
		}
		if card.RulesText != profile.TacticalText {
			t.Fatalf("%s divergiu texto público/perfil", card.ID)
		}
		if strings.Contains(strings.ToLower(card.RulesText), "ruleset legado") {
			t.Fatalf("%s vazou texto de adaptação técnica ao jogador: %q", card.ID, card.RulesText)
		}
		if card.Type == engine.TypeGuarda && !profile.PreventAll && profile.Prevention < 1 {
			t.Fatalf("%s é Guarda competitiva sem Prevenção: %+v", card.ID, profile)
		}
	}
	if got := rs.Cards["VR-001"].Confront.Power; got != 7 {
		t.Fatalf("Poder público deve incluir bônus garantido após o custo: %d", got)
	}
	if text := rs.Cards["VR-015"].RulesText; !strings.Contains(text, "Guarda é proibida") || !strings.Contains(text, "+2 Poder") {
		t.Fatalf("VR-015 não explica o Selo da Guarda: %q", text)
	}
}

func tacticalDeckWithFirst(t *testing.T, avatar string, ids ...string) []string {
	t.Helper()
	rs := engine.CompetitiveRuleset()
	deck, err := rs.PreconstructedDeck(avatar)
	if err != nil {
		t.Fatal(err)
	}
	for targetIndex, id := range ids {
		found := -1
		for index := targetIndex; index < len(deck); index++ {
			if deck[index] == id {
				found = index
				break
			}
		}
		if found < 0 {
			for index := targetIndex; index < len(deck); index++ {
				if rs.Cards[deck[index]].Type == rs.Cards[id].Type {
					deck[index] = id
					found = index
					break
				}
			}
		}
		if found < 0 {
			t.Fatalf("não foi possível inserir %s no baralho", id)
		}
		deck[targetIndex], deck[found] = deck[found], deck[targetIndex]
	}
	if err := rs.ValidateDeck(avatar, deck); err != nil {
		t.Fatalf("baralho tático de teste inválido: %v", err)
	}
	return deck
}

func instanceByDef(t *testing.T, game *engine.Game, player int, def string) string {
	t.Helper()
	for _, id := range game.State().Players[player].Hand {
		if game.State().Cards[id].Def == def {
			return id
		}
	}
	t.Fatalf("%s não está na mão do jogador %d", def, player)
	return ""
}

func TestTacticalAssaultSealSkipsOnlyNextAssaultWindow(t *testing.T) {
	deck0 := tacticalDeckWithFirst(t, "CH-VH-01", "VR-018")
	deck1 := tacticalDeckWithFirst(t, "CH-SO-01")
	cfg := engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 5101, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}}
	g, err := engine.NewGame(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-018")}); err != nil {
		t.Fatal(err)
	}
	if g.State().Round != 2 || g.State().Active != 1 || g.State().Phase != engine.PhaseRite {
		t.Fatalf("Selo do Assalto deveria levar o rival direto ao Rito: turno=%d ativo=%d fase=%s",
			g.State().Round, g.State().Active, g.State().Phase)
	}
	if g.State().Players[1].AssaultSealUntil != 0 {
		t.Fatal("Selo do Assalto não foi consumido")
	}
	applied, consumed := false, false
	for _, event := range g.Log {
		applied = applied || (event.Kind == engine.EvStatusApplied && event.S == engine.ConfrontAssaultSeal)
		consumed = consumed || (event.Kind == engine.EvStatusExpired && event.S == engine.ConfrontAssaultSeal)
	}
	if !applied || !consumed {
		t.Fatalf("ciclo público do Selo ausente: aplicado=%v consumido=%v", applied, consumed)
	}
	replayed, err := engine.Replay(cfg, g.CommandLog)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(g.Log)
	got, _ := json.Marshal(replayed.Log)
	if string(want) != string(got) {
		t.Fatal("replay do Selo do Assalto divergiu")
	}
}

func TestTacticalExposedForbidsGuardAndAddsPowerBonus(t *testing.T) {
	deck0 := tacticalDeckWithFirst(t, "CH-VH-01", "VR-015", "VR-013")
	deck1 := tacticalDeckWithFirst(t, "CH-SO-01")
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 5102,
		SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass})
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-015")}); err != nil {
		t.Fatal(err)
	}
	if !g.State().Players[1].Exposto {
		t.Fatal("Marca do Caçador não aplicou Exposto")
	}
	_, _ = g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPass})
	_, _ = g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPass})
	before := g.State().Players[1].Vitality
	attack := instanceByDef(t, g, 0, "VR-013")
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: attack}); err != nil {
		t.Fatal(err)
	}
	if g.State().Phase != engine.PhaseRite || g.State().Active != 0 || g.State().Confront != nil {
		t.Fatalf("Exposto deveria resolver sem abrir Guarda: ativo=%d fase=%s", g.State().Active, g.State().Phase)
	}
	wantDamage := engine.CompetitiveRuleset().Cards["VR-013"].Confront.Power + engine.CompetitiveRuleset().ConfrontRules.ExposedPowerBonus
	if got := before - g.State().Players[1].Vitality; got != wantDamage {
		t.Fatalf("Exposto deve somar o bônus versionado: dano=%d esperado=%d", got, wantDamage)
	}
}

func TestTacticalCounterGuardPreventsCurrentRiteWindow(t *testing.T) {
	deck0 := tacticalDeckWithFirst(t, "CH-VH-01", "VR-013")
	deck1 := tacticalDeckWithFirst(t, "CH-SO-01", "VR-121")
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 5103,
		SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-013")}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 1, "VR-121")}); err != nil {
		t.Fatal(err)
	}
	if g.State().Round != 2 || g.State().Active != 1 || g.State().Phase != engine.PhaseAssault {
		t.Fatalf("Selo do Rito deveria entregar imediatamente o turno: turno=%d ativo=%d fase=%s",
			g.State().Round, g.State().Active, g.State().Phase)
	}
	if g.State().Players[0].RiteSealUntil != 0 {
		t.Fatal("Selo do Rito não foi consumido")
	}
	if got := engine.CompetitiveRuleset().Cards["VR-121"].Confront.Prevention; got != 2 {
		t.Fatalf("Aparo de Viela deveria ter Prevenção 2, recebeu %d", got)
	}
}

func TestConfrontFlowCostGuardResolutionAndReplay(t *testing.T) {
	rs, deck := confrontSetup(t)
	var assault, guard string
	for _, card := range rs.CardList {
		if card.Confront == nil || !card.Confront.Legal {
			continue
		}
		if assault == "" && card.Type == engine.TypeAssalto {
			assault = card.ID
		}
		if guard == "" && card.Type == engine.TypeGuarda {
			guard = card.ID
		}
	}
	putFirst := func(base []string, id string) []string {
		result := append([]string{}, base...)
		for i, item := range result {
			if item == id {
				result[0], result[i] = result[i], result[0]
				return result
			}
		}
		result[0] = id
		return result
	}
	cfg := engine.Config{RulesetVersion: engine.ConfrontRulesetVersion, Seed: 77,
		SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: putFirst(deck, assault)},
			{ChampionID: "CH-SO-01", Deck: putFirst(deck, guard)},
		}}
	g, err := engine.NewGame(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if g.State().Phase != engine.PhaseAssault || g.State().Active != 0 || len(g.State().Players[0].Hand) != 6 {
		t.Fatalf("abertura inválida: fase=%s ativo=%d mão=%d", g.State().Phase, g.State().Active, len(g.State().Players[0].Hand))
	}
	attackInst := g.State().Players[0].Hand[0]
	attackDef := rs.Cards[g.State().Cards[attackInst].Def]
	beforeAttack := g.State().Players[0].Vitality
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: attackInst}); err != nil {
		t.Fatal(err)
	}
	if g.State().Phase != engine.PhaseGuard || g.State().Confront == nil || g.State().Players[0].Vitality != beforeAttack-attackDef.Cost {
		t.Fatalf("Assalto não abriu Guarda/cobrou custo corretamente")
	}
	guardInst := g.State().Players[1].Hand[0]
	if _, err := g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: guardInst}); err != nil {
		t.Fatal(err)
	}
	if g.State().Phase != engine.PhaseRite || g.State().Active != 0 || g.State().Confront != nil {
		t.Fatalf("confronto não terminou na fase de Rito")
	}
	if g.State().Cards[attackInst].Zone != engine.ZoneDiscard || g.State().Cards[guardInst].Zone != engine.ZoneDiscard {
		t.Fatalf("cartas do confronto não foram descartadas")
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	if g.State().Active != 1 || g.State().Phase != engine.PhaseAssault || g.State().Round != 2 {
		t.Fatalf("turno não alternou: ativo=%d fase=%s turno=%d", g.State().Active, g.State().Phase, g.State().Round)
	}

	replayed, err := engine.Replay(cfg, g.CommandLog)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(g.Log)
	got, _ := json.Marshal(replayed.Log)
	if string(want) != string(got) {
		t.Fatal("replay do Confronto divergiu")
	}
}

func TestConfrontCannotSpendLastVitality(t *testing.T) {
	_, deck := confrontSetup(t)
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.ConfrontRulesetVersion,
		Seed: 1, SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck}, {ChampionID: "CH-SO-01", Deck: deck},
		}})
	if err != nil {
		t.Fatal(err)
	}
	g.State().Players[0].Vitality = 1
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: g.State().Players[0].Hand[0]}); err == nil {
		t.Fatal("engine aceitou carta sem preservar 1 de Vitalidade")
	}
}

func TestConfrontPressureHitsBothPlayersFromTurn25(t *testing.T) {
	_, deck := confrontSetup(t)
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.ConfrontRulesetVersion,
		Seed: 1, SkipShuffle: true, FirstPlayer: 0, Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-VH-01", Deck: deck}, {ChampionID: "CH-SO-01", Deck: deck},
		}})
	if err != nil {
		t.Fatal(err)
	}
	g.State().Round = engine.ConfrontPressureStartTurn - 1
	g.State().Players[0].Vitality = 3
	g.State().Players[1].Vitality = 3
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	if g.State().Round != engine.ConfrontPressureStartTurn || g.State().Players[0].Vitality != 1 || g.State().Players[1].Vitality != 1 {
		t.Fatalf("pressão não atingiu ambos: turno=%d vida=%d/%d", g.State().Round,
			g.State().Players[0].Vitality, g.State().Players[1].Vitality)
	}
	found := false
	for _, event := range g.Log {
		if event.Kind == engine.EvStatusApplied && event.S == "Pressão de Nythara" && event.N == 2 {
			found = true
		}
	}
	if !found {
		t.Fatal("evento público da pressão não foi emitido")
	}
}
