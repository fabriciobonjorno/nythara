package engine

// Intérprete da DSL: executa []Op num contexto (jogador dono da carta, carta
// fonte, contexto de combate/gatilho e carta escolhida em decisão anterior).

type opCtx struct {
	player int
	source string // def id da carta fonte
	inst   string // instância da carta fonte
	guard  *GuardCtx
	curse  *TimedN // em gatilhos curse_applied
	picked string  // instância escolhida na decisão que originou estes ops
	depth  int
}

func (g *Game) runOps(ops []Op, ctx *opCtx) {
	for i := range ops {
		if g.s.Over {
			return
		}
		g.runOp(&ops[i], ctx)
	}
}

func (g *Game) target(op *Op, ctx *opCtx) int {
	if op.Who == "opponent" {
		return 1 - ctx.player
	}
	return ctx.player
}

func (g *Game) runOp(op *Op, ctx *opCtx) {
	s := g.s
	tgt := g.target(op, ctx)
	switch op.Op {
	case "damage":
		g.loseVitality(tgt, op.N, ctx.source)
	case "heal":
		g.heal(tgt, op.N, ctx.source)
	case "draw":
		for i := 0; i < op.N && !s.Over; i++ {
			g.drawOne(tgt)
		}
	case "both_draw":
		for _, p := range []int{s.Initiative, 1 - s.Initiative} {
			if !s.Over {
				g.drawOne(p)
			}
		}
	case "ward":
		g.gainWard(tgt, op.N, ctx.source)
	case "temp_essence":
		g.gainTempEssence(tgt, op.N, ctx.source)
	case "lose_vitality":
		n := op.N
		if op.NFrom == "picked_cost" && ctx.picked != "" {
			n = g.rs.Cards[s.Cards[ctx.picked].Def].Cost
		}
		g.loseVitality(tgt, n, ctx.source)
	case "shift":
		g.shiftEclipse(op.N, ctx.source, ctx.player)
	case "set_eclipse":
		g.shiftEclipse(op.N-s.Eclipse, ctx.source, ctx.player)
	case "status":
		g.applyStatusOp(op, tgt, ctx)
	case "remove_own_bleeds":
		p := s.Players[tgt]
		if len(p.Bleeds) > 0 {
			p.Bleeds = nil
			g.emit(Event{Kind: EvStatusExpired, P: tgt, S: "Sangramento", Def: ctx.source})
		}
	case "remove_own_curse":
		p := s.Players[tgt]
		if len(p.Curses) > 0 {
			p.Curses = p.Curses[1:]
			g.emit(Event{Kind: EvStatusExpired, P: tgt, S: "Maldição", Def: ctx.source})
		}
	case "remove_curse_smart":
		// VR-074: prioriza remover a própria Maldição; senão remove do
		// oponente e compra 1.
		self := s.Players[ctx.player]
		opp := s.Players[1-ctx.player]
		if len(self.Curses) > 0 {
			self.Curses = self.Curses[1:]
			g.emit(Event{Kind: EvStatusExpired, P: ctx.player, S: "Maldição", Def: ctx.source})
		} else if len(opp.Curses) > 0 {
			opp.Curses = opp.Curses[1:]
			g.emit(Event{Kind: EvStatusExpired, P: 1 - ctx.player, S: "Maldição", Def: ctx.source})
			g.drawOne(ctx.player)
		}
	case "remove_veil_expose":
		opp := s.Players[1-ctx.player]
		if veilActive(opp, s.Round) {
			opp.VeilRound = 0
			g.emit(Event{Kind: EvStatusExpired, P: 1 - ctx.player, S: "Véu", Def: ctx.source})
			opp.Exposto = true
			g.emit(Event{Kind: EvStatusApplied, P: 1 - ctx.player, S: "Exposto", Def: ctx.source})
		}
	case "curse_duration_delta":
		if ctx.curse != nil {
			ctx.curse.Round += op.N
		}
	case "cost_mod":
		p := s.Players[tgt]
		m := CostMod{
			Type: CardType(op.Type), Faction: op.Faction, Delta: op.Delta,
			Uses: op.Uses, Source: ctx.source,
		}
		if op.ThisRound {
			m.Round = s.Round
		}
		if op.SelfInst {
			m.Instance = ctx.inst
		}
		p.CostMods = append(p.CostMods, m)
		g.emit(Event{Kind: EvCostModAdded, P: tgt, N: op.Delta, S: ctx.source})
	case "suppress_opp_manifs":
		opp := s.Players[1-ctx.player]
		until := s.Round + max(op.N, 1)
		if until > opp.ManifsSuppressedUntil {
			opp.ManifsSuppressedUntil = until
		}
		g.emit(Event{Kind: EvStatusApplied, P: 1 - ctx.player, S: "Silêncio", Def: ctx.source, N: op.N})
	case "lua_no_sangue":
		s.Players[ctx.player].LuaNoSangue = true
	case "emit_sigil":
		g.emitSigil(ctx.player, Sigil(op.Sigil), ctx.source, ctx.depth+1)
	case "end_assault_window":
		if ctx.guard != nil {
			ctx.guard.EndWindow = true
		}
	case "open_extra_window":
		s.QueuedExtra = append(s.QueuedExtra, ExtraWindow{
			Player: ctx.player, MaxCost: op.N, Source: ctx.source,
		})
	case "emit_last_card_sigil":
		// VR-030: copia o Sigilo da última carta que você jogou (antes dele).
		if prev := s.Players[ctx.player].PrevPlayed; prev != "" {
			g.emitSigil(ctx.player, g.rs.Cards[prev].Sigil, ctx.source, ctx.depth+1)
		}
	case "moon_return":
		// VR-048: o Assalto retorna à mão no fim da rodada e custa +1.
		if ctx.guard != nil {
			if inst := s.Cards[ctx.guard.AssaultInst]; inst != nil {
				p := s.Players[ctx.player]
				p.PendingReturns = append(p.PendingReturns, inst.ID)
				p.CostMods = append(p.CostMods, CostMod{
					Instance: inst.ID, Delta: 1, Uses: 1, Source: ctx.source,
				})
				g.emit(Event{Kind: EvCostModAdded, P: ctx.player, N: 1, S: ctx.source})
			}
		}
	case "conditional":
		if g.evalCond(*op.If, ctx) {
			g.runOps(op.Then, ctx)
		} else {
			g.runOps(op.Else, ctx)
		}
	default:
		g.runDecisionOp(op, ctx)
	}
}

