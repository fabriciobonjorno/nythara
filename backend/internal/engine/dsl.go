package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// A DSL de efeitos (Fase 2). Cada carta é descrita por dados em
// data/effects_alpha.json; o compilador (dsl_compile.go) transforma as
// definições nos registros executáveis e o validador abaixo rejeita
// definições malformadas no boot. Cartas fora da DSL constam em "unsupported"
// com motivo — nunca são ignoradas em silêncio.

//go:embed data/effects_alpha.json
var effectsAlphaJSON []byte

// EffectsFile é o arquivo versionado de definições.
type EffectsFile struct {
	Version     string              `json:"version"`
	Cards       map[string]*CardFx  `json:"cards"`
	Unsupported map[string]string   `json:"unsupported"`
}

// CardFx é a definição de efeitos de uma carta (exatamente uma seção,
// correspondente ao tipo da carta).
type CardFx struct {
	Assault   *AssaultFx `json:"assault,omitempty"`
	Guard     *GuardFx   `json:"guard,omitempty"`
	Rite      *RiteFx    `json:"rite,omitempty"`
	Permanent *PermFx    `json:"permanent,omitempty"`
}

// AssaultFx descreve um Assalto: dano base por instância, bônus condicionais
// e efeitos posteriores.
type AssaultFx struct {
	Damage         int     `json:"damage"`
	Instances      int     `json:"instances,omitempty"`
	IgnoreWard     int     `json:"ignore_ward,omitempty"`
	PreExile       bool    `json:"pre_exile,omitempty"` // VR-055
	Bonuses        []Bonus `json:"bonuses,omitempty"`
	UndefendableIf *Cond   `json:"undefendable_if,omitempty"`
	After          []Op    `json:"after,omitempty"`
}

// GuardFx descreve uma Guarda.
type GuardFx struct {
	Prevent        int     `json:"prevent"`
	PreventAll     bool    `json:"prevent_all,omitempty"`
	CounterRite    bool    `json:"counter_rite,omitempty"` // VR-035
	Bonuses        []Bonus `json:"bonuses,omitempty"`
	OnPlay         []Op    `json:"on_play,omitempty"`
	After          []Op    `json:"after,omitempty"`
	ToHandAfter    bool    `json:"to_hand_after,omitempty"` // VR-056
	SelfCostAfter  int     `json:"self_cost_after,omitempty"`
}

// RiteFx descreve um Rito.
type RiteFx struct {
	Sacrifice       int    `json:"sacrifice,omitempty"`
	TargetsOpponent bool   `json:"targets_opponent,omitempty"`
	RevealsOppHand  bool   `json:"reveals_opp_hand,omitempty"` // VR-006/VR-018
	DealsDamage     bool   `json:"deals_damage,omitempty"`
	OwnShift        bool   `json:"own_shift,omitempty"`
	Requires        []Cond `json:"requires,omitempty"`
	Steps           []Op   `json:"steps,omitempty"`
}

// ActivatedFx é uma habilidade ativada de permanente (comando activate).
type ActivatedFx struct {
	OncePerRound bool `json:"once_per_round,omitempty"`
	DiscardCost  int  `json:"discard_cost,omitempty"`
	ExileCost    int  `json:"exile_cost,omitempty"`
	Do           []Op `json:"do"`
}

// PermFx descreve uma Relíquia ou Manifestação.
type PermFx struct {
	OnEnter          []Op         `json:"on_enter,omitempty"`
	Triggers         []Trigger    `json:"triggers,omitempty"`
	SacrificeDelta   int          `json:"sacrifice_delta,omitempty"`    // VR-007
	BlocksHandReveal bool         `json:"blocks_hand_reveal,omitempty"` // VR-010
	Activated        *ActivatedFx `json:"activated,omitempty"`          // VR-058/VR-072
}

// Trigger é um gatilho de permanente.
type Trigger struct {
	On           string `json:"on"` // round_start | sigil_emitted | assault_resolved | heal | curse_applied | exile | opponent_draw | damage_dealt | guard_played
	OncePerRound bool   `json:"once_per_round,omitempty"`
	If           *Cond  `json:"if,omitempty"`
	Sigil        string `json:"sigil,omitempty"`     // filtro para sigil_emitted
	MaxCost      int    `json:"max_cost,omitempty"`  // filtro para assault_resolved (0 = sem filtro)
	NEquals      int    `json:"n_equals,omitempty"`  // filtro para damage_dealt / opponent_draw
	Do           []Op   `json:"do"`
}

