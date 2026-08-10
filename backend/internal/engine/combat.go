package engine

// applyPlay valida e executa a jogada de uma carta, roteando pelo tipo e pela
// janela atual. Toda validação acontece antes de qualquer mutação.
func (g *Game) applyPlay(cmd Command) error {
	s := g.s
	inst := s.Cards[cmd.Card]
	if inst == nil || inst.Owner != cmd.Player || inst.Zone != ZoneHand {
		return errCmd(ErrInvalidCard, "carta %q não está na sua mão", cmd.Card)
	}
	def := g.rs.Cards[inst.Def]

	// Janela de reação a Rito (VR-035): só o defensor, só Guarda de counter.
	if s.RiteReact != nil {
		if cmd.Player != 1-s.RiteReact.Caster {
			return errCmd(ErrWrongPlayer, "a janela de reação é do jogador %d", 1-s.RiteReact.Caster)
		}
		if def.Type != TypeGuarda {
			return errCmd(ErrWrongPhase, "apenas Guarda pode reagir a um Rito")
		}
		return g.playCounterGuard(cmd.Player, inst, def)
	}

	// Janela de Guarda aberta: só o defensor pode agir, e só com Guarda.
	if s.Guard != nil {
		if cmd.Player != s.Guard.Defender {
			return errCmd(ErrWrongPlayer, "a janela de Guarda é do jogador %d", s.Guard.Defender)
		}
		if def.Type != TypeGuarda {
			return errCmd(ErrWrongPhase, "apenas Guarda pode ser jogada na janela de Guarda")
		}
		return g.playGuard(cmd.Player, inst, def)
	}

	switch s.Phase {
	case PhaseRite:
		if cmd.Player != s.Active {
			return errCmd(ErrWrongPlayer, "a janela é do jogador %d", s.Active)
		}
		switch def.Type {
		case TypeRito:
			return g.playRite(cmd.Player, inst, def)
		case TypeReliquia, TypeManifestacao:
			return g.playPermanent(cmd.Player, inst, def)
		default:
			return errCmd(ErrWrongPhase, "%s não pode ser jogada na fase de Rito", def.Type)
		}
	case PhaseConfront:
		if cmd.Player != s.Active {
			return errCmd(ErrWrongPlayer, "a janela é do jogador %d", s.Active)
		}
		if def.Type != TypeAssalto {
			return errCmd(ErrWrongPhase, "apenas Assalto pode ser jogado no Confronto")
		}
		return g.playAssault(cmd.Player, inst, def)
	default:
		return errCmd(ErrWrongPhase, "não é possível jogar cartas na fase %s", s.Phase)
	}
}

// payAndAnnounce paga o custo efetivo, tira a carta da mão e anuncia a jogada.
func (g *Game) payAndAnnounce(player int, inst *CardInstance, def *CardDef) int {
	cost := g.effectiveCost(player, def, inst.ID)
	g.consumeCostMods(player, def, inst.ID)
	p := g.s.Players[player]
	p.Hand, _ = removeID(p.Hand, inst.ID)
	inst.Zone = ZonePlay
	p.PrevPlayed = p.LastPlayed
	p.LastPlayed = def.ID
	g.emit(Event{Kind: EvCardPlayed, P: player, Card: inst.ID, Def: def.ID, N: cost})
	g.spendEssence(player, cost)
	return cost
}

// --- Assalto ---

func (g *Game) playAssault(player int, inst *CardInstance, def *CardDef) error {
	impl := g.rs.assault[def.ID]
	if impl == nil {
		return errCmd(ErrNotImplemented, "%s (%s) ainda não implementada no alpha", def.ID, def.Name)
	}
	if inst.LockedRound == g.s.Round {
		return errCmd(ErrIllegalTarget, "%s está travada nesta rodada", def.ID)
	}
	if g.s.Extra != nil && g.s.Extra.MaxCost > 0 && def.Cost > g.s.Extra.MaxCost {
		return errCmd(ErrWrongPhase, "a janela extra aceita apenas custo %d ou menos", g.s.Extra.MaxCost)
	}
	if impl.preExile && len(g.s.Players[player].Discard) == 0 {
		return errCmd(ErrRequirement, "%s exige uma carta no descarte para exilar", def.ID)
	}
	if !g.canAfford(player, g.effectiveCost(player, def, inst.ID)) {
		return errCmd(ErrCantAfford, "essência insuficiente para %s", def.ID)
	}

	g.payAndAnnounce(player, inst, def)
	g.startAssault(player, inst.ID, def, impl)
	return nil
}