// applyStatusOp aplica um status. Aplicações direcionadas ao oponente sob Véu
// por um Rito fracassam (fizzle) — cartas cujo único efeito é direcionado usam
// targets_opponent e já são rejeitadas ao jogar.
func (g *Game) applyStatusOp(op *Op, tgt int, ctx *opCtx) {
	s := g.s
	if tgt != ctx.player && veilActive(s.Players[tgt], s.Round) &&
		g.rs.Cards[ctx.source] != nil && g.rs.Cards[ctx.source].Type == TypeRito {
		g.emit(Event{Kind: EvStatusFizzled, P: tgt, S: op.Status, Def: ctx.source})
		return
	}
	p := s.Players[tgt]
	switch op.Status {
	case "exposto":
		p.Exposto = true
		g.emit(Event{Kind: EvStatusApplied, P: tgt, S: "Exposto", Def: ctx.source})
	case "veu":
		p.VeilRound = s.Round + 1
		g.emit(Event{Kind: EvStatusApplied, P: tgt, S: "Véu", Def: ctx.source})
	case "sangramento":
		p.Bleeds = append(p.Bleeds, TimedN{N: op.N, Round: s.Round + 1})
		g.emit(Event{Kind: EvStatusApplied, P: tgt, N: op.N, S: "Sangramento", Def: ctx.source})
	case "maldicao":
		g.applyCurse(tgt, TimedN{N: op.N, Round: s.Round + 1, Kind: op.Kind}, ctx.source, ctx.player)
	}
}

