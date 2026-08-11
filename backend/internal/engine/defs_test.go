package engine_test

import (
	"testing"

	"veurubro/backend/internal/engine"
)

func TestCatalogLoads(t *testing.T) {
	if got := len(engine.CardList); got != 130 {
		t.Fatalf("cartas carregadas: %d; esperado 130 (Set 1 + Set 2)", got)
	}
	if got := len(engine.Champions); got != 10 {
		t.Fatalf("campeões carregados: %d; esperado 10", got)
	}

	byFaction := map[string]int{}
	legendaries := 0
	for _, c := range engine.CardList {
		byFaction[c.Faction]++
		if c.Rarity == engine.RarityLendaria {
			legendaries++
		}
	}
	// Set 1 (12×5 + 20) + Set 2 (8×5 + 10).
	want := map[string]int{
		"Casa Vhal": 20, "Ordem Solara": 20, "Conclave Mirr": 20,
		"Matilha Varka": 20, "Sínodo Cinéreo": 20, "Errantes": 30,
	}
	for f, n := range want {
		if byFaction[f] != n {
			t.Errorf("facção %s: %d cartas; esperado %d", f, byFaction[f], n)
		}
	}
	if legendaries != 8 {
		t.Errorf("lendárias: %d; esperado 8 (7 do Set 1 + O Peso de Nythara)", legendaries)
	}

	champsByFaction := map[string]int{}
	for _, ch := range engine.Champions {
		champsByFaction[ch.Faction]++
	}
	for _, f := range []string{"Casa Vhal", "Ordem Solara", "Conclave Mirr", "Matilha Varka", "Sínodo Cinéreo"} {
		if champsByFaction[f] != 2 {
			t.Errorf("facção %s: %d campeões; esperado 2", f, champsByFaction[f])
		}
	}
}

func TestImplementationReportGate(t *testing.T) {
	r := engine.ImplementationReport()
	if len(r.ImplementedCards)+len(r.MissingCards) != 130 {
		t.Fatalf("relatório não cobre as 130 cartas: %d + %d",
			len(r.ImplementedCards), len(r.MissingCards))
	}
	// Gate da Fase 1: pelo menos 20 cartas totalmente funcionais.
	if len(r.ImplementedCards) < 20 {
		t.Fatalf("apenas %d cartas implementadas; gate exige 20", len(r.ImplementedCards))
	}
	for _, id := range r.ImplementedCards {
		if engine.Cards[id] == nil {
			t.Errorf("implementação órfã: %s não existe no catálogo", id)
		}
	}
	if len(r.ImplementedChampions) < 5 {
		t.Errorf("apenas %d campeões com passiva implementada; esperado ≥5", len(r.ImplementedChampions))
	}
	t.Logf("cartas implementadas: %d/80; campeões: %d/10",
		len(r.ImplementedCards), len(r.ImplementedChampions))
}
