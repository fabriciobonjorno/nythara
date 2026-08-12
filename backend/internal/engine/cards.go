package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/cards_alpha.json
var cardsAlphaJSON []byte

//go:embed data/champions_alpha.json
var championsAlphaJSON []byte

// Sigil, tipos e enums do GDD. Os valores em string preservam a grafia dos
// arquivos de dados (com acentos), que são a fonte de verdade do catálogo.
type Sigil string

const (
	SigilPresa   Sigil = "Presa"
	SigilSol     Sigil = "Sol"
	SigilEspelho Sigil = "Espelho"
	SigilGarra   Sigil = "Garra"
	SigilCinza   Sigil = "Cinza"
	SigilCoroa   Sigil = "Coroa"
)

type CardType string

const (
	TypeAssalto      CardType = "Assalto"
	TypeGuarda       CardType = "Guarda"
	TypeRito         CardType = "Rito"
	TypeReliquia     CardType = "Relíquia"
	TypeManifestacao CardType = "Manifestação"
)

type Rarity string

const (
	RarityComum    Rarity = "Comum"
	RarityIncomum  Rarity = "Incomum"
	RarityRara     Rarity = "Rara"
	RarityEpica    Rarity = "Épica"
	RarityLendaria Rarity = "Lendária"
)

var validSigils = map[Sigil]bool{
	SigilPresa: true, SigilSol: true, SigilEspelho: true,
	SigilGarra: true, SigilCinza: true, SigilCoroa: true,
}

var validTypes = map[CardType]bool{
	TypeAssalto: true, TypeGuarda: true, TypeRito: true,
	TypeReliquia: true, TypeManifestacao: true,
}

var validRarities = map[Rarity]bool{
	RarityComum: true, RarityIncomum: true, RarityRara: true,
	RarityEpica: true, RarityLendaria: true,
}

var validFactions = map[string]bool{
	"Casa Vhal": true, "Ordem Solara": true, "Conclave Mirr": true,
	"Matilha Varka": true, "Sínodo Cinéreo": true, "Errantes": true,
}

// CardDef é a definição estática de uma carta (catálogo alpha).
type CardDef struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Faction      string   `json:"faction"`
	Type         CardType `json:"type"`
	Rarity       Rarity   `json:"rarity"`
	Cost         int      `json:"cost"`
	EclipseShift int      `json:"eclipse_shift"`
	Sigil        Sigil    `json:"sigil"`
	RulesText    string   `json:"rules_text"`
	Flavor       string   `json:"flavor"`
	DesignRole   string   `json:"design_role"`
	// Set identifica a coleção: 1 = núcleo (precons), 2 = expansão
	// (deckbuilding). 0 = fora de sets (drafts do admin) — nunca entra em
	// precon (ADR-032).
	Set int `json:"set,omitempty"`
	// Confront descreve a implementação desta carta no ruleset alpha-0.9.x.
	// Ausente nos payloads históricos; preenchido ao compilar o modo atual.
	Confront *ConfrontCardProfile `json:"confront,omitempty"`
}

// ConfrontCardProfile é informação pública e executável do Modo Confronto.
// Legal=false sempre traz o motivo explícito da exclusão do pool. Texto,
// papel e palavras-chave são derivados da mesma DSL que a engine executa.
type ConfrontCardProfile struct {
	Legal         bool     `json:"legal"`
	Adapted       bool     `json:"adapted,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Power         int      `json:"power,omitempty"`
	PowerMax      int      `json:"power_max,omitempty"`
	Prevention    int      `json:"prevention,omitempty"`
	PreventionMax int      `json:"prevention_max,omitempty"`
	PreventAll    bool     `json:"prevent_all,omitempty"`
	Defendable    bool     `json:"defendable,omitempty"`
	TacticalText  string   `json:"tactical_text,omitempty"`
	Role          string   `json:"role,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
}

