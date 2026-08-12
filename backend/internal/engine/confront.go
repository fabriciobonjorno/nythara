package engine

import (
	"fmt"
	"sort"
)

const (
	ConfrontAssaultSeal = "Selo do Assalto"
	ConfrontGuardSeal   = "Exposto"
	ConfrontRiteSeal    = "Selo do Rito"
)

// ImplementationEntry torna explícito por que cada carta participa ou não
// do Modo Confronto. É usado por testes, LiveOps e pela interface.
type ImplementationEntry struct {
	CardID string `json:"card_id"`
	Legal  bool   `json:"legal"`
	Reason string `json:"reason,omitempty"`
}

// ConfrontImplementationReport devolve a cobertura integral e ordenada do
// catálogo; nenhuma exclusão fica implícita.
func (rs *Ruleset) ConfrontImplementationReport() []ImplementationEntry {
	out := make([]ImplementationEntry, 0, len(rs.CardList))
	for _, card := range rs.CardList {
		entry := ImplementationEntry{CardID: card.ID}
		if card.Confront != nil {
			entry.Legal, entry.Reason = card.Confront.Legal, card.Confront.Reason
		} else {
			entry.Reason = "carta não compilada para o Modo Confronto"
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CardID < out[j].CardID })
	return out
}

// applyCardAdjustments reescreve valores de carta apenas nesta versão. Cada
// Ruleset compila suas próprias cópias do catálogo, então mexer aqui não vaza
// para as versões já publicadas nem muda o significado de um replay antigo.
func applyCardAdjustments(rs *Ruleset) {
	for id, adjustment := range rs.ConfrontRules.CardAdjustments {
		card, fx := rs.Cards[id], rs.Effects.Cards[id]
		if card == nil || fx == nil {
			continue
		}
		if adjustment.Cost != nil {
			card.Cost = max(0, *adjustment.Cost)
		}
		if adjustment.Damage != nil && fx.Assault != nil {
			fx.Assault.Damage = max(0, *adjustment.Damage)
		}
		if adjustment.Prevent != nil && fx.Guard != nil {
			fx.Guard.Prevent = max(0, *adjustment.Prevent)
		}
	}
}

