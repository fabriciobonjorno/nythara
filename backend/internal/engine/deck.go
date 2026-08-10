package engine

import "fmt"

const (
	NeutralFaction  = "Errantes"
	MaxCopies       = 2
	MaxLegendary    = 1
	MinCoreFaction  = 24 // cartas da facção do Campeão ou neutras
	MaxAlliedCards  = 12
)

// ValidateDeck aplica as regras de construção do GDD §4 sob o ruleset
// embutido. No alpha, qualquer facção única pode ser a aliada da temporada.
func ValidateDeck(championID string, deck []string) error {
	return builtin.ValidateDeck(championID, deck)
}

// ValidateDeck aplica as regras de construção sob este Ruleset.
func (rs *Ruleset) ValidateDeck(championID string, deck []string) error {
	champ := rs.Champions[championID]
	if champ == nil {
		return fmt.Errorf("campeão desconhecido: %q", championID)
	}
	if len(deck) != DeckSize {
		return fmt.Errorf("deck com %d cartas; exigido %d", len(deck), DeckSize)
	}

	counts := map[string]int{}
	core := 0
	allied := map[string]int{}
	for _, id := range deck {
		def := rs.Cards[id]
		if def == nil {
			return fmt.Errorf("carta desconhecida: %q", id)
		}
		counts[id]++
		limit := MaxCopies
		if def.Rarity == RarityLendaria {
			limit = MaxLegendary
		}
		if counts[id] > limit {
			return fmt.Errorf("%s (%s): máximo de %d cópia(s)", id, def.Name, limit)
		}
		if def.Faction == champ.Faction || def.Faction == NeutralFaction {
			core++
		} else {
			allied[def.Faction]++
		}
	}
	if core < MinCoreFaction {
		return fmt.Errorf("pelo menos %d cartas devem ser da facção do Campeão ou neutras (há %d)", MinCoreFaction, core)
	}
	if len(allied) > 1 {
		return fmt.Errorf("apenas uma facção aliada é permitida (há %d)", len(allied))
	}
	for f, n := range allied {
		if n > MaxAlliedCards {
			return fmt.Errorf("máximo de %d cartas da facção aliada %s (há %d)", MaxAlliedCards, f, n)
		}
	}
	return nil
}
