package engine

// Sistemas avançados: habilidades ativadas, cópias, reação a Ritos (counter),
// janelas extras de Assalto e o funil central de perda de Vitalidade.

// --- Perda de Vitalidade centralizada ---

// applyVitalityLoss é o único caminho de dano/perda direta. Aplica o escudo
// da Saela (primeiro dano da rodada → 0) e a Recusa da Morte de Kaedor
// (uma vez por partida: fica com 1 e descarta a mão). Retorna o valor aplicado.
func (g *Game) applyVitalityLoss(player, n int, cause, card, def string) int {
	if n <= 0 || g.s.Over {
		return 0
	}
	p := g.s.Players[player]
	if p.SaelaShield {
		p.SaelaShield = false
		g.emit(Event{Kind: EvPrevented, P: player, N: n, S: "CH-VA-02"})
		return 0
	}
	from := p.Vitality
	p.Vitality -= n
	g.emit(Event{Kind: EvDamage, P: player, N: n, From: from, To: p.Vitality, S: cause, Card: card, Def: def})
	p.DamageTakenRound += n

	if p.Vitality <= 0 && p.Champion == "CH-VH-02" && !p.UltimateUsed {
		p.UltimateUsed = true
		p.Vitality = 1
		g.emit(Event{Kind: EvUltimate, P: player, S: "CH-VH-02"})
		for len(p.Hand) > 0 {
			g.discardFromHand(player, p.Hand[0])
		}
		return n
	}
	g.checkWin()
	return n
}

// --- Habilidades ativadas de permanentes ---

// CanActivate valida a ativação sem executá-la (fonte única para Apply e bots).
func (g *Game) CanActivate(player int, instID string) error {
	s := g.s
	inst := s.Cards[instID]
	if inst == nil || inst.Owner != player || inst.Zone != ZonePlay {
		return errCmd(ErrInvalidCard, "permanente %q não está ativo", instID)
	}
	pi := permImpls[inst.Def]
	if pi == nil || pi.activated == nil {
		return errCmd(ErrNotImplemented, "%s não tem habilidade ativada", inst.Def)
	}
	if s.Pending != nil || s.Guard != nil || s.RiteReact != nil {
		return errCmd(ErrWrongPhase, "há uma janela/decisão em aberto")
	}
	if s.Phase != PhaseRite || s.Active != player {
		return errCmd(ErrWrongPhase, "ativação só na sua janela de Rito")
	}
	if g.permSuppressed(inst) {
		return errCmd(ErrIllegalTarget, "%s está com o texto suprimido", inst.Def)
	}
	ai := pi.activated
	if ai.oncePerRound && inst.UsedRound == s.Round {
		return errCmd(ErrBadCommand, "%s já foi ativada nesta rodada", inst.Def)
	}
	p := s.Players[player]
	if ai.discardCost > 0 && len(p.Hand) < ai.discardCost {
		return errCmd(ErrCantAfford, "descarte de %d exige cartas na mão", ai.discardCost)
	}
	if ai.exileCost > 0 && len(p.Discard) < ai.exileCost {
		return errCmd(ErrCantAfford, "exílio de %d exige cartas no descarte", ai.exileCost)
	}
	return nil
}

func (g *Game) applyActivate(cmd Command) error {
	if err := g.CanActivate(cmd.Player, cmd.Card); err != nil {
		return err
	}
	s := g.s
	inst := s.Cards[cmd.Card]
	ai := permImpls[inst.Def].activated
	if ai.oncePerRound {
		inst.UsedRound = s.Round
	}
	g.emit(Event{Kind: EvActivated, P: cmd.Player, Card: inst.ID, Def: inst.Def})
	p := s.Players[cmd.Player]
	switch {
	case ai.discardCost > 0:
		g.requestDecision(&Decision{
			Player: cmd.Player, Kind: DecDiscardN,
			Options: append([]string{}, p.Hand...), N: ai.discardCost,
			Source: inst.Def, Card: inst.ID, Then: ai.do,
		})
	case ai.exileCost > 0:
		g.requestDecision(&Decision{
			Player: cmd.Player, Kind: DecExilePick,
			Options: append([]string{}, p.Discard...), N: ai.exileCost,
			Source: inst.Def, Card: inst.ID, Then: ai.do,
		})
	default:
		g.runOps(ai.do, &opCtx{player: cmd.Player, source: inst.Def, inst: inst.ID})
	}
	return nil
}

// --- Cópias ---