func prepareConfrontRuleset(rs *Ruleset) {
	applyCardAdjustments(rs)
	for _, card := range rs.CardList {
		legacyText := card.RulesText
		profile := &ConfrontCardProfile{Defendable: true}
		fx := rs.Effects.Cards[card.ID]
		switch card.Type {
		case TypeReliquia, TypeManifestacao:
			profile.Reason = "Relíquias e Manifestações pertencem ao ruleset legado"
		case TypeAssalto:
			if fx == nil || fx.Assault == nil {
				profile.Reason = "Assalto sem efeito executável"
				break
			}
			a := fx.Assault
			profile.Legal = true
			profile.Power = a.Damage*max(1, a.Instances) + rs.ConfrontRules.PowerBonus
			profile.PowerMax = profile.Power
			if rs.ConfrontRules.TacticalSeals {
				profile.Power, profile.PowerMax = confrontAssaultPowerRange(card, a, rs.ConfrontRules.PowerBonus)
			}
			if a.UndefendableIf != nil && a.UndefendableIf.Cond == "always" {
				profile.Defendable = false
			}
			if a.PreExile || confrontUnsupportedBonuses(a.Bonuses) != "" ||
				(a.UndefendableIf != nil && !confrontCondAllowed(*a.UndefendableIf)) ||
				confrontUnsupportedOps(a.After, rs.ConfrontRules.TacticalSeals, rs.ConfrontRules.Decisions) != "" {
				profile.Adapted = true
				profile.Reason = "efeito secundário do ruleset legado removido; Poder base preservado"
			}
			card.RulesText = fmt.Sprintf("Poder base %d. Custa %d de Vitalidade.", profile.Power, card.Cost)
			if !profile.Adapted {
				card.RulesText += " " + legacyText
			}
		case TypeGuarda:
			if fx == nil || fx.Guard == nil {
				profile.Reason = "Guarda sem efeito executável"
				break
			}
			gd := fx.Guard
			profile.Legal = true
			profile.Prevention, profile.PreventAll = gd.Prevent, gd.PreventAll
			profile.PreventionMax = profile.Prevention
			if rs.ConfrontRules.TacticalSeals && !gd.PreventAll {
				profile.Prevention, profile.PreventionMax = confrontBonusRange(card.Cost, gd.Prevent, gd.Bonuses)
			}
			if rs.ConfrontRules.TacticalSeals && gd.CounterRite {
				profile.Prevention, profile.PreventionMax = max(2, gd.Prevent), max(2, gd.Prevent)
				profile.Adapted = true
				profile.Reason = "contra-Rito convertido em Selo do Rito com Prevenção 2"
			}
			if (!rs.ConfrontRules.TacticalSeals && gd.CounterRite) || gd.ToHandAfter || gd.SelfCostAfter != 0 ||
				confrontUnsupportedBonuses(gd.Bonuses) != "" ||
				confrontUnsupportedOps(append(append([]Op{}, gd.OnPlay...), gd.After...), rs.ConfrontRules.TacticalSeals, rs.ConfrontRules.Decisions) != "" {
				profile.Adapted = true
				if profile.Reason == "" {
					profile.Reason = "efeito secundário do ruleset legado removido; Prevenção preservada"
				}
			}
			counterRite := rs.ConfrontRules.TacticalSeals && gd.CounterRite
			if !gd.PreventAll && !counterRite && rs.ConfrontRules.GuardBonus != 0 {
				profile.Prevention += rs.ConfrontRules.GuardBonus
				profile.PreventionMax += rs.ConfrontRules.GuardBonus
			}
			label := fmt.Sprintf("Prevenção %d", profile.Prevention)
			if gd.PreventAll {
				label = "Prevenção total"
			}
			card.RulesText = fmt.Sprintf("%s. Custa %d de Vitalidade.", label, card.Cost)
			if !profile.Adapted {
				card.RulesText += " " + legacyText
			}
		case TypeRito:
			if fx == nil || fx.Rite == nil {
				profile.Reason = "Rito sem efeito executável"
				break
			}
			rite := fx.Rite
			for _, cond := range rite.Requires {
				if !confrontCondAllowed(cond) {
					profile.Reason = "requisito depende de sistema removido"
					break
				}
			}
			if profile.Reason != "" {
				break
			}
			if reason := confrontUnsupportedOps(rite.Steps, rs.ConfrontRules.TacticalSeals, rs.ConfrontRules.Decisions); reason != "" {
				profile.Reason = reason
				break
			}
			profile.Legal = true
			if rs.ConfrontRules.TacticalSeals && confrontHasOp(rite.Steps, "reveal_lock_assault") {
				profile.Adapted = true
				profile.Reason = "revelação de mão convertida em Selo do Assalto público"
			}
			card.RulesText = fmt.Sprintf("Custa %d de Vitalidade. %s", card.Cost+rite.Sacrifice, card.RulesText)
		}
		if !profile.Legal {
			card.RulesText = "Fora do Modo Confronto: " + profile.Reason + "."
		}
		if profile.Legal && rs.ConfrontRules.TacticalSeals {
			prepareConfrontCardCopy(card, fx, profile, rs.ConfrontRules)
			card.RulesText = profile.TacticalText
		}
		card.Confront = profile
	}
}

func confrontUnsupportedBonuses(bonuses []Bonus) string {
	for _, bonus := range bonuses {
		if !confrontCondAllowed(bonus.If) {
			return "bônus depende de Eclipse, Ressonância, permanentes ou outra regra removida"
		}
	}
	return ""
}

func confrontUnsupportedOps(ops []Op, tacticalSeals, decisions bool) string {
	for _, op := range ops {
		switch op.Op {
		case "damage", "heal", "draw", "both_draw", "ward", "lose_vitality",
			"status", "remove_own_bleeds", "remove_own_curse", "remove_curse_smart",
			"remove_veil_expose":
		case "reveal_lock_assault":
			if !tacticalSeals {
				return "operação reveal_lock_assault depende do ruleset legado"
			}
		case "choose_discard":
			// Sem decisões, o motivo é o texto histórico EXATO: as versões já
			// publicadas foram sincronizadas com ele, e o guard do catálogo
			// rejeita — corretamente — qualquer byte diferente sob o mesmo
			// número de versão.
			if !decisions {
				return fmt.Sprintf("operação %s depende do ruleset legado", op.Op)
			}
		case "conditional":
			if op.If == nil || !confrontCondAllowed(*op.If) {
				return "efeito condicional depende de sistema removido"
			}
		default:
			return fmt.Sprintf("operação %s depende do ruleset legado", op.Op)
		}
		if reason := confrontUnsupportedOps(op.Then, tacticalSeals, decisions); reason != "" {
			return reason
		}
		if reason := confrontUnsupportedOps(op.Else, tacticalSeals, decisions); reason != "" {
			return reason
		}
	}
	return ""
}

