package battle

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

const snapshotInterval int64 = 10
const persistenceTimeout = 5 * time.Second

type Manager struct {
	store         Store
	progress      atomic.Value // ProgressRecorder
	activeRuleset atomic.Value // string: versão competitiva corrente
	mu            sync.Mutex
	queue         []QueuedPlayer
	queued        map[string]bool
	rooms         map[string]*room
	tickets       map[string]Ticket
	readyTimeout  time.Duration
	actionTimeout time.Duration
	now           func() time.Time
}

func NewManager(store Store) *Manager {
	m := &Manager{store: store, queued: map[string]bool{}, rooms: map[string]*room{},
		tickets: map[string]Ticket{}, readyTimeout: 30 * time.Second, actionTimeout: 45 * time.Second,
		now: time.Now}
	m.activeRuleset.Store(engine.RulesetVersion)
	return m
}

// SetProgressRecorder instala o gravador de progressão (rituais, maestria,
// rating) chamado a cada partida encerrada.
func (m *Manager) SetProgressRecorder(recorder ProgressRecorder) {
	if recorder != nil {
		m.progress.Store(recorder)
	}
}

func (m *Manager) progressRecorder() ProgressRecorder {
	if value := m.progress.Load(); value != nil {
		return value.(ProgressRecorder)
	}
	return nil
}

// SetActiveRuleset aponta o matchmaking para outra versão publicada
// (ativação/rollback via admin). Partidas em andamento não são afetadas.
func (m *Manager) SetActiveRuleset(version string) {
	if version != "" {
		m.activeRuleset.Store(version)
	}
}

// ActiveRuleset devolve a versão competitiva corrente.
func (m *Manager) ActiveRuleset() string {
	return m.activeRuleset.Load().(string)
}

