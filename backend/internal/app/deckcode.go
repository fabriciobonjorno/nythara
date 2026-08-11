package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// Códigos de deck — a moeda social dos jogos de duelo: qualquer lista vira
// uma string curta para colar no chat. Formato versionado por prefixo
// ("VR1."); o payload é JSON compacto em base64url. Importar NUNCA confia no
// código: o deck reconstruído passa pelo mesmo funil de validação e posse do
// SaveDeck (o servidor decide a legalidade, jamais o código).

const deckCodePrefix = "VR1."

type deckCodePayload struct {
	Version  string      `json:"v"`
	Champion string      `json:"c"`
	Cards    [][2]any    `json:"k"` // [id, quantidade] — compacto e estável
}

// EncodeDeckCode serializa um deck num código compartilhável e determinístico
// (cartas ordenadas por id: o mesmo deck gera sempre o mesmo código).
func EncodeDeckCode(deck domain.Deck) string {
	cards := append([]domain.DeckCard{}, deck.Cards...)
	sort.Slice(cards, func(i, j int) bool { return cards[i].CardID < cards[j].CardID })
	payload := deckCodePayload{Version: deck.RulesetVersion, Champion: deck.ChampionID}
	for _, card := range cards {
		payload.Cards = append(payload.Cards, [2]any{card.CardID, card.Quantity})
	}
	raw, _ := json.Marshal(payload)
	return deckCodePrefix + base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeDeckCode reconstrói campeão e cartas a partir de um código. Erros são
// de formato; legalidade/posse ficam com o SaveDeck.
func DecodeDeckCode(code string) (championID, rulesetVersion string, cards []domain.DeckCard, err error) {
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, deckCodePrefix) {
		return "", "", nil, fmt.Errorf("%w: código de deck não reconhecido (prefixo VR1. ausente)", domain.ErrInvalid)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, deckCodePrefix))
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: código de deck corrompido", domain.ErrInvalid)
	}
	var payload deckCodePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", nil, fmt.Errorf("%w: código de deck corrompido", domain.ErrInvalid)
	}
	if payload.Champion == "" || len(payload.Cards) == 0 {
		return "", "", nil, fmt.Errorf("%w: código de deck incompleto", domain.ErrInvalid)
	}
	for _, entry := range payload.Cards {
		id, okID := entry[0].(string)
		quantityFloat, okQ := entry[1].(float64)
		quantity := int(quantityFloat)
		if !okID || !okQ || id == "" || quantity < 1 || quantity > engine.DeckSize ||
			float64(quantity) != quantityFloat {
			return "", "", nil, fmt.Errorf("%w: código de deck com entrada inválida", domain.ErrInvalid)
		}
		cards = append(cards, domain.DeckCard{CardID: id, Quantity: quantity})
	}
	return payload.Champion, payload.Version, cards, nil
}

// DeckCode exporta o código de um deck do próprio jogador.
func (s *Service) DeckCode(ctx context.Context, principal domain.Principal, deckID string) (string, error) {
	deck, err := s.store.Deck(ctx, principal.UserID, deckID)
	if err != nil {
		return "", err
	}
	return EncodeDeckCode(deck), nil
}

// ImportDeck cria um deck a partir de um código. A versão embutida é apenas
// informativa: o deck nasce no ruleset ativo e falha com mensagem clara se
// alguma carta não existir nele.
func (s *Service) ImportDeck(ctx context.Context, principal domain.Principal,
	code, name, idempotencyKey string) (domain.Deck, bool, error) {
	championID, _, cards, err := DecodeDeckCode(code)
	if err != nil {
		return domain.Deck{}, false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Deck importado"
	}
	deck := domain.Deck{ChampionID: championID, Name: name, Cards: cards}
	return s.SaveDeck(ctx, principal, deck, nil, idempotencyKey, []byte(code))
}