func confrontCondAllowed(cond Cond) bool {
	switch cond.Cond {
	case "always", "sacrificed_this_round", "took_damage_round", "hand_less_than_opp",
		"hand_ge", "opp_hand_ge", "opp_vitality_le", "prevented_all", "opp_veiled",
		"discard_ge", "opp_assaults_eq", "own_assaults_eq", "opp_discard_nonempty":
		return true
	default:
		return false
	}
}

func (g *Game) applyConfront(cmd Command) error {
	// Decisão pendente trava a mesa: aceitar outra ação no meio deixaria o
	// estado ambíguo no replay.
	if g.s.Pending != nil && cmd.Kind != CmdKindChoose {
		return errCmd(ErrPendingChoice, "há uma decisão pendente do jogador %d", g.s.Pending.Player)
	}
	switch cmd.Kind {
	case CmdKindPlay:
		return g.applyConfrontPlay(cmd)
	case CmdKindPass:
		return g.applyConfrontPass(cmd)
	case CmdKindChoose:
		return g.applyConfrontChoose(cmd)
	default:
		return errCmd(ErrBadCommand, "comando %q não existe no Modo Confronto", cmd.Kind)
	}
}

func (g *Game) applyConfrontPlay(cmd Command) error {
	s := g.s
	if cmd.Player != s.Active {
		return errCmd(ErrWrongPlayer, "a ação pertence ao jogador %d", s.Active)
	}
	p := s.Players[cmd.Player]
	if !containsID(p.Hand, cmd.Card) {
		return errCmd(ErrInvalidCard, "carta %q não está na mão", cmd.Card)
	}
	inst := s.Cards[cmd.Card]
	def := g.rs.Cards[inst.Def]
	if def == nil || def.Confront == nil || !def.Confront.Legal {
		return errCmd(ErrNotImplemented, "%s não pertence ao pool do Modo Confronto", inst.Def)
	}

	switch s.Phase {
	case PhaseAssault:
		if def.Type != TypeAssalto {
			return errCmd(ErrWrongPhase, "na fase de Assalto só um Assalto pode ser jogado")
		}
		if err := g.payConfrontCost(cmd.Player, def.Cost, def.ID); err != nil {
			return err
		}
		g.moveToClash(cmd.Player, inst)
		p.AssaultsRound++
		fx := g.rs.Effects.Cards[def.ID].Assault
		ctx := &GuardCtx{Attacker: cmd.Player, Defender: 1 - cmd.Player,
			AssaultInst: inst.ID, AssaultDef: def.ID}
		power := g.confrontApplyBonuses(fx.Damage, fx.Bonuses,
			&opCtx{player: cmd.Player, source: def.ID, inst: inst.ID, guard: ctx})*max(1, fx.Instances) + g.rs.ConfrontRules.PowerBonus
		if s.Round == 1 && cmd.Player == s.Initiative {
			power = max(0, power-g.rs.ConfrontRules.FirstTurnPenalty)
		}
		defendable := true
		if fx.UndefendableIf != nil && confrontCondAllowed(*fx.UndefendableIf) {
			defendable = !g.evalCond(*fx.UndefendableIf, &opCtx{player: cmd.Player, source: def.ID, guard: ctx})
		}
		if g.rs.ConfrontRules.TacticalSeals && s.Players[1-cmd.Player].Exposto {
			defendable = false
			power += g.rs.ConfrontRules.ExposedPowerBonus
			s.Players[1-cmd.Player].Exposto = false
			g.emit(Event{Kind: EvStatusExpired, P: 1 - cmd.Player, S: ConfrontGuardSeal, Def: def.ID})
		}
		s.Confront = &ConfrontCtx{Attacker: cmd.Player, Defender: 1 - cmd.Player,
			AssaultInst: inst.ID, AssaultDef: def.ID, Power: power, Defendable: defendable}
		g.emit(Event{Kind: EvCardPlayed, P: cmd.Player, Card: inst.ID, Def: def.ID, N: def.Cost})
		g.emit(Event{Kind: EvConfrontationOpened, P: cmd.Player, Card: inst.ID, Def: def.ID, N: power, To: 1 - cmd.Player})
		if !defendable {
			g.resolveConfront(true)
			return nil
		}
		s.Phase, s.Active = PhaseGuard, 1-cmd.Player
		g.emit(Event{Kind: EvWindowOpened, P: s.Active, S: string(PhaseGuard)})
		return nil

	case PhaseGuard:
		if s.Confront == nil || cmd.Player != s.Confront.Defender {
			return errCmd(ErrWrongPlayer, "a Guarda pertence ao defensor")
		}
		if def.Type != TypeGuarda {
			return errCmd(ErrWrongPhase, "na janela de Guarda só uma Guarda pode ser jogada")
		}
		if err := g.payConfrontCost(cmd.Player, def.Cost, def.ID); err != nil {
			return err
		}
		g.moveToClash(cmd.Player, inst)
		p.GuardsRound++
		fx := g.rs.Effects.Cards[def.ID].Guard
		legacyCtx := &GuardCtx{Attacker: s.Confront.Attacker, Defender: s.Confront.Defender,
			AssaultInst: s.Confront.AssaultInst, AssaultDef: s.Confront.AssaultDef,
			BaseDamage: s.Confront.Power, Instances: 1, GuardInst: inst.ID}
		prevention := g.confrontApplyBonuses(fx.Prevent, fx.Bonuses,
			&opCtx{player: cmd.Player, source: def.ID, inst: inst.ID, guard: legacyCtx})
		switch {
		case fx.PreventAll:
			prevention = s.Confront.Power
		case g.rs.ConfrontRules.TacticalSeals && fx.CounterRite:
			// O valor desta Guarda está no Selo do Rito, não na Prevenção: dar
			// o bônus aqui a tornaria estritamente superior às demais.
			prevention = max(2, prevention)
		default:
			prevention += g.rs.ConfrontRules.GuardBonus
		}
		s.Confront.GuardInst, s.Confront.GuardDef, s.Confront.Prevention = inst.ID, def.ID, prevention
		g.emit(Event{Kind: EvCardPlayed, P: cmd.Player, Card: inst.ID, Def: def.ID, N: def.Cost})
		g.emit(Event{Kind: EvGuardCommitted, P: cmd.Player, Card: inst.ID, Def: def.ID, N: prevention,
			From: s.Confront.Power, To: s.Confront.Attacker})
		g.confrontRunOps(fx.OnPlay, &opCtx{player: cmd.Player, source: def.ID, inst: inst.ID, guard: legacyCtx})
		if g.rs.ConfrontRules.TacticalSeals && fx.CounterRite {
			attacker := s.Confront.Attacker
			s.Players[attacker].RiteSealUntil = max(s.Players[attacker].RiteSealUntil, s.Round)
			g.emit(Event{Kind: EvStatusApplied, P: attacker, S: ConfrontRiteSeal, Def: def.ID, N: s.Round})
		}
		g.resolveConfront(false)
		return nil

	case PhaseRite:
		if def.Type != TypeRito {
			return errCmd(ErrWrongPhase, "na fase de Rito só um Rito pode ser jogado")
		}
		rite := g.rs.Effects.Cards[def.ID].Rite
		if rite.TargetsOpponent && veilActive(s.Players[1-cmd.Player], s.Round) {
			return errCmd(ErrIllegalTarget, "o rival está sob Véu")
		}
		for _, cond := range rite.Requires {
			if !g.evalCond(cond, &opCtx{player: cmd.Player, source: def.ID}) {
				return errCmd(ErrRequirement, "%s: requisito não satisfeito", def.ID)
			}
		}
		if err := g.payConfrontCost(cmd.Player, def.Cost+rite.Sacrifice, def.ID); err != nil {
			return err
		}
		p.RitesRound++
		g.moveToClash(cmd.Player, inst)
		g.emit(Event{Kind: EvCardPlayed, P: cmd.Player, Card: inst.ID, Def: def.ID, N: def.Cost + rite.Sacrifice})
		g.confrontRunOps(rite.Steps, &opCtx{player: cmd.Player, source: def.ID, inst: inst.ID})
		if s.Pending != nil || len(s.DecQueue) > 0 {
			s.PendingConfrontRite = &ConfrontRiteFinalize{Player: cmd.Player, Inst: inst.ID}
			return nil
		}
		g.finalizeConfrontRite(cmd.Player, inst.ID)
		return nil
	default:
		return errCmd(ErrWrongPhase, "não é possível jogar carta na fase %s", s.Phase)
	}
}

