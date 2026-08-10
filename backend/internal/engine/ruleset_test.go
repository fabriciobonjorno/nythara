package engine_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"veurubro/backend/internal/engine"
)

// builtinPayload devolve os três documentos do ruleset embutido, para os
// testes derivarem versões modificadas.
func builtinPayload(t *testing.T) (cards, champs, effects []byte) {
	t.Helper()
	var err error
	if cards, err = json.Marshal(engine.CardList); err != nil {
		t.Fatal(err)
	}
	var champList []*engine.ChampionDef
	for _, id := range []string{"CH-VH-01", "CH-VH-02", "CH-SO-01", "CH-SO-02", "CH-MI-01",
		"CH-MI-02", "CH-VA-01", "CH-VA-02", "CH-CI-01", "CH-CI-02"} {
		champList = append(champList, engine.Champions[id])
	}
	if champs, err = json.Marshal(champList); err != nil {
		t.Fatal(err)
	}
	if effects, err = json.Marshal(engine.Effects); err != nil {
		t.Fatal(err)
	}
	return cards, champs, effects
}

// TestCompileAndPlayModifiedRuleset publica uma versão com VR-013 nerfada
// (dano 1) e prova que: a nova versão executa com o novo valor, o embutido
// segue intacto, e cada partida faz replay sob sua própria versão.
func TestCompileAndPlayModifiedRuleset(t *testing.T) {
	cards, champs, effects := builtinPayload(t)
	patched := bytes.Replace(effects,
		[]byte(`"VR-013":{"assault":{"damage":2,`),
		[]byte(`"VR-013":{"assault":{"damage":1,`), 1)
	if bytes.Equal(patched, effects) {
		t.Fatal("patch do VR-013 não encontrou o alvo — formato mudou?")
	}

	const version = "test-nerf-vr013"
	rs, err := engine.CompileRuleset(version, cards, champs, patched)
	if err != nil {
		t.Fatalf("CompileRuleset: %v", err)
	}
	if err := engine.RegisterRuleset(rs); err != nil {
		t.Fatalf("RegisterRuleset: %v", err)
	}
	t.Cleanup(func() { engine.UnregisterRuleset(version) })

	// Registrar de novo o mesmo objeto é idempotente; conteúdo diferente não.
	if err := engine.RegisterRuleset(rs); err != nil {
		t.Fatalf("re-registro idempotente falhou: %v", err)
	}
	other, _ := engine.CompileRuleset(version, cards, champs, effects)
	if err := engine.RegisterRuleset(other); err == nil {
		t.Fatal("registro de conteúdo divergente sob a mesma versão deveria falhar")
	}

	play := func(rulesetVersion string) (*engine.Game, int) {
		t.Helper()
		cfg := engine.Config{
			RulesetVersion: rulesetVersion, Seed: 5, SkipShuffle: true, FirstPlayer: 0,
			Players: [2]engine.PlayerSetup{
				{ChampionID: "CH-CI-01", Deck: deckWith("VR-013")},
				{ChampionID: "CH-CI-01", Deck: deckWith()},
			},
		}
		g, err := engine.NewGame(cfg)
		if err != nil {
			t.Fatal(err)
		}
		h := &harness{t: t, g: g}
		h.keepAll()
		h.stances(engine.StanceArcano, engine.StanceArcano)
		h.bothPassRite()
		h.play(0, h.handInst(0, "VR-013"))
		h.pass(1)
		return g, 30 - g.State().Players[1].Vitality
	}

	gOld, dmgOld := play("") // embutido
	gNew, dmgNew := play(version)
	if dmgOld != 3 {
		t.Fatalf("embutido: dano %d; esperado 3 (2 + eclipse ≤0)", dmgOld)
	}
	if dmgNew != 2 {
		t.Fatalf("versão nerfada: dano %d; esperado 2 (1 + eclipse ≤0)", dmgNew)
	}

	// Replays respeitam a versão gravada na config de cada partida.
	for _, g := range []*engine.Game{gOld, gNew} {
		replayed, err := engine.Replay(g.Cfg, g.CommandLog)
		if err != nil {
			t.Fatalf("replay %s: %v", g.Cfg.RulesetVersion, err)
		}
		a, _ := json.Marshal(g.Log)
		b, _ := json.Marshal(replayed.Log)
		if !bytes.Equal(a, b) {
			t.Fatalf("replay divergiu sob %s", g.Cfg.RulesetVersion)
		}
	}
}

func TestUnknownRulesetRejected(t *testing.T) {
	_, err := engine.NewGame(engine.Config{
		RulesetVersion: "nunca-publicado",
		Players: [2]engine.PlayerSetup{
			{ChampionID: "CH-CI-01", Deck: deckWith()},
			{ChampionID: "CH-CI-01", Deck: deckWith()},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "não registrado") {
		t.Fatalf("esperava rejeição de ruleset desconhecido; err=%v", err)
	}
}

func TestCompileRulesetRejectsInvalidEffects(t *testing.T) {
	cards, champs, effects := builtinPayload(t)
	broken := bytes.Replace(effects, []byte(`"op":"draw"`), []byte(`"op":"explodir"`), 1)
	if _, err := engine.CompileRuleset("test-quebrado", cards, champs, broken); err == nil {
		t.Fatal("efeitos inválidos deveriam falhar a compilação")
	}
	// Cobertura: efeito órfão (carta fora do catálogo) também falha.
	orphan := bytes.Replace(effects, []byte(`"VR-013"`), []byte(`"VR-999"`), 1)
	if _, err := engine.CompileRuleset("test-orfao", cards, champs, orphan); err == nil {
		t.Fatal("efeito de carta inexistente deveria falhar a compilação")
	}
}
