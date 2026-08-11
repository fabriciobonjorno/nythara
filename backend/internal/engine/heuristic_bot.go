package engine

import (
	"sort"
	"strings"
)

// HeuristicBot é o nível 2 do plano de balanceamento. Ele não antecipa RNG
// nem chama Apply em cópias: pontua somente comandos já enumerados como
// legais, mas o scoring lê o estado real e a DSL declarativa das cartas —
// dano esperado com bônus avaliados agora, cadeias de Sigilos, plano de
// Eclipse do kit, economia de Guarda e timing de Ultimate. A rodada 1 de
// balanceamento (ADR-024) provou que o scorer antigo, cego a combo,
// distorcia a régua; este é o instrumento da rodada 2 (ADR-029).
type HeuristicBot struct {
	RNG *RNG
}

func (b *HeuristicBot) Next(g *Game) (Command, bool) {
	player, ok := RequiredPlayer(g)
	if !ok {
		return Command{}, false
	}
	return b.NextFor(g, player)
}

// NextFor escolhe a ação somente para o lado informado.
func (b *HeuristicBot) NextFor(g *Game, player int) (Command, bool) {
	s := g.State()
	required, ok := RequiredPlayer(g)
	if !ok || required != player {
		return Command{}, false
	}
	if b.RNG == nil {
		b.RNG = NewRNG(s.Seed ^ 0x9e3779b97f4a7c15)
	}
	if d := s.Pending; d != nil {
		return b.answerDecision(g, d), true
	}
	if g.rs.IsConfront() {
		actions := g.legalPlays(player)
		actions = append(actions, Command{Player: player, Kind: CmdKindPass})
		return b.bestAction(g, actions), true
	}
	switch s.Phase {
	case PhaseMulligan:
		return b.mulligan(g, player), true
	case PhaseStance:
		return Command{Player: player, Kind: CmdKindStance, Stance: b.stance(g, player)}, true
	case PhaseRite, PhaseConfront:
		actor := player
		actions := g.legalPlays(actor)
		if s.Guard == nil && s.RiteReact == nil && s.Extra == nil && actor == s.Active {
			if s.Phase == PhaseRite {
				p := s.Players[actor]
				for _, id := range append(append([]string{}, p.Relics...), p.Manifs...) {
					if g.CanActivate(actor, id) == nil {
						actions = append(actions, Command{Player: actor, Kind: CmdKindActivate, Card: id})
					}
				}
			}
			if g.CanUltimate(actor) == nil {
				actions = append(actions, Command{Player: actor, Kind: CmdKindUltimate})
			}
		}
		actions = append(actions, Command{Player: actor, Kind: CmdKindPass})
		return b.bestAction(g, actions), true
	case PhaseAssault, PhaseGuard:
		actions := g.legalPlays(player)
		actions = append(actions, Command{Player: player, Kind: CmdKindPass})
		return b.bestAction(g, actions), true
	}
	return Command{}, false
}

// --- Consciência de kit (introspecção dos impls e da mão — sem tabela por ID) ---

// kitChasesSigils detecta kits que pagam por encadear Sigilos (passiva de 3
// Sigilos, Sigilo extra em cópias) ou mãos densas em emissores/condições de
// trilha.
func (b *HeuristicBot) kitChasesSigils(g *Game, player int) bool {
	if ci := champImpl(g, player); ci != nil && (ci.on3Sigils != nil || ci.copyExtraSigil != nil) {
		return true
	}
	density := 0
	for _, id := range g.State().Players[player].Hand {
		fx := b.cardFx(g, g.State().Cards[id].Def)
		if fx == nil {
			continue
		}
		if b.fxEmitsSigil(fx) || b.fxWantsTrail(fx) {
			density++
		}
	}
	return density >= 3
}

func (b *HeuristicBot) kitDrawsExtra(g *Game, player int) bool {
	ci := champImpl(g, player)
	return ci != nil && ci.onExtraDraw != nil
}

