package engine

// Bot é a superfície comum dos agentes headless. A implementação pode ler o
// estado, mas só devolve Command; Apply continua sendo a autoridade.
type Bot interface {
	Next(*Game) (Command, bool)
}

// PlayerBot também permite ao simulador usar políticas diferentes em cada
// lado sem deixar um bot agir pelo adversário.
type PlayerBot interface {
	Bot
	NextFor(*Game, int) (Command, bool)
}

// RequiredPlayer devolve quem precisa fornecer a próxima intenção.
func RequiredPlayer(g *Game) (int, bool) {
	s := g.State()
	if s.Over {
		return 0, false
	}
	if s.Pending != nil {
		return s.Pending.Player, true
	}
	order := [2]int{s.Initiative, 1 - s.Initiative}
	switch s.Phase {
	case PhaseMulligan:
		for _, player := range order {
			if !s.Players[player].MulliganDone {
				return player, true
			}
		}
	case PhaseStance:
		for _, player := range order {
			if !s.Players[player].StanceCommitted {
				return player, true
			}
		}
	case PhaseRite, PhaseConfront:
		return expectedBotActor(s), true
	}
	return 0, false
}

func expectedBotActor(s *GameState) int {
	switch {
	case s.Pending != nil:
		return s.Pending.Player
	case s.RiteReact != nil:
		return 1 - s.RiteReact.Caster
	case s.Guard != nil:
		return s.Guard.Defender
	case s.Extra != nil:
		return s.Extra.Player
	default:
		return s.Active
	}
}

// legalPlays enumera as jogadas de carta legais do jogador na janela atual.
// É a mesma sequência de checagens de applyPlay; o teste de simulação garante
// que nenhuma jogada enumerada seja rejeitada pela engine.
func (g *Game) legalPlays(player int) []Command {
	s := g.s
	if s.Over || s.Pending != nil {
		return nil
	}
	var out []Command
	p := s.Players[player]

	add := func(inst string) {
		out = append(out, Command{Player: player, Kind: CmdKindPlay, Card: inst})
	}

	if s.RiteReact != nil {
		if player != 1-s.RiteReact.Caster {
			return nil
		}
		for _, id := range p.Hand {
			def := g.rs.Cards[s.Cards[id].Def]
			gi := g.rs.guard[def.ID]
			if def.Type == TypeGuarda && gi != nil && gi.counterRite &&
				g.canAfford(player, g.effectiveCost(player, def, id)) {
				add(id)
			}
		}
		return out
	}

	if s.Guard != nil {
		if player != s.Guard.Defender || s.Guard.GuardInst != "" || !s.Guard.Announced {
			return nil
		}
		for _, id := range p.Hand {
			def := g.rs.Cards[s.Cards[id].Def]
			gi := g.rs.guard[def.ID]
			if def.Type == TypeGuarda && gi != nil && !gi.counterRite &&
				g.canAfford(player, g.effectiveCost(player, def, id)) {
				add(id)
			}
		}
		return out
	}

	if s.Extra != nil {
		if player != s.Extra.Player {
			return nil
		}
		for _, id := range p.Hand {
			def := g.rs.Cards[s.Cards[id].Def]
			impl := g.rs.assault[def.ID]
			if def.Type != TypeAssalto || impl == nil ||
				s.Cards[id].LockedRound == s.Round {
				continue
			}
			if s.Extra.MaxCost > 0 && def.Cost > s.Extra.MaxCost {
				continue
			}
			if impl.preExile && len(p.Discard) == 0 {
				continue
			}
			if g.canAfford(player, g.effectiveCost(player, def, id)) {
				add(id)
			}
		}
		return out
	}

	if player != s.Active {
		return nil
	}
	switch s.Phase {
	case PhaseRite:
		for _, id := range p.Hand {
			def := g.rs.Cards[s.Cards[id].Def]
			switch def.Type {
			case TypeRito:
				impl := g.rs.rite[def.ID]
				if impl == nil || !g.canAfford(player, g.effectiveCost(player, def, id)) {
					continue
				}
				if impl.sacrifice > 0 {
					effSac := max(impl.sacrifice+g.permSacrificeDelta(player, false), 0)
					if effSac > 0 && p.Vitality <= effSac {
						continue
					}
				}
				if impl.targetsOpponent && s.Players[1-player].VeilRound == s.Round {
					continue
				}
				if impl.validate != nil && impl.validate(g, player) != nil {
					continue
				}
				add(id)
			case TypeReliquia:
				if g.rs.perm[def.ID] != nil && len(p.Relics) < 2 &&
					g.canAfford(player, g.effectiveCost(player, def, id)) {
					add(id)
				}
			case TypeManifestacao:
				if g.rs.perm[def.ID] != nil && len(p.Manifs) < 2 &&
					g.canAfford(player, g.effectiveCost(player, def, id)) {
					add(id)
				}
			}
		}
	case PhaseConfront:
		for _, id := range p.Hand {
			def := g.rs.Cards[s.Cards[id].Def]
			impl := g.rs.assault[def.ID]
			if def.Type != TypeAssalto || impl == nil ||
				s.Cards[id].LockedRound == s.Round {
				continue
			}
			if impl.preExile && len(p.Discard) == 0 {
				continue
			}
			if g.canAfford(player, g.effectiveCost(player, def, id)) {
				add(id)
			}
		}
	}
	return out
}

