package engine_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"veurubro/backend/internal/engine"
)

// O duelo longo (ADR-035) sustenta a partida em duas alavancas novas: a Guarda
// recebe o mesmo tipo de bônus fixo que o Assalto já tinha, e o formato declara
// a faixa de duração que promete. Ambas mudam resultado de jogo, então são
// testadas pelo efeito e não pela configuração.

func TestLongDuelGuardBonusReducesDamageAtResolution(t *testing.T) {
	// VR-001 (Assalto) contra VR-003 (Guarda) no ruleset publicado.
	deck0 := tacticalDeckWithFirst(t, "CH-VH-01", "VR-001")
	deck1 := tacticalDeckWithFirst(t, "CH-SO-01", "VR-003")
	cfg := engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 7301, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}}
	g, err := engine.NewGame(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-001")}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 1, "VR-003")}); err != nil {
		t.Fatal(err)
	}
	var resolved *engine.Event
	for index := range g.Log {
		if g.Log[index].Kind == engine.EvConfrontationResolved {
			resolved = &g.Log[index]
		}
	}
	if resolved == nil {
		t.Fatal("confronto não foi resolvido")
	}
	rs := engine.CompetitiveRuleset()
	base := rs.Effects.Cards["VR-003"].Guard.Prevent
	if want := base + engine.LongDuelGuardBonus; resolved.To != want {
		t.Fatalf("Prevenção aplicada %d, esperado base %d + bônus %d", resolved.To, base, engine.LongDuelGuardBonus)
	}
	if want := max(0, resolved.From-resolved.To); resolved.N != want {
		t.Fatalf("dano %d não corresponde a Poder %d menos Prevenção %d", resolved.N, resolved.From, resolved.To)
	}
	// Sem o bônus o mesmo confronto passaria mais dano: é essa diferença que
	// alonga a partida em vez de inflar Vitalidade.
	if resolved.N >= resolved.From-base {
		t.Fatalf("bônus de Guarda não reduziu o dano: %d contra %d sem bônus", resolved.N, resolved.From-base)
	}
}

// O teto de vazamento é o que torna defender confiável contra o topo da curva
// de Poder. Sem ele, um Assalto forte contra uma Guarda fraca atravessa inteiro
// e a duração da partida passa a depender da agressividade do baralho rival.
func TestGuardLeakCapLimitsDamageThroughACommittedGuard(t *testing.T) {
	const version = "test-leak-cap"
	rs := compileConfrontVariant(t, version, func(cfg *engine.ConfrontRulesConfig) {
		cfg.GuardBonus = 0 // isola o teto: Prevenção crua contra Poder cheio
		cfg.GuardLeakCap = 1
	})
	if err := engine.RegisterRuleset(rs); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engine.UnregisterRuleset(version) })

	deck0 := deckForVersion(t, rs, "CH-VH-01", "VR-001")
	deck1 := deckForVersion(t, rs, "CH-SO-01", "VR-003")
	g, err := engine.NewGame(engine.Config{RulesetVersion: version, Seed: 7303, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-001")}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 1, "VR-003")}); err != nil {
		t.Fatal(err)
	}
	var resolved *engine.Event
	for index := range g.Log {
		if g.Log[index].Kind == engine.EvConfrontationResolved {
			resolved = &g.Log[index]
		}
	}
	if resolved == nil {
		t.Fatal("confronto não foi resolvido")
	}
	if resolved.From-resolved.To <= 1 {
		t.Fatalf("cenário inválido: a subtração crua já daria %d, o teto não seria exercido", resolved.From-resolved.To)
	}
	if resolved.N != 1 {
		t.Fatalf("dano atravessou o teto: %d (Poder %d, Prevenção %d)", resolved.N, resolved.From, resolved.To)
	}
}

// Sem Guarda comprometida o teto não se aplica: quem não responde leva o golpe
// inteiro, que é o que mantém o Assalto relevante.
func TestGuardLeakCapDoesNotProtectWhoDoesNotGuard(t *testing.T) {
	deck0 := tacticalDeckWithFirst(t, "CH-VH-01", "VR-001")
	deck1 := tacticalDeckWithFirst(t, "CH-SO-01")
	g, err := engine.NewGame(engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 7304, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck0}, {ChampionID: "CH-SO-01", Deck: deck1}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: instanceByDef(t, g, 0, "VR-001")}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Apply(engine.Command{Player: 1, Kind: engine.CmdKindPass}); err != nil {
		t.Fatal(err)
	}
	for index := range g.Log {
		event := g.Log[index]
		if event.Kind != engine.EvConfrontationResolved {
			continue
		}
		if event.N != event.From {
			t.Fatalf("Assalto sem Guarda deveria passar inteiro: %d de %d", event.N, event.From)
		}
		return
	}
	t.Fatal("confronto sem Guarda não foi resolvido")
}