// Bonus condiciona dano/prevenção. "add" soma; "instead" substitui o valor.
// "resonance" marca o bônus como bônus de Ressonância (VR-017 interage).
type Bonus struct {
	If        Cond `json:"if"`
	Add       int  `json:"add,omitempty"`
	Instead   int  `json:"instead,omitempty"`
	Resonance bool `json:"resonance,omitempty"`
}

// Cond é uma condição avaliável no contexto do jogador da carta.
type Cond struct {
	Cond   string   `json:"cond"`
	N      int      `json:"n,omitempty"`
	Sigils []string `json:"sigils,omitempty"`
	Type   string   `json:"type,omitempty"` // para discard_has
}

// Op é um efeito atômico. Um único struct discriminado por "op"; o validador
// garante os campos exigidos por cada operação.
type Op struct {
	Op  string `json:"op"`
	N   int    `json:"n,omitempty"`
	Who string `json:"who,omitempty"` // self (default) | opponent

	Status string `json:"status,omitempty"` // exposto | sangramento | maldicao | veu
	Kind   string `json:"kind,omitempty"`   // subtipo de maldição (ex.: VR-053)
	Sigil  string `json:"sigil,omitempty"`

	// cost_mod
	Type      string `json:"type,omitempty"`
	Faction   string `json:"faction,omitempty"`
	Delta     int    `json:"delta,omitempty"`
	Uses      int    `json:"uses,omitempty"`
	ThisRound bool   `json:"this_round,omitempty"`
	SelfInst  bool   `json:"self_inst,omitempty"` // modificador preso à própria instância

	If   *Cond `json:"if,omitempty"`
	Then []Op  `json:"then,omitempty"`
	Else []Op  `json:"else,omitempty"`

	// escolhas
	Filter *Filter `json:"filter,omitempty"`
	NFrom  string  `json:"n_from,omitempty"` // "picked_cost": usa o custo da carta escolhida
}

// Filter restringe opções de escolha em zonas.
type Filter struct {
	Type    string `json:"type,omitempty"`
	Cost    int    `json:"cost,omitempty"`     // custo exato (-1 = ignora)
	MaxCost int    `json:"max_cost,omitempty"` // 0 = ignora
	HasCost bool   `json:"has_cost,omitempty"` // true quando Cost é significativo
}

var knownOps = map[string]bool{
	"damage": true, "heal": true, "draw": true, "ward": true,
	"temp_essence": true, "lose_vitality": true,
	"shift": true, "set_eclipse": true, "swap_eclipse_confirm": true,
	"status": true, "remove_own_bleeds": true, "remove_own_curse": true,
	"remove_curse_smart": true,
	"remove_veil_expose": true, "curse_duration_delta": true,
	"cost_mod": true, "suppress_opp_manifs": true, "lua_no_sangue": true,
	"conditional": true, "emit_sigil": true, "end_assault_window": true,
	"choose_discard": true, "pick_top2": true, "reorder_top": true,
	"recover_pick": true, "draw_reveal_opp_picks": true,
	"mill_top_confirm": true, "exile_self_heal_confirm": true,
	"destroy_relic_pick": true, "neutral_sigil_choice": true,
	"tax_type_pick": true, "both_draw": true,
	"open_extra_window": true, "emit_last_card_sigil": true, "moon_return": true,
	"shift_chosen": true, "peek_opp_hand_tax": true, "reveal_lock_assault": true,
	"declare_reveal_top": true, "mirror_relic_pick": true, "copy_played_pick": true,
	"exile_all_choices": true, "re_emit_sigils": true, "clock_direction_choice": true,
}

var knownConds = map[string]bool{
	"always": true, "sacrificed_this_round": true,
	"eclipse_le": true, "eclipse_ge": true, "eclipse_eq": true, "eclipse_abs_eq": true,
	"eclipse_night": true, "eclipse_aurora": true,
	"second_assault": true, "took_damage_round": true, "exiled_round": true,
	"trail_ends": true, "distinct_sigils_ge": true, "last_global_sigil_in": true,
	"sigil_count_eq": true,
	"hand_less_than_opp": true, "hand_ge": true, "opp_has_manif": true,
	"opp_has_relic": true, "opp_discard_nonempty": true,
	"opp_vitality_le": true, "prevented_all": true, "resonance_bonus": true,
	"relic_destroyed_round": true,
	"opp_veiled": true, "discard_has": true, "discard_ge": true,
	"opp_hand_ge": true, "opp_assaults_eq": true,
	"picked_was_rite": true, "own_assaults_eq": true,
}