// startAssault monta o contexto de combate (também usado por cópias, com
// instância virtual) e abre a janela de Guarda ou resolve direto.
func (g *Game) startAssault(player int, instID string, def *CardDef, impl *assaultImpl) {
	s := g.s
	p := s.Players[player]
	p.AssaultsRound++

	ctx := &GuardCtx{
		Attacker:    player,
		Defender:    1 - player,
		AssaultInst: instID,
		AssaultDef:  def.ID,
		Instances:   max(impl.instances, 1),
		IgnoreWard:  impl.ignoreWard,
	}
	// Forma de Kaedor: o primeiro Assalto na Noite atravessa Ward.
	if champFirstAssaultPierces(g, player) {
		ctx.IgnoreWard = 99
	}
	ctx.BaseDamage = impl.base(g, ctx)

	bonus := 0
	if p.Stance == StancePredacao && p.AssaultsRound == 1 {
		bonus++
	}
	bonus += champAssaultBonus(g, player)
	if ctx.Instances == 1 {
		ctx.BaseDamage += bonus
	} else {
		ctx.FirstHitBonus = bonus
	}

	s.Guard = ctx
	if impl.preExile {
		// VR-055: escolha de exílio antes da janela de Guarda.
		g.requestDecision(&Decision{
			Player: player, Kind: DecExilePick,
			Options: append([]string{}, p.Discard...), N: 1,
			Source: def.ID, Card: instID,
		})
		return
	}
	g.announceAssault()
}

// announceAssault abre a janela de Guarda (ou resolve direto se o Assalto for
// indefensável no estado atual).
func (g *Game) announceAssault() {
	s := g.s
	ctx := s.Guard
	ctx.Announced = true
	impl := g.rs.assault[ctx.AssaultDef]
	if impl.undefendable != nil && impl.undefendable(g, ctx) {
		g.resolveAssault()
		return
	}
	g.emit(Event{Kind: EvWindowOpened, P: ctx.Defender, S: "guarda", Card: ctx.AssaultInst,
		Def: ctx.AssaultDef, N: ctx.BaseDamage*ctx.Instances + ctx.FirstHitBonus})
}

func (g *Game) playGuard(player int, inst *CardInstance, def *CardDef) error {
	impl := g.rs.guard[def.ID]
	if impl == nil {
		return errCmd(ErrNotImplemented, "%s (%s) ainda não implementada no alpha", def.ID, def.Name)
	}
	if impl.counterRite {
		return errCmd(ErrWrongPhase, "%s só reage a Ritos direcionados", def.ID)
	}
	if g.s.Guard.GuardInst != "" {
		return errCmd(ErrBadCommand, "apenas uma Guarda por janela")
	}
	if !g.canAfford(player, g.effectiveCost(player, def, inst.ID)) {
		return errCmd(ErrCantAfford, "essência insuficiente para %s", def.ID)
	}

	ctx := g.s.Guard
	g.payAndAnnounce(player, inst, def)
	p := g.s.Players[player]
	p.GuardsRound++
	ctx.GuardInst = inst.ID

	if impl.onPlay != nil {
		impl.onPlay(g, ctx)
	}
	prev := impl.prevent(g, ctx)
	prev += champGuardPreventBonus(g, player)
	ctx.Prevention = prev

	// A Guarda resolve como reação: emite Sigilo e desloca o Eclipse antes
	// de o dano do Assalto ser aplicado.
	g.emitCardSigil(player, def, inst.ID)
	g.shiftEclipse(def.EclipseShift, def.ID, player)
	champOnGuardResolved(g, player)
	g.firePerms(player, func(pi *permImpl, pinst *CardInstance) {
		if pi.onGuardPlayed != nil {
			pi.onGuardPlayed(g, pinst)
		}
	})

	g.resolveAssault()
	return nil
}