func (g *Game) confrontApplyBonuses(base int, bonuses []Bonus, ctx *opCtx) int {
	value := base
	for _, bonus := range bonuses {
		if !confrontCondAllowed(bonus.If) || !g.evalCond(bonus.If, ctx) {
			continue
		}
		if bonus.Instead != 0 {
			value = bonus.Instead
		} else {
			value += bonus.Add
		}
	}
	return value
}

func (g *Game) applyConfrontPass(cmd Command) error {
	s := g.s
	if cmd.Player != s.Active {
		return errCmd(ErrWrongPlayer, "a ação pertence ao jogador %d", s.Active)
	}
	switch s.Phase {
	case PhaseAssault:
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: string(PhaseAssault)})
		g.openConfrontRite(cmd.Player)
	case PhaseGuard:
		if s.Confront == nil || cmd.Player != s.Confront.Defender {
			return errCmd(ErrWrongPlayer, "somente o defensor pode passar a Guarda")
		}
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: string(PhaseGuard)})
		g.resolveConfront(false)
	case PhaseRite:
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: string(PhaseRite)})
		g.finishConfrontTurn(cmd.Player)
	default:
		return errCmd(ErrWrongPhase, "não há janela para passar")
	}
	return nil
}

func (g *Game) payConfrontCost(player, cost int, source string) error {
	p := g.s.Players[player]
	if discount := g.championDiscountFor(player, source); discount > 0 {
		cost = max(0, cost-discount)
		g.emit(Event{Kind: EvStatusApplied, P: player, N: discount, S: championPowerEvent, Def: source})
	}
	if cost < 0 || p.Vitality-cost < 1 {
		return errCmd(ErrCantAfford, "%s custa %d de Vitalidade e deve restar pelo menos 1", source, cost)
	}
	if cost == 0 {
		return nil
	}
	from := p.Vitality
	p.Vitality -= cost
	p.SacrificesRound++
	g.emit(Event{Kind: EvVitalitySpent, P: player, N: cost, From: from, To: p.Vitality, S: source})
	return nil
}

