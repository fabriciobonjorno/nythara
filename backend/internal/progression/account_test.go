package progression

import (
	"testing"

	"veurubro/backend/internal/engine"
)

func TestAccountLevelIsDerivedAndCapped(t *testing.T) {
	tests := []struct {
		xp, level int
	}{
		{-1, 1}, {0, 1}, {99, 1}, {100, 2}, {219, 2}, {220, 3},
		{MaxAccountXP, 50}, {MaxAccountXP + 1_000_000, 50},
	}
	for _, test := range tests {
		if got := LevelForXP(test.xp); got != test.level {
			t.Fatalf("LevelForXP(%d)=%d; esperado %d", test.xp, got, test.level)
		}
	}
	max := AccountForXP(MaxAccountXP + 1)
	if max.XP != MaxAccountXP || max.LevelXPRequired != 0 {
		t.Fatalf("projeção no teto inesperada: %+v", max)
	}
}

func TestAccountXPComesFromMatchResult(t *testing.T) {
	tests := []struct {
		won, pvp bool
		want     int
	}{
		{false, false, 0}, {true, false, 0}, {false, true, 15}, {true, true, 30},
	}
	for _, test := range tests {
		if got := XPForMatch(test.won, test.pvp); got != test.want {
			t.Fatalf("XPForMatch(%t,%t)=%d; esperado %d", test.won, test.pvp, got, test.want)
		}
	}
}

func TestLegendaryScheduleStartsAfterNewcomerMatchRange(t *testing.T) {
	lowest := MaxAccountLevel
	for id, level := range LegendaryUnlockLevels() {
		if level < 2 || level > MaxAccountLevel {
			t.Fatalf("%s tem nível inválido %d", id, level)
		}
		if level < lowest {
			lowest = level
		}
	}
	if lowest <= MinAccountLevel+MaxMatchLevelGap {
		t.Fatalf("primeira Lendária no nível %d alcança a faixa de um iniciante", lowest)
	}
}

func TestEveryAlphaLegendaryHasExplicitUnlockLevel(t *testing.T) {
	schedule := LegendaryUnlockLevels()
	for _, card := range engine.CompetitiveRuleset().CardList {
		level, scheduled := schedule[card.ID]
		if card.Rarity == engine.RarityLendaria && !scheduled {
			t.Fatalf("Lendária %s sem nível explícito", card.ID)
		}
		if scheduled && card.Rarity != engine.RarityLendaria {
			t.Fatalf("%s agendada no nível %d mas tem raridade %s", card.ID, level, card.Rarity)
		}
	}
}

func TestLevelsCanMatchAtMostFiveLevelsApart(t *testing.T) {
	for _, test := range []struct {
		left, right int
		want        bool
	}{{1, 1, true}, {1, 6, true}, {1, 7, false}, {20, 15, true}, {20, 26, false}, {0, 1, false}, {50, 51, false}} {
		if got := LevelsCanMatch(test.left, test.right); got != test.want {
			t.Fatalf("LevelsCanMatch(%d,%d)=%t; esperado %t", test.left, test.right, got, test.want)
		}
	}
}
