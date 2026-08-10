package engine

import (
	"fmt"
	"sort"
	"sync"
)

// Ruleset é um conjunto completo e imutável de regras executáveis: catálogo,
// Campeões e efeitos compilados de uma versão. Partidas referenciam um
// Ruleset; versões publicadas ficam num registro e partidas históricas
// continuam reproduzíveis sob a versão original (Fase 7 / ADR-022).
//
// As passivas/ultimates de Campeão permanecem em Go (championImpls) e NÃO são
// versionadas por Ruleset — apenas os atributos (vitalidade etc.). Mudar
// comportamento de Campeão exige release do binário e bump de ruleset.
type Ruleset struct {
	Version   string
	Cards     map[string]*CardDef
	CardList  []*CardDef
	Champions map[string]*ChampionDef
	Effects   *EffectsFile

	assault map[string]*assaultImpl
	guard   map[string]*guardImpl
	rite    map[string]*riteImpl
	perm    map[string]*permImpl
}

// CompileRuleset valida catálogo + efeitos e compila os registros executáveis.
// É o único caminho de construção de um Ruleset — o embutido passa por aqui.
func CompileRuleset(version string, cardsJSON, championsJSON, effectsJSON []byte) (*Ruleset, error) {
	if version == "" {
		return nil, fmt.Errorf("ruleset sem versão")
	}
	cards, list, champs, err := loadCatalog(cardsJSON, championsJSON)
	if err != nil {
		return nil, fmt.Errorf("catálogo %s: %w", version, err)
	}
	fx, err := loadEffects(effectsJSON, cards, list)
	if err != nil {
		return nil, fmt.Errorf("efeitos %s: %w", version, err)
	}
	rs := &Ruleset{
		Version:   version,
		Cards:     cards,
		CardList:  list,
		Champions: champs,
		Effects:   fx,
		assault:   map[string]*assaultImpl{},
		guard:     map[string]*guardImpl{},
		rite:      map[string]*riteImpl{},
		perm:      map[string]*permImpl{},
	}
	compileEffects(fx, rs)
	return rs, nil
}

// cardImplemented informa se a carta tem efeitos executáveis neste Ruleset.
func (rs *Ruleset) cardImplemented(defID string) bool {
	return rs.assault[defID] != nil || rs.guard[defID] != nil ||
		rs.rite[defID] != nil || rs.perm[defID] != nil
}

// cardDealsDamage informa se a carta causa dano direto (restrição do Arcano).
func (rs *Ruleset) cardDealsDamage(defID string) bool {
	if rs.assault[defID] != nil {
		return true
	}
	if r := rs.rite[defID]; r != nil {
		return r.dealsDamage
	}
	return false
}

// --- Registro de versões ---

var (
	rulesetMu sync.RWMutex
	rulesets  = map[string]*Ruleset{}
	builtin   *Ruleset
)

// Builtin devolve o Ruleset embutido no binário.
func Builtin() *Ruleset { return builtin }

// RegisterRuleset disponibiliza uma versão para NewGame/Replay. Registrar a
// mesma versão com conteúdo diferente é rejeitado — versões são imutáveis.
func RegisterRuleset(rs *Ruleset) error {
	if rs == nil || rs.Version == "" {
		return fmt.Errorf("ruleset inválido")
	}
	rulesetMu.Lock()
	defer rulesetMu.Unlock()
	if existing, ok := rulesets[rs.Version]; ok {
		if existing != rs {
			return fmt.Errorf("versão %s já registrada", rs.Version)
		}
		return nil
	}
	rulesets[rs.Version] = rs
	return nil
}

// UnregisterRuleset remove uma versão do registro (uso administrativo, ex.:
// rulesets efêmeros de simulação de draft). O embutido não pode ser removido.
func UnregisterRuleset(version string) {
	if builtin != nil && version == builtin.Version {
		return
	}
	rulesetMu.Lock()
	defer rulesetMu.Unlock()
	delete(rulesets, version)
}

// RulesetByVersion resolve uma versão registrada; vazio resolve o embutido.
func RulesetByVersion(version string) (*Ruleset, error) {
	if version == "" {
		return builtin, nil
	}
	rulesetMu.RLock()
	defer rulesetMu.RUnlock()
	if rs, ok := rulesets[version]; ok {
		return rs, nil
	}
	return nil, fmt.Errorf("ruleset %q não registrado", version)
}

// RegisteredVersions lista as versões disponíveis, ordenadas.
func RegisteredVersions() []string {
	rulesetMu.RLock()
	defer rulesetMu.RUnlock()
	out := make([]string, 0, len(rulesets))
	for v := range rulesets {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// A construção do embutido substitui os init de cards.go/dsl.go: um único
// pipeline, com os globais servindo de visão do builtin para pacotes externos
// e testes.
func init() {
	rs, err := CompileRuleset(RulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset embutido inválido: %v", err))
	}
	builtin = rs
	rulesets[rs.Version] = rs
	Cards = rs.Cards
	CardList = rs.CardList
	Champions = rs.Champions
	Effects = rs.Effects
	assaultImpls = rs.assault
	guardImpls = rs.guard
	riteImpls = rs.rite
	permImpls = rs.perm
}