// eclipsePole devolve o polo desejado do Eclipse: +1 Noite, -1 Aurora, 0
// indiferente. Parte da facção e refina pela mão atual (condições
// eclipse_le/ge nas cartas que o jogador realmente segura).
func (b *HeuristicBot) eclipsePole(g *Game, player int) int {
	s := g.State()
	pole := factionPole(g.rs.Champions[s.Players[player].Champion].Faction)
	night, aurora := 0, 0
	for _, id := range s.Players[player].Hand {
		fx := b.cardFx(g, s.Cards[id].Def)
		if fx == nil {
			continue
		}
		for _, cond := range b.fxConds(fx) {
			switch cond.Cond {
			case "eclipse_ge", "eclipse_night":
				night++
			case "eclipse_le", "eclipse_aurora":
				aurora++
			}
		}
	}
	if night > aurora+1 {
		return 1
	}
	if aurora > night+1 {
		return -1
	}
	return pole
}

func factionPole(faction string) int {
	switch faction {
	case "Casa Vhal", "Matilha Varka":
		return 1
	case "Ordem Solara":
		return -1
	}
	return 0
}

// --- Mineração da DSL ---

func (b *HeuristicBot) cardFx(g *Game, defID string) *CardFx {
	if g.rs.Effects == nil {
		return nil
	}
	return g.rs.Effects.Cards[defID]
}

func (b *HeuristicBot) fxEmitsSigil(fx *CardFx) bool {
	found := false
	b.walkOps(b.allOps(fx), func(op *Op) {
		if op.Op == "emit_sigil" || op.Op == "emit_last_card_sigil" {
			found = true
		}
	})
	return found
}

func (b *HeuristicBot) fxWantsTrail(fx *CardFx) bool {
	for _, cond := range b.fxConds(fx) {
		switch cond.Cond {
		case "trail_ends", "distinct_sigils_ge", "sigil_count_eq", "last_global_sigil_in":
			return true
		}
	}
	return false
}

// fxConds coleta todas as condições declaradas na carta (bônus, requisitos e
// condicionais aninhadas).
func (b *HeuristicBot) fxConds(fx *CardFx) []Cond {
	var out []Cond
	add := func(c *Cond) {
		if c != nil {
			out = append(out, *c)
		}
	}
	if fx.Assault != nil {
		for i := range fx.Assault.Bonuses {
			out = append(out, fx.Assault.Bonuses[i].If)
		}
		add(fx.Assault.UndefendableIf)
	}
	if fx.Guard != nil {
		for i := range fx.Guard.Bonuses {
			out = append(out, fx.Guard.Bonuses[i].If)
		}
	}
	if fx.Rite != nil {
		out = append(out, fx.Rite.Requires...)
	}
	if fx.Permanent != nil {
		for i := range fx.Permanent.Triggers {
			add(fx.Permanent.Triggers[i].If)
		}
	}
	b.walkOps(b.allOps(fx), func(op *Op) { add(op.If) })
	return out
}

func (b *HeuristicBot) allOps(fx *CardFx) []Op {
	var ops []Op
	if fx.Assault != nil {
		ops = append(ops, fx.Assault.After...)
	}
	if fx.Guard != nil {
		ops = append(ops, fx.Guard.OnPlay...)
		ops = append(ops, fx.Guard.After...)
	}
	if fx.Rite != nil {
		ops = append(ops, fx.Rite.Steps...)
	}
	if fx.Permanent != nil {
		ops = append(ops, fx.Permanent.OnEnter...)
		for i := range fx.Permanent.Triggers {
			ops = append(ops, fx.Permanent.Triggers[i].Do...)
		}
		if fx.Permanent.Activated != nil {
			ops = append(ops, fx.Permanent.Activated.Do...)
		}
	}
	return ops
}

func (b *HeuristicBot) walkOps(ops []Op, visit func(*Op)) {
	for i := range ops {
		visit(&ops[i])
		b.walkOps(ops[i].Then, visit)
		b.walkOps(ops[i].Else, visit)
	}
}