// applyCurse aplica uma Maldição e dispara gatilhos do alvo (que podem alterar
// a duração — VR-016 — ou reagir — VR-064). A forma de Edda alonga as
// Maldições que ela aplica na Aurora Total.
func (g *Game) applyCurse(victim int, tn TimedN, source string, caster int) {
	tn.Round += champCurseDurationBonus(g, caster)
	p := g.s.Players[victim]
	p.Curses = append(p.Curses, tn)
	g.emit(Event{Kind: EvStatusApplied, P: victim, N: tn.N, S: "Maldição", Def: source})
	curse := &p.Curses[len(p.Curses)-1]
	g.firePerms(victim, func(pi *permImpl, inst *CardInstance) {
		if pi.onCurseApplied != nil {
			pi.onCurseApplied(g, inst, curse)
		}
	})
}

// --- Condições ---

func (g *Game) evalCond(c Cond, ctx *opCtx) bool {
	s := g.s
	p := s.Players[ctx.player]
	opp := s.Players[1-ctx.player]
	switch c.Cond {
	case "always":
		return true
	case "sacrificed_this_round":
		return p.SacrificesRound > 0
	case "eclipse_le":
		return s.Eclipse <= c.N
	case "eclipse_ge":
		return s.Eclipse >= c.N
	case "eclipse_eq":
		return s.Eclipse == c.N
	case "eclipse_abs_eq":
		return s.Eclipse == c.N || s.Eclipse == -c.N
	case "eclipse_night":
		return s.EclipseState == EclipseNight
	case "eclipse_aurora":
		return s.EclipseState == EclipseAurora
	case "second_assault":
		return p.AssaultsRound == 2
	case "took_damage_round":
		return p.DamageTakenRound > 0
	case "exiled_round":
		return p.ExiledRound
	case "trail_ends":
		// "*" é curinga (VR-047: Garra→Garra→qualquer).
		if len(p.Trail) < len(c.Sigils) {
			return false
		}
		off := len(p.Trail) - len(c.Sigils)
		for i, sg := range c.Sigils {
			if sg != "*" && p.Trail[off+i] != Sigil(sg) {
				return false
			}
		}
		return true
	case "distinct_sigils_ge":
		return distinctSigils(p.Trail) >= c.N
	case "sigil_count_eq":
		count := 0
		for _, sg := range p.Trail {
			if sg == Sigil(c.Sigils[0]) {
				count++
			}
		}
		return count == c.N
	case "last_global_sigil_in":
		rs := s.RoundSigils
		if len(rs) == 0 {
			return false
		}
		last := rs[len(rs)-1].S
		for _, sg := range c.Sigils {
			if Sigil(sg) == last {
				return true
			}
		}
		return false
	case "hand_less_than_opp":
		return len(p.Hand) < len(opp.Hand)
	case "hand_ge":
		return len(p.Hand) >= c.N
	case "opp_hand_ge":
		return len(opp.Hand) >= c.N
	case "opp_has_manif":
		return len(opp.Manifs) > 0
	case "opp_has_relic":
		return len(opp.Relics) > 0
	case "opp_discard_nonempty":
		return len(opp.Discard) > 0
	case "opp_vitality_le":
		return opp.Vitality <= c.N
	case "prevented_all":
		return ctx.guard != nil && ctx.guard.PreventedAll
	case "resonance_bonus":
		return ctx.guard != nil && ctx.guard.ResonanceBonus
	case "relic_destroyed_round":
		return p.RelicDestroyedRound
	case "opp_veiled":
		return veilActive(opp, s.Round)
	case "discard_has":
		for _, id := range p.Discard {
			def := g.rs.Cards[s.Cards[id].Def]
			if c.Type == "" || string(def.Type) == c.Type {
				return true
			}
		}
		return false
	case "discard_ge":
		return len(p.Discard) >= c.N
	case "opp_assaults_eq":
		return opp.AssaultsRound == c.N
	case "own_assaults_eq":
		return p.AssaultsRound == c.N
	case "picked_was_rite":
		return ctx.guard != nil && ctx.guard.PickedExile != "" &&
			g.rs.Cards[s.Cards[ctx.guard.PickedExile].Def].Type == TypeRito
	}
	return false
}

