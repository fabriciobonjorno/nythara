package engine

// Implementações dos 10 Campeões: passiva, Forma de Eclipse e ultimate.
// Campeões permanecem em Go (a DSL de Campeões chega com o painel admin);
// interpretações registradas em DECISIONS.md (ADR-015).

type championImpl struct {
	costDelta         func(g *Game, player int, def *CardDef) int
	assaultBonus      func(g *Game, player int) int
	guardPreventBonus func(g *Game, player int) int
	onSacrifice       func(g *Game, player int)
	onGuardResolved   func(g *Game, player int)
	resonanceCapDelta func(g *Game, player int) int

	healBonus           func(g *Game, player int) int                                     // Seris: Noturno
	firstAssaultPierces func(g *Game, player int) bool                                    // Kaedor: Noturno
	afterFullPrevent    func(g *Game, player int)                                         // Mara: Aurora Total
	onOwnSigil          func(g *Game, player int, sig Sigil, fromAssault bool, depth int) // Saela: Noturno
	onRecover           func(g *Game, player int, inst string)                            // Voren
	onCostDiscard       func(g *Game, player int)                                         // Edda
	onExileSelf         func(g *Game, player int)                                         // Voren: forma
	onOppNightShift     func(g *Game, player int)                                         // Ilyan: passiva
	curseDurationBonus  func(g *Game, player int) int                                     // Edda: Aurora Total
	onExtraDraw         func(g *Game, player int, inst string)                            // Oren: forma
	copyExtraSigil      func(g *Game, player int) bool                                    // Nyra: forma
	preMulliganScry     bool                                                              // Oren: passiva
	fatigueStepDelta    func(g *Game, player int) int                                     // Oren: passiva (ADR-029)
	on3Sigils           func(g *Game, player int)                                         // Nyra: passiva

	canUltimate func(g *Game, player int) error
	ultimate    func(g *Game, player int)
}

func anyTotal(g *Game) bool { return g.s.EclipseState != EclipseNone }

var championImpls = map[string]*championImpl{}

