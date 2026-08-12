package engine

import "fmt"

const (
	NeutralFaction = "Errantes"
	MaxCopies      = 2
	MaxLegendary   = 1
	MinCoreFaction = 24 // cartas da facção do Campeão ou neutras
	MaxAlliedCards = 12
)

// ValidateDeck aplica as regras de construção do GDD §4 sob o ruleset
// embutido. No alpha, qualquer facção única pode ser a aliada da temporada.
func ValidateDeck(championID string, deck []string) error {
	return builtin.ValidateDeck(championID, deck)
}

// ValidateDeckForVersion valida sem reinterpretar o tamanho e as restrições
// de versões históricas.
func ValidateDeckForVersion(version, avatarID string, deck []string) error {
	rs, err := RulesetByVersion(version)
	if err != nil {
		return err
	}
	return rs.ValidateDeck(avatarID, deck)
}

// ValidateDeck aplica as regras de construção sob este Ruleset.
func (rs *Ruleset) ValidateDeck(championID string, deck []string) error {
	champ := rs.Champions[championID]
	if champ == nil {
		return fmt.Errorf("campeão desconhecido: %q", championID)
	}
	if rs.IsConfront() {
		return rs.validateConfrontDeck(deck)
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

func (rs *Ruleset) validateConfrontDeck(deck []string) error {
	if len(deck) != ConfrontDeckSize {
		return fmt.Errorf("deck com %d cartas; exigido %d", len(deck), ConfrontDeckSize)
	}
	counts := map[string]int{}
	types := map[CardType]int{}
	for _, id := range deck {
		def := rs.Cards[id]
		if def == nil {
			return fmt.Errorf("carta desconhecida: %q", id)
		}
		if def.Type != TypeAssalto && def.Type != TypeGuarda && def.Type != TypeRito {
			return fmt.Errorf("%s (%s): tipo fora do Modo Confronto", id, def.Name)
		}
		if def.Confront == nil || !def.Confront.Legal {
			reason := "efeito não compilado"
			if def.Confront != nil && def.Confront.Reason != "" {
				reason = def.Confront.Reason
			}
			return fmt.Errorf("%s (%s): %s", id, def.Name, reason)
		}
		counts[id]++
		types[def.Type]++
		limit := MaxCopies
		if def.Rarity == RarityLendaria {
			limit = MaxLegendary
		}
		if counts[id] > limit {
			return fmt.Errorf("%s (%s): máximo de %d cópia(s)", id, def.Name, limit)
		}
	}
	minimums := rs.ConfrontRules
	if types[TypeAssalto] < minimums.MinAssaults || types[TypeGuarda] < minimums.MinGuards || types[TypeRito] < minimums.MinRites {
		return fmt.Errorf("composição mínima: %d Assaltos, %d Guardas e %d Ritos (há %d/%d/%d)",
			minimums.MinAssaults, minimums.MinGuards, minimums.MinRites,
			types[TypeAssalto], types[TypeGuarda], types[TypeRito])
	}
	return nil
}
