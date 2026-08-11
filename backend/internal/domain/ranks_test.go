package domain

import "testing"

func TestTierForRatingBands(t *testing.T) {
	cases := []struct {
		rating int
		key    string
		nextAt int
	}{
		{0, "errante", 900},
		{899, "errante", 900},
		{900, "iniciado", 1000},
		{1000, "lamina", 1100},   // rating inicial: Lâmina Velada
		{1099, "lamina", 1100},
		{1100, "guardiao", 1200},
		{1200, "arauto", 1350},
		{1349, "arauto", 1350},
		{1350, "soberano", 1500},
		{1500, "voz", 0},
		{2400, "voz", 0},
	}
	for _, c := range cases {
		tier := TierForRating(c.rating)
		if tier.Key != c.key || tier.NextAt != c.nextAt {
			t.Fatalf("rating %d: %s (próxima em %d); esperado %s (%d)",
				c.rating, tier.Key, tier.NextAt, c.key, c.nextAt)
		}
		if tier.Name == "" {
			t.Fatalf("patente %s sem nome", tier.Key)
		}
	}
}

func TestTiersMonotonicAndNamed(t *testing.T) {
	previous := -1
	seen := map[string]bool{}
	for _, tier := range rankTiers {
		if tier.MinRating <= previous {
			t.Fatalf("faixas fora de ordem em %s", tier.Key)
		}
		if seen[tier.Key] || seen[tier.Name] {
			t.Fatalf("patente duplicada: %s", tier.Key)
		}
		seen[tier.Key], seen[tier.Name] = true, true
		previous = tier.MinRating
	}
}

func TestMasteryTitleLadder(t *testing.T) {
	cases := map[int]string{
		1: "Aprendiz", 4: "Aprendiz", 5: "Adepto", 9: "Adepto",
		10: "Discípulo", 20: "Mestre", 35: "Avatar", 50: "Encarnação do Véu",
	}
	for level, want := range cases {
		if got := MasteryTitle(level); got != want {
			t.Fatalf("nível %d: %q; esperado %q", level, got, want)
		}
	}
}

func TestSeasonRewardsGrowWithTier(t *testing.T) {
	previous := -1
	for _, tier := range rankTiers {
		_, reward := SeasonRewardForRating(tier.MinRating)
		if reward <= previous {
			t.Fatalf("recompensa de %s (%d) não cresce sobre a anterior (%d)",
				tier.Key, reward, previous)
		}
		previous = reward
	}
	if _, reward := SeasonRewardForRating(1000); reward != 60 {
		t.Fatalf("rating inicial deveria valer 60 (Lâmina Velada); veio %d", reward)
	}
}
