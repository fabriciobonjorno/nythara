package engine

import (
	"fmt"
	"sort"
)

// PreconstructedDeck monta o deck oficial determinístico de um Campeão sob
// este Ruleset: prioriza a facção do Campeão (por id), completa com neutras e
// respeita limites de cópia. É a fonte única dos precons — produto, bots de
// treino e gate de balanceamento usam a mesma lista.
func (rs *Ruleset) PreconstructedDeck(championID string) ([]string, error) {
	champion := rs.Champions[championID]
	if champion == nil {
		return nil, fmt.Errorf("campeão desconhecido: %s", championID)
	}
	cards := append([]*CardDef{}, rs.CardList...)
	sort.Slice(cards, func(i, j int) bool {
		leftCore := cards[i].Faction == champion.Faction
		rightCore := cards[j].Faction == champion.Faction
		if leftCore != rightCore {
			return leftCore
		}
		return cards[i].ID < cards[j].ID
	})
	deck := make([]string, 0, DeckSize)
	for _, card := range cards {
		if card.Faction != champion.Faction && card.Faction != NeutralFaction {
			continue
		}
		copies := MaxCopies
		if card.Rarity == RarityLendaria {
			copies = MaxLegendary
		}
		for range copies {
			if len(deck) == DeckSize {
				break
			}
			deck = append(deck, card.ID)
		}
		if len(deck) == DeckSize {
			break
		}
	}
	if err := rs.ValidateDeck(championID, deck); err != nil {
		return nil, fmt.Errorf("precon %s: %w", championID, err)
	}
	return deck, nil
}
