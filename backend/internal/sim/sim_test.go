package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"veurubro/backend/internal/engine"
)

func TestPreconstructedDecksAreLegal(t *testing.T) {
	for _, champion := range championIDs(engine.Builtin()) {
		deck, err := PreconstructedDeck(champion)
		if err != nil {
			t.Fatalf("precon %s: %v", champion, err)
		}
		if len(deck) != engine.DeckSize {
			t.Fatalf("precon %s tem %d cartas", champion, len(deck))
		}
		if err := engine.ValidateDeck(champion, deck); err != nil {
			t.Fatalf("precon %s ilegal: %v", champion, err)
		}
	}
}

func TestRunIsDeterministicAcrossWorkerCounts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Games = 400
	cfg.Workers = 1
	one, err := Run(cfg)
	if err != nil {
		t.Fatalf("worker único: %v (falhas: %+v)", err, one.Failures)
	}
	cfg.Workers = 4
	four, err := Run(cfg)
	if err != nil {
		t.Fatalf("quatro workers: %v (falhas: %+v)", err, four.Failures)
	}
	one.Config.Workers = 0
	four.Config.Workers = 0
	if !reflect.DeepEqual(one, four) {
		left, _ := json.Marshal(one)
		right, _ := json.Marshal(four)
		t.Fatalf("relatórios divergiram por paralelismo\n1: %s\n4: %s", left, right)
	}
}

func TestRandomVersusHeuristicHealthAndReports(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Games = 200
	cfg.Workers = 2
	cfg.Bot0 = BotRandom
	cfg.Bot1 = BotHeuristic
	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("simulação: %v (falhas: %+v)", err, report.Failures)
	}
	if report.Health.Completed != cfg.Games || len(report.Champions) != len(engine.Champions) {
		t.Fatalf("cobertura incompleta: health=%+v champions=%d", report.Health, len(report.Champions))
	}
	ended := 0
	for _, reason := range report.EndReasons {
		if reason.Reason == "" || reason.Games == 0 {
			t.Fatalf("causa de término inválida: %+v", reason)
		}
		ended += reason.Games
	}
	if ended != cfg.Games {
		t.Fatalf("causas de término cobrem %d/%d partidas", ended, cfg.Games)
	}
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "balance.json")
	markdownPath := filepath.Join(dir, "balance.md")
	if err := WriteJSON(jsonPath, report); err != nil {
		t.Fatal(err)
	}
	if err := WriteMarkdown(markdownPath, report); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), engine.CompetitiveRulesetVersion) || !strings.Contains(string(raw), "Win rate por Campeão") {
		t.Fatalf("relatório Markdown incompleto: %s", raw)
	}
}

func TestVariedGateExercisesEveryCard(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Games = 5000
	cfg.DeckMode = DeckVaried
	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("gate variado: %v (falhas: %+v)", err, report.Failures)
	}
	if len(report.Cards) != len(engine.CardList) {
		t.Fatalf("relatório cobriu %d/%d cartas", len(report.Cards), len(engine.CardList))
	}
	for _, card := range report.Cards {
		definition := engine.CompetitiveRuleset().Cards[card.ID]
		if definition.Confront == nil || !definition.Confront.Legal {
			continue
		}
		if card.InclusionGames == 0 || card.DrawnGames == 0 || card.PlayedGames == 0 {
			t.Errorf("%s não foi exercitada ponta a ponta: inclusão=%d compra=%d uso=%d", card.ID,
				card.InclusionGames, card.DrawnGames, card.PlayedGames)
		}
	}
}