// O registro vive num init para permitir referências cruzadas (ultimates
// chamam resolveCopy, que consulta o próprio registro).
func init() {
	for id, ci := range map[string]*championImpl{
		// Seris Vhal, Herdeira Carmesim — sacrifício vira Essência; Dreno +1 na
		// Noite; ultimate: cura metade do último dano causado.
		"CH-VH-01": {
			onSacrifice: func(g *Game, player int) {
				if g.s.Players[player].SacrificesRound == 1 {
					g.gainTempEssence(player, 1, "CH-VH-01")
				}
			},
			healBonus: func(g *Game, player int) int {
				if g.s.EclipseState == EclipseNight {
					return 1
				}
				return 0
			},
			canUltimate: func(g *Game, player int) error {
				if g.s.Players[player].LastDamageDealt <= 0 {
					return errCmd(ErrRequirement, "Banquete da Lua exige dano causado nesta rodada")
				}
				return nil
			},
			ultimate: func(g *Game, player int) {
				g.heal(player, g.s.Players[player].LastDamageDealt/2, "CH-VH-01")
			},
		},

		// Kaedor, o Sem-Pulso — desespero barato; primeiro Assalto fura Ward na
		// Noite; ultimate automática: Recusa da Morte (ver applyVitalityLoss).
		"CH-VH-02": {
			costDelta: func(g *Game, player int, def *CardDef) int {
				p := g.s.Players[player]
				if def.Type == TypeAssalto && p.Vitality <= 15 && p.AssaultsRound == 0 {
					return -1
				}
				return 0
			},
			firstAssaultPierces: func(g *Game, player int) bool {
				return g.s.EclipseState == EclipseNight && g.s.Players[player].AssaultsRound == 1
			},
			canUltimate: func(g *Game, player int) error {
				return errCmd(ErrBadCommand, "Recusa da Morte é automática")
			},
		},

		// Mara Vale, Lâmina do Primeiro Sol — primeira Guarda +1; compra ao
		// prevenir tudo na Aurora Total; ultimate: Aurora Implacável.
		"CH-SO-01": {
			// alpha-0.5.0: era "+1 de prevenção na 1ª Guarda" — dominava o
			// meta (91% com bots). Vira vantagem de tempo, não de muralha.
			// alpha-0.6.0: o desconto só vale com o Eclipse do lado da Aurora
			// (ADR-029) — o motor recorrente compunha por 30+ rodadas sem
			// counterplay; agora o rival desliga a passiva contestando o céu.
			costDelta: func(g *Game, player int, def *CardDef) int {
				if def.Type == TypeGuarda && g.s.Players[player].GuardsRound == 0 &&
					g.s.EclipseState == EclipseAurora {
					return -1
				}
				return 0
			},
			afterFullPrevent: func(g *Game, player int) {
				// alpha-0.6.0 (ADR-029): era compra — vantagem de cartas
				// compunha; cura preserva a identidade de muralha sem bola
				// de neve.
				if g.s.EclipseState == EclipseAurora {
					g.heal(player, 1, "CH-SO-01")
				}
			},
			canUltimate: func(g *Game, player int) error { return nil },
			ultimate: func(g *Game, player int) {
				g.shiftEclipse(-2, "CH-SO-01", player)
				g.loseVitality(1-player, 1, "CH-SO-01") // alpha-0.6.0: era 2
			},
		},

		// Ilyan, Portador da Cicatriz Dourada — pune o 2º empurrão à Noite;
		// Ritos -1 na Aurora Total; ultimate: anula o próximo Eclipse adversário.
		"CH-SO-02": {
			onOppNightShift: func(g *Game, player int) {
				// alpha-0.5.0: era 2. alpha-0.6.0: não acumula (ADR-029) — o
				// fluxo por rodada virava uma montanha de Ward contra decks
				// de Noite, que empurram o Eclipse pelo próprio plano.
				if g.s.Players[player].Ward == 0 {
					g.gainWard(player, 1, "CH-SO-02")
				}
			},
			costDelta: func(g *Game, player int, def *CardDef) int {
				if def.Type == TypeRito && g.s.EclipseState == EclipseAurora {
					return -1
				}
				return 0
			},
			canUltimate: func(g *Game, player int) error { return nil },
			ultimate: func(g *Game, player int) {
				g.s.Players[player].NullifyOppShift = true
			},
		},

		// Nyra dos Sete Reflexos — scry ao completar 3 Sigilos; cópias emitem
		// Sigilo extra em Eclipse Total; ultimate: copia o último Rito rival.
		"CH-MI-01": {
			on3Sigils: func(g *Game, player int) {
				// alpha-0.5.0: era só scry — vira compra direta. alpha-0.6.0
				// (ADR-029): a tríade também gera 1 Essência temporária — o
				// combo produz o tempo do próximo elo em vez de consumi-lo.
				g.drawOne(player)
				g.gainTempEssence(player, 1, "CH-MI-01")
				g.heal(player, 1, "CH-MI-01")
			},
			// alpha-0.6.0: sempre, não só em Eclipse Total (ADR-029) — os
			// Sete Reflexos são a identidade dela; o gatilho raro deixava a
			// passiva quase nunca ligada.
			copyExtraSigil: func(g *Game, player int) bool { return true },
			canUltimate: func(g *Game, player int) error {
				def := g.s.LastRite[1-player]
				if def == "" {
					return errCmd(ErrRequirement, "o oponente ainda não resolveu um Rito")
				}
				if g.rs.rite[def] == nil {
					return errCmd(ErrRequirement, "o último Rito rival não é copiável")
				}
				return nil
			},
			ultimate: func(g *Game, player int) {
				g.resolveCopy(player, g.s.LastRite[1-player], 1)
			},
		},

		// Oren, Leitor do Véu — reordena o topo no início; compra extra mais
		// barata em Eclipse Total; ultimate: Fenda de Probabilidade.
		"CH-MI-02": {
			preMulliganScry: true,
			// alpha-0.6.0 (ADR-029): quem lê o Véu não se queima nele — a
			// Fadiga de Oren cresce 1 a menos por ciclo. O arquétipo de
			// compra esvazia o próprio baralho primeiro e perdia a corrida
			// de atrito que o próprio plano dele alonga.
			fatigueStepDelta: func(g *Game, player int) int { return -2 },
			onExtraDraw: func(g *Game, player int, inst string) {
				// alpha-0.5.0: sem exigir Eclipse Total (era só em totais).
				// alpha-0.6.0 (ADR-029): desconto de 2 — o conhecimento paga
				// o dobro; era -1 e Oren seguia no fundo (20-24%).
				p := g.s.Players[player]
				p.CostMods = append(p.CostMods, CostMod{
					Instance: inst, Delta: -2, Uses: 1, Round: g.s.Round, Source: "CH-MI-02",
				})
				g.emit(Event{Kind: EvCostModAdded, P: player, N: -2, S: "CH-MI-02"})
				g.heal(player, 1, "CH-MI-02") // conhecimento também sustenta (ADR-029)
			},
			canUltimate: func(g *Game, player int) error { return nil },
			ultimate: func(g *Game, player int) {
				g.requestDecision(&Decision{
					Player: player, Kind: DecOrenChoice,
					Options: []string{"comprar_2", "embaralhar_2"}, N: 1, Source: "CH-MI-02",
				})
			},
		},

		// Rauk Fenclaw, Alfa da Fenda — 2º Assalto +1; Ressonância 6 na Noite;
		// ultimate: Caçada Sem Lua (janela extra de Assalto).
		"CH-VA-01": {
			assaultBonus: func(g *Game, player int) int {
				if g.s.Players[player].AssaultsRound == 2 {
					return 1
				}
				return 0
			},
			resonanceCapDelta: func(g *Game, player int) int {
				if g.s.EclipseState == EclipseNight {
					return 1
				}
				return 0
			},
			canUltimate: func(g *Game, player int) error { return nil },
			ultimate: func(g *Game, player int) {
				g.s.QueuedExtra = append(g.s.QueuedExtra, ExtraWindow{Player: player, Source: "CH-VA-01"})
			},
		},

		// Saela, Pele-de-Lua — Guarda alimenta Assalto; Garra ressoa em dobro na
		// Noite; ultimate: Forma da Fera Branca.
		"CH-VA-02": {
			onGuardResolved: func(g *Game, player int) {
				p := g.s.Players[player]
				p.CostMods = append(p.CostMods, CostMod{
					Type: TypeAssalto, Delta: -1, Uses: 1, Round: g.s.Round, Source: "CH-VA-02",
				})
				g.emit(Event{Kind: EvCostModAdded, P: player, N: -1, S: "CH-VA-02"})
			},
			onOwnSigil: func(g *Game, player int, sig Sigil, fromAssault bool, depth int) {
				if g.s.EclipseState == EclipseNight && fromAssault && sig == SigilGarra {
					g.emitSigil(player, SigilGarra, "CH-VA-02", depth+1)
				}
			},
			canUltimate: func(g *Game, player int) error { return nil },
			ultimate: func(g *Game, player int) {
				p := g.s.Players[player]
				p.SaelaShield = true
				p.NextAssaultBonus = 2
			},
		},

		// Voren Ashhand, Mestre da Retorta — recuperação barata; Ward por exílio
		// em Eclipse Total; ultimate: Destilação Negra.
		"CH-CI-01": {
			onRecover: func(g *Game, player int, inst string) {
				p := g.s.Players[player]
				if p.RecoversRound != 1 {
					return
				}
				p.CostMods = append(p.CostMods, CostMod{
					Instance: inst, Delta: -1, Uses: 1, Round: g.s.Round, Source: "CH-CI-01",
				})
				g.emit(Event{Kind: EvCostModAdded, P: player, N: -1, S: "CH-CI-01"})
			},
			onExileSelf: func(g *Game, player int) {
				if anyTotal(g) {
					g.gainWard(player, 1, "CH-CI-01")
				}
			},
			canUltimate: func(g *Game, player int) error {
				if len(g.s.Players[player].Discard) == 0 {
					return errCmd(ErrRequirement, "Destilação Negra exige cartas no descarte")
				}
				return nil
			},
			ultimate: func(g *Game, player int) {
				p := g.s.Players[player]
				n := min(3, len(p.Discard))
				g.requestDecision(&Decision{
					Player: player, Kind: DecExilePick,
					Options: append([]string{}, p.Discard...), N: n, MinN: 1, Source: "CH-CI-01",
				})
			},
		},

		// Edda, Escriba das Cinzas — marcadores por descarte; Maldições mais
		// longas na Aurora Total; ultimate: Página Reescrita.
		"CH-CI-02": {
			onCostDiscard: func(g *Game, player int) {
				p := g.s.Players[player]
				if p.CinzaMarkers < 3 {
					p.CinzaMarkers++
					g.emit(Event{Kind: EvMarkerGained, P: player, N: p.CinzaMarkers, S: "Cinza"})
				}
			},
			curseDurationBonus: func(g *Game, player int) int {
				if g.s.EclipseState == EclipseAurora {
					return 1
				}
				return 0
			},
			canUltimate: func(g *Game, player int) error {
				p := g.s.Players[player]
				if p.CinzaMarkers < 3 {
					return errCmd(ErrRequirement, "Página Reescrita exige 3 marcadores de Cinza")
				}
				var ok bool
				for _, id := range p.Discard {
					if g.rs.Cards[g.s.Cards[id].Def].Rarity != RarityLendaria {
						ok = true
						break
					}
				}
				if !ok {
					return errCmd(ErrRequirement, "nenhuma carta não-Lendária no descarte")
				}
				return nil
			},
			ultimate: func(g *Game, player int) {
				p := g.s.Players[player]
				p.CinzaMarkers -= 3
				var opts []string
				for _, id := range p.Discard {
					if g.rs.Cards[g.s.Cards[id].Def].Rarity != RarityLendaria {
						opts = append(opts, id)
					}
				}
				g.requestDecision(&Decision{
					Player: player, Kind: DecEddaReturn, Options: opts, N: 1, Source: "CH-CI-02",
				})
			},
		},
	} {
		championImpls[id] = ci
	}
}