func (g *Game) moveToClash(player int, inst *CardInstance) {
	p := g.s.Players[player]
	p.Hand, _ = removeID(p.Hand, inst.ID)
	inst.Zone = ZoneClash
}

func (g *Game) discardClash(id string) {
	inst := g.s.Cards[id]
	if inst == nil || inst.Zone != ZoneClash {
		return
	}
	inst.Zone = ZoneDiscard
	p := g.s.Players[inst.Owner]
	p.Discard = append(p.Discard, id)
	g.emit(Event{Kind: EvCardDiscarded, P: inst.Owner, Card: id, Def: inst.Def})
}

func (g *Game) resolveConfront(undefendable bool) {
	s := g.s
	ctx := s.Confront
	if ctx == nil {
		return
	}
	legacyCtx := &GuardCtx{Attacker: ctx.Attacker, Defender: ctx.Defender,
		AssaultInst: ctx.AssaultInst, AssaultDef: ctx.AssaultDef, BaseDamage: ctx.Power,
		Instances: 1, GuardInst: ctx.GuardInst, Prevention: ctx.Prevention}
	raw := max(0, ctx.Power-ctx.Prevention)
	// Teto de vazamento: uma Guarda comprometida nunca deixa passar mais que
	// este valor. Um bônus fixo de Prevenção não acompanha uma curva de Poder
	// larga — sobra contra o Assalto fraco e falta contra o forte —, e é isso
	// que faz a duração da partida depender do quanto o baralho é agressivo.
	// O teto torna defender confiável em toda a curva sem mexer em carta.
	if cap := g.rs.ConfrontRules.GuardLeakCap; cap > 0 && ctx.GuardInst != "" && raw > cap {
		raw = cap
	}
	net := g.confrontDealDamage(ctx.Defender, raw, ctx.AssaultDef)
	if net > 0 {
		g.championOnTrigger(ctx.Attacker, "connected_hit")
		if ctx.GuardInst == "" {
			g.championOnTrigger(ctx.Defender, "undefended_hit")
		}
	}
	outcome := "assault"
	shattered := ctx.GuardInst
	if ctx.GuardInst == "" {
		outcome = "direct"
		shattered = ""
	} else if net == 0 {
		outcome = "guard"
		shattered = ctx.AssaultInst
		legacyCtx.PreventedAll = true
	}
	legacyCtx.NetTotal = net
	g.emit(Event{Kind: EvConfrontationResolved, P: ctx.Attacker, Card: ctx.AssaultInst,
		Def: ctx.AssaultDef, N: net, From: ctx.Power, To: ctx.Prevention, S: outcome})
	if shattered != "" {
		inst := s.Cards[shattered]
		g.emit(Event{Kind: EvCardShattered, P: inst.Owner, Card: inst.ID, Def: inst.Def, S: outcome})
		g.championOnTrigger(inst.Owner, "own_shatter")
	}
	assaultFx := g.rs.Effects.Cards[ctx.AssaultDef].Assault
	g.confrontRunOps(assaultFx.After, &opCtx{player: ctx.Attacker, source: ctx.AssaultDef,
		inst: ctx.AssaultInst, guard: legacyCtx})
	if ctx.GuardDef != "" {
		guardFx := g.rs.Effects.Cards[ctx.GuardDef].Guard
		g.confrontRunOps(guardFx.After, &opCtx{player: ctx.Defender, source: ctx.GuardDef,
			inst: ctx.GuardInst, guard: legacyCtx})
	}
	g.discardClash(ctx.AssaultInst)
	if ctx.GuardInst != "" {
		g.discardClash(ctx.GuardInst)
	}
	attacker := ctx.Attacker
	s.Confront = nil
	g.checkWin()
	if s.Over {
		return
	}
	g.openConfrontRite(attacker)
	_ = undefendable // preserva no log pela ausência de guard_committed
}