func TestLongDuelDeclaresDurationTargetAndKeepsOlderRulesetsIntact(t *testing.T) {
	current := engine.CompetitiveRuleset()
	if current.ConfrontRules.TargetMinP50Rounds != engine.LongDuelTargetMinP50Rounds ||
		current.ConfrontRules.TargetP95Rounds != engine.LongDuelTargetP95Rounds {
		t.Fatalf("faixa de duração não declarada: %+v", current.ConfrontRules)
	}
	if current.ConfrontRules.TargetMinP50Rounds >= current.ConfrontRules.TargetP95Rounds {
		t.Fatal("piso de duração deveria ser menor que o teto")
	}
	for _, version := range []string{engine.ConfrontRulesetVersion, engine.TacticalInitialRulesetVersion, engine.TacticalRulesetVersion} {
		rs, err := engine.RulesetByVersion(version)
		if err != nil {
			t.Fatal(err)
		}
		if rs.ConfrontRules.GuardBonus != 0 || rs.ConfrontRules.TargetP95Rounds != 0 {
			t.Fatalf("%s foi reinterpretado pelo duelo longo: %+v", version, rs.ConfrontRules)
		}
	}
}

// A compensação de iniciativa fica desligada na calibragem publicada, mas é uma
// alavanca de ritmo: se ela quebrar em silêncio, a próxima rotação de formato
// perde a ferramenta que corrige o viés de primeiro jogador.
func TestSecondPlayerBonusDrawGivesCardsToWhoDoesNotOpen(t *testing.T) {
	const version = "test-second-draw"
	rs := compileConfrontVariant(t, version, func(cfg *engine.ConfrontRulesConfig) {
		cfg.SecondPlayerBonusDraw = 2
	})
	if err := engine.RegisterRuleset(rs); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engine.UnregisterRuleset(version) })

	deck, err := rs.PreconstructedDeck("CH-VH-01")
	if err != nil {
		t.Fatal(err)
	}
	other, err := rs.PreconstructedDeck("CH-SO-01")
	if err != nil {
		t.Fatal(err)
	}
	g, err := engine.NewGame(engine.Config{RulesetVersion: version, Seed: 7302, SkipShuffle: true, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: deck}, {ChampionID: "CH-SO-01", Deck: other}}})
	if err != nil {
		t.Fatal(err)
	}
	// O jogador 0 abre e ainda compra no próprio turno; o 1 recebe as extras.
	opening := len(g.State().Players[0].Hand)
	if got, want := len(g.State().Players[1].Hand), engine.OpeningHandSize+2; got != want {
		t.Fatalf("mão de quem não abre: %d, esperado %d", got, want)
	}
	if opening >= len(g.State().Players[1].Hand) {
		t.Fatalf("compensação não favoreceu quem não abre: %d contra %d", opening, len(g.State().Players[1].Hand))
	}
}

// deckForVersion monta um baralho legal na versão pedida, com as cartas do
// teste no topo para caírem na mão inicial.
func deckForVersion(t *testing.T, rs *engine.Ruleset, avatar string, first ...string) []string {
	t.Helper()
	deck, err := rs.PreconstructedDeck(avatar)
	if err != nil {
		t.Fatal(err)
	}
	for target, id := range first {
		found := -1
		for index := target; index < len(deck); index++ {
			if deck[index] == id {
				found = index
				break
			}
		}
		if found < 0 {
			for index := target; index < len(deck); index++ {
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
		deck[target], deck[found] = deck[found], deck[target]
	}
	if err := rs.ValidateDeck(avatar, deck); err != nil {
		t.Fatalf("baralho de teste inválido: %v", err)
	}
	return deck
}

// compileConfrontVariant monta um ruleset efêmero a partir dos mesmos dados do
// produto, mudando apenas as alavancas pedidas.
func compileConfrontVariant(t *testing.T, version string, tune func(*engine.ConfrontRulesConfig)) *engine.Ruleset {
	t.Helper()
	dir := filepath.Join("data")
	read := func(name string) []byte {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	var file map[string]any
	if err := json.Unmarshal(read("effects_alpha.json"), &file); err != nil {
		t.Fatal(err)
	}
	cfg := engine.CompetitiveRuleset().ConfrontRules
	tune(&cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var rules map[string]any
	if err := json.Unmarshal(raw, &rules); err != nil {
		t.Fatal(err)
	}
	file["version"] = version
	file["mode"] = "confront"
	file["confront_rules"] = rules
	patched, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	rs, err := engine.CompileRuleset(version, read("cards_alpha.json"), read("champions_alpha.json"), patched)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}