func champImpl(g *Game, player int) *championImpl {
	return championImpls[g.s.Players[player].Champion]
}

func champCostDelta(g *Game, player int, def *CardDef) int {
	if ci := champImpl(g, player); ci != nil && ci.costDelta != nil {
		return ci.costDelta(g, player, def)
	}
	return 0
}

func champAssaultBonus(g *Game, player int) int {
	bonus := 0
	if ci := champImpl(g, player); ci != nil && ci.assaultBonus != nil {
		bonus += ci.assaultBonus(g, player)
	}
	// Ultimate de Saela: próximo Assalto +2.
	p := g.s.Players[player]
	if p.NextAssaultBonus > 0 {
		bonus += p.NextAssaultBonus
		p.NextAssaultBonus = 0
	}
	return bonus
}

func champGuardPreventBonus(g *Game, player int) int {
	if ci := champImpl(g, player); ci != nil && ci.guardPreventBonus != nil {
		return ci.guardPreventBonus(g, player)
	}
	return 0
}

func champOnSacrifice(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.onSacrifice != nil {
		ci.onSacrifice(g, player)
	}
}

func champOnGuardResolved(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.onGuardResolved != nil {
		ci.onGuardResolved(g, player)
	}
}

func champResonanceCapDelta(g *Game, player int) int {
	if ci := champImpl(g, player); ci != nil && ci.resonanceCapDelta != nil {
		return ci.resonanceCapDelta(g, player)
	}
	return 0
}