// --- Ops de decisão ---

func filterMatches(def *CardDef, f *Filter) bool {
	if f == nil {
		return true
	}
	if f.Type != "" && string(def.Type) != f.Type {
		return false
	}
	if f.HasCost && def.Cost != f.Cost {
		return false
	}
	if f.MaxCost > 0 && def.Cost > f.MaxCost {
		return false
	}
	return true
}

func (g *Game) runDecisionOp(op *Op, ctx *opCtx) {
	s := g.s
	p := s.Players[ctx.player]
	d := &Decision{
		Player: ctx.player, Card: ctx.inst, Source: ctx.source, Then: op.Then,
	}
	switch op.Op {
	case "choose_discard":
		if len(p.Hand) == 0 {
			g.runOps(op.Then, ctx)
			return
		}
		d.Kind = DecDiscardN
		d.Options = append([]string{}, p.Hand...)
		d.N = min(op.N, len(p.Hand))
	case "pick_top2":
		switch {
		case len(p.Deck) == 0:
			return
		case len(p.Deck) == 1:
			id := p.Deck[0]
			p.Deck = p.Deck[1:]
			p.Hand = append(p.Hand, id)
			s.Cards[id].Zone = ZoneHand
			g.emit(Event{Kind: EvCardToHand, P: ctx.player, Card: id, Def: s.Cards[id].Def})
			return
		default:
			d.Kind = DecPickTop2
			d.Options = []string{p.Deck[0], p.Deck[1]}
			d.N = 1
		}
	case "reorder_top":
		n := min(op.N, len(p.Deck))
		if n <= 1 {
			return
		}
		d.Kind = DecReorderTop
		d.Options = append([]string{}, p.Deck[:n]...)
		d.N = n
	case "recover_pick":
		var opts []string
		for _, id := range p.Discard {
			if filterMatches(g.rs.Cards[s.Cards[id].Def], op.Filter) {
				opts = append(opts, id)
			}
		}
		if len(opts) == 0 {
			return
		}
		d.Kind = DecRecoverPick
		d.Options = opts
		d.N = min(op.N, len(opts))
	case "draw_reveal_opp_picks":
		var drawn []string
		for i := 0; i < op.N && !s.Over; i++ {
			if inst := g.drawOne(ctx.player); inst != nil {
				drawn = append(drawn, inst.ID)
			}
		}
		// Gatilhos de compra podem mover a mão antes da escolha (ex.: Recusa da
		// Morte de Kaedor descarta tudo). Só ofereça cartas que continuam na mão.
		remaining := drawn[:0]
		for _, id := range drawn {
			if s.Cards[id].Zone == ZoneHand {
				remaining = append(remaining, id)
			}
		}
		drawn = remaining
		if s.Over || len(drawn) == 0 {
			return
		}
		d.Kind = DecOppDiscardPick
		d.Player = 1 - ctx.player
		d.Options = drawn
		d.N = 1
	case "mill_top_confirm":
		if len(p.Deck) == 0 {
			return
		}
		d.Kind = DecMillTop
		d.Options = []string{"yes", "no"}
		d.N = 1
	case "exile_self_heal_confirm":
		d.Kind = DecExileSelfHeal
		d.Options = []string{"yes", "no"}
		d.N = 1
	case "swap_eclipse_confirm":
		if s.Eclipse == 0 {
			return
		}
		d.Kind = DecSwapEclipse
		d.Options = []string{"yes", "no"}
		d.N = 1
	case "destroy_relic_pick":
		opp := s.Players[1-ctx.player]
		if len(opp.Relics) == 0 {
			return
		}
		d.Kind = DecDestroyRelicPick
		d.Options = append([]string{}, opp.Relics...)
		d.N = 1
	case "tax_type_pick":
		opp := s.Players[1-ctx.player]
		if len(opp.Discard) == 0 {
			return
		}
		d.Kind = DecTaxTypePick
		d.Options = append([]string{}, opp.Discard...)
		d.N = 1
	case "neutral_sigil_choice":
		d.Kind = DecPickSigil
		d.Options = []string{"Presa", "Sol", "Espelho", "Garra", "Cinza", "Coroa"}
		d.N = 1
	case "peek_opp_hand_tax":
		// VR-006: olha 2 cartas aleatórias da mão adversária.
		opp := s.Players[1-ctx.player]
		if len(opp.Hand) == 0 {
			return
		}
		ids := append([]string{}, opp.Hand...)
		s.RNG.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		n := min(op.N, len(ids))
		for _, id := range ids[:n] {
			g.emit(Event{Kind: EvHandRevealed, P: ctx.player, Card: id, Def: s.Cards[id].Def})
		}
		d.Kind = DecRevealTax
		d.Options = ids[:n]
		d.N = 1
	case "reveal_lock_assault":
		// VR-018: revela a mão adversária e trava um Assalto.
		opp := s.Players[1-ctx.player]
		var assaults []string
		for _, id := range opp.Hand {
			g.emit(Event{Kind: EvHandRevealed, P: ctx.player, Card: id, Def: s.Cards[id].Def})
			if g.rs.Cards[s.Cards[id].Def].Type == TypeAssalto {
				assaults = append(assaults, id)
			}
		}
		if len(assaults) == 0 {
			return
		}
		d.Kind = DecLockAssault
		d.Options = assaults
		d.N = 1
	case "declare_reveal_top":
		d.Kind = DecDeclareType
		d.Options = []string{"Assalto", "Guarda", "Rito"}
		d.N = 1
	case "mirror_relic_pick":
		opp := s.Players[1-ctx.player]
		if len(opp.Relics) == 0 {
			return
		}
		d.Kind = DecMirrorRelic
		d.Options = append([]string{}, opp.Relics...)
		d.N = 1
	case "copy_played_pick":
		// VR-036: copia uma carta não-Lendária (Assalto/Rito) resolvida na rodada.
		seen := map[string]bool{}
		var opts []string
		for _, pc := range s.PlayedRound {
			def := g.rs.Cards[pc.Def]
			if pc.IsCopy || seen[pc.Def] || def.Rarity == RarityLendaria {
				continue
			}
			if def.Type != TypeAssalto && def.Type != TypeRito {
				continue
			}
			if def.ID == ctx.source {
				continue
			}
			seen[pc.Def] = true
			opts = append(opts, pc.Def)
		}
		if len(opts) == 0 {
			return
		}
		d.Kind = DecCopyPlayed
		d.Options = opts
		d.N = 1
	case "re_emit_sigils":
		// VR-034: gere novamente até N Sigilos distintos já emitidos.
		seen := map[string]bool{}
		var opts []string
		for _, sg := range p.Trail {
			if !seen[string(sg)] {
				seen[string(sg)] = true
				opts = append(opts, string(sg))
			}
		}
		if len(opts) == 0 {
			return
		}
		d.Kind = DecReEmitSigils
		d.Options = opts
		d.N = min(op.N, len(opts))
		d.MinN = 1
	case "shift_chosen":
		d.Kind = DecDirection
		d.Options = []string{"noite", "aurora"}
		d.N = 1
	case "clock_direction_choice":
		// VR-076: só em rodadas em que o Eclipse não se moveu; escolhe o
		// jogador com menos Vitalidade (empate: o controlador da Relíquia).
		if s.EclipseMovedRound {
			return
		}
		chooser := ctx.player
		if s.Players[0].Vitality != s.Players[1].Vitality {
			chooser = 0
			if s.Players[1].Vitality < s.Players[0].Vitality {
				chooser = 1
			}
		}
		d.Player = chooser
		d.Kind = DecDirection
		d.Options = []string{"noite", "aurora"}
		d.N = 1
	case "exile_all_choices":
		// VR-060: exila o descarte inteiro; 1 escolha por lote (máx. 3).
		count := len(p.Discard)
		batch := op.N
		if batch <= 0 {
			batch = 4
		}
		for len(p.Discard) > 0 {
			g.exileFromDiscard(ctx.player, p.Discard[0])
		}
		for i := 0; i < min(count/batch, 3); i++ {
			g.requestDecision(&Decision{
				Player: ctx.player, Kind: DecFormulaChoice,
				Options: []string{"dano_2", "cura_2", "compra_1"}, N: 1, Source: ctx.source,
			})
		}
		return
	default:
		panic("engine: op desconhecida em runtime (validador falhou): " + op.Op)
	}
	g.requestDecision(d)
}