// Ops que criam decisões. Uma cadeia pode ter no máximo DUAS decisões
// sequenciais (ex.: VR-059 recupera e depois descarta); a segunda não pode
// encadear uma terceira — regra anti-recursão do validador.
var decisionOps = map[string]bool{
	"choose_discard": true, "pick_top2": true, "reorder_top": true,
	"recover_pick": true, "draw_reveal_opp_picks": true,
	"mill_top_confirm": true, "exile_self_heal_confirm": true,
	"destroy_relic_pick": true, "neutral_sigil_choice": true,
	"swap_eclipse_confirm": true, "tax_type_pick": true,
	"peek_opp_hand_tax": true, "reveal_lock_assault": true,
	"declare_reveal_top": true, "mirror_relic_pick": true,
	"copy_played_pick": true, "re_emit_sigils": true,
	"shift_chosen": true, "exile_all_choices": true,
}

// Effects é o arquivo carregado e validado no boot.
var Effects *EffectsFile

func init() {
	fx, err := loadEffects(effectsAlphaJSON)
	if err != nil {
		panic(fmt.Sprintf("engine: effects_alpha.json inválido: %v", err))
	}
	Effects = fx
	compileEffects(fx)
}

// LoadEffectsForTest expõe o parser/validador da DSL para testes.
func LoadEffectsForTest(raw []byte) (*EffectsFile, error) { return loadEffects(raw) }

func loadEffects(raw []byte) (*EffectsFile, error) {
	var fx EffectsFile
	if err := json.Unmarshal(raw, &fx); err != nil {
		return nil, err
	}
	if fx.Version == "" {
		return nil, fmt.Errorf("sem campo version")
	}
	// Cobertura exata: cada carta do catálogo está na DSL OU em unsupported.
	for _, c := range CardList {
		_, inDSL := fx.Cards[c.ID]
		reason, inUnsup := fx.Unsupported[c.ID]
		switch {
		case inDSL && inUnsup:
			return nil, fmt.Errorf("%s: presente em cards e unsupported", c.ID)
		case !inDSL && !inUnsup:
			return nil, fmt.Errorf("%s: ausente de cards e de unsupported", c.ID)
		case inUnsup && reason == "":
			return nil, fmt.Errorf("%s: unsupported sem motivo", c.ID)
		}
	}
	for id, cfx := range fx.Cards {
		def := Cards[id]
		if def == nil {
			return nil, fmt.Errorf("%s: não existe no catálogo", id)
		}
		if err := validateCardFx(def, cfx); err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
	}
	return &fx, nil
}