func (g *Game) startConfrontTurn(player int, first bool) {
	s := g.s
	s.Round++
	s.Active = player
	for _, p := range s.Players {
		p.AssaultsRound, p.GuardsRound, p.RitesRound = 0, 0, 0
		p.SacrificesRound, p.DamageTakenRound, p.DrawsRound = 0, 0, 0
	}
	g.resolveConfrontDawn(player)
	pressureStart := g.rs.ConfrontRules.PressureStartTurn
	if pressureStart > 0 && s.Round >= pressureStart && !s.Over {
		pressure := (s.Round - pressureStart + 1) * g.rs.ConfrontRules.PressureBaseLoss
		g.emit(Event{Kind: EvStatusApplied, P: -1, N: pressure, S: "Pressão de Nythara"})
		g.confrontLoseVitality(0, pressure, "Pressão de Nythara")
		g.confrontLoseVitality(1, pressure, "Pressão de Nythara")
		g.checkConfrontPressureWin()
	}
	if (!first || g.rs.ConfrontRules.DrawOnFirstTurn) && !s.Over {
		g.drawConfront(player)
	}
	g.checkWin()
	if s.Over {
		return
	}
	g.emit(Event{Kind: EvTurnStarted, P: player, N: s.Round})
	s.Phase = PhaseAssault
	if g.rs.ConfrontRules.TacticalSeals && s.Players[player].AssaultSealUntil >= s.Round {
		s.Players[player].AssaultSealUntil = 0
		g.emit(Event{Kind: EvStatusExpired, P: player, S: ConfrontAssaultSeal})
		g.emit(Event{Kind: EvPass, P: player, S: string(PhaseAssault), Def: ConfrontAssaultSeal})
		g.openConfrontRite(player)
		return
	}
	g.emit(Event{Kind: EvWindowOpened, P: player, S: string(PhaseAssault)})
}

func (g *Game) openConfrontRite(player int) {
	if g.s.Over {
		return
	}
	g.s.Active, g.s.Phase = player, PhaseRite
	if g.rs.ConfrontRules.TacticalSeals && g.s.Players[player].RiteSealUntil >= g.s.Round {
		g.s.Players[player].RiteSealUntil = 0
		g.emit(Event{Kind: EvStatusExpired, P: player, S: ConfrontRiteSeal})
		g.emit(Event{Kind: EvPass, P: player, S: string(PhaseRite), Def: ConfrontRiteSeal})
		g.finishConfrontTurn(player)
		return
	}
	g.emit(Event{Kind: EvWindowOpened, P: player, S: string(PhaseRite)})
}