func champFatigueStepDelta(g *Game, player int) int {
	if ci := champImpl(g, player); ci != nil && ci.fatigueStepDelta != nil {
		return ci.fatigueStepDelta(g, player)
	}
	return 0
}

func champHealBonus(g *Game, player int) int {
	if ci := champImpl(g, player); ci != nil && ci.healBonus != nil {
		return ci.healBonus(g, player)
	}
	return 0
}

func champFirstAssaultPierces(g *Game, player int) bool {
	if ci := champImpl(g, player); ci != nil && ci.firstAssaultPierces != nil {
		return ci.firstAssaultPierces(g, player)
	}
	return false
}

func champAfterFullPrevent(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.afterFullPrevent != nil {
		ci.afterFullPrevent(g, player)
	}
}

func champOnOwnSigil(g *Game, player int, sig Sigil, fromAssault bool, depth int) {
	if ci := champImpl(g, player); ci != nil && ci.onOwnSigil != nil {
		ci.onOwnSigil(g, player, sig, fromAssault, depth)
	}
}

func champOnRecover(g *Game, player int, inst string) {
	if ci := champImpl(g, player); ci != nil && ci.onRecover != nil {
		ci.onRecover(g, player, inst)
	}
}

func champOnCostDiscard(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.onCostDiscard != nil {
		ci.onCostDiscard(g, player)
	}
}

