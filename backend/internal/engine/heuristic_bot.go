package engine

import (
	"sort"
	"strings"
)

// HeuristicBot é o nível 2 do plano de balanceamento. Ele não antecipa RNG ou
// chama Apply em cópias: pontua somente comandos já enumerados como legais.
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
	}
	return Command{}, false
}

func (b *HeuristicBot) mulligan(g *Game, player int) Command {
	s := g.State()
	picks := make([]string, 0, 2)
	guardsKept := 0
	for _, id := range s.Players[player].Hand {
		def := g.rs.Cards[s.Cards[id].Def]
		if def.Type == TypeGuarda && guardsKept == 0 {
			guardsKept++
			continue
		}
		if def.Cost >= 4 && len(picks) < 2 {
			picks = append(picks, id)
		}
	}
	return Command{Player: player, Kind: CmdKindMulligan, Cards: picks}
}

func (b *HeuristicBot) stance(g *Game, player int) Stance {
	s := g.State()
	assaults, rites, guards := 0, 0, 0
	for _, id := range s.Players[player].Hand {
		switch g.rs.Cards[s.Cards[id].Def].Type {
		case TypeAssalto:
			assaults++
		case TypeRito:
			rites++
		case TypeGuarda:
			guards++
		}
	}
	if s.Players[player].Vitality*3 <= s.Players[player].MaxVitality && guards > 0 {
		return StanceVigilia
	}
	if assaults >= rites && assaults >= 2 {
		return StancePredacao
	}
	if rites >= 2 {
		return StanceArcano
	}
	return StanceVigilia
}

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
	if cmd.Kind == CmdKindPass {
		return 0
	}
	if cmd.Kind == CmdKindUltimate {
		return 125
	}
	if cmd.Kind == CmdKindActivate {
		return 95
	}
	inst := s.Cards[cmd.Card]
	if inst == nil {
		return -1000
	}
	def := g.rs.Cards[inst.Def]
	score := 0
	switch def.Type {
	case TypeAssalto:
		score = 80 + def.Cost*8
	case TypeGuarda:
		score = 105 - def.Cost*3
		if s.Guard != nil && s.Guard.BaseDamage >= 4 {
			score += 20
		}
	case TypeRito:
		score = 48 + def.Cost*4
	case TypeReliquia, TypeManifestacao:
		score = 55 + def.Cost*2
	}
	champFaction := g.rs.Champions[s.Players[cmd.Player].Champion].Faction
	if prefersNight(champFaction) {
		score += def.EclipseShift * 5
	} else if champFaction == "Ordem Solara" {
		score -= def.EclipseShift * 5
	}
	text := strings.ToLower(def.RulesText)
	if s.Players[cmd.Player].Vitality <= 10 && (strings.Contains(text, "sacrifique") || strings.Contains(text, "perca vitalidade")) {
		score -= 100
	}
	if s.Players[cmd.Player].Vitality*2 <= s.Players[cmd.Player].MaxVitality && strings.Contains(text, "cure") {
		score += 18
	}
	return score
}

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
		wantSwap := (prefersNight(g.rs.Champions[g.State().Players[d.Player].Champion].Faction) && g.State().Eclipse < 0) ||
			(g.rs.Champions[g.State().Players[d.Player].Champion].Faction == "Ordem Solara" && g.State().Eclipse > 0)
		cmd.Cards = []string{pickOption(options, wantSwap, "yes", "no")}
	case DecDirection:
		if prefersNight(g.rs.Champions[g.State().Players[d.Player].Champion].Faction) {
			cmd.Cards = []string{findOption(options, "noite")}
		} else {
			cmd.Cards = []string{findOption(options, "aurora")}
		}
	case DecFormulaChoice, DecOrenChoice, DecVorenChoice:
		cmd.Cards = []string{preferLabel(options, "dano", "compr", "cura")}
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

func prefersNight(faction string) bool { return faction == "Casa Vhal" || faction == "Matilha Varka" }

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