// resolveAssault aplica o dano (com Exposto, prevenção e Ward), efeitos
// posteriores e fecha a janela de Guarda.
func (g *Game) resolveAssault() {
	s := g.s
	ctx := s.Guard
	impl := g.rs.assault[ctx.AssaultDef]
	att := s.Players[ctx.Attacker]
	defP := s.Players[ctx.Defender]

	for i := 0; i < ctx.Instances && !s.Over; i++ {
		amt := ctx.BaseDamage
		if i == 0 {
			amt += ctx.FirstHitBonus
		}
		if defP.Exposto && amt > 0 {
			amt += 2
			defP.Exposto = false
			g.emit(Event{Kind: EvStatusExpired, P: ctx.Defender, S: "Exposto"})
		}
		if ctx.Prevention > 0 && amt > 0 {
			use := min(ctx.Prevention, amt)
			amt -= use
			ctx.Prevention -= use
			g.emit(Event{Kind: EvPrevented, P: ctx.Defender, N: use, Card: ctx.GuardInst})
		}
		if defP.Ward > 0 && amt > 0 {
			effWard := max(defP.Ward-ctx.IgnoreWard, 0)
			use := min(effWard, amt)
			if use > 0 {
				defP.Ward -= use
				amt -= use
				g.emit(Event{Kind: EvWardConsumed, P: ctx.Defender, N: use, To: defP.Ward})
			}
		}
		if amt > 0 {
			applied := g.applyVitalityLoss(ctx.Defender, amt, "", ctx.AssaultInst, ctx.AssaultDef)
			ctx.NetTotal += applied
			if applied > 0 && !s.Over {
				net := applied
				g.firePerms(ctx.Attacker, func(pi *permImpl, pinst *CardInstance) {
					if pi.onDamageDealt != nil {
						pi.onDamageDealt(g, pinst, net)
					}
				})
			}
		}
	}
	ctx.PreventedAll = ctx.NetTotal == 0
	if ctx.NetTotal > 0 {
		att.LastDamageDealt = ctx.NetTotal
	}

	// Efeito posterior da Guarda (sabe se preveniu tudo).
	var gImpl *guardImpl
	if ctx.GuardInst != "" {
		gImpl = g.rs.guard[s.Cards[ctx.GuardInst].Def]
	}
	if gImpl != nil && gImpl.after != nil && !s.Over {
		gImpl.after(g, ctx)
	}
	// Forma de Mara: prevenir todo o dano na Aurora Total compra 1.
	if ctx.PreventedAll && !s.Over {
		champAfterFullPrevent(g, ctx.Defender)
	}
	// Destino da Guarda: descarte, ou mão (VR-056), com sobretaxa opcional.
	if ctx.GuardInst != "" {
		gInst := s.Cards[ctx.GuardInst]
		defPl := s.Players[gInst.Owner]
		if gImpl != nil && gImpl.toHandAfter {
			gInst.Zone = ZoneHand
			defPl.Hand = append(defPl.Hand, gInst.ID)
			g.emit(Event{Kind: EvCardToHand, P: gInst.Owner, Card: gInst.ID, Def: gInst.Def})
			if gImpl.selfCostAfter != 0 {
				defPl.CostMods = append(defPl.CostMods, CostMod{
					Instance: gInst.ID, Delta: gImpl.selfCostAfter, Uses: 1, Source: gInst.Def,
				})
				g.emit(Event{Kind: EvCostModAdded, P: gInst.Owner, N: gImpl.selfCostAfter, S: gInst.Def})
			}
		} else {
			gInst.Zone = ZoneDiscard
			defPl.Discard = append(defPl.Discard, gInst.ID)
		}
	}

	if !s.Over {
		// Sigilo e deslocamento intrínseco do Assalto, efeitos posteriores
		// e gatilhos de permanentes, nessa ordem.
		aDef := g.rs.Cards[ctx.AssaultDef]
		g.emitCardSigil(ctx.Attacker, aDef, ctx.AssaultInst)
		g.shiftEclipse(aDef.EclipseShift, aDef.ID, ctx.Attacker)
		if impl.after != nil {
			impl.after(g, ctx)
		}
		g.fireAssaultResolved(ctx)
		// VR-045: força emprestada cobra no corpo.
		if att.LuaNoSangue && !s.Over {
			g.loseVitality(ctx.Attacker, 1, "VR-045")
		}
	}

	// Assalto vai ao descarte (cópias não têm instância física).
	if aInst := s.Cards[ctx.AssaultInst]; aInst != nil {
		aInst.Zone = ZoneDiscard
		att.Discard = append(att.Discard, aInst.ID)
		s.PlayedRound = append(s.PlayedRound, PlayedCard{Def: ctx.AssaultDef, P: ctx.Attacker})
	}

	s.Guard = nil
	if s.Over {
		return
	}
	if ctx.EndWindow {
		// VR-078: encerra a janela de Assalto do atacante.
		g.emit(Event{Kind: EvPass, P: ctx.Attacker, S: "janela_encerrada"})
		if s.Extra != nil && s.Extra.Player == ctx.Attacker {
			s.Extra = nil
			g.openExtraOrTwilight()
			return
		}
		s.Passed[ctx.Attacker] = true
		other := 1 - ctx.Attacker
		if s.Passed[other] {
			g.openExtraOrTwilight()
		} else {
			s.Active = other
			g.emit(Event{Kind: EvWindowOpened, P: other, S: string(PhaseConfront)})
		}
		return
	}
	s.Active = ctx.Attacker
}

