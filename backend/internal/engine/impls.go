package engine

import "sort"

// Registros executáveis de efeitos. Desde a Fase 2, as cartas são compiladas
// de data/effects_alpha.json (ver dsl.go/dsl_compile.go) — nenhuma carta é
// codificada à mão aqui. Handlers especializados em Go continuam possíveis
// (registrados explicitamente neste arquivo), mas hoje não há nenhum.
// Campeões permanecem em Go até a DSL de Campeões (Fase 9).

type assaultImpl struct {
	base         func(g *Game, ctx *GuardCtx) int // dano por instância
	instances    int                              // 0 => 1
	ignoreWard   int
	preExile     bool // VR-055: exila 1 do descarte antes da janela de Guarda
	undefendable func(g *Game, ctx *GuardCtx) bool
	after        func(g *Game, ctx *GuardCtx)
}

type guardImpl struct {
	onPlay        func(g *Game, ctx *GuardCtx)
	prevent       func(g *Game, ctx *GuardCtx) int
	after         func(g *Game, ctx *GuardCtx) // roda após o dano; sabe PreventedAll
	toHandAfter   bool                         // VR-056: volta à mão em vez do descarte
	selfCostAfter int
	counterRite   bool // VR-035: jogável apenas na janela de reação a Rito
}

type riteImpl struct {
	sacrifice       int
	targetsOpponent bool
	dealsDamage     bool
	ownShift        bool // o texto da carta já move o Eclipse; suprime o intrínseco
	revealsOppHand  bool // VR-006/VR-018: abre reação e é bloqueável por VR-010
	validate        func(g *Game, player int) error
	resolve         func(g *Game, player int, inst string)
}

type activatedImpl struct {
	oncePerRound bool
	discardCost  int
	exileCost    int
	do           []Op
}

type permImpl struct {
	onRoundStart      func(g *Game, inst *CardInstance)
	onSigilEmitted    func(g *Game, inst *CardInstance, sig Sigil, depth int)
	onAssaultResolved func(g *Game, inst *CardInstance, ctx *GuardCtx)
	onEnter           func(g *Game, inst *CardInstance)
	onHeal            func(g *Game, inst *CardInstance)
	onCurseApplied    func(g *Game, inst *CardInstance, curse *TimedN)
	onExile           func(g *Game, inst *CardInstance)
	onOppDraw         func(g *Game, inst *CardInstance, drawer int)
	onDamageDealt     func(g *Game, inst *CardInstance, net int)
	onGuardPlayed     func(g *Game, inst *CardInstance)
	sacrificeDelta    int  // VR-007
	blocksHandReveal  bool // VR-010
	onRevealBlocked   func(g *Game, inst *CardInstance)
	onTwilight        func(g *Game, inst *CardInstance) // VR-076
	activated         *activatedImpl // VR-058/VR-072
}

var (
	assaultImpls = map[string]*assaultImpl{}
	guardImpls   = map[string]*guardImpl{}
	riteImpls    = map[string]*riteImpl{}
	permImpls    = map[string]*permImpl{}
)

func cardImplemented(defID string) bool {
	return assaultImpls[defID] != nil || guardImpls[defID] != nil ||
		riteImpls[defID] != nil || permImpls[defID] != nil
}

// cardDealsDamage informa se a carta causa dano direto (restrição do Arcano).
func cardDealsDamage(defID string) bool {
	if assaultImpls[defID] != nil {
		return true
	}
	if r := riteImpls[defID]; r != nil {
		return r.dealsDamage
	}
	return false
}

// --- Relatório de cobertura ---

// Report lista o que está e o que não está implementado, com motivo. Gate da
// Fase 2: efeitos não suportados aparecem em relatório, nunca são ignorados.
type Report struct {
	ImplementedCards     []string          `json:"implemented_cards"`
	MissingCards         []string          `json:"missing_cards"`
	MissingReasons       map[string]string `json:"missing_reasons"`
	ImplementedChampions []string          `json:"implemented_champions"`
	MissingChampions     []string          `json:"missing_champions"`
	EffectsVersion       string            `json:"effects_version"`
}

func ImplementationReport() Report {
	r := Report{
		MissingReasons: map[string]string{},
		EffectsVersion: Effects.Version,
	}
	for _, c := range CardList {
		if cardImplemented(c.ID) {
			r.ImplementedCards = append(r.ImplementedCards, c.ID)
		} else {
			r.MissingCards = append(r.MissingCards, c.ID)
			r.MissingReasons[c.ID] = Effects.Unsupported[c.ID]
		}
	}
	for id := range Champions {
		if championImpls[id] != nil {
			r.ImplementedChampions = append(r.ImplementedChampions, id)
		} else {
			r.MissingChampions = append(r.MissingChampions, id)
		}
	}
	sort.Strings(r.ImplementedCards)
	sort.Strings(r.MissingCards)
	sort.Strings(r.ImplementedChampions)
	sort.Strings(r.MissingChampions)
	return r
}
