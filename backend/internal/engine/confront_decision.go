package engine

// Decisões no Modo Confronto (ADR-058).
//
// A máquina de decisão do motor legado é genérica — fila, validação e eventos
// já existiam —, mas o Confronto tinha executor próprio e recusava qualquer
// comando que não fosse jogar ou passar. Isso mantinha fora do modo toda carta
// que pede uma escolha ao jogador, justamente o eixo mais fino do formato.
//
// Este arquivo abre esse caminho sem misturar os dois fluxos: o Confronto pede
// a decisão, valida a resposta e roda a continuação com o seu próprio executor.
// A decisão pendente trava a mesa: nenhuma outra ação é aceita até ela ser
// respondida, porque a alternativa seria estado ambíguo em replay.

// confrontRequestDiscard abre uma escolha de descarte. Mão vazia não vira
// decisão pendente — a continuação roda direto, senão a partida travaria
// esperando uma resposta impossível.
func (g *Game) confrontRequestDiscard(player, n int, then []Op, source string) {
	hand := g.s.Players[player].Hand
	if len(hand) == 0 || n <= 0 {
		g.confrontRunOps(then, &opCtx{player: player, source: source})
		return
	}
	g.requestDecision(&Decision{
		Player:  player,
		Kind:    DecDiscardN,
		Options: append([]string{}, hand...),
		N:       min(n, len(hand)),
		Source:  source,
		Then:    then,
	})
}

// applyConfrontChoose resolve a decisão pendente do Modo Confronto.
func (g *Game) applyConfrontChoose(cmd Command) error {
	s := g.s
	d := s.Pending
	if d == nil {
		return errCmd(ErrBadCommand, "não há decisão pendente")
	}
	if cmd.Player != d.Player {
		return errCmd(ErrWrongPlayer, "a decisão pendente é do jogador %d", d.Player)
	}
	if cmd.DecisionID != d.ID {
		return errCmd(ErrBadCommand, "decisão %d não corresponde à pendente %d", cmd.DecisionID, d.ID)
	}
	if d.Kind != DecDiscardN {
		return errCmd(ErrNotImplemented, "decisão %q ainda não existe no Modo Confronto", d.Kind)
	}
	if len(cmd.Cards) != d.N {
		return errCmd(ErrBadCommand, "escolha exatamente %d carta(s)", d.N)
	}
	// A escolha precisa ser de cartas realmente oferecidas, e sem repetição:
	// aceitar a mesma carta duas vezes descartaria uma carta só e deixaria o
	// estado divergir do log.
	seen := map[string]bool{}
	for _, id := range cmd.Cards {
		if seen[id] {
			return errCmd(ErrBadCommand, "carta repetida na escolha: %s", id)
		}
		seen[id] = true
		if !containsID(d.Options, id) {
			return errCmd(ErrInvalidCard, "carta %q não está entre as opções", id)
		}
		if inst := s.Cards[id]; inst == nil || inst.Owner != d.Player || inst.Zone != ZoneHand {
			return errCmd(ErrInvalidCard, "carta %q não está mais na mão", id)
		}
	}

	pending := d
	s.Pending = nil
	for _, id := range cmd.Cards {
		g.confrontDiscardFromHand(pending.Player, id)
	}
	g.emit(Event{Kind: EvDecisionResolved, P: pending.Player, N: pending.ID,
		S: string(pending.Kind), Def: pending.Source})
	g.confrontRunOps(pending.Then, &opCtx{player: pending.Player, source: pending.Source})
	g.promoteConfrontDecision()
	if s.Pending == nil && len(s.DecQueue) == 0 && s.PendingConfrontRite != nil {
		g.finalizePendingConfrontRite()
	} else {
		// A continuação pode causar Fadiga ou dano letal mesmo sem pertencer a
		// um Rito suspenso; a vitória precisa ser observada antes da próxima
		// ação da mesa.
		g.checkWin()
		if s.Over {
			g.finalizePendingConfrontRite()
		}
	}
	return nil
}

// finalizeConfrontRite encerra a carta e só então avança o turno. Separar a
// finalização do executor impede que uma escolha aberta entregue a vez ao
// rival antes de o efeito terminar.
func (g *Game) finalizeConfrontRite(player int, inst string) {
	g.discardClash(inst)
	g.checkWin()
	if !g.s.Over {
		g.finishConfrontTurn(player)
	}
}

// finalizePendingConfrontRite retoma o Rito quando a cadeia de decisões
// termina. Se a continuação já encerrou a partida, ainda limpa a zona da carta
// sem iniciar um novo turno.
func (g *Game) finalizePendingConfrontRite() {
	pending := g.s.PendingConfrontRite
	if pending == nil {
		return
	}
	if !g.s.Over && (g.s.Pending != nil || len(g.s.DecQueue) > 0) {
		return
	}
	g.s.PendingConfrontRite = nil
	g.finalizeConfrontRite(pending.Player, pending.Inst)
}

// promoteConfrontDecision puxa a próxima decisão da fila, se houver.
func (g *Game) promoteConfrontDecision() {
	s := g.s
	if s.Pending != nil || len(s.DecQueue) == 0 {
		return
	}
	next := s.DecQueue[0]
	s.DecQueue = s.DecQueue[1:]
	s.Pending = next
	g.emit(Event{Kind: EvDecisionRequested, P: next.Player, N: next.N,
		S: string(next.Kind), Card: next.Card})
}

// confrontDiscardFromHand move uma carta da mão para o descarte.
func (g *Game) confrontDiscardFromHand(player int, id string) {
	inst := g.s.Cards[id]
	if inst == nil || inst.Zone != ZoneHand {
		return
	}
	p := g.s.Players[player]
	p.Hand, _ = removeID(p.Hand, id)
	inst.Zone = ZoneDiscard
	p.Discard = append(p.Discard, id)
	g.emit(Event{Kind: EvCardDiscarded, P: player, Card: id, Def: inst.Def})
}