// condNow avalia uma condição no estado atual com contexto mínimo; condições
// que exigem contexto de combate (prevented_all etc.) leem como falsas —
// conservador por construção.
func (b *HeuristicBot) condNow(g *Game, player int, c Cond) bool {
	return g.evalCond(c, &opCtx{player: player})
}

// opsValue estima o valor de uma lista de ops no estado atual. Sem RNG e sem
// lookahead: pesos fixos sobre fatos do estado.
func (b *HeuristicBot) opsValue(g *Game, player int, ops []Op, depth int) int {
	if depth > 3 {
		return 0
	}
	s := g.State()
	p := s.Players[player]
	opp := s.Players[1-player]
	total := 0
	for i := range ops {
		op := &ops[i]
		if op.If != nil {
			if b.condNow(g, player, *op.If) {
				total += b.opsValue(g, player, op.Then, depth+1)
			} else {
				total += b.opsValue(g, player, op.Else, depth+1)
			}
			if op.Op == "conditional" {
				continue
			}
		}
		switch op.Op {
		case "damage":
			if op.Who == "self" {
				total -= op.N * 9
			} else {
				total += op.N * 11
				if op.N >= opp.Vitality+opp.Ward {
					total += 400
				}
			}
		case "lose_vitality":
			if op.Who == "opponent" {
				total += op.N * 10
			} else {
				total -= op.N * 8
			}
		case "heal":
			missing := p.MaxVitality - p.Vitality
			total += min(op.N, missing) * 8
		case "draw", "both_draw":
			if len(p.Deck) == 0 {
				total -= op.N * 12 // comprar sem baralho é Fadiga imediata
			} else {
				// Comprar acelera o próximo ciclo de Fadiga: o valor decai
				// conforme o baralho encolhe (13 → ~4 com 3 cartas restando).
				perCard := 4 + min(9, len(p.Deck))
				total += op.N * perCard
				if b.kitDrawsExtra(g, player) {
					total += op.N * 4
				}
			}
			if op.Op == "both_draw" {
				total -= op.N * 6 // o rival também compra
			}
		case "ward":
			total += op.N * 7
		case "temp_essence":
			total += op.N * 9
		case "emit_sigil", "emit_last_card_sigil":
			total += b.sigilValue(g, player)
		case "shift":
			total += b.shiftValue(g, player, op.N)
		case "set_eclipse":
			total += b.shiftValue(g, player, op.N-s.Eclipse)
		case "shift_chosen", "clock_direction_choice":
			total += 10
		case "status":
			if g.rs.ConfrontRules.TacticalSeals && op.Status == "exposto" {
				total += 22
			} else if op.Who == "opponent" || op.Status == "exposto" || op.Status == "sangramento" || op.Status == "maldicao" {
				total += max(1, op.N) * 8
			} else if op.Status == "veu" {
				total += 10
			}
		case "remove_own_bleeds":
			if len(p.Bleeds) > 0 {
				total += 14
			}
		case "remove_own_curse", "remove_curse_smart":
			if len(p.Curses) > 0 {
				total += 14
			}
		case "cost_mod":
			total += b.costModValue(g, player, op)
		case "open_extra_window":
			total += 12
		case "choose_discard":
			if op.Who == "opponent" {
				total += op.N * 9
			} else {
				total -= op.N * 5
			}
		case "recover_pick":
			total += max(1, op.N) * 10
		case "pick_top2", "reorder_top", "moon_return":
			total += 10
		case "destroy_relic_pick":
			if len(opp.Relics) > 0 {
				total += 22
			}
		case "suppress_opp_manifs":
			if len(opp.Manifs) > 0 {
				total += 18
			}
		case "peek_opp_hand_tax", "reveal_lock_assault", "declare_reveal_top":
			total += 9
		case "mirror_relic_pick", "copy_played_pick":
			total += 16
		case "exile_all_choices":
			// VR-060 só gera recompensa a cada bloco completo de quatro
			// cartas; jogar com 1–3 descartes seria desperdício real.
			batch := op.N
			if batch <= 0 {
				batch = 4
			}
			choices := min(len(p.Discard)/batch, 3)
			total += choices*16 - len(p.Discard)
		}
		// Operações de decisão continuam em `then` depois da escolha. Esse
		// valor fazia VR-049 parecer apenas um descarte, ocultando a compra 2.
		if op.Op != "conditional" && len(op.Then) > 0 {
			total += b.opsValue(g, player, op.Then, depth+1)
		}
	}
	return total
}

