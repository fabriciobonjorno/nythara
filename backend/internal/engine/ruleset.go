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
	avatar, err := CompileRuleset(AvatarRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset de Avatares inválido: %v", err))
	}
	configureConfrontRuleset(avatar, avatarConfrontRules())
	decisions, err := CompileRuleset(DecisionRulesetVersion, cardsAlphaJSON, championsAlphaJSON, effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: ruleset com decisões inválido: %v", err))
	}
	configureConfrontRuleset(decisions, decisionConfrontRules())

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
	rulesets[avatar.Version] = avatar
	rulesets[decisions.Version] = decisions
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
	if err := validateChampionPowers(rs); err != nil {
		panic(fmt.Sprintf("engine: %s: %v", rs.Version, err))
	}
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

// avatarConfrontRules parte do duelo longo e devolve poder aos Avatares. Cada
// poder usa só sistemas que o Confronto tem — custo em Vitalidade, janelas,
// Ward, estilhaço e Fadiga —, porque os poderes históricos dependiam de
// Essência, Eclipse e Ressonância, que este modo removeu. Ver ADR-057.
func avatarConfrontRules() ConfrontRulesConfig {
	cfg := longDuelConfrontRules()
	// Os poderes de Avatar somam sustentação (cura, Ward, desconto) dos dois
	// lados, e com 56 de Vitalidade a partida passava a ser decidida pela Fadiga:
	// o gate reprovou com 34,4% de desfechos por dano. Baixar a Vitalidade
	// devolve a decisão para a mesa sem encurtar o formato.
	cfg.StartingVitality = AvatarStartingVitality
	// As magnitudes abaixo saíram de medição, não de intuição. Duas descobertas
	// mudaram o desenho: um desconto que vale todo turno soma ~20 de Vitalidade
	// numa partida de 44 rodadas (o primeiro rascunho deu 82% de win rate), e
	// comprar carta virou desvantagem — o baralho é o relógio, então acelerar a
	// compra aproxima a Fadiga. Por isso nenhum Avatar compra cartas.
	// As magnitudes abaixo saíram de medição, não de intuição. Três descobertas
	// moldaram o desenho: um desconto que vale todo turno soma ~20 de Vitalidade
	// numa partida de 44 rodadas (o primeiro rascunho deu 82% de win rate);
	// comprar carta virou desvantagem, porque o baralho é o relógio e acelerar a
	// compra aproxima a Fadiga; e cura vale pouco enquanto se está com a vida
	// cheia, então poder curativo só compensa se disparar depois do dano.
	// Atribuição escolhida por medição de estabilidade entre modos. Só dois
	// gatilhos se mostraram estáveis entre baralho oficial e baralho montado à
	// mão: "quando seu Assalto conecta" e "quando a Vitalidade cai até um
	// limiar". Gatilhos ligados à frequência de Guarda ou de estilhaço variam
	// com a agressividade do baralho e chegaram a 11 pontos de diferença entre
	// os modos. O limiar de Vitalidade funciona como dial fino: cada ponto vale
	// cerca de meio ponto de win rate, então ele calibra sem inventar sistema.
	cfg.ChampionPowers = map[string]ChampionPower{
		"CH-VH-01": {Text: "Quando seu Assalto causa dano, recupere 1 de Vitalidade.",
			Trigger: "connected_hit", Effect: "heal", N: 1},
		"CH-VH-02": {Text: "Com 24 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 24},
		"CH-SO-01": {Text: "Com 32 ou menos de Vitalidade, recupere 1 sempre que levar dano.",
			Trigger: "low_vitality", Effect: "heal", N: 1, At: 32},
		"CH-SO-02": {Text: "Quando seu Assalto causa dano, ganhe 1 de Ward.",
			Trigger: "connected_hit", Effect: "ward", N: 1},
		"CH-MI-01": {Text: "Com 30 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 30},
		"CH-MI-02": {Text: "Com 34 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 34},
		"CH-VA-01": {Text: "Com 28 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 28},
		"CH-VA-02": {Text: "Com 26 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 26},
		"CH-CI-01": {Text: "Com 36 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 36},
		"CH-CI-02": {Text: "Com 32 ou menos de Vitalidade, seus Assaltos custam 1 a menos.",
			Trigger: "low_vitality", Effect: "discount", N: 1, At: 32},
	}
	// Os poderes acima são majoritariamente reativos, e quem joga em segundo é
	// quem responde primeiro: sem mexer aqui, o first-player cai para 44,8%. A
	// penalidade do primeiro turno some para devolver a simetria.
	cfg.FirstTurnPenalty = 1
	// Dial diferencial de iniciativa: uma carta extra é ativo no jogo curto e
	// passivo no jogo longo, porque lá o baralho é o relógio. É a única alavanca
	// que empurra precon e deck livre em direções opostas ao mesmo tempo.
	cfg.SecondPlayerBonusDraw = 1
	// As duas Guardas mais fracas morriam na mão em ~56% das partidas: numa
	// janela de uma Guarda só, a pior nunca é escolhida. De graça, elas passam a
	// valer justamente quando você quer guardar a boa para o golpe grande.
	zero := 0
	cfg.CardAdjustments = map[string]CardAdjustment{
		"VR-038": {Cost: &zero, Reason: "Guarda gratuita: a carta precisa valer quando não é a melhor da mão"},
		"VR-062": {Cost: &zero, Reason: "Guarda gratuita: a carta precisa valer quando não é a melhor da mão"},
	}
	return cfg
}

// decisionConfrontRules abre o comando de escolha e, com ele, as cartas de
// lapidação de mão que estavam fora do modo por pedirem uma decisão ao jogador.
// Duas cartas a mais mudam a composição dos precons e o ritmo cai de 43,7 para
// 31,9 rodadas, então esta versão nasce separada: a capacidade está pronta e
// verificada, a calibragem dela é o próximo trabalho. Ver ADR-058.
func decisionConfrontRules() ConfrontRulesConfig {
	cfg := avatarConfrontRules()
	cfg.Decisions = true
	// Calibragem própria (ADR-059). Os dois motores de carta viram custo 2:
	// de graça eles queimam o baralho rápido demais e entregam o formato ao
	// segundo jogador — a 1 de custo o first-player caiu a 45,5% no oficial e
	// 47,1% no livre. A penalidade de primeiro turno zera pelo mesmo motivo.
	cfg.FirstTurnPenalty = 0
	adjustments := make(map[string]CardAdjustment, len(cfg.CardAdjustments)+2)
	for id, adjustment := range cfg.CardAdjustments {
		adjustments[id] = adjustment
	}
	two := 2
	adjustments["VR-002"] = CardAdjustment{Cost: &two, Reason: "motor de carta pago no ritmo do formato"}
	adjustments["VR-049"] = CardAdjustment{Cost: &two, Reason: "motor de carta pago no ritmo do formato"}
	cfg.CardAdjustments = adjustments
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