// Se a pressão ambiental derrubar os dois com a mesma Vitalidade, o último
// vencedor de um confronto leva a partida. É um desempate baseado em jogo,
// não em assento/iniciativa; sem confronto anterior, vale o desempate
// histórico do motor.
func (g *Game) checkConfrontPressureWin() {
	s := g.s
	if s.Players[0].Vitality <= 0 && s.Players[1].Vitality <= 0 && s.Players[0].Vitality == s.Players[1].Vitality {
		for i := len(g.Log) - 1; i >= 0; i-- {
			event := g.Log[i]
			if event.Kind != EvConfrontationResolved {
				continue
			}
			winner := event.P
			if event.S == "guard" {
				winner = 1 - event.P
			}
			g.endMatch(winner, "pressao_de_nythara")
			return
		}
	}
	g.checkWin()
}

func (g *Game) finishConfrontTurn(player int) {
	if g.s.Over {
		return
	}
	p := g.s.Players[player]
	if p.VeilRound > 0 && p.VeilRound <= g.s.Round {
		p.VeilRound = 0
		g.emit(Event{Kind: EvStatusExpired, P: player, S: "Véu"})
	}
	g.startConfrontTurn(1-player, false)
}

func (g *Game) resolveConfrontDawn(player int) {
	p := g.s.Players[player]
	keepBleeds := p.Bleeds[:0]
	for _, bleed := range p.Bleeds {
		if bleed.Round <= g.s.Round {
			g.emit(Event{Kind: EvBleedTriggered, P: player, N: bleed.N, S: "Sangramento"})
			g.confrontLoseVitality(player, bleed.N, "Sangramento")
		} else {
			keepBleeds = append(keepBleeds, bleed)
		}
	}
	p.Bleeds = keepBleeds
	keepCurses := p.Curses[:0]
	for _, curse := range p.Curses {
		if curse.Round <= g.s.Round {
			g.confrontLoseVitality(player, curse.N, "Maldição")
		} else {
			keepCurses = append(keepCurses, curse)
		}
	}
	p.Curses = keepCurses
}

func (g *Game) drawConfront(player int) *CardInstance {
	s, p := g.s, g.s.Players[player]
	if len(p.Deck) == 0 {
		p.Fatigue += max(1, 2-g.championFatigueRelief(player))
		g.emit(Event{Kind: EvFatigue, P: player, N: p.Fatigue})
		g.confrontLoseVitality(player, p.Fatigue, "Fadiga")
		return nil
	}
	id := p.Deck[0]
	p.Deck = p.Deck[1:]
	inst := s.Cards[id]
	p.DrawsRound++
	if len(p.Hand) >= HandLimit {
		inst.Zone = ZoneDiscard
		p.Discard = append(p.Discard, id)
		g.emit(Event{Kind: EvCardBurned, P: player, Card: id, Def: inst.Def, N: HandLimit})
		g.emit(Event{Kind: EvCardDiscarded, P: player, Card: id, Def: inst.Def})
		return inst
	}
	inst.Zone = ZoneHand
	p.Hand = append(p.Hand, id)
	g.emit(Event{Kind: EvCardDrawn, P: player, Card: id, Def: inst.Def})
	return inst
}

func (g *Game) confrontDealDamage(player, amount int, source string) int {
	if amount <= 0 {
		return 0
	}
	p := g.s.Players[player]
	if p.Exposto && !g.rs.ConfrontRules.TacticalSeals {
		amount += 2
		p.Exposto = false
		g.emit(Event{Kind: EvStatusExpired, P: player, S: "Exposto", Def: source})
	}
	if p.Ward > 0 {
		blocked := min(p.Ward, amount)
		p.Ward -= blocked
		amount -= blocked
		g.emit(Event{Kind: EvWardConsumed, P: player, N: blocked, To: p.Ward, S: source})
		g.emit(Event{Kind: EvPrevented, P: player, N: blocked, S: source})
	}
	if amount <= 0 {
		return 0
	}
	from := p.Vitality
	p.Vitality -= amount
	p.DamageTakenRound += amount
	g.emit(Event{Kind: EvDamage, P: player, N: amount, From: from, To: p.Vitality, S: source})
	// Poderes de resiliência: reagem ao golpe, mas só depois que a Vitalidade
	// desce até o limiar do Avatar. É o que os torna estáveis entre baralhos
	// agressivos e defensivos — o gatilho depende do placar, não do estilo.
	g.championOnTrigger(player, "low_vitality")
	return amount
}