// --- Rito ---

func (g *Game) playRite(player int, inst *CardInstance, def *CardDef) error {
	impl := g.rs.rite[def.ID]
	if impl == nil {
		return errCmd(ErrNotImplemented, "%s (%s) ainda não implementada no alpha", def.ID, def.Name)
	}
	if !g.canAfford(player, g.effectiveCost(player, def, inst.ID)) {
		return errCmd(ErrCantAfford, "essência insuficiente para %s", def.ID)
	}
	p := g.s.Players[player]
	effSac := impl.sacrifice
	if effSac > 0 {
		effSac = max(effSac+g.permSacrificeDelta(player, false), 0)
	}
	if effSac > 0 && p.Vitality <= effSac {
		return errCmd(ErrCantAfford, "vitalidade insuficiente para sacrificar %d", effSac)
	}
	if impl.targetsOpponent && g.s.Players[1-player].VeilRound == g.s.Round {
		return errCmd(ErrIllegalTarget, "o oponente está sob Véu")
	}
	if impl.validate != nil {
		if err := impl.validate(g, player); err != nil {
			return err
		}
	}

	cost := g.payAndAnnounce(player, inst, def)

	// Ritos direcionados podem ser respondidos (VR-035); a janela só abre se
	// o defensor tiver um counter jogável (ADR-014).
	if (impl.targetsOpponent || impl.revealsOppHand) && g.defenderHoldsCounter(1-player) {
		g.s.RiteReact = &RiteReact{Caster: player, Inst: inst.ID, Def: def.ID, Cost: cost}
		g.emit(Event{Kind: EvRiteReactWindow, P: 1 - player, Card: inst.ID, Def: def.ID})
		return nil
	}
	g.resolveRiteBody(player, inst, def, impl)
	return nil
}

// resolveRiteBody resolve um Rito já pago (fluxo direto ou retomado após a
// janela de reação).
func (g *Game) resolveRiteBody(player int, inst *CardInstance, def *CardDef, impl *riteImpl) {
	stackBase := len(g.s.PendingRites)
	if impl.sacrifice > 0 {
		effSac := max(impl.sacrifice+g.permSacrificeDelta(player, true), 0)
		if effSac > 0 {
			g.sacrificeVitality(player, effSac, def.ID)
		}
	}
	if !g.s.Over {
		// VR-010: Ritos que olhariam/revelariam a mão protegida falham.
		if impl.revealsOppHand && g.oppBlocksReveal(player) {
			g.emit(Event{Kind: EvRiteFizzled, P: player, Card: inst.ID, Def: def.ID, S: "VR-010"})
		} else {
			impl.resolve(g, player, inst.ID)
		}
	}
	if g.s.Pending != nil || len(g.s.DecQueue) > 0 {
		// Insere antes de continuações aninhadas abertas durante impl.resolve:
		// a pilha é LIFO, logo a cópia interna finaliza antes do Rito externo.
		pending := RiteFinalize{Player: player, Inst: inst.ID, Def: def.ID}
		g.s.PendingRites = append(g.s.PendingRites, RiteFinalize{})
		copy(g.s.PendingRites[stackBase+1:], g.s.PendingRites[stackBase:])
		g.s.PendingRites[stackBase] = pending
		return
	}
	g.finalizeRite(player, inst, def, impl)
}