// resolveCopy resolve uma cópia virtual de uma carta (Rito ou Assalto), sem
// instância física, sem custos e sem entrar em zonas. Cópias não são
// registradas em PlayedRound (não podem ser recopiadas) e respeitam o limite
// de profundidade.
func (g *Game) resolveCopy(player int, defID string, depth int) {
	if depth > 2 {
		g.emit(Event{Kind: EvChainTruncated, P: player, S: defID})
		return
	}
	def := Cards[defID]
	if def == nil || def.Rarity == RarityLendaria {
		return
	}
	g.emit(Event{Kind: EvCopyResolved, P: player, Def: defID, N: depth})
	virtual := "copy-" + defID

	switch def.Type {
	case TypeRito:
		impl := riteImpls[defID]
		if impl == nil {
			return
		}
		if impl.targetsOpponent && g.s.Players[1-player].VeilRound == g.s.Round {
			g.emit(Event{Kind: EvRiteFizzled, P: player, Def: defID, S: "veu"})
			return
		}
		if impl.validate != nil && impl.validate(g, player) != nil {
			g.emit(Event{Kind: EvRiteFizzled, P: player, Def: defID, S: "requisito"})
			return
		}
		impl.resolve(g, player, virtual)
		if g.s.Pending != nil || len(g.s.DecQueue) > 0 {
			g.s.PendingRites = append(g.s.PendingRites, RiteFinalize{
				Player: player, Def: defID, CopyDepth: depth,
			})
			return
		}
		g.finalizeRiteCopy(player, def, impl, depth)
	case TypeAssalto:
		impl := assaultImpls[defID]
		if impl == nil || g.s.Guard != nil {
			return
		}
		g.startAssault(player, virtual, def, impl)
	}
}

func (g *Game) finalizeRiteCopy(player int, def *CardDef, impl *riteImpl, depth int) {
	if g.s.Over {
		return
	}
	virtual := "copy-" + def.ID
	g.emitSigil(player, def.Sigil, virtual, depth)
	if champCopyExtraSigil(g, player) {
		g.emitSigil(player, def.Sigil, "CH-MI-01", depth+1)
	}
	if !impl.ownShift {
		g.shiftEclipse(def.EclipseShift, def.ID, player)
	}
}

// --- Janelas extras de Assalto ---

// openExtraOrTwilight ativa a próxima janela extra enfileirada; sem nenhuma,
// encerra a rodada.
func (g *Game) openExtraOrTwilight() {
	s := g.s
	if len(s.QueuedExtra) > 0 {
		w := s.QueuedExtra[0]
		s.QueuedExtra = s.QueuedExtra[1:]
		s.Extra = &w
		s.Active = w.Player
		g.emit(Event{Kind: EvExtraWindow, P: w.Player, N: w.MaxCost, S: w.Source})
		return
	}
	g.startTwilight()
}

// --- Reação a Ritos (VR-035) ---

// defenderHoldsCounter responde se o defensor pode reagir a um Rito
// direcionado. A janela só abre nesse caso (ADR-014: evita atrasar todo Rito;
// o vazamento de informação é aceito no alpha e revisitado na Fase 4).
func (g *Game) defenderHoldsCounter(defender int) bool {
	p := g.s.Players[defender]
	for _, id := range p.Hand {
		def := Cards[g.s.Cards[id].Def]
		if def.Type != TypeGuarda {
			continue
		}
		gi := guardImpls[def.ID]
		if gi != nil && gi.counterRite && g.canAfford(defender, g.effectiveCost(defender, def, id)) {
			return true
		}
	}
	return false
}

// playCounterGuard resolve um counter na janela de reação.
func (g *Game) playCounterGuard(player int, inst *CardInstance, def *CardDef) error {
	react := g.s.RiteReact
	gi := guardImpls[def.ID]
	if gi == nil || !gi.counterRite {
		return errCmd(ErrWrongPhase, "apenas Guardas de counter na janela de reação")
	}
	if !g.canAfford(player, g.effectiveCost(player, def, inst.ID)) {
		return errCmd(ErrCantAfford, "essência insuficiente para %s", def.ID)
	}
	g.payAndAnnounce(player, inst, def)
	p := g.s.Players[player]
	p.GuardsRound++
	g.emitCardSigil(player, def, inst.ID)
	g.shiftEclipse(def.EclipseShift, def.ID, player)

	// O Rito é anulado: vai ao descarte sem resolver; o conjurador recupera
	// metade do custo como Essência temporária.
	caster := react.Caster
	riteInst := g.s.Cards[react.Inst]
	riteInst.Zone = ZoneDiscard
	cp := g.s.Players[caster]
	cp.Discard = append(cp.Discard, riteInst.ID)
	g.emit(Event{Kind: EvRiteCountered, P: caster, Card: react.Inst, Def: react.Def, N: react.Cost / 2})
	if refund := react.Cost / 2; refund > 0 {
		g.gainTempEssence(caster, refund, def.ID)
	}

	inst.Zone = ZoneDiscard
	p.Discard = append(p.Discard, inst.ID)
	g.s.RiteReact = nil
	return nil
}

// resumeSuspendedRite retoma um Rito após o defensor passar a reação.
func (g *Game) resumeSuspendedRite() {
	react := g.s.RiteReact
	g.s.RiteReact = nil
	inst := g.s.Cards[react.Inst]
	g.resolveRiteBody(react.Caster, inst, Cards[react.Def], riteImpls[react.Def])
}

// --- Crepúsculo: retornos do Coração da Lua Feral (VR-048) ---

func (g *Game) processPendingReturns() {
	s := g.s
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		p := s.Players[i]
		for _, id := range p.PendingReturns {
			inst := s.Cards[id]
			if inst == nil || inst.Zone != ZoneDiscard {
				continue
			}
			var ok bool
			p.Discard, ok = removeID(p.Discard, id)
			if !ok {
				continue
			}
			p.Hand = append(p.Hand, id)
			inst.Zone = ZoneHand
			g.emit(Event{Kind: EvCardReturned, P: i, Card: id, Def: inst.Def, S: "VR-048"})
		}
		p.PendingReturns = nil
	}
}
