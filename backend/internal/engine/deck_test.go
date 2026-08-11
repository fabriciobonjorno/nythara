package engine_test

import (
	"strings"
	"testing"

	"veurubro/backend/internal/engine"
)

// legalVhalDeck monta um deck válido para um Campeão da Casa Vhal a partir do
// Set 1 (o Set 2 tornaria o preenchimento maior que 36; a legalidade não
// depende de set — só a montagem determinística do teste).
func legalVhalDeck() []string {
	var deck []string
	for _, c := range engine.CardList {
		if c.Faction == "Casa Vhal" && c.Set == 1 {
			if c.Rarity == engine.RarityLendaria {
				deck = append(deck, c.ID)
			} else {
				deck = append(deck, c.ID, c.ID)
			}
		}
	}
	for _, c := range engine.CardList {
		if len(deck) == engine.DeckSize {
			break
		}
		if c.Faction == engine.NeutralFaction && c.Set == 1 && c.Rarity != engine.RarityLendaria {
			deck = append(deck, c.ID)
		}
	}
	return deck
}

func TestValidateDeckLegal(t *testing.T) {
	deck := legalVhalDeck()
	if err := engine.ValidateDeck("CH-VH-01", deck); err != nil {
		t.Fatalf("deck legal rejeitado: %v", err)
	}
}

func TestValidateDeckRejections(t *testing.T) {
	base := legalVhalDeck()

	cases := []struct {
		name    string
		mutate  func([]string) []string
		wantErr string
	}{
		{"tamanho errado", func(d []string) []string { return d[:35] }, "35 cartas"},
		{"3 cópias", func(d []string) []string {
			d = append([]string{}, d...)
			d[len(d)-1] = "VR-001" // VR-001 já aparece 2x
			return d
		}, "máximo de 2"},
		{"2 lendárias iguais", func(d []string) []string {
			d = append([]string{}, d...)
			d[len(d)-1] = "VR-012" // lendária Vhal já presente 1x
			return d
		}, "máximo de 1"},
		{"duas facções aliadas", func(d []string) []string {
			d = append([]string{}, d...)
			d[len(d)-1] = "VR-013" // Ordem Solara
			d[len(d)-2] = "VR-025" // Conclave Mirr
			return d
		}, "facção aliada"},
		{"campeão desconhecido", func(d []string) []string { return d }, "campeão desconhecido"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			champ := "CH-VH-01"
			if tc.name == "campeão desconhecido" {
				champ = "CH-XX-99"
			}
			err := engine.ValidateDeck(champ, tc.mutate(base))
			if err == nil {
				t.Fatalf("deck ilegal aceito")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("erro %q não contém %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateDeckCoreMinimum(t *testing.T) {
	// 24 Solara (12 aliadas seria acima do limite) — na verdade monta um deck
	// Vhal com 13 cartas Solara: estoura o limite de 12 aliadas.
	var deck []string
	for _, c := range engine.CardList {
		if c.Faction == "Casa Vhal" && c.Rarity != engine.RarityLendaria && len(deck) < 23 {
			deck = append(deck, c.ID, c.ID)
		}
	}
	deck = deck[:23]
	for _, c := range engine.CardList {
		if c.Faction == "Ordem Solara" && c.Rarity != engine.RarityLendaria && len(deck) < engine.DeckSize {
			deck = append(deck, c.ID, c.ID)
		}
	}
	deck = deck[:engine.DeckSize] // 23 core + 13 aliadas
	err := engine.ValidateDeck("CH-VH-01", deck)
	if err == nil {
		t.Fatal("deck com 13 cartas aliadas e 23 core aceito")
	}
}