func (b *HeuristicBot) costModValue(g *Game, player int, op *Op) int {
	if op.Delta >= 0 {
		return 0
	}
	s := g.State()
	best := 0
	for _, id := range s.Players[player].Hand {
		inst := s.Cards[id]
		if inst == nil {
			continue
		}
		def := g.rs.Cards[inst.Def]
		if def == nil || (op.Type != "" && string(def.Type) != op.Type) ||
			(op.Faction != "" && def.Faction != op.Faction) {
			continue
		}
		saving := min(-op.Delta, g.effectiveCost(player, def, id))
		best = max(best, saving*14)
	}
	return best
}

// sigilValue: emitir Sigilo vale mais quando o kit persegue a tríade da
// rodada (passiva de Nyra, rituais) ou quando a mão tem condições de trilha
// prestes a fechar.
func (b *HeuristicBot) sigilValue(g *Game, player int) int {
	s := g.State()
	p := s.Players[player]
	if len(p.Trail) >= g.resonanceCap(player) {
		return 2 // trilha cheia: quase nada acontece
	}
	value := 9
	roundOwn := 0
	for _, rs := range s.RoundSigils {
		if rs.P == player {
			roundOwn++
		}
	}
	if b.kitChasesSigils(g, player) {
		value += 5
		if len(p.Trail) == 2 { // a próxima emissão dispara a passiva de tríade
			value += 14
		}
	}
	if roundOwn == 2 {
		value += 6
	}
	for _, id := range p.Hand {
		fx := b.cardFx(g, s.Cards[id].Def)
		if fx != nil && b.fxWantsTrail(fx) {
			value += 4
			break
		}
	}
	return value
}

// shiftValue: mover o Eclipse vale pelo alinhamento com o polo do kit; fechar
// o Total próprio vale muito, entregar o Total do rival custa caro.
func (b *HeuristicBot) shiftValue(g *Game, player int, delta int) int {
	if delta == 0 {
		return 0
	}
	s := g.State()
	pole := b.eclipsePole(g, player)
	oppPole := factionPole(g.rs.Champions[s.Players[1-player].Champion].Faction)
	next := max(-3, min(3, s.Eclipse+delta))
	moved := next - s.Eclipse
	value := 0
	if pole != 0 {
		value += moved * pole * 6
		if next == 3*pole {
			value += 28
		}
	}
	if oppPole != 0 && next == 3*oppPole {
		value -= 26
	}
	return value
}

// --- Escolha de ação ---

func (b *HeuristicBot) bestAction(g *Game, actions []Command) Command {
	bestScore := -1 << 30
	best := make([]Command, 0, len(actions))
	for _, action := range actions {
		score := b.scoreAction(g, action)
		switch {
		case score > bestScore:
			bestScore = score
			best = append(best[:0], action)
		case score == bestScore:
			best = append(best, action)
		}
	}
	if len(best) == 0 {
		return Command{}
	}
	return best[b.RNG.Intn(len(best))]
}

