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
	if rs.IsConfront() {
		return rs.confrontPreconstructedDeck(championID)
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
		// Precons são o produto de entrada e a régua do gate de balanceamento:
		// só o Set 1 entra (ADR-032). Sets de expansão vivem no deckbuilding.
		if card.Set != 1 {
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
	// Cirurgia de precon por Campeão (ADR-031): substituições declaradas nos
	// dados do Campeão, aplicadas cópia a cópia. A validação integral abaixo
	// continua sendo o único juiz da legalidade do resultado.
	for i, id := range deck {
		if swap, ok := champion.PreconSwaps[id]; ok {
			if rs.Cards[swap] == nil {
				return nil, fmt.Errorf("precon %s: swap para carta desconhecida %q", championID, swap)
			}
			deck[i] = swap
		}
	}
	if err := rs.ValidateDeck(championID, deck); err != nil {
		return nil, fmt.Errorf("precon %s: %w", championID, err)
	}
	return deck, nil
}

func (rs *Ruleset) confrontPreconstructedDeck(avatarID string) ([]string, error) {
	byType := map[CardType][]*CardDef{}
	for _, card := range rs.CardList {
		if card.Confront == nil || !card.Confront.Legal || card.Rarity == RarityLendaria {
			continue
		}
		if card.Type == TypeAssalto || card.Type == TypeGuarda || card.Type == TypeRito {
			byType[card.Type] = append(byType[card.Type], card)
		}
	}
	deck := make([]string, 0, ConfrontDeckSize)
	for _, cardType := range []CardType{TypeAssalto, TypeGuarda, TypeRito} {
		cards := byType[cardType]
		sort.Slice(cards, func(i, j int) bool {
			if cards[i].Cost != cards[j].Cost {
				return cards[i].Cost < cards[j].Cost
			}
			return cards[i].ID < cards[j].ID
		})
		for _, card := range cards {
			deck = append(deck, card.ID, card.ID)
			if len(deck)%10 == 0 {
				break
			}
		}
	}
	if err := rs.ValidateDeck(avatarID, deck); err != nil {
		return nil, fmt.Errorf("baralho inicial Confronto: %w", err)
	}
	return deck, nil
}