// finalizeRite executa a continuação intrínseca que só pode ocorrer depois do
// texto do Rito e de todas as escolhas abertas por ele.
func (g *Game) finalizeRite(player int, inst *CardInstance, def *CardDef, impl *riteImpl) {
	p := g.s.Players[player]
	if !g.s.Over {
		g.emitCardSigil(player, def, inst.ID)
		if !impl.ownShift {
			g.shiftEclipse(def.EclipseShift, def.ID, player)
		}
	}

	inst.Zone = ZoneDiscard
	p.Discard = append(p.Discard, inst.ID)
	g.emit(Event{Kind: EvCardDiscarded, P: player, Card: inst.ID, Def: def.ID})
	g.s.PlayedRound = append(g.s.PlayedRound, PlayedCard{Def: def.ID, P: player})
	g.s.LastRite[player] = def.ID
}

// finalizePendingRite retoma um Rito suspenso quando nenhuma decisão anterior
// resta. A finalização pode abrir novas decisões por gatilhos de Sigilo.
func (g *Game) finalizePendingRite() {
	for len(g.s.PendingRites) > 0 && g.s.Pending == nil && len(g.s.DecQueue) == 0 {
		index := len(g.s.PendingRites) - 1
		pending := g.s.PendingRites[index]
		g.s.PendingRites = g.s.PendingRites[:index]
		def := g.rs.Cards[pending.Def]
		impl := g.rs.rite[pending.Def]
		if def == nil || impl == nil {
			panic("engine: continuação de Rito inválida: " + pending.Def)
		}
		if pending.CopyDepth > 0 {
			g.finalizeRiteCopy(pending.Player, def, impl, pending.CopyDepth)
			continue
		}
		inst := g.s.Cards[pending.Inst]
		if inst == nil {
			panic("engine: instância da continuação de Rito ausente: " + pending.Inst)
		}
		g.finalizeRite(pending.Player, inst, def, impl)
	}
}

// oppBlocksReveal responde se o oponente tem uma Relíquia anti-revelação
// ativa (VR-010) e dispara o gatilho dela.
func (g *Game) oppBlocksReveal(caster int) bool {
	blocked := false
	g.firePerms(1-caster, func(pi *permImpl, inst *CardInstance) {
		if pi.blocksHandReveal && !blocked {
			blocked = true
			if pi.onRevealBlocked != nil {
				pi.onRevealBlocked(g, inst)
			}
		}
	})
	return blocked
}

// --- Permanentes ---

func (g *Game) playPermanent(player int, inst *CardInstance, def *CardDef) error {
	impl := g.rs.perm[def.ID]
	if impl == nil {
		return errCmd(ErrNotImplemented, "%s (%s) ainda não implementada no alpha", def.ID, def.Name)
	}
	if !g.canAfford(player, g.effectiveCost(player, def, inst.ID)) {
		return errCmd(ErrCantAfford, "essência insuficiente para %s", def.ID)
	}
	p := g.s.Players[player]
	if def.Type == TypeReliquia && len(p.Relics) >= 2 {
		return errCmd(ErrZoneFull, "máximo de 2 Relíquias ativas (substituição chega em iteração futura)")
	}
	if def.Type == TypeManifestacao && len(p.Manifs) >= 2 {
		return errCmd(ErrZoneFull, "máximo de 2 Manifestações ativas (substituição chega em iteração futura)")
	}

	g.payAndAnnounce(player, inst, def)
	if def.Type == TypeReliquia {
		p.Relics = append(p.Relics, inst.ID)
	} else {
		p.Manifs = append(p.Manifs, inst.ID)
	}
	g.emit(Event{Kind: EvPermanentEntered, P: player, Card: inst.ID, Def: def.ID, S: string(def.Type)})
	g.emitCardSigil(player, def, inst.ID)
	g.shiftEclipse(def.EclipseShift, def.ID, player)
	if impl.onEnter != nil && !g.s.Over && !g.permSuppressed(inst) {
		impl.onEnter(g, inst)
	}
	return nil
}