func (b *HeuristicBot) scoreAction(g *Game, cmd Command) int {
	s := g.State()
	switch cmd.Kind {
	case CmdKindPass:
		return 0
	case CmdKindUltimate:
		if s.Round >= 3 || s.Players[1-cmd.Player].Vitality <= 8 {
			return 118
		}
		return 12
	case CmdKindActivate:
		return b.scoreActivate(g, cmd)
	}
	inst := s.Cards[cmd.Card]
	if inst == nil {
		return -1000
	}
	def := g.rs.Cards[inst.Def]
	if g.rs.IsConfront() {
		costPenalty := def.Cost * 7
		switch s.Phase {
		case PhaseAssault:
			score := def.Confront.Power*12 - costPenalty
			if g.rs.ConfrontRules.TacticalSeals && s.Players[1-cmd.Player].Exposto {
				score += 24
			}
			return score
		case PhaseGuard:
			incoming := 0
			if s.Confront != nil {
				incoming = s.Confront.Power
			}
			prevent := def.Confront.Prevention
			if def.Confront.PreventAll {
				prevent = incoming
			}
			sealValue := 0
			if fx := g.rs.Effects.Cards[def.ID]; g.rs.ConfrontRules.TacticalSeals && fx != nil && fx.Guard != nil && fx.Guard.CounterRite {
				sealValue = 18
			}
			if incoming >= s.Players[cmd.Player].Vitality {
				return min(incoming, prevent)*20 - costPenalty + sealValue + 300
			}
			return min(incoming, prevent)*13 - costPenalty + sealValue
		case PhaseRite:
			fx := g.rs.Effects.Cards[def.ID]
			return 8 + b.opsValue(g, cmd.Player, fx.Rite.Steps, 0) - costPenalty
		}
	}
	fx := b.cardFx(g, inst.Def)
	player := cmd.Player
	var score int
	switch {
	case s.RiteReact != nil:
		// Counter: vale pelo custo do Rito anulado.
		riteCost := g.rs.Cards[s.RiteReact.Def].Cost
		score = 34 + riteCost*9 - def.Cost*4
	case s.Guard != nil:
		score = b.scoreGuardWindow(g, player, def, fx)
	default:
		score = b.scoreMainPlay(g, player, def, fx)
	}
	return score
}

func (b *HeuristicBot) scoreActivate(g *Game, cmd Command) int {
	s := g.State()
	inst := s.Cards[cmd.Card]
	if inst == nil {
		return -1000
	}
	fx := b.cardFx(g, inst.Def)
	if fx == nil || fx.Permanent == nil || fx.Permanent.Activated == nil {
		return 60
	}
	act := fx.Permanent.Activated
	value := 2 + b.opsValue(g, cmd.Player, act.Do, 0)
	value -= act.DiscardCost * 12
	value -= act.ExileCost * 4
	return value
}

func (b *HeuristicBot) scoreGuardWindow(g *Game, player int, def *CardDef, fx *CardFx) int {
	s := g.State()
	ctx := s.Guard
	incoming := ctx.BaseDamage*max(1, ctx.Instances) + ctx.FirstHitBonus
	p := s.Players[player]
	prevent := 0
	extras := 0
	if fx != nil && fx.Guard != nil {
		prevent = fx.Guard.Prevent
		for _, bonus := range fx.Guard.Bonuses {
			if b.condNow(g, player, bonus.If) {
				if bonus.Instead > 0 {
					prevent = bonus.Instead
				} else {
					prevent += bonus.Add
				}
			}
		}
		if fx.Guard.PreventAll {
			prevent = incoming
		}
		extras = b.opsValue(g, player, fx.Guard.OnPlay, 0) + b.opsValue(g, player, fx.Guard.After, 0)
	}
	blocked := min(prevent, incoming)
	waste := max(0, prevent-incoming)
	score := blocked*13 - waste*4 - def.Cost*5 + extras
	if incoming >= p.Vitality {
		score += 300 // defender ou morrer
	}
	if incoming <= 2 && p.Vitality > 14 {
		score -= 28 // economize a Guarda: aguente o arranhão
	}
	return score
}