func (g *Game) confrontLoseVitality(player, amount int, source string) {
	if amount <= 0 {
		return
	}
	p := g.s.Players[player]
	from := p.Vitality
	p.Vitality -= amount
	g.emit(Event{Kind: EvDamage, P: player, N: amount, From: from, To: p.Vitality, S: source})
}

func (g *Game) confrontRunOps(ops []Op, ctx *opCtx) {
	for i := range ops {
		op := &ops[i]
		target := ctx.player
		if op.Who == "opponent" {
			target = 1 - ctx.player
		}
		switch op.Op {
		case "damage":
			g.confrontDealDamage(target, op.N, ctx.source)
		case "lose_vitality":
			g.confrontLoseVitality(target, op.N, ctx.source)
		case "heal":
			p := g.s.Players[target]
			from := p.Vitality
			p.Vitality = min(p.MaxVitality, p.Vitality+op.N)
			if p.Vitality > from {
				g.emit(Event{Kind: EvHealed, P: target, N: p.Vitality - from, From: from, To: p.Vitality, S: ctx.source})
			}
		case "choose_discard":
			g.confrontRequestDiscard(target, op.N, op.Then, ctx.source)
		case "draw":
			for range op.N {
				g.drawConfront(target)
			}
		case "both_draw":
			g.drawConfront(ctx.player)
			g.drawConfront(1 - ctx.player)
		case "ward":
			p := g.s.Players[target]
			p.Ward += op.N
			g.emit(Event{Kind: EvWardGained, P: target, N: op.N, To: p.Ward, S: ctx.source})
		case "status":
			g.applyConfrontStatus(op, target, ctx)
		case "remove_own_bleeds":
			g.s.Players[target].Bleeds = nil
			g.emit(Event{Kind: EvStatusExpired, P: target, S: "Sangramento", Def: ctx.source})
		case "remove_own_curse":
			p := g.s.Players[target]
			if len(p.Curses) > 0 {
				p.Curses = p.Curses[1:]
				g.emit(Event{Kind: EvStatusExpired, P: target, S: "Maldição", Def: ctx.source})
			}
		case "remove_curse_smart":
			self, opp := g.s.Players[ctx.player], g.s.Players[1-ctx.player]
			if len(self.Curses) > 0 {
				self.Curses = self.Curses[1:]
			} else if len(opp.Curses) > 0 {
				opp.Curses = opp.Curses[1:]
				g.drawConfront(ctx.player)
			}
		case "remove_veil_expose":
			opp := g.s.Players[1-ctx.player]
			if veilActive(opp, g.s.Round) {
				opp.VeilRound, opp.Exposto = 0, true
				g.emit(Event{Kind: EvStatusApplied, P: 1 - ctx.player, S: "Exposto", Def: ctx.source})
			}
		case "reveal_lock_assault":
			if g.rs.ConfrontRules.TacticalSeals {
				opp := g.s.Players[1-ctx.player]
				opp.AssaultSealUntil = max(opp.AssaultSealUntil, g.s.Round+1)
				g.emit(Event{Kind: EvStatusApplied, P: 1 - ctx.player, N: opp.AssaultSealUntil,
					S: ConfrontAssaultSeal, Def: ctx.source})
			}
		case "conditional":
			if g.evalCond(*op.If, ctx) {
				g.confrontRunOps(op.Then, ctx)
			} else {
				g.confrontRunOps(op.Else, ctx)
			}
		}
	}
}

func (g *Game) applyConfrontStatus(op *Op, target int, ctx *opCtx) {
	if target != ctx.player && veilActive(g.s.Players[target], g.s.Round) {
		g.emit(Event{Kind: EvStatusFizzled, P: target, S: op.Status, Def: ctx.source})
		return
	}
	p := g.s.Players[target]
	switch op.Status {
	case "exposto":
		p.Exposto = true
	case "veu":
		p.VeilRound = g.s.Round + 1
	case "sangramento":
		p.Bleeds = append(p.Bleeds, TimedN{N: op.N, Round: g.s.Round + 1})
	case "maldicao":
		p.Curses = append(p.Curses, TimedN{N: max(1, op.N), Round: g.s.Round + 1, Kind: op.Kind})
	}
	g.emit(Event{Kind: EvStatusApplied, P: target, N: op.N, S: op.Status, Def: ctx.source})
}
