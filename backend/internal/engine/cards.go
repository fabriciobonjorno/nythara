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
}

// Cards e Champions são os catálogos carregados dos dados embutidos.
var (
	Cards     map[string]*CardDef
	CardList  []*CardDef
	Champions map[string]*ChampionDef
)

func init() {
	var err error
	Cards, CardList, Champions, err = loadCatalog(cardsAlphaJSON, championsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: catálogo alpha inválido: %v", err))
	}
}

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
