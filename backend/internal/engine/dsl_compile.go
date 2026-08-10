package engine

// compileEffects transforma as definições da DSL nos registros executáveis
// (assaultImpls/guardImpls/riteImpls/permImpls). Roda no boot, após o
// validador. As cartas viram closures pequenas sobre o intérprete — nenhum
// switch por carta no código de jogo.
func compileEffects(fx *EffectsFile, rs *Ruleset) {
	for id, cfx := range fx.Cards {
		switch {
		case cfx.Assault != nil:
			rs.assault[id] = compileAssault(id, cfx.Assault)
		case cfx.Guard != nil:
			rs.guard[id] = compileGuard(id, cfx.Guard)
		case cfx.Rite != nil:
			rs.rite[id] = compileRite(id, cfx.Rite)
		case cfx.Permanent != nil:
			rs.perm[id] = compilePerm(id, cfx.Permanent)
		}
	}
}

func applyBonuses(g *Game, base int, bonuses []Bonus, ctx *opCtx) int {
	v := base
	for _, b := range bonuses {
		if g.evalCond(b.If, ctx) {
			if b.Instead != 0 {
				v = b.Instead
			} else {
				v += b.Add
			}
			if b.Resonance && ctx.guard != nil {
				ctx.guard.ResonanceBonus = true
			}
		}
	}
	return v
}

func compileAssault(id string, a *AssaultFx) *assaultImpl {
	impl := &assaultImpl{
		instances:  a.Instances,
		ignoreWard: a.IgnoreWard,
		preExile:   a.PreExile,
	}
	impl.base = func(g *Game, ctx *GuardCtx) int {
		oc := &opCtx{player: ctx.Attacker, source: id, inst: ctx.AssaultInst, guard: ctx}
		return applyBonuses(g, a.Damage, a.Bonuses, oc)
	}
	if a.UndefendableIf != nil {
		cond := *a.UndefendableIf
		impl.undefendable = func(g *Game, ctx *GuardCtx) bool {
			return g.evalCond(cond, &opCtx{player: ctx.Attacker, source: id, guard: ctx})
		}
	}
	if len(a.After) > 0 {
		after := a.After
		impl.after = func(g *Game, ctx *GuardCtx) {
			g.runOps(after, &opCtx{player: ctx.Attacker, source: id, inst: ctx.AssaultInst, guard: ctx})
		}
	}
	return impl
}

func compileGuard(id string, gf *GuardFx) *guardImpl {
	impl := &guardImpl{
		toHandAfter:   gf.ToHandAfter,
		selfCostAfter: gf.SelfCostAfter,
		counterRite:   gf.CounterRite,
	}
	impl.prevent = func(g *Game, ctx *GuardCtx) int {
		if gf.PreventAll {
			return 9999
		}
		oc := &opCtx{player: ctx.Defender, source: id, inst: ctx.GuardInst, guard: ctx}
		return applyBonuses(g, gf.Prevent, gf.Bonuses, oc)
	}
	if len(gf.OnPlay) > 0 {
		ops := gf.OnPlay
		impl.onPlay = func(g *Game, ctx *GuardCtx) {
			g.runOps(ops, &opCtx{player: ctx.Defender, source: id, inst: ctx.GuardInst, guard: ctx})
		}
	}
	if len(gf.After) > 0 {
		ops := gf.After
		impl.after = func(g *Game, ctx *GuardCtx) {
			g.runOps(ops, &opCtx{player: ctx.Defender, source: id, inst: ctx.GuardInst, guard: ctx})
		}
	}
	return impl
}

func compileRite(id string, r *RiteFx) *riteImpl {
	impl := &riteImpl{
		sacrifice:       r.Sacrifice,
		targetsOpponent: r.TargetsOpponent,
		revealsOppHand:  r.RevealsOppHand,
		dealsDamage:     r.DealsDamage,
		ownShift:        r.OwnShift,
	}
	if len(r.Requires) > 0 {
		reqs := r.Requires
		impl.validate = func(g *Game, player int) error {
			for _, c := range reqs {
				if !g.evalCond(c, &opCtx{player: player, source: id}) {
					return errCmd(ErrRequirement, "%s: requisito %q não satisfeito", id, c.Cond)
				}
			}
			return nil
		}
	}
	steps := r.Steps
	impl.resolve = func(g *Game, player int, inst string) {
		g.runOps(steps, &opCtx{player: player, source: id, inst: inst})
	}
	return impl
}

func compilePerm(id string, p *PermFx) *permImpl {
	impl := &permImpl{
		sacrificeDelta:   p.SacrificeDelta,
		blocksHandReveal: p.BlocksHandReveal,
	}
	if len(p.OnEnter) > 0 {
		ops := p.OnEnter
		impl.onEnter = func(g *Game, inst *CardInstance) {
			g.runOps(ops, &opCtx{player: inst.Owner, source: id, inst: inst.ID})
		}
	}
	if a := p.Activated; a != nil {
		impl.activated = &activatedImpl{
			oncePerRound: a.OncePerRound,
			discardCost:  a.DiscardCost,
			exileCost:    a.ExileCost,
			do:           a.Do,
		}
	}
	for i := range p.Triggers {
		tr := p.Triggers[i]
		run := func(g *Game, inst *CardInstance, extra *opCtx) {
			if tr.OncePerRound && inst.UsedRound == g.s.Round {
				return
			}
			oc := &opCtx{player: inst.Owner, source: id, inst: inst.ID}
			if extra != nil {
				oc.curse = extra.curse
				oc.depth = extra.depth
				oc.guard = extra.guard
			}
			if tr.If != nil && !g.evalCond(*tr.If, oc) {
				return
			}
			if tr.OncePerRound {
				inst.UsedRound = g.s.Round
			}
			g.runOps(tr.Do, oc)
		}
		switch tr.On {
		case "round_start":
			impl.onRoundStart = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		case "sigil_emitted":
			sig := Sigil(tr.Sigil)
			impl.onSigilEmitted = func(g *Game, inst *CardInstance, s Sigil, depth int) {
				if sig != "" && s != sig {
					return
				}
				run(g, inst, &opCtx{depth: depth})
			}
		case "assault_resolved":
			costEq := tr.NEquals
			impl.onAssaultResolved = func(g *Game, inst *CardInstance, ctx *GuardCtx) {
				if costEq > 0 && g.rs.Cards[ctx.AssaultDef].Cost != costEq {
					return
				}
				run(g, inst, &opCtx{guard: ctx})
			}
		case "reveal_blocked":
			impl.onRevealBlocked = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		case "twilight":
			impl.onTwilight = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		case "heal":
			impl.onHeal = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		case "curse_applied":
			impl.onCurseApplied = func(g *Game, inst *CardInstance, curse *TimedN) {
				run(g, inst, &opCtx{curse: curse})
			}
		case "exile":
			impl.onExile = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		case "opponent_draw":
			nEq := tr.NEquals
			impl.onOppDraw = func(g *Game, inst *CardInstance, drawer int) {
				if nEq > 0 && g.s.Players[drawer].DrawsRound != nEq {
					return
				}
				run(g, inst, nil)
			}
		case "damage_dealt":
			nEq := tr.NEquals
			impl.onDamageDealt = func(g *Game, inst *CardInstance, net int) {
				if nEq > 0 && net != nEq {
					return
				}
				run(g, inst, nil)
			}
		case "guard_played":
			impl.onGuardPlayed = func(g *Game, inst *CardInstance) { run(g, inst, nil) }
		}
	}
	return impl
}
