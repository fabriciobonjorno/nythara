package app

import (
	"context"
	"fmt"
	"sort"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/sim"
)

// Fase 9 — decks preconstruídos oficiais: um por Campeão, determinísticos,
// gerados do catálogo do ruleset embutido (mesmos precons usados no gate de
// balanceamento da Fase 6).

// PreconDeck é a descrição pública de um precon oficial.
type PreconDeck struct {
	ChampionID   string            `json:"champion_id"`
	ChampionName string            `json:"champion_name"`
	Name         string            `json:"name"`
	Cards        []domain.DeckCard `json:"cards"`
}

// Precons lista os 10 decks oficiais do Alpha, ordenados por Campeão.
func Precons() ([]PreconDeck, error) {
	rs := engine.Builtin()
	ids := make([]string, 0, len(rs.Champions))
	for id := range rs.Champions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]PreconDeck, 0, len(ids))
	for _, id := range ids {
		deck, err := sim.PreconstructedDeckFor(rs, id)
		if err != nil {
			return nil, err
		}
		out = append(out, PreconDeck{
			ChampionID:   id,
			ChampionName: rs.Champions[id].Name,
			Name:         "Precon — " + rs.Champions[id].Name,
			Cards:        collapseDeck(deck),
		})
	}
	return out, nil
}

func collapseDeck(cards []string) []domain.DeckCard {
	counts := map[string]int{}
	var order []string
	for _, id := range cards {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}
	out := make([]domain.DeckCard, 0, len(order))
	for _, id := range order {
		out = append(out, domain.DeckCard{CardID: id, Quantity: counts[id]})
	}
	return out
}

// CreatePreconDeck copia o precon oficial do Campeão para os decks do
// jogador, passando pela validação normal de SaveDeck.
func (s *Service) CreatePreconDeck(ctx context.Context, principal domain.Principal,
	championID, key string, rawBody []byte) (domain.Deck, bool, error) {
	rs := engine.Builtin()
	champion := rs.Champions[championID]
	if champion == nil {
		return domain.Deck{}, false, fmt.Errorf("%w: campeão desconhecido %q", domain.ErrInvalid, championID)
	}
	cards, err := sim.PreconstructedDeckFor(rs, championID)
	if err != nil {
		return domain.Deck{}, false, err
	}
	deck := domain.Deck{
		Name:       "Precon — " + champion.Name,
		ChampionID: championID,
		Cards:      collapseDeck(cards),
	}
	return s.SaveDeck(ctx, principal, deck, nil, key, rawBody)
}
