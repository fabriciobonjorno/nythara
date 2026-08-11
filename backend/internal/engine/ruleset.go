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
	Version       string
	Mode          string
	ConfrontRules ConfrontRulesConfig
	Cards         map[string]*CardDef
	CardList      []*CardDef
	Champions     map[string]*ChampionDef
	Effects       *EffectsFile

	assault map[string]*assaultImpl
	guard   map[string]*guardImpl
	rite    map[string]*riteImpl
	perm    map[string]*permImpl
}

const (
	RulesModeLegacy   = "legacy"
	RulesModeConfront = "confront"
)

// IsConfront informa se este conjunto usa o fluxo direto do ADR-044.
func (rs *Ruleset) IsConfront() bool { return rs != nil && rs.Mode == RulesModeConfront }

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
		Mode:      fx.Mode,
		Cards:     cards,
		CardList:  list,
		Champions: champs,
		Effects:   fx,
		assault:   map[string]*assaultImpl{},
		guard:     map[string]*guardImpl{},
		rite:      map[string]*riteImpl{},
		perm:      map[string]*permImpl{},
	}
	if rs.Mode == "" {
		rs.Mode = RulesModeLegacy
	}
	if rs.IsConfront() {
		rs.ConfrontRules = initialConfrontRules()
		if fx.Confront != nil {
			rs.ConfrontRules = *fx.Confront
		}
	}
	compileEffects(fx, rs)
	if rs.IsConfront() {
		prepareConfrontRuleset(rs)
	}
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
	legacy, err := CompileRuleset(RulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset embutido inválido: %v", err))
	}
	confrontInitial, err := CompileRuleset(ConfrontInitialRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset Confronto inicial inválido: %v", err))
	}
	configureConfrontRuleset(confrontInitial, initialConfrontRules())
	confront, err := CompileRuleset(ConfrontRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset Confronto competitivo inválido: %v", err))
	}
	configureConfrontRuleset(confront, competitiveConfrontRules())
	tacticalInitial, err := CompileRuleset(TacticalInitialRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset Confronto tático inicial inválido: %v", err))
	}
	configureConfrontRuleset(tacticalInitial, initialTacticalConfrontRules())
	tactical, err := CompileRuleset(TacticalRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset Confronto tático balanceado inválido: %v", err))
	}
	configureConfrontRuleset(tactical, tacticalConfrontRules())
	longDuel, err := CompileRuleset(LongDuelRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset de duelo longo inválido: %v", err))
	}
	configureConfrontRuleset(longDuel, longDuelConfrontRules())

	// `builtin` e os globais permanecem apontando para 0.8.3 para que replays
	// históricos e integrações antigas nunca mudem de significado. Produto e
	// matchmaking escolhem explicitamente CompetitiveRulesetVersion.
	builtin = legacy
	rulesets[legacy.Version] = legacy
	rulesets[confrontInitial.Version] = confrontInitial
	rulesets[confront.Version] = confront
	rulesets[tacticalInitial.Version] = tacticalInitial
	rulesets[tactical.Version] = tactical
	rulesets[longDuel.Version] = longDuel
	Cards = legacy.Cards
	CardList = legacy.CardList
	Champions = legacy.Champions
	Effects = legacy.Effects
	assaultImpls = legacy.assault
	guardImpls = legacy.guard
	riteImpls = legacy.rite
	permImpls = legacy.perm
}

func configureConfrontRuleset(rs *Ruleset, cfg ConfrontRulesConfig) {
	rs.Mode = RulesModeConfront
	rs.ConfrontRules = cfg
	rs.Effects.Mode = RulesModeConfront
	rs.Effects.Confront = &rs.ConfrontRules
	prepareConfrontRuleset(rs)
}

func initialConfrontRules() ConfrontRulesConfig {
	return ConfrontRulesConfig{StartingVitality: 30}
}

func competitiveConfrontRules() ConfrontRulesConfig {
	return ConfrontRulesConfig{StartingVitality: ConfrontStartingVitality,
		PowerBonus: ConfrontPowerBonus, FirstTurnPenalty: ConfrontFirstTurnPenalty,
		DrawOnFirstTurn: true, PressureStartTurn: ConfrontPressureStartTurn,
		PressureBaseLoss: ConfrontPressureBaseLoss, MinAssaults: ConfrontMinAssaults,
		MinGuards: ConfrontMinGuards, MinRites: ConfrontMinRites}
}

func initialTacticalConfrontRules() ConfrontRulesConfig {
	cfg := competitiveConfrontRules()
	cfg.TacticalSeals = true
	return cfg
}

func tacticalConfrontRules() ConfrontRulesConfig {
	cfg := initialTacticalConfrontRules()
	cfg.ExposedPowerBonus = 2
	return cfg
}

// longDuelConfrontRules parte do tático e corrige a assimetria que encurtava a
// partida: o Assalto recebia bônus fixo de Poder e a Guarda, nenhum. Ver ADR-035.
func longDuelConfrontRules() ConfrontRulesConfig {
	cfg := tacticalConfrontRules()
	cfg.StartingVitality = LongDuelStartingVitality
	cfg.GuardBonus = LongDuelGuardBonus
	cfg.PressureStartTurn = LongDuelPressureStartTurn
	cfg.PressureBaseLoss = LongDuelPressureBaseLoss
	cfg.SecondPlayerBonusDraw = LongDuelSecondPlayerDraw
	cfg.TargetP95Rounds = LongDuelTargetP95Rounds
	cfg.TargetMinP50Rounds = LongDuelTargetMinP50Rounds
	cfg.MinGuards = LongDuelMinGuards
	cfg.GuardLeakCap = LongDuelGuardLeakCap
	return cfg
}

// CompetitiveRuleset devolve o ruleset atualmente servido pelo produto.
func CompetitiveRuleset() *Ruleset {
	rs, err := RulesetByVersion(CompetitiveRulesetVersion)
	if err != nil {
		panic(err)
	}
	return rs
}