func champOnExileSelf(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.onExileSelf != nil {
		ci.onExileSelf(g, player)
	}
}

func champOnOppNightShift(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.onOppNightShift != nil {
		ci.onOppNightShift(g, player)
	}
}

func champCurseDurationBonus(g *Game, player int) int {
	if ci := champImpl(g, player); ci != nil && ci.curseDurationBonus != nil {
		return ci.curseDurationBonus(g, player)
	}
	return 0
}

func champOnExtraDraw(g *Game, player int, inst string) {
	if ci := champImpl(g, player); ci != nil && ci.onExtraDraw != nil {
		ci.onExtraDraw(g, player, inst)
	}
}

func champCopyExtraSigil(g *Game, player int) bool {
	if ci := champImpl(g, player); ci != nil && ci.copyExtraSigil != nil {
		return ci.copyExtraSigil(g, player)
	}
	return false
}

func champOn3Sigils(g *Game, player int) {
	if ci := champImpl(g, player); ci != nil && ci.on3Sigils != nil {
		ci.on3Sigils(g, player)
	}
}

// CanUltimate valida a ultimate sem executá-la (fonte única para Apply e bots).
func (g *Game) CanUltimate(player int) error {
	s := g.s
	p := s.Players[player]
	if p.UltimateUsed {
		return errCmd(ErrBadCommand, "ultimate já usada nesta partida")
	}
	ci := champImpl(g, player)
	if ci == nil || ci.ultimate == nil {
		if ci != nil && ci.canUltimate != nil {
			return ci.canUltimate(g, player)
		}
		return errCmd(ErrNotImplemented, "ultimate não implementada")
	}
	if s.Pending != nil || s.Guard != nil || s.RiteReact != nil {
		return errCmd(ErrWrongPhase, "há uma janela/decisão em aberto")
	}
	if (s.Phase != PhaseRite && s.Phase != PhaseConfront) || s.Active != player {
		return errCmd(ErrWrongPhase, "ultimate só na sua janela de Rito ou Confronto")
	}
	if ci.canUltimate != nil {
		return ci.canUltimate(g, player)
	}
	return nil
}

func (g *Game) applyUltimate(cmd Command) error {
	if err := g.CanUltimate(cmd.Player); err != nil {
		return err
	}
	p := g.s.Players[cmd.Player]
	p.UltimateUsed = true
	g.emit(Event{Kind: EvUltimate, P: cmd.Player, S: p.Champion})
	championImpls[p.Champion].ultimate(g, cmd.Player)
	return nil
}