func (b *HeuristicBot) scoreMainPlay(g *Game, player int, def *CardDef, fx *CardFx) int {
	s := g.State()
	p := s.Players[player]
	opp := s.Players[1-player]
	if fx == nil {
		// Sem definição na DSL (não deve ocorrer com o set completo).
		return 40 + def.Cost*4
	}
	score := 0
	switch {
	case fx.Assault != nil:
		ax := fx.Assault
		damage := ax.Damage
		for _, bonus := range ax.Bonuses {
			if b.condNow(g, player, bonus.If) {
				if bonus.Instead > 0 {
					damage = bonus.Instead
				} else {
					damage += bonus.Add
				}
			}
		}
		total := damage * max(1, ax.Instances)
		score = total*11 + b.opsValue(g, player, ax.After, 0)
		if total >= opp.Vitality+opp.Ward-max(0, ax.IgnoreWard) {
			score += 400
		}
		if ax.PreExile {
			score -= 6
		}
	case fx.Rite != nil:
		rx := fx.Rite
		score = 6 + b.opsValue(g, player, rx.Steps, 0)
		sacPenalty := 6
		if p.Vitality*3 <= p.MaxVitality {
			sacPenalty = 15
		}
		score -= rx.Sacrifice * sacPenalty
	case fx.Permanent != nil:
		px := fx.Permanent
		score = 12 + b.opsValue(g, player, px.OnEnter, 0)
		for i := range px.Triggers {
			trigger := &px.Triggers[i]
			per := b.opsValue(g, player, trigger.Do, 0)
			switch trigger.On {
			case "round_start":
				score += per
			case "sigil_emitted":
				if b.kitChasesSigils(g, player) {
					score += per * 4 / 5
				} else {
					score += per / 3
				}
			default:
				score += per * 2 / 5
			}
			// Gatilho condicionado a um estado ainda distante vale menos —
			// mas kits que perseguem a condição (Sigilos) chegam lá.
			if trigger.If != nil && !b.condNow(g, player, *trigger.If) {
				if !b.kitChasesSigils(g, player) {
					score -= per / 2
				}
			}
		}
		if px.Activated != nil {
			score += 14
		}
		score += max(0, (7-s.Round)*3) // permanentes pagam com o tempo
	case fx.Guard != nil:
		// Guarda fora de janela não é legal; valor defensivo só reativo.
		score = 10
	}
	// Eficiência: gastar tudo numa carta média perde para duas jogadas boas.
	score -= def.Cost * 2
	return score
}

// --- Postura ---

func (b *HeuristicBot) stance(g *Game, player int) Stance {
	s := g.State()
	p := s.Players[player]
	assaultValue, riteValue, guards := 0, 0, 0
	for _, id := range p.Hand {
		def := g.rs.Cards[s.Cards[id].Def]
		fx := b.cardFx(g, s.Cards[id].Def)
		switch def.Type {
		case TypeAssalto:
			if fx != nil && fx.Assault != nil {
				assaultValue += fx.Assault.Damage * max(1, fx.Assault.Instances)
			} else {
				assaultValue += 2
			}
		case TypeRito, TypeReliquia, TypeManifestacao:
			riteValue += 2 + def.Cost
		case TypeGuarda:
			guards++
		}
	}
	if p.Vitality*3 <= p.MaxVitality && guards > 0 {
		return StanceVigilia
	}
	if opp := s.Players[1-player]; opp.Vitality <= 8 && assaultValue >= 4 {
		return StancePredacao // feche o jogo
	}
	if assaultValue >= 6 && assaultValue >= riteValue {
		return StancePredacao
	}
	if riteValue >= 6 {
		return StanceArcano
	}
	if guards >= 2 {
		return StanceVigilia
	}
	if assaultValue >= riteValue {
		return StancePredacao
	}
	return StanceArcano
}

// --- Mulligan ---

func (b *HeuristicBot) mulligan(g *Game, player int) Command {
	s := g.State()
	chasesSigils := b.kitChasesSigils(g, player)
	picks := make([]string, 0, 2)
	guardsKept := 0
	for _, id := range s.Players[player].Hand {
		def := g.rs.Cards[s.Cards[id].Def]
		if def.Type == TypeGuarda && guardsKept == 0 {
			guardsKept++
			continue
		}
		if len(picks) >= 2 {
			break
		}
		fx := b.cardFx(g, s.Cards[id].Def)
		if chasesSigils && fx != nil && b.fxEmitsSigil(fx) && def.Cost <= 3 {
			continue // combustível de cadeia fica
		}
		if def.Cost >= 5 {
			picks = append(picks, id)
			continue
		}
		if def.Cost == 4 && len(picks) == 0 {
			picks = append(picks, id)
		}
	}
	return Command{Player: player, Kind: CmdKindMulligan, Cards: picks}
}