// RandomBot joga qualquer comando legal, com RNG próprio (independente do
// RNG da partida). Nível 1 do plano de balanceamento.
type RandomBot struct {
	RNG *RNG
}

// Next produz o próximo comando legal da partida, decidindo por qualquer um
// dos jogadores que precise agir. Retorna false se ninguém pode agir.
func (b *RandomBot) Next(g *Game) (Command, bool) {
	player, ok := RequiredPlayer(g)
	if !ok {
		return Command{}, false
	}
	return b.NextFor(g, player)
}

// NextFor produz a intenção do lado informado apenas quando ele é o ator.
func (b *RandomBot) NextFor(g *Game, player int) (Command, bool) {
	s := g.State()
	required, ok := RequiredPlayer(g)
	if !ok || required != player {
		return Command{}, false
	}

	if d := s.Pending; d != nil {
		return b.answerDecision(d), true
	}

	switch s.Phase {
	case PhaseMulligan:
		p := s.Players[player]
		cmd := Command{Player: player, Kind: CmdKindMulligan}
		if len(p.Hand) > 0 && b.RNG.Intn(2) == 0 {
			cmd.Cards = []string{p.Hand[b.RNG.Intn(len(p.Hand))]}
		}
		return cmd, true
	case PhaseStance:
		stances := []Stance{StancePredacao, StanceVigilia, StanceArcano}
		return Command{Player: player, Kind: CmdKindStance, Stance: stances[b.RNG.Intn(3)]}, true
	case PhaseRite, PhaseConfront:
		actor := player
		plays := g.legalPlays(actor)
		// Ativações e ultimate entram no sorteio junto com as cartas.
		if (s.Phase == PhaseRite || s.Phase == PhaseConfront) && s.Guard == nil && s.RiteReact == nil && actor == s.Active {
			p := s.Players[actor]
			if s.Phase == PhaseRite {
				for _, id := range append(append([]string{}, p.Relics...), p.Manifs...) {
					if g.CanActivate(actor, id) == nil {
						plays = append(plays, Command{Player: actor, Kind: CmdKindActivate, Card: id})
					}
				}
			}
			if g.CanUltimate(actor) == nil {
				plays = append(plays, Command{Player: actor, Kind: CmdKindUltimate})
			}
		}
		// Passar sempre é legal em janelas; dá um leve peso a jogar.
		if len(plays) > 0 && b.RNG.Intn(4) != 0 {
			return plays[b.RNG.Intn(len(plays))], true
		}
		return Command{Player: actor, Kind: CmdKindPass}, true
	}
	return Command{}, false
}

func (b *RandomBot) answerDecision(d *Decision) Command {
	cmd := Command{Player: d.Player, Kind: CmdKindChoose, DecisionID: d.ID}
	switch d.Kind {
	case DecDiscardN, DecRecoverPick, DecExilePick, DecReEmitSigils:
		opts := append([]string{}, d.Options...)
		b.RNG.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
		n := d.N
		if d.MinN > 0 && d.MinN < d.N {
			n = d.MinN + b.RNG.Intn(d.N-d.MinN+1)
		}
		cmd.Cards = opts[:n]
	case DecReorderTop:
		opts := append([]string{}, d.Options...)
		b.RNG.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
		cmd.Cards = opts
	case DecPickTop2, DecOppDiscardPick, DecDestroyRelicPick, DecTaxTypePick, DecPickSigil,
		DecRevealTax, DecLockAssault, DecDeclareType, DecDirection, DecFormulaChoice,
		DecMirrorRelic, DecCopyPlayed, DecOrenChoice, DecVorenChoice, DecEddaReturn:
		cmd.Cards = []string{d.Options[b.RNG.Intn(len(d.Options))]}
	case DecExileSelfHeal, DecMillTop, DecSwapEclipse, DecScryBottom:
		cmd.Cards = []string{[]string{"yes", "no"}[b.RNG.Intn(2)]}
	}
	return cmd
}

// Replay reconstrói uma partida a partir de config + log de comandos.
func Replay(cfg Config, commands []Command) (*Game, error) {
	g, err := NewGame(cfg)
	if err != nil {
		return nil, err
	}
	for i, cmd := range commands {
		if _, err := g.Apply(cmd); err != nil {
			return nil, errCmd(ErrBadCommand, "replay divergiu no comando %d: %v", i, err)
		}
	}
	return g, nil
}