func validateCardFx(def *CardDef, fx *CardFx) error {
	sections := 0
	if fx.Assault != nil {
		sections++
		if def.Type != TypeAssalto {
			return fmt.Errorf("seção assault em carta %s", def.Type)
		}
		a := fx.Assault
		if a.Damage < 0 || a.Damage > 10 {
			return fmt.Errorf("dano fora do intervalo: %d", a.Damage)
		}
		if a.Instances < 0 || a.Instances > 5 {
			return fmt.Errorf("instances fora do intervalo: %d", a.Instances)
		}
		if a.IgnoreWard < 0 || a.IgnoreWard > 5 {
			return fmt.Errorf("ignore_ward fora do intervalo: %d", a.IgnoreWard)
		}
		for _, b := range a.Bonuses {
			if err := validateBonus(b); err != nil {
				return err
			}
		}
		if a.UndefendableIf != nil {
			if err := validateCond(*a.UndefendableIf); err != nil {
				return err
			}
		}
		if err := validateOps(a.After, 0); err != nil {
			return err
		}
	}
	if fx.Guard != nil {
		sections++
		if def.Type != TypeGuarda {
			return fmt.Errorf("seção guard em carta %s", def.Type)
		}
		gd := fx.Guard
		if gd.Prevent < 0 || gd.Prevent > 10 {
			return fmt.Errorf("prevenção fora do intervalo: %d", gd.Prevent)
		}
		for _, b := range gd.Bonuses {
			if err := validateBonus(b); err != nil {
				return err
			}
		}
		if err := validateOps(gd.OnPlay, 0); err != nil {
			return err
		}
		if err := validateOps(gd.After, 0); err != nil {
			return err
		}
		if gd.SelfCostAfter != 0 && !gd.ToHandAfter {
			return fmt.Errorf("self_cost_after exige to_hand_after")
		}
	}
	if fx.Rite != nil {
		sections++
		if def.Type != TypeRito {
			return fmt.Errorf("seção rite em carta %s", def.Type)
		}
		r := fx.Rite
		if r.Sacrifice < 0 || r.Sacrifice > 5 {
			return fmt.Errorf("sacrifício fora do intervalo: %d", r.Sacrifice)
		}
		for _, c := range r.Requires {
			if err := validateCond(c); err != nil {
				return err
			}
		}
		if err := validateOps(r.Steps, 0); err != nil {
			return err
		}
	}
	if fx.Permanent != nil {
		sections++
		if def.Type != TypeReliquia && def.Type != TypeManifestacao {
			return fmt.Errorf("seção permanent em carta %s", def.Type)
		}
		p := fx.Permanent
		if p.SacrificeDelta < -2 || p.SacrificeDelta > 0 {
			return fmt.Errorf("sacrifice_delta fora do intervalo: %d", p.SacrificeDelta)
		}
		if err := validateOps(p.OnEnter, 0); err != nil {
			return err
		}
		for _, tr := range p.Triggers {
			if err := validateTrigger(tr); err != nil {
				return err
			}
		}
		if a := p.Activated; a != nil {
			if len(a.Do) == 0 {
				return fmt.Errorf("habilidade ativada sem efeitos")
			}
			if a.DiscardCost < 0 || a.DiscardCost > 3 || a.ExileCost < 0 || a.ExileCost > 3 {
				return fmt.Errorf("custos de ativação fora do intervalo")
			}
			if err := validateOps(a.Do, 0); err != nil {
				return err
			}
		}
	}
	if sections != 1 {
		return fmt.Errorf("exatamente uma seção de efeito é exigida (há %d)", sections)
	}
	return nil
}

var knownTriggers = map[string]bool{
	"round_start": true, "sigil_emitted": true, "assault_resolved": true,
	"heal": true, "curse_applied": true, "exile": true,
	"opponent_draw": true, "damage_dealt": true, "guard_played": true,
	"reveal_blocked": true, "twilight": true,
}

func validateTrigger(tr Trigger) error {
	if !knownTriggers[tr.On] {
		return fmt.Errorf("gatilho desconhecido: %q", tr.On)
	}
	if tr.If != nil {
		if err := validateCond(*tr.If); err != nil {
			return err
		}
	}
	if len(tr.Do) == 0 {
		return fmt.Errorf("gatilho %s sem efeitos", tr.On)
	}
	if err := validateOps(tr.Do, 0); err != nil {
		return err
	}
	// Anti-loop estático: gatilho de sigilo que emite sigilo precisa ser 1x/rodada.
	if tr.On == "sigil_emitted" && !tr.OncePerRound {
		for _, op := range tr.Do {
			if op.Op == "emit_sigil" {
				return fmt.Errorf("gatilho sigil_emitted que emite sigilo exige once_per_round")
			}
		}
	}
	return nil
}

func validateBonus(b Bonus) error {
	if err := validateCond(b.If); err != nil {
		return err
	}
	// Um bônus pode existir só para marcar Ressonância (VR-005).
	if b.Add == 0 && b.Instead == 0 && !b.Resonance {
		return fmt.Errorf("bônus sem add, instead ou resonance")
	}
	if b.Add != 0 && b.Instead != 0 {
		return fmt.Errorf("bônus com add e instead simultâneos")
	}
	if b.Add < -5 || b.Add > 5 || b.Instead < 0 || b.Instead > 12 {
		return fmt.Errorf("bônus fora do intervalo")
	}
	return nil
}