// --- Decisões ---

func (b *HeuristicBot) answerDecision(g *Game, d *Decision) Command {
	cmd := Command{Player: d.Player, Kind: CmdKindChoose, DecisionID: d.ID}
	options := append([]string{}, d.Options...)
	if len(options) == 0 {
		return cmd
	}
	value := func(option string) int {
		if inst := g.State().Cards[option]; inst != nil {
			def := g.rs.Cards[inst.Def]
			score := def.Cost * 10
			if def.Type == TypeAssalto {
				score += 15
			}
			if fx := b.cardFx(g, inst.Def); fx != nil && b.fxEmitsSigil(fx) && b.kitChasesSigils(g, d.Player) {
				score += 18
			}
			return score
		}
		return 0
	}
	switch d.Kind {
	case DecDiscardN, DecExilePick:
		sort.SliceStable(options, func(i, j int) bool { return value(options[i]) < value(options[j]) })
		cmd.Cards = options[:decisionCount(d, len(options))]
	case DecRecoverPick, DecReEmitSigils:
		sort.SliceStable(options, func(i, j int) bool { return value(options[i]) > value(options[j]) })
		cmd.Cards = options[:decisionCount(d, len(options))]
	case DecReorderTop:
		sort.SliceStable(options, func(i, j int) bool { return value(options[i]) > value(options[j]) })
		cmd.Cards = options
	case DecExileSelfHeal:
		cmd.Cards = []string{pickOption(options, g.State().Players[d.Player].Vitality*2 <= g.State().Players[d.Player].MaxVitality, "yes", "no")}
	case DecMillTop, DecScryBottom:
		cmd.Cards = []string{pickOption(options, false, "yes", "no")}
	case DecSwapEclipse:
		pole := b.eclipsePole(g, d.Player)
		wantSwap := (pole > 0 && g.State().Eclipse < 0) || (pole < 0 && g.State().Eclipse > 0)
		cmd.Cards = []string{pickOption(options, wantSwap, "yes", "no")}
	case DecDirection:
		if b.eclipsePole(g, d.Player) >= 0 {
			cmd.Cards = []string{findOption(options, "noite")}
		} else {
			cmd.Cards = []string{findOption(options, "aurora")}
		}
	case DecFormulaChoice, DecOrenChoice, DecVorenChoice:
		if opp := g.State().Players[1-d.Player]; opp.Vitality <= 6 {
			cmd.Cards = []string{preferLabel(options, "dano", "compr", "cura")}
		} else if p := g.State().Players[d.Player]; p.Vitality*2 <= p.MaxVitality {
			cmd.Cards = []string{preferLabel(options, "cura", "compr", "dano")}
		} else {
			cmd.Cards = []string{preferLabel(options, "compr", "dano", "cura")}
		}
	case DecPickTop2, DecOppDiscardPick, DecDestroyRelicPick, DecTaxTypePick, DecPickSigil,
		DecRevealTax, DecLockAssault, DecDeclareType, DecMirrorRelic, DecCopyPlayed, DecEddaReturn:
		sort.SliceStable(options, func(i, j int) bool { return value(options[i]) > value(options[j]) })
		cmd.Cards = options[:1]
	default:
		cmd.Cards = options[:decisionCount(d, len(options))]
	}
	return cmd
}

func decisionCount(d *Decision, available int) int {
	n := d.N
	if d.MinN > 0 {
		n = d.MinN
	}
	if n < 1 {
		n = 1
	}
	return min(n, available)
}

func pickOption(options []string, condition bool, yes, no string) string {
	if condition {
		return findOption(options, yes)
	}
	return findOption(options, no)
}

func findOption(options []string, needle string) string {
	for _, option := range options {
		if strings.Contains(strings.ToLower(option), needle) {
			return option
		}
	}
	return options[0]
}

func preferLabel(options []string, needles ...string) string {
	for _, needle := range needles {
		for _, option := range options {
			if strings.Contains(strings.ToLower(option), needle) {
				return option
			}
		}
	}
	return options[0]
}