// ChampionDef é a definição estática de um Campeão.
type ChampionDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Faction     string `json:"faction"`
	Vitality    int    `json:"vitality"`
	Passive     string `json:"passive"`
	Ultimate    string `json:"ultimate"`
	EclipseForm string `json:"eclipse_form"`
	// PreconSwaps ajusta o precon DESTE Campeão: cada cópia da carta-chave é
	// substituída pela carta-valor (ADR-031 — cirurgia de precon separa a
	// força do deck da força do kit sem tocar em regra). Dados, não código.
	PreconSwaps map[string]string `json:"precon_swaps,omitempty"`
}

// Cards e Champions são os catálogos carregados dos dados embutidos.
var (
	Cards     map[string]*CardDef
	CardList  []*CardDef
	Champions map[string]*ChampionDef
)

func loadCatalog(cardsRaw, champsRaw []byte) (map[string]*CardDef, []*CardDef, map[string]*ChampionDef, error) {
	var list []*CardDef
	if err := json.Unmarshal(cardsRaw, &list); err != nil {
		return nil, nil, nil, fmt.Errorf("cards_alpha.json: %w", err)
	}
	cards := make(map[string]*CardDef, len(list))
	for _, c := range list {
		if err := validateCardDef(c); err != nil {
			return nil, nil, nil, err
		}
		if cards[c.ID] != nil {
			return nil, nil, nil, fmt.Errorf("carta duplicada: %s", c.ID)
		}
		cards[c.ID] = c
	}

	var champList []*ChampionDef
	if err := json.Unmarshal(champsRaw, &champList); err != nil {
		return nil, nil, nil, fmt.Errorf("champions_alpha.json: %w", err)
	}
	champs := make(map[string]*ChampionDef, len(champList))
	for _, ch := range champList {
		if err := validateChampionDef(ch); err != nil {
			return nil, nil, nil, err
		}
		if champs[ch.ID] != nil {
			return nil, nil, nil, fmt.Errorf("campeão duplicado: %s", ch.ID)
		}
		champs[ch.ID] = ch
	}
	return cards, list, champs, nil
}

func validateCardDef(c *CardDef) error {
	switch {
	case c.ID == "":
		return fmt.Errorf("carta sem id")
	case c.Name == "":
		return fmt.Errorf("%s: sem nome", c.ID)
	case !validFactions[c.Faction]:
		return fmt.Errorf("%s: facção inválida %q", c.ID, c.Faction)
	case !validTypes[c.Type]:
		return fmt.Errorf("%s: tipo inválido %q", c.ID, c.Type)
	case !validRarities[c.Rarity]:
		return fmt.Errorf("%s: raridade inválida %q", c.ID, c.Rarity)
	case c.Cost < 0 || c.Cost > 10:
		return fmt.Errorf("%s: custo fora do intervalo: %d", c.ID, c.Cost)
	case c.EclipseShift < -2 || c.EclipseShift > 2:
		return fmt.Errorf("%s: eclipse_shift fora do intervalo: %d", c.ID, c.EclipseShift)
	case !validSigils[c.Sigil]:
		return fmt.Errorf("%s: sigilo inválido %q", c.ID, c.Sigil)
	case c.RulesText == "":
		return fmt.Errorf("%s: sem texto de regras", c.ID)
	case c.Set < 0 || c.Set > 2:
		return fmt.Errorf("%s: set fora do intervalo: %d", c.ID, c.Set)
	}
	return nil
}

func validateChampionDef(ch *ChampionDef) error {
	switch {
	case ch.ID == "":
		return fmt.Errorf("campeão sem id")
	case ch.Name == "":
		return fmt.Errorf("%s: sem nome", ch.ID)
	case !validFactions[ch.Faction]:
		return fmt.Errorf("%s: facção inválida %q", ch.ID, ch.Faction)
	case ch.Vitality < 25 || ch.Vitality > 40:
		return fmt.Errorf("%s: vitalidade fora do intervalo: %d", ch.ID, ch.Vitality)
	case ch.Passive == "" || ch.Ultimate == "" || ch.EclipseForm == "":
		return fmt.Errorf("%s: campos de habilidade incompletos", ch.ID)
	}
	return nil
}