// --- Gatilhos de permanentes ---

// permSuppressed responde se um permanente está com texto passivo suprimido
// (VR-021 silencia Manifestações adversárias).
func (g *Game) permSuppressed(inst *CardInstance) bool {
	if g.rs.Cards[inst.Def].Type != TypeManifestacao {
		return false
	}
	return g.s.Players[inst.Owner].ManifsSuppressedUntil >= g.s.Round
}

// firePerms percorre Relíquias e Manifestações ativas do jogador na ordem de
// entrada (mais a Relíquia espelhada por VR-032), pulando as suprimidas.
func (g *Game) firePerms(owner int, fn func(pi *permImpl, inst *CardInstance)) {
	p := g.s.Players[owner]
	for _, id := range append(append([]string{}, p.Relics...), p.Manifs...) {
		if g.s.Over {
			return
		}
		inst := g.s.Cards[id]
		pi := g.rs.perm[inst.Def]
		if pi == nil || g.permSuppressed(inst) {
			continue
		}
		fn(pi, inst)
	}
	// VR-032: cópia temporária do texto de uma Relíquia adversária.
	if p.MirroredRelic != "" && p.MirrorUntil >= g.s.Round && !g.s.Over {
		if pi := g.rs.perm[p.MirroredRelic]; pi != nil {
			virtual := &CardInstance{
				ID: "mirror-" + p.MirroredRelic, Def: p.MirroredRelic,
				Owner: owner, Zone: ZonePlay, UsedRound: p.MirrorUsedRound,
			}
			fn(pi, virtual)
			p.MirrorUsedRound = virtual.UsedRound
		}
	}
}

// permSacrificeDelta soma descontos de sacrifício (VR-007). Com consume=true,
// marca o uso da rodada.
func (g *Game) permSacrificeDelta(player int, consume bool) int {
	delta := 0
	g.firePerms(player, func(pi *permImpl, inst *CardInstance) {
		if pi.sacrificeDelta != 0 && inst.UsedRound != g.s.Round {
			delta += pi.sacrificeDelta
			if consume {
				inst.UsedRound = g.s.Round
			}
		}
	})
	return delta
}

// emitCardSigil emite o Sigilo impresso da carta, o extra do Juramento do
// Errante (VR-077) e os ecos de Campeão (forma de Saela).
func (g *Game) emitCardSigil(player int, def *CardDef, instID string) {
	g.emitSigil(player, def.Sigil, instID, 0)
	if def.Type == TypeAssalto {
		champOnOwnSigil(g, player, def.Sigil, true, 0)
	}
	p := g.s.Players[player]
	if p.ExtraNeutralSigil != "" && def.Faction == NeutralFaction {
		g.emitSigil(player, p.ExtraNeutralSigil, "VR-077", 1)
	}
}

func (g *Game) fireRoundStart(instID string) {
	inst := g.s.Cards[instID]
	pi := g.rs.perm[inst.Def]
	if pi == nil || pi.onRoundStart == nil || g.permSuppressed(inst) {
		return
	}
	pi.onRoundStart(g, inst)
}

func (g *Game) fireSigilEmitted(player int, sig Sigil, depth int) {
	g.firePerms(player, func(pi *permImpl, inst *CardInstance) {
		if pi.onSigilEmitted != nil {
			pi.onSigilEmitted(g, inst, sig, depth)
		}
	})
}

func (g *Game) fireAssaultResolved(ctx *GuardCtx) {
	g.firePerms(ctx.Attacker, func(pi *permImpl, inst *CardInstance) {
		if pi.onAssaultResolved != nil {
			pi.onAssaultResolved(g, inst, ctx)
		}
	})
}

func (g *Game) gainWard(player, n int, source string) {
	if n <= 0 {
		return
	}
	p := g.s.Players[player]
	p.Ward += n
	g.emit(Event{Kind: EvWardGained, P: player, N: n, To: p.Ward, S: source})
}
