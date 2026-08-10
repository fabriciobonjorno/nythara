package battle

import (
	"context"
	"errors"
	"fmt"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// Modo treino: o assento 1 é ocupado pelo bot heurístico. A partida usa o
// mesmo pipeline authoritative (persistência, snapshots, replay, reconexão);
// os comandos do bot entram com origem "bot" e sem client_sequence. Partidas
// de treino não passam pelo filtro de bans (não são competitivas) e ficam
// fora da telemetria de win rate PvP.

const (
	ModePvP      = "pvp"
	ModePractice = "practice"

	// practicePumpCap limita comandos do bot por rajada; o simulador prova
	// que partidas terminam muito antes disso — é um corta-fogo, não um teto.
	practicePumpCap = 700
)

// StartPractice cria uma partida de treino contra o bot. O deck do bot vem da
// camada de aplicação (decks oficiais semeados da conta reservada).
func (m *Manager) StartPractice(ctx context.Context, principal domain.Principal,
	deck, botDeck domain.Deck) (QueueResult, error) {
	if deck.UserID != principal.UserID || deck.RulesetVersion != m.ActiveRuleset() {
		return QueueResult{}, domain.ErrForbidden
	}
	if err := validateStoredDeck(deck); err != nil {
		return QueueResult{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if botDeck.UserID != domain.BotUserID || botDeck.RulesetVersion != deck.RulesetVersion {
		return QueueResult{}, fmt.Errorf("%w: deck do bot inválido", domain.ErrInvalid)
	}
	if err := validateStoredDeck(botDeck); err != nil {
		return QueueResult{}, fmt.Errorf("%w: deck do bot: %v", domain.ErrInvalid, err)
	}
	if existing, slot, err := m.store.ActiveMatchForUser(ctx, principal.UserID); err == nil {
		return QueueResult{Status: "matched", MatchID: existing.ID, Slot: &slot}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return QueueResult{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queued[principal.UserID] {
		return QueueResult{}, ErrAlreadyQueued
	}
	if existing, slot, err := m.store.ActiveMatchForUser(ctx, principal.UserID); err == nil {
		return QueueResult{Status: "matched", MatchID: existing.ID, Slot: &slot}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return QueueResult{}, err
	}

	human := QueuedPlayer{Principal: principal, Deck: deck}
	bot := QueuedPlayer{Principal: domain.Principal{UserID: domain.BotUserID,
		Role: domain.RolePlayer, DisplayName: "Treinador do Véu"}, Deck: botDeck}
	match, err := newMatch(human, bot, m.now().UTC())
	if err != nil {
		return QueueResult{}, err
	}
	match.Mode = ModePractice
	if err := m.store.CreateMatch(ctx, match); err != nil {
		return QueueResult{}, err
	}
	// O bot está sempre pronto; o jogo começa quando o humano confirmar.
	if err := m.store.MarkReady(ctx, match.ID, 1); err != nil {
		return QueueResult{}, err
	}
	match.Players[1].Ready = true

	r, err := newRoom(m, LoadedMatch{Match: match})
	if err != nil {
		return QueueResult{}, err
	}
	m.rooms[match.ID] = r
	r.armReadyTimer()
	go r.loop()
	zero := 0
	return QueueResult{Status: "matched", MatchID: match.ID, Slot: &zero}, nil
}

// botFor devolve o cérebro do bot da sala (determinístico pela seed da
// partida — replays de treino também reproduzem, pois os comandos do bot são
// persistidos como qualquer outro).
func (r *room) botFor() *engine.HeuristicBot {
	if r.bot == nil {
		r.bot = &engine.HeuristicBot{RNG: engine.NewRNG(r.match.Config.Seed ^ 0xB07B07B07)}
	}
	return r.bot
}

// pumpBot faz o bot agir até a vez voltar ao humano (ou a partida acabar).
// Roda sempre dentro do goroutine da sala — sem concorrência sobre o estado.
func (r *room) pumpBot() {
	if r.match.Mode != ModePractice || r.match.Status != StatusActive || r.game == nil {
		return
	}
	const botSlot = 1
	bot := r.botFor()
	for step := 0; step < practicePumpCap; step++ {
		if r.match.Status != StatusActive || r.game.State().Over {
			return
		}
		if expectedActor(r.game.State()) != botSlot {
			r.armActionTimer()
			return
		}
		command, ok := bot.NextFor(r.game, botSlot)
		if !ok {
			return
		}
		if err := r.applyServerCommand(botSlot, "bot", command); err != nil {
			// Persistência falhou: mantém o estado consistente e tenta na
			// próxima interação; o timer protege o humano.
			r.armActionTimer()
			return
		}
	}
	// Corta-fogo: jamais deixar a sala presa num bot confuso.
	_ = r.applyServerCommand(botSlot, "bot",
		engine.Command{Player: 1, Kind: engine.CmdKindConcede, Reason: "bot_guard"})
}

// applyServerCommand aplica e persiste um comando de origem do servidor
// (bot), com o mesmo contrato transacional do caminho do cliente: falha de
// persistência reverte a engine por replay.
func (r *room) applyServerCommand(slot int, origin string, command engine.Command) error {
	previousCommands := append([]engine.Command{}, r.game.CommandLog...)
	events, err := r.game.Apply(command)
	if err != nil {
		return err
	}
	nextIndex := r.commandIndex + 1
	step := Step{MatchID: r.match.ID, CommandIndex: nextIndex, PlayerSlot: slot,
		Origin: origin, Command: command, Events: events}
	if nextIndex%snapshotInterval == 0 || r.game.State().Over {
		data, snapshotErr := r.game.SnapshotJSON()
		if snapshotErr != nil {
			r.game, _ = engine.Replay(r.match.Config, previousCommands)
			return snapshotErr
		}
		step.Snapshot = &Snapshot{CommandIndex: nextIndex, EventSeq: len(r.game.Log) - 1, Data: data}
	}
	if r.game.State().Over {
		winner := r.game.State().Winner
		step.Finished, step.Winner, step.EndReason = true, &winner, r.game.State().EndReason
	}
	ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
	err = r.manager.store.PersistStep(ctx, step)
	cancel()
	if err != nil {
		r.game, _ = engine.Replay(r.match.Config, previousCommands)
		return err
	}
	r.commandIndex = nextIndex
	if step.Finished {
		r.match.Status, r.match.Winner, r.match.EndReason = StatusFinished, step.Winner, step.EndReason
		r.stopTimer()
		r.notifyFinished()
	}
	r.broadcastEvents(events, "", 0, false)
	return nil
}