// resolveDecisionPicks executa a semântica de cada tipo de decisão após a
// validação em applyChoose. Retorna a instância "escolhida" para binding.
func (g *Game) resolveDecisionPicks(d *Decision, picks []string) string {
	s := g.s
	switch d.Kind {
	case DecDiscardN:
		for _, id := range picks {
			g.discardFromHand(d.Player, id)
			if d.Source != "" {
				champOnCostDiscard(g, d.Player) // marcadores de Edda
			}
		}
	case DecPickTop2:
		p := s.Players[d.Player]
		pick := picks[0]
		for _, id := range d.Options {
			p.Deck, _ = removeID(p.Deck, id)
		}
		for _, id := range d.Options {
			if id == pick {
				p.Hand = append(p.Hand, id)
				s.Cards[id].Zone = ZoneHand
				g.emit(Event{Kind: EvCardToHand, P: d.Player, Card: id, Def: s.Cards[id].Def})
			} else {
				p.Deck = append(p.Deck, id)
				s.Cards[id].Zone = ZoneDeck
				g.emit(Event{Kind: EvCardToBottom, P: d.Player, Card: id, Def: s.Cards[id].Def})
			}
		}
		return pick
	case DecReorderTop:
		p := s.Players[d.Player]
		// Defesa em profundidade: só reescreve cartas que AINDA estão no
		// baralho (a cadeia pode ter comprado uma delas — ADR-032).
		kept := picks[:0]
		for _, id := range picks {
			if s.Cards[id] != nil && s.Cards[id].Zone == ZoneDeck {
				kept = append(kept, id)
			}
		}
		copy(p.Deck[:len(kept)], kept)
		g.emit(Event{Kind: EvDeckReordered, P: d.Player, N: len(kept)})
	case DecRecoverPick:
		p := s.Players[d.Player]
		for _, id := range picks {
			p.Discard, _ = removeID(p.Discard, id)
			p.Hand = append(p.Hand, id)
			s.Cards[id].Zone = ZoneHand
			g.emit(Event{Kind: EvCardToHand, P: d.Player, Card: id, Def: s.Cards[id].Def})
			p.RecoversRound++
			if p.RecoversRound == 1 {
				champOnRecover(g, d.Player, id) // passiva de Voren
			}
		}
		return picks[0]
	case DecOppDiscardPick:
		victim := 1 - d.Player
		g.discardFromHand(victim, picks[0])
		champOnCostDiscard(g, victim)
		return picks[0]
	case DecMillTop:
		if picks[0] == "yes" {
			p := s.Players[d.Player]
			if len(p.Deck) > 0 {
				id := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, id)
				s.Cards[id].Zone = ZoneDiscard
				g.emit(Event{Kind: EvCardDiscarded, P: d.Player, Card: id, Def: s.Cards[id].Def})
			}
		}
	case DecExileSelfHeal:
		if picks[0] == "yes" {
			g.exileFromDiscard(d.Player, d.Card)
			g.heal(d.Player, 1, d.Card)
		}
	case DecSwapEclipse:
		if picks[0] == "yes" && s.Eclipse != 0 {
			g.shiftEclipse(-2*s.Eclipse, d.Source, d.Player)
		}
	case DecDestroyRelicPick:
		pick := picks[0]
		owner := s.Cards[pick].Owner
		op := s.Players[owner]
		op.Relics, _ = removeID(op.Relics, pick)
		op.Discard = append(op.Discard, pick)
		s.Cards[pick].Zone = ZoneDiscard
		op.RelicDestroyedRound = true
		g.emit(Event{Kind: EvPermanentDestroyed, P: owner, Card: pick, Def: s.Cards[pick].Def, S: string(TypeReliquia)})
		// VR-046: se custava 3+, o controlador sofre 2.
		if g.rs.Cards[s.Cards[pick].Def].Cost >= 3 {
			g.loseVitality(owner, 2, d.Source)
		}
		return pick
	case DecTaxTypePick:
		pick := picks[0]
		owner := s.Cards[pick].Owner
		taxed := g.rs.Cards[s.Cards[pick].Def].Type
		s.Players[owner].CostMods = append(s.Players[owner].CostMods, CostMod{
			Type: taxed, Delta: 1, Uses: 999, Round: s.Round, Source: d.Source,
		})
		g.emit(Event{Kind: EvCostModAdded, P: owner, N: 1, S: d.Source})
		return pick
	case DecPickSigil:
		p := s.Players[d.Player]
		p.ExtraNeutralSigil = Sigil(picks[0])
		g.emit(Event{Kind: EvStatusApplied, P: d.Player, S: "Juramento:" + picks[0], Def: d.Source})

	case DecExilePick:
		for _, id := range picks {
			g.exileFromDiscard(d.Player, id)
		}
		// VR-055: o exílio precede a janela de Guarda do próprio Assalto.
		if s.Guard != nil && !s.Guard.Announced && s.Guard.AssaultDef == d.Source {
			s.Guard.PickedExile = picks[0]
			g.announceAssault()
		}
		// Ultimate de Voren: cada exilada vira uma escolha cura/dano.
		if d.Source == "CH-CI-01" {
			for _, id := range picks {
				g.requestDecision(&Decision{
					Player: d.Player, Kind: DecVorenChoice,
					Options: []string{"cura_1", "dano_1"}, N: 1, Card: id, Source: d.Source,
				})
			}
		}
		return picks[0]
	case DecReEmitSigils:
		for _, sg := range picks {
			g.emitSigil(d.Player, Sigil(sg), d.Source, 1)
		}
	case DecRevealTax:
		// VR-006: a escolhida custa +1 até o fim da próxima rodada.
		pick := picks[0]
		owner := s.Cards[pick].Owner
		s.Players[owner].CostMods = append(s.Players[owner].CostMods, CostMod{
			Instance: pick, Delta: 1, Uses: 1, UntilRound: s.Round + 1, Source: d.Source,
		})
		g.emit(Event{Kind: EvCostModAdded, P: owner, N: 1, S: d.Source})
		return pick
	case DecLockAssault:
		pick := picks[0]
		s.Cards[pick].LockedRound = s.Round
		g.emit(Event{Kind: EvCardLocked, P: s.Cards[pick].Owner, Card: pick, Def: s.Cards[pick].Def, S: d.Source})
		return pick
	case DecDeclareType:
		// VR-029: revela o topo do deck rival; acerto compra 1 e taxa a carta.
		opp := s.Players[1-d.Player]
		if len(opp.Deck) == 0 {
			return ""
		}
		top := opp.Deck[0]
		g.emit(Event{Kind: EvDeckTopRevealed, P: d.Player, Card: top, Def: s.Cards[top].Def})
		if string(g.rs.Cards[s.Cards[top].Def].Type) == picks[0] {
			g.drawOne(d.Player)
			opp.CostMods = append(opp.CostMods, CostMod{
				Instance: top, Delta: 1, Uses: 1, Round: s.Round, Source: d.Source,
			})
			g.emit(Event{Kind: EvCostModAdded, P: 1 - d.Player, N: 1, S: d.Source})
		}
	case DecDirection:
		n := 1
		if picks[0] == "aurora" {
			n = -1
		}
		g.shiftEclipse(n, d.Source, d.Player)
	case DecFormulaChoice:
		switch picks[0] {
		case "dano_2":
			g.loseVitality(1-d.Player, 2, d.Source)
		case "cura_2":
			g.heal(d.Player, 2, d.Source)
		case "compra_1":
			g.drawOne(d.Player)
		}
	case DecMirrorRelic:
		pick := picks[0]
		p := s.Players[d.Player]
		p.MirroredRelic = s.Cards[pick].Def
		p.MirrorUntil = s.Round
		p.MirrorUsedRound = 0
		g.emit(Event{Kind: EvCopyResolved, P: d.Player, Card: pick, Def: s.Cards[pick].Def, S: "espelho"})
		return pick
	case DecCopyPlayed:
		g.resolveCopy(d.Player, picks[0], 1)
	case DecScryBottom:
		if picks[0] == "yes" {
			p := s.Players[d.Player]
			if len(p.Deck) > 0 {
				id := p.Deck[0]
				p.Deck = append(p.Deck[1:], id)
				g.emit(Event{Kind: EvCardToBottom, P: d.Player, Card: id, Def: s.Cards[id].Def})
			}
		}
	case DecOrenChoice:
		if picks[0] == "comprar_2" {
			g.drawOne(d.Player)
			g.drawOne(d.Player)
		} else {
			opp := s.Players[1-d.Player]
			ids := append([]string{}, opp.Hand...)
			s.RNG.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
			for _, id := range ids[:min(2, len(ids))] {
				opp.Hand, _ = removeID(opp.Hand, id)
				opp.Deck = append(opp.Deck, id)
				s.Cards[id].Zone = ZoneDeck
				g.emit(Event{Kind: EvCardToBottom, P: 1 - d.Player, Card: id, Def: s.Cards[id].Def})
			}
			s.RNG.Shuffle(len(opp.Deck), func(i, j int) { opp.Deck[i], opp.Deck[j] = opp.Deck[j], opp.Deck[i] })
			g.emit(Event{Kind: EvDeckReshuffled, P: 1 - d.Player, N: len(opp.Deck)})
		}
	case DecVorenChoice:
		if picks[0] == "cura_1" {
			g.heal(d.Player, 1, d.Source)
		} else {
			g.loseVitality(1-d.Player, 1, d.Source)
		}
	case DecEddaReturn:
		pick := picks[0]
		p := s.Players[d.Player]
		p.Discard, _ = removeID(p.Discard, pick)
		p.Hand = append(p.Hand, pick)
		s.Cards[pick].Zone = ZoneHand
		g.emit(Event{Kind: EvCardToHand, P: d.Player, Card: pick, Def: s.Cards[pick].Def})
		p.RecoversRound++
		if p.RecoversRound == 1 {
			champOnRecover(g, d.Player, pick)
		}
		p.CostMods = append(p.CostMods, CostMod{
			Instance: pick, Delta: -9, Uses: 1, Round: s.Round, Source: d.Source,
		})
		g.emit(Event{Kind: EvCostModAdded, P: d.Player, N: -9, S: d.Source})
		return pick
	}
	return ""
}
