package app

import (
	"testing"

	"veurubro/backend/internal/engine"
)

// Alpha 0.9: os 10 Avatares recebem baralhos de 30 cartas legais no Confronto.
func TestPreconsCoverAllChampionsAndValidate(t *testing.T) {
	precons, err := Precons()
	if err != nil {
		t.Fatal(err)
	}
	if len(precons) != 10 {
		t.Fatalf("precons: %d; esperado 10", len(precons))
	}
	seen := map[string]bool{}
	for _, precon := range precons {
		if seen[precon.ChampionID] {
			t.Fatalf("campeão duplicado: %s", precon.ChampionID)
		}
		seen[precon.ChampionID] = true
		total := 0
		var expanded []string
		for _, card := range precon.Cards {
			total += card.Quantity
			for i := 0; i < card.Quantity; i++ {
				expanded = append(expanded, card.CardID)
			}
		}
		if total != engine.ConfrontDeckSize {
			t.Fatalf("%s: %d cartas; esperado %d", precon.ChampionID, total, engine.ConfrontDeckSize)
		}
		if err := engine.ValidateDeckForVersion(engine.ConfrontRulesetVersion, precon.ChampionID, expanded); err != nil {
			t.Fatalf("%s: precon ilegal: %v", precon.ChampionID, err)
		}
	}
}