func validateCond(c Cond) error {
	if !knownConds[c.Cond] {
		return fmt.Errorf("condição desconhecida: %q", c.Cond)
	}
	if c.Cond == "trail_ends" || c.Cond == "last_global_sigil_in" || c.Cond == "sigil_count_eq" {
		if len(c.Sigils) == 0 {
			return fmt.Errorf("condição %s exige sigils", c.Cond)
		}
		for _, s := range c.Sigils {
			if s == "*" && c.Cond == "trail_ends" {
				continue // curinga: qualquer Sigilo (VR-047)
			}
			if !validSigils[Sigil(s)] {
				return fmt.Errorf("sigilo inválido em condição: %q", s)
			}
		}
	}
	return nil
}

const maxOpDepth = 4

func validateOps(ops []Op, depth int) error {
	if depth > maxOpDepth {
		return fmt.Errorf("profundidade de efeitos acima de %d", maxOpDepth)
	}
	for _, op := range ops {
		if !knownOps[op.Op] {
			return fmt.Errorf("operação desconhecida: %q", op.Op)
		}
		if op.Who != "" && op.Who != "self" && op.Who != "opponent" {
			return fmt.Errorf("%s: who inválido %q", op.Op, op.Who)
		}
		switch op.Op {
		case "damage", "heal", "draw", "ward", "temp_essence":
			if op.N <= 0 || op.N > 9 {
				return fmt.Errorf("%s: n fora do intervalo: %d", op.Op, op.N)
			}
		case "lose_vitality":
			if op.N <= 0 && op.NFrom != "picked_cost" {
				return fmt.Errorf("lose_vitality exige n ou n_from")
			}
		case "shift":
			if op.N == 0 || op.N < -3 || op.N > 3 {
				return fmt.Errorf("shift: n inválido: %d", op.N)
			}
		case "set_eclipse":
			if op.N < EclipseMin || op.N > EclipseMax {
				return fmt.Errorf("set_eclipse fora do medidor: %d", op.N)
			}
		case "status":
			switch op.Status {
			case "exposto", "veu":
			case "sangramento", "maldicao":
				if op.N <= 0 {
					return fmt.Errorf("status %s exige n", op.Status)
				}
			default:
				return fmt.Errorf("status desconhecido: %q", op.Status)
			}
		case "cost_mod":
			if op.Delta < -3 || op.Delta > 3 || op.Delta == 0 {
				return fmt.Errorf("cost_mod: delta inválido: %d", op.Delta)
			}
			if op.Uses <= 0 || op.Uses > 999 {
				return fmt.Errorf("cost_mod: uses inválido: %d", op.Uses)
			}
			if op.Type != "" && !validTypes[CardType(op.Type)] {
				return fmt.Errorf("cost_mod: tipo inválido: %q", op.Type)
			}
			if op.Faction != "" && !validFactions[op.Faction] {
				return fmt.Errorf("cost_mod: facção inválida: %q", op.Faction)
			}
		case "conditional":
			if op.If == nil {
				return fmt.Errorf("conditional sem if")
			}
			if err := validateCond(*op.If); err != nil {
				return err
			}
		case "emit_sigil":
			if !validSigils[Sigil(op.Sigil)] {
				return fmt.Errorf("emit_sigil: sigilo inválido: %q", op.Sigil)
			}
		case "choose_discard", "recover_pick", "reorder_top":
			if op.N <= 0 || op.N > 3 {
				return fmt.Errorf("%s: n fora do intervalo: %d", op.Op, op.N)
			}
		case "draw_reveal_opp_picks":
			if op.N != 2 {
				return fmt.Errorf("draw_reveal_opp_picks: apenas n=2 suportado")
			}
		}
		// Anti-recursão: no máximo 2 decisões em cadeia; a 2ª não encadeia 3ª.
		if decisionOps[op.Op] {
			for _, t := range flattenOps(op.Then) {
				if decisionOps[t.Op] {
					for _, t2 := range flattenOps(t.Then) {
						if decisionOps[t2.Op] {
							return fmt.Errorf("%s: três decisões encadeadas (%s → %s)", op.Op, t.Op, t2.Op)
						}
					}
				}
			}
		}
		if err := validateOps(op.Then, depth+1); err != nil {
			return err
		}
		if err := validateOps(op.Else, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func flattenOps(ops []Op) []Op {
	var out []Op
	for _, op := range ops {
		out = append(out, op)
		out = append(out, flattenOps(op.Then)...)
		out = append(out, flattenOps(op.Else)...)
	}
	return out
}
