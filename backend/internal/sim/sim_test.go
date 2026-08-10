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
	for _, champion := range championIDs() {
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
	if !strings.Contains(string(raw), engine.RulesetVersion) || !strings.Contains(string(raw), "Win rate por Campeão") {
		t.Fatalf("relatório Markdown incompleto: %s", raw)
	}
}