func (m *Manager) Queue(ctx context.Context, principal domain.Principal, deck domain.Deck) (QueueResult, error) {
	if deck.UserID != principal.UserID || deck.RulesetVersion != m.ActiveRuleset() {
		return QueueResult{}, domain.ErrForbidden
	}
	if err := validateStoredDeck(deck); err != nil {
		return QueueResult{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	// Bans emergenciais (Fase 7): deck com carta desativada não entra na fila.
	bans, err := m.store.ActiveBans(ctx)
	if err != nil {
		return QueueResult{}, err
	}
	banned := make(map[string]string, len(bans))
	for _, ban := range bans {
		banned[ban.CardID] = ban.Reason
	}
	for _, card := range deck.Cards {
		if reason, ok := banned[card.CardID]; ok {
			return QueueResult{}, fmt.Errorf("%w: %s está temporariamente desativada do competitivo (%s)",
				domain.ErrInvalid, card.CardID, reason)
		}
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
	// Repete sob o lock local para fechar a janela entre a consulta acima e
	// outra requisição concorrente do mesmo usuário formar uma partida.
	if existing, slot, err := m.store.ActiveMatchForUser(ctx, principal.UserID); err == nil {
		return QueueResult{Status: "matched", MatchID: existing.ID, Slot: &slot}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return QueueResult{}, err
	}
	matchAt := -1
	for i, waiting := range m.queue {
		if waiting.Principal.UserID != principal.UserID && waiting.Deck.RulesetVersion == deck.RulesetVersion {
			matchAt = i
			break
		}
	}
	if matchAt < 0 {
		m.queue = append(m.queue, QueuedPlayer{Principal: principal, Deck: deck})
		m.queued[principal.UserID] = true
		return QueueResult{Status: "queued"}, nil
	}
	opponent := m.queue[matchAt]
	m.queue = append(m.queue[:matchAt], m.queue[matchAt+1:]...)
	delete(m.queued, opponent.Principal.UserID)
	match, err := newMatch(opponent, QueuedPlayer{Principal: principal, Deck: deck}, m.now().UTC())
	if err != nil {
		return QueueResult{}, err
	}
	if err := m.store.CreateMatch(ctx, match); err != nil {
		m.queue = append([]QueuedPlayer{opponent}, m.queue...)
		m.queued[opponent.Principal.UserID] = true
		return QueueResult{}, err
	}
	r, err := newRoom(m, LoadedMatch{Match: match})
	if err != nil {
		return QueueResult{}, err
	}
	m.rooms[match.ID] = r
	r.armReadyTimer()
	go r.loop()
	one := 1
	return QueueResult{Status: "matched", MatchID: match.ID, Slot: &one}, nil
}

func (m *Manager) CancelQueue(userID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, queued := range m.queue {
		if queued.Principal.UserID == userID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			delete(m.queued, userID)
			return true
		}
	}
	return false
}

func (m *Manager) QueueStatus(ctx context.Context, userID string) (QueueResult, error) {
	m.mu.Lock()
	if m.queued[userID] {
		m.mu.Unlock()
		return QueueResult{Status: "queued"}, nil
	}
	rooms := make([]*room, 0, len(m.rooms))
	for _, r := range m.rooms {
		rooms = append(rooms, r)
	}
	m.mu.Unlock()
	for _, r := range rooms {
		response := make(chan QueueResult, 1)
		select {
		case r.requests <- statusRequest{userID: userID, response: response}:
		case <-ctx.Done():
			return QueueResult{}, ctx.Err()
		}
		select {
		case result := <-response:
			if result.Status == "matched" {
				return result, nil
			}
		case <-ctx.Done():
			return QueueResult{}, ctx.Err()
		}
	}
	match, slot, err := m.store.ActiveMatchForUser(ctx, userID)
	if err == nil {
		return QueueResult{Status: "matched", MatchID: match.ID, Slot: &slot}, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return QueueResult{}, err
	}
	return QueueResult{Status: "idle"}, nil
}

func (m *Manager) IssueTicket(ctx context.Context, principal domain.Principal, matchID string,
	mode TicketMode) (Ticket, error) {
	r, err := m.getRoom(ctx, matchID)
	if err != nil {
		return Ticket{}, err
	}
	slotResponse := make(chan int, 1)
	select {
	case r.requests <- participantRequest{userID: principal.UserID, response: slotResponse}:
	case <-ctx.Done():
		return Ticket{}, ctx.Err()
	}
	var slot int
	select {
	case slot = <-slotResponse:
	case <-ctx.Done():
		return Ticket{}, ctx.Err()
	}
	if mode == TicketPlayer && slot < 0 {
		return Ticket{}, ErrNotParticipant
	}
	if mode == TicketSpectator && principal.Role != domain.RoleAdmin {
		return Ticket{}, domain.ErrForbidden
	}
	plain, _, err := security.NewToken()
	if err != nil {
		return Ticket{}, err
	}
	ticket := Ticket{Token: plain, MatchID: matchID, UserID: principal.UserID, Mode: mode,
		Slot: slot, ExpiresAt: m.now().UTC().Add(time.Minute)}
	m.mu.Lock()
	for token, existing := range m.tickets {
		if !existing.ExpiresAt.After(m.now().UTC()) {
			delete(m.tickets, token)
		}
	}
	m.tickets[plain] = ticket
	m.mu.Unlock()
	return ticket, nil
}

func (m *Manager) Connect(ctx context.Context, token, matchID string, afterEvent int) (*Connection, error) {
	m.mu.Lock()
	ticket, ok := m.tickets[token]
	delete(m.tickets, token)
	m.mu.Unlock()
	if !ok || ticket.MatchID != matchID || !ticket.ExpiresAt.After(m.now().UTC()) {
		return nil, ErrTicketInvalid
	}
	r, err := m.getRoom(ctx, matchID)
	if err != nil {
		return nil, err
	}
	id, err := security.NewID()
	if err != nil {
		return nil, err
	}
	sub := &subscriber{id: id, userID: ticket.UserID, slot: ticket.Slot, mode: ticket.Mode,
		out: make(chan ServerMessage, 64)}
	response := make(chan error, 1)
	select {
	case r.requests <- subscribeRequest{sub: sub, afterEvent: afterEvent, response: response}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case err := <-response:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &Connection{room: r, sub: sub, Messages: sub.out}, nil
}

func (m *Manager) getRoom(ctx context.Context, matchID string) (*room, error) {
	m.mu.Lock()
	if existing := m.rooms[matchID]; existing != nil {
		m.mu.Unlock()
		return existing, nil
	}
	m.mu.Unlock()
	loaded, err := m.store.LoadMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	r, err := newRoom(m, loaded)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if existing := m.rooms[matchID]; existing != nil {
		m.mu.Unlock()
		return existing, nil
	}
	m.rooms[matchID] = r
	m.mu.Unlock()
	if r.match.Status == StatusWaitingReady {
		r.armReadyTimer()
	} else if r.match.Status == StatusActive {
		r.armActionTimer()
	}
	go r.loop()
	if r.match.Mode == ModePractice && r.match.Status == StatusActive {
		// Sala restaurada mid-partida: o bot pode estar devendo a jogada.
		r.requests <- pumpRequest{}
	}
	return r, nil
}

type Connection struct {
	room     *room
	sub      *subscriber
	Messages <-chan ServerMessage
	once     sync.Once
}

func (c *Connection) Ready(ctx context.Context) error {
	response := make(chan error, 1)
	select {
	case c.room.requests <- readyRequest{subID: c.sub.id, response: response}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-response:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Connection) Submit(ctx context.Context, sequence int64, intent Intent) error {
	response := make(chan error, 1)
	select {
	case c.room.requests <- commandRequest{subID: c.sub.id, sequence: sequence, intent: intent, response: response}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-response:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Connection) Close() {
	c.once.Do(func() { c.room.requests <- unsubscribeRequest{subID: c.sub.id} })
}

type subscriber struct {
	id     string
	userID string
	slot   int
	mode   TicketMode
	out    chan ServerMessage
}

type cachedCommand struct {
	payload []byte
	events  []engine.Event
}

type room struct {
	manager      *Manager
	match        Match
	game         *engine.Game
	commandIndex int64
	requests     chan any
	subscribers  map[string]*subscriber
	cache        [2]map[int64]cachedCommand
	timer        *time.Timer
	timerGen     int64
	deadline     *time.Time
	bot          *engine.HeuristicBot
}

type subscribeRequest struct {
	sub        *subscriber
	afterEvent int
	response   chan error
}
type unsubscribeRequest struct{ subID string }
type readyRequest struct {
	subID    string
	response chan error
}
type commandRequest struct {
	subID    string
	sequence int64
	intent   Intent
	response chan error
}
type timeoutRequest struct{ generation int64 }

type pumpRequest struct{}
type statusRequest struct {
	userID   string
	response chan QueueResult
}
type participantRequest struct {
	userID   string
	response chan int
}

func newRoom(manager *Manager, loaded LoadedMatch) (*room, error) {
	r := &room{manager: manager, match: loaded.Match, requests: make(chan any, 128),
		subscribers: map[string]*subscriber{}, cache: [2]map[int64]cachedCommand{{}, {}}}
	if loaded.Match.Status != StatusActive && loaded.Match.Status != StatusFinished {
		return r, nil
	}
	if loaded.Snapshot == nil {
		return nil, fmt.Errorf("partida %s ativa sem snapshot", loaded.Match.ID)
	}
	var prefixEvents []engine.Event
	for _, event := range loaded.Events {
		if event.Seq <= loaded.Snapshot.EventSeq {
			prefixEvents = append(prefixEvents, event)
		}
	}
	var prefixCommands []engine.Command
	for _, command := range loaded.Commands {
		if command.Index > 0 && command.Index <= loaded.Snapshot.CommandIndex {
			prefixCommands = append(prefixCommands, command.Command)
		}
	}
	game, err := engine.RestoreSnapshot(loaded.Match.Config, loaded.Snapshot.Data, prefixEvents, prefixCommands)
	if err != nil {
		return nil, err
	}
	for _, stored := range loaded.Commands {
		if stored.Index <= loaded.Snapshot.CommandIndex || stored.Index == 0 {
			continue
		}
		generated, err := game.Apply(stored.Command)
		if err != nil {
			return nil, fmt.Errorf("catch-up comando %d: %w", stored.Index, err)
		}
		persisted := eventsForCommand(loaded.Events, loaded.Commands, stored.Index)
		if !sameEvents(generated, persisted) {
			return nil, fmt.Errorf("catch-up divergiu no comando %d", stored.Index)
		}
	}
	r.game = game
	if len(loaded.Commands) > 0 {
		r.commandIndex = loaded.Commands[len(loaded.Commands)-1].Index
	}
	for _, stored := range loaded.Commands {
		if stored.ClientSequence == nil || stored.PlayerSlot < 0 {
			continue
		}
		payload, _ := json.Marshal(stored.Command)
		r.cache[stored.PlayerSlot][*stored.ClientSequence] = cachedCommand{
			payload: payload, events: eventsForCommand(loaded.Events, loaded.Commands, stored.Index),
		}
	}
	return r, nil
}

func (r *room) loop() {
	for request := range r.requests {
		switch value := request.(type) {
		case subscribeRequest:
			r.subscribers[value.sub.id] = value.sub
			r.sendSync(value.sub, value.afterEvent)
			value.response <- nil
		case unsubscribeRequest:
			if sub := r.subscribers[value.subID]; sub != nil {
				delete(r.subscribers, value.subID)
				close(sub.out)
			}
		case readyRequest:
			value.response <- r.handleReady(value.subID)
		case commandRequest:
			value.response <- r.handleCommand(value)
		case timeoutRequest:
			if value.generation == r.timerGen {
				r.handleTimeout()
			}
		case statusRequest:
			result := QueueResult{Status: "idle"}
			if r.match.Status != StatusFinished && r.match.Status != StatusCancelled {
				if slot := r.slotFor(value.userID); slot >= 0 {
					result = QueueResult{Status: "matched", MatchID: r.match.ID, Slot: &slot}
				}
			}
			value.response <- result
		case participantRequest:
			value.response <- r.slotFor(value.userID)
		case pumpRequest:
			r.pumpBot()
		}
	}
}

func (r *room) handleReady(subID string) error {
	sub := r.subscribers[subID]
	if sub == nil || sub.mode != TicketPlayer || sub.slot < 0 {
		return ErrSpectatorWrite
	}
	if r.match.Status != StatusWaitingReady {
		return nil
	}
	wasReady := r.match.Players[sub.slot].Ready
	if !wasReady {
		r.match.Players[sub.slot].Ready = true
	}
	both := r.match.Players[0].Ready && r.match.Players[1].Ready
	if !wasReady {
		ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
		err := r.manager.store.MarkReady(ctx, r.match.ID, sub.slot)
		cancel()
		if err != nil {
			r.match.Players[sub.slot].Ready = false
			return err
		}
	}
	if both && r.game == nil {
		game, err := engine.NewGame(r.match.Config)
		if err != nil {
			return err
		}
		snapshotData, err := game.SnapshotJSON()
		if err != nil {
			return err
		}
		snapshot := Snapshot{CommandIndex: 0, EventSeq: len(game.Log) - 1, Data: snapshotData}
		ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
		err = r.manager.store.StartMatch(ctx, r.match.ID, game.Log, snapshot)
		cancel()
		if err != nil {
			return err
		}
		r.game = game
		r.match.Status = StatusActive
		now := r.manager.now().UTC()
		r.match.StartedAt = &now
		r.armActionTimer()
	}
	r.broadcastState("ready")
	r.pumpBot()
	return nil
}

func (r *room) handleCommand(request commandRequest) error {
	sub := r.subscribers[request.subID]
	if sub == nil || sub.mode != TicketPlayer || sub.slot < 0 {
		return ErrSpectatorWrite
	}
	if r.match.Status != StatusActive || r.game == nil {
		return ErrNotReady
	}
	if request.sequence <= 0 {
		return ErrSequenceGap
	}
	if err := request.intent.Validate(); err != nil {
		return err
	}
	command := request.intent.Command(sub.slot)
	payload, _ := json.Marshal(command)
	last := r.match.Players[sub.slot].LastSequence
	if request.sequence <= last {
		cached, ok := r.cache[sub.slot][request.sequence]
		if !ok || !bytes.Equal(cached.payload, payload) {
			return ErrSequenceReuse
		}
		r.send(sub, ServerMessage{Type: "events", MatchID: r.match.ID, Status: r.match.Status,
			Events: RedactEvents(cached.events, sub.slot, false), ClientSequence: request.sequence, Duplicate: true})
		return nil
	}
	if request.sequence != last+1 {
		return ErrSequenceGap
	}
	previousCommands := append([]engine.Command{}, r.game.CommandLog...)
	events, err := r.game.Apply(command)
	if err != nil {
		return err
	}
	nextIndex := r.commandIndex + 1
	step := Step{MatchID: r.match.ID, CommandIndex: nextIndex, PlayerSlot: sub.slot,
		ClientSequence: &request.sequence, Origin: "client", Command: command, Events: events}
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
	r.match.Players[sub.slot].LastSequence = request.sequence
	r.cache[sub.slot][request.sequence] = cachedCommand{payload: payload, events: append([]engine.Event{}, events...)}
	if step.Finished {
		r.match.Status, r.match.Winner, r.match.EndReason = StatusFinished, step.Winner, step.EndReason
		r.stopTimer()
		r.notifyFinished()
	} else {
		r.armActionTimer()
	}
	r.broadcastEvents(events, sub.id, request.sequence, false)
	r.pumpBot()
	return nil
}

func (r *room) handleTimeout() {
	if r.match.Status == StatusWaitingReady {
		ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
		err := r.manager.store.CancelMatch(ctx, r.match.ID, "ready_timeout")
		cancel()
		if err != nil {
			r.armReadyTimer()
			return
		}
		r.match.Status, r.match.EndReason = StatusCancelled, "ready_timeout"
		r.broadcastState("match_cancelled")
		r.stopTimer()
		return
	}
	if r.match.Status != StatusActive || r.game == nil {
		return
	}
	actor := expectedActor(r.game.State())
	previousCommands := append([]engine.Command{}, r.game.CommandLog...)
	command := engine.Command{Player: actor, Kind: engine.CmdKindConcede, Reason: "timeout"}
	events, err := r.game.Apply(command)
	if err != nil {
		return
	}
	nextIndex := r.commandIndex + 1
	winner := r.game.State().Winner
	data, err := r.game.SnapshotJSON()
	if err != nil {
		r.game, _ = engine.Replay(r.match.Config, previousCommands)
		r.armActionTimer()
		return
	}
	step := Step{MatchID: r.match.ID, CommandIndex: nextIndex, PlayerSlot: actor,
		Origin: "timeout", Command: command, Events: events, Finished: true, Winner: &winner,
		EndReason: "timeout", Snapshot: &Snapshot{CommandIndex: nextIndex,
			EventSeq: len(r.game.Log) - 1, Data: data}}
	ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
	err = r.manager.store.PersistStep(ctx, step)
	cancel()
	if err != nil {
		r.game, _ = engine.Replay(r.match.Config, previousCommands)
		r.armActionTimer()
		return
	}
	r.commandIndex = nextIndex
	r.match.Status, r.match.Winner, r.match.EndReason = StatusFinished, &winner, "timeout"
	r.broadcastEvents(events, "", 0, false)
	r.stopTimer()
	r.notifyFinished()
}

// notifyFinished entrega a partida encerrada ao gravador de progresso, com
// cópias feitas dentro do goroutine da sala (sem corrida) e execução fora
// dele (sem atrasar a sala).
func (r *room) notifyFinished() {
	recorder := r.manager.progressRecorder()
	if recorder == nil || r.game == nil {
		return
	}
	finished := FinishedMatch{
		MatchID:        r.match.ID,
		Mode:           r.match.Mode,
		RulesetVersion: r.match.Config.RulesetVersion,
		Winner:         r.match.Winner,
	}
	for slot := 0; slot < 2; slot++ {
		finished.Players[slot] = FinishedParticipant{
			UserID:     r.match.Players[slot].UserID,
			ChampionID: r.match.Config.Players[slot].ChampionID,
		}
	}
	events := append([]engine.Event{}, r.game.Log...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
		defer cancel()
		recorder(ctx, finished, events)
	}()
}

func (r *room) sendSync(sub *subscriber, after int) {
	message := ServerMessage{Type: "sync", MatchID: r.match.ID, Status: r.match.Status,
		Ready: [2]bool{r.match.Players[0].Ready, r.match.Players[1].Ready}, Deadline: r.deadline}
	if sub.mode == TicketPlayer {
		slot := sub.slot
		message.Slot = &slot
		message.ClientSequence = r.match.Players[slot].LastSequence
	}
	if r.game != nil {
		view := ViewFor(r.game.State(), sub.slot, sub.mode == TicketSpectator)
		message.State = &view
		if after >= -1 && after < len(r.game.Log)-1 {
			message.Events = RedactEvents(r.game.Log[after+1:], sub.slot, sub.mode == TicketSpectator)
		}
	}
	r.send(sub, message)
}

func (r *room) broadcastState(kind string) {
	for _, sub := range r.subscribers {
		message := ServerMessage{Type: kind, MatchID: r.match.ID, Status: r.match.Status,
			Ready: [2]bool{r.match.Players[0].Ready, r.match.Players[1].Ready}, Deadline: r.deadline}
		if r.game != nil {
			view := ViewFor(r.game.State(), sub.slot, sub.mode == TicketSpectator)
			message.State = &view
		}
		r.send(sub, message)
	}
}

func (r *room) broadcastEvents(events []engine.Event, source string, sequence int64, duplicate bool) {
	for _, sub := range r.subscribers {
		message := ServerMessage{Type: "events", MatchID: r.match.ID, Status: r.match.Status,
			Events: RedactEvents(events, sub.slot, sub.mode == TicketSpectator), Deadline: r.deadline,
			Duplicate: duplicate}
		if sub.id == source {
			message.ClientSequence = sequence
		}
		view := ViewFor(r.game.State(), sub.slot, sub.mode == TicketSpectator)
		message.State = &view
		r.send(sub, message)
	}
}

func (r *room) send(sub *subscriber, message ServerMessage) {
	select {
	case sub.out <- message:
	default:
		delete(r.subscribers, sub.id)
		close(sub.out)
	}
}

func (r *room) slotFor(userID string) int {
	for _, player := range r.match.Players {
		if player.UserID == userID {
			return player.Slot
		}
	}
	return -1
}

func (r *room) armReadyTimer() {
	r.armTimer(r.manager.readyTimeout)
}

func (r *room) armActionTimer() {
	r.armTimer(r.manager.actionTimeout)
}

func (r *room) armTimer(duration time.Duration) {
	r.timerGen++
	generation := r.timerGen
	if r.timer != nil {
		r.timer.Stop()
	}
	deadline := r.manager.now().UTC().Add(duration)
	r.deadline = &deadline
	r.timer = time.AfterFunc(duration, func() { r.requests <- timeoutRequest{generation: generation} })
}

func (r *room) stopTimer() {
	if r.timer != nil {
		r.timer.Stop()
	}
	r.deadline = nil
}

func expectedActor(state *engine.GameState) int {
	if state.Pending != nil {
		return state.Pending.Player
	}
	order := [2]int{state.Initiative, 1 - state.Initiative}
	if state.Phase == engine.PhaseMulligan {
		for _, slot := range order {
			if !state.Players[slot].MulliganDone {
				return slot
			}
		}
	}
	if state.Phase == engine.PhaseStance {
		for _, slot := range order {
			if !state.Players[slot].StanceCommitted {
				return slot
			}
		}
	}
	if state.RiteReact != nil {
		return 1 - state.RiteReact.Caster
	}
	if state.Guard != nil {
		return state.Guard.Defender
	}
	if state.Extra != nil {
		return state.Extra.Player
	}
	return state.Active
}

func newMatch(first, second QueuedPlayer, now time.Time) (Match, error) {
	id, err := security.NewID()
	if err != nil {
		return Match{}, err
	}
	seed, err := randomSeed()
	if err != nil {
		return Match{}, err
	}
	config := engine.Config{RulesetVersion: first.Deck.RulesetVersion, Seed: seed, FirstPlayer: -1,
		Players: [2]engine.PlayerSetup{
			{ChampionID: first.Deck.ChampionID, Deck: expandDeck(first.Deck)},
			{ChampionID: second.Deck.ChampionID, Deck: expandDeck(second.Deck)},
		}}
	return Match{ID: id, Config: config, Status: StatusWaitingReady, CreatedAt: now,
		Players: [2]Participant{
			{UserID: first.Principal.UserID, DeckID: first.Deck.ID, Slot: 0},
			{UserID: second.Principal.UserID, DeckID: second.Deck.ID, Slot: 1},
		}}, nil
}

func randomSeed() (uint64, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(raw[:]) & (1<<63 - 1), nil
}

func validateStoredDeck(deck domain.Deck) error {
	return engine.ValidateDeck(deck.ChampionID, expandDeck(deck))
}

func expandDeck(deck domain.Deck) []string {
	result := make([]string, 0, engine.DeckSize)
	for _, card := range deck.Cards {
		for range card.Quantity {
			result = append(result, card.CardID)
		}
	}
	return result
}

func eventsForCommand(events []engine.Event, commands []StoredCommand, index int64) []engine.Event {
	first, last := -1, -1
	for _, command := range commands {
		if command.Index == index {
			first, last = command.FirstEventSeq, command.LastEventSeq
			break
		}
	}
	var result []engine.Event
	for _, event := range events {
		if event.Seq >= first && event.Seq <= last {
			result = append(result, event)
		}
	}
	return result
}

func sameEvents(a, b []engine.Event) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}

func ProtocolCode(err error) string {
	var commandErr *engine.CommandError
	if errors.As(err, &commandErr) {
		return commandErr.Code
	}
	switch {
	case errors.Is(err, ErrSequenceGap):
		return "sequence_gap"
	case errors.Is(err, ErrSequenceReuse):
		return "sequence_reuse"
	case errors.Is(err, ErrSpectatorWrite):
		return "spectator_read_only"
	case errors.Is(err, ErrNotReady):
		return "not_ready"
	case errors.Is(err, ErrInvalidIntent):
		return "invalid_message"
	default:
		return "battle_error"
	}
}
