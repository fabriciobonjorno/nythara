package battle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

func TestBattleCommandsAreAuthoritativeIdempotentAndReconnectable(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.RulesetVersion)
	manager.readyTimeout = time.Hour
	manager.actionTimeout = time.Hour
	p0 := domain.Principal{UserID: "00000000-0000-4000-8000-000000000101", Role: domain.RolePlayer}
	p1 := domain.Principal{UserID: "00000000-0000-4000-8000-000000000102", Role: domain.RolePlayer}
	d0 := testDeck(t, p0.UserID, "00000000-0000-4000-8000-000000000201", "CH-VH-01")
	d1 := testDeck(t, p1.UserID, "00000000-0000-4000-8000-000000000202", "CH-SO-01")
	if result, err := manager.Queue(ctx, p0, d0); err != nil || result.Status != "queued" {
		t.Fatalf("primeiro na fila: result=%+v err=%v", result, err)
	}
	matched, err := manager.Queue(ctx, p1, d1)
	if err != nil || matched.Status != "matched" {
		t.Fatalf("match: result=%+v err=%v", matched, err)
	}
	ticket0, err := manager.IssueTicket(ctx, p0, matched.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	ticket1, err := manager.IssueTicket(ctx, p1, matched.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	c0, err := manager.Connect(ctx, ticket0.Token, matched.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer c0.Close()
	c1, err := manager.Connect(ctx, ticket1.Token, matched.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	waitMessage(t, c0.Messages, "sync")
	waitMessage(t, c1.Messages, "sync")
	if err := c0.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c1.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, c0.Messages, StatusActive)
	waitStatus(t, c1.Messages, StatusActive)
	admin := domain.Principal{UserID: "00000000-0000-4000-8000-000000000999", Role: domain.RoleAdmin}
	spectatorTicket, err := manager.IssueTicket(ctx, admin, matched.MatchID, TicketSpectator)
	if err != nil {
		t.Fatal(err)
	}
	spectator, err := manager.Connect(ctx, spectatorTicket.Token, matched.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer spectator.Close()
	spectatorSync := waitMessage(t, spectator.Messages, "sync")
	if spectatorSync.State == nil || len(spectatorSync.State.Players[0].Hand) != 0 ||
		len(spectatorSync.State.Players[1].Hand) != 0 {
		t.Fatal("espectador recebeu informação privada")
	}
	if err := spectator.Submit(ctx, 1, Intent{Kind: engine.CmdKindConcede}); !errors.Is(err, ErrSpectatorWrite) {
		t.Fatalf("espectador conseguiu escrever: %v", err)
	}

	keep := Intent{Kind: engine.CmdKindMulligan}
	if err := c0.Submit(ctx, 1, keep); err != nil {
		t.Fatal(err)
	}
	waitSequence(t, c0.Messages, 1, false)
	if err := c0.Submit(ctx, 1, keep); err != nil {
		t.Fatal(err)
	}
	waitSequence(t, c0.Messages, 1, true)
	if err := c0.Submit(ctx, 1, Intent{Kind: engine.CmdKindMulligan, Cards: []string{"forjada"}}); !errors.Is(err, ErrSequenceReuse) {
		t.Fatalf("reuso divergente: %v", err)
	}
	if err := c0.Submit(ctx, 3, keep); !errors.Is(err, ErrSequenceGap) {
		t.Fatalf("salto de sequência: %v", err)
	}
	if err := c1.Submit(ctx, 1, keep); err != nil {
		t.Fatal(err)
	}
	waitSequence(t, c1.Messages, 1, false)
	if err := c0.Submit(ctx, 2, Intent{Kind: engine.CmdKindStance, Stance: engine.StancePredacao}); err != nil {
		t.Fatal(err)
	}
	if err := c1.Submit(ctx, 2, Intent{Kind: engine.CmdKindStance, Stance: engine.StanceVigilia}); err != nil {
		t.Fatal(err)
	}

	r := manager.rooms[matched.MatchID]
	active := r.game.State().Active
	nonActiveConnection := c0
	nonActiveSequence := int64(3)
	if active == 0 {
		nonActiveConnection = c1
	}
	if err := nonActiveConnection.Submit(ctx, nonActiveSequence, Intent{Kind: engine.CmdKindPass}); err == nil {
		t.Fatal("comando fora de turno foi aceito")
	} else {
		var commandErr *engine.CommandError
		if !errors.As(err, &commandErr) || commandErr.Code != engine.ErrWrongPlayer {
			t.Fatalf("erro fora de turno inesperado: %v", err)
		}
	}

	lastSeqBefore := r.match.Players[0].LastSequence
	c0.Close()
	if err := c0.Submit(ctx, lastSeqBefore+1, Intent{Kind: engine.CmdKindPass}); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("conexão encerrada devolveu diagnóstico incorreto: %v", err)
	}
	reconnectTicket, err := manager.IssueTicket(ctx, p0, matched.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	reconnected, err := manager.Connect(ctx, reconnectTicket.Token, matched.MatchID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer reconnected.Close()
	syncMessage := waitMessage(t, reconnected.Messages, "sync")
	if syncMessage.State == nil || len(syncMessage.Events) == 0 {
		t.Fatal("reconexão não recebeu estado + catch-up")
	}
	if syncMessage.ClientSequence != lastSeqBefore {
		t.Fatalf("reconexão informou sequência %d, esperada %d", syncMessage.ClientSequence, lastSeqBefore)
	}
	if len(syncMessage.State.Players[1].Hand) != 0 {
		t.Fatal("reconexão vazou mão adversária")
	}
	if r.match.Players[0].LastSequence != lastSeqBefore {
		t.Fatal("reconexão alterou sequência")
	}

	// Simula restart do processo: nova Manager restaura snapshot inicial e
	// reaplica os comandos persistidos posteriores.
	manager2 := NewManager(store)
	manager2.SetActiveRuleset(engine.RulesetVersion)
	manager2.readyTimeout = time.Hour
	manager2.actionTimeout = time.Hour
	restartTicket, err := manager2.IssueTicket(ctx, p0, matched.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	afterRestart, err := manager2.Connect(ctx, restartTicket.Token, matched.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer afterRestart.Close()
	restored := waitMessage(t, afterRestart.Messages, "sync")
	if restored.State == nil || restored.State.Phase != r.game.State().Phase || restored.State.Round != r.game.State().Round {
		t.Fatalf("estado restaurado divergiu: %+v", restored.State)
	}
	if restored.ClientSequence != lastSeqBefore {
		t.Fatalf("restart informou sequência %d, esperada %d", restored.ClientSequence, lastSeqBefore)
	}
	if err := afterRestart.Submit(ctx, 1, keep); err != nil {
		t.Fatalf("duplicata após restart: %v", err)
	}
	waitSequence(t, afterRestart.Messages, 1, true)
	restartedRoom := manager2.rooms[matched.MatchID]
	restartedRoom.requests <- timeoutRequest{generation: restartedRoom.timerGen}
	finished := waitStatus(t, afterRestart.Messages, StatusFinished)
	if finished.State == nil || finished.State.EndReason != "timeout" {
		t.Fatalf("timer não encerrou authoritative: %+v", finished.State)
	}
	persisted, err := store.LoadMatch(ctx, matched.MatchID)
	if err != nil || persisted.Match.Status != StatusFinished || persisted.Match.EndReason != "timeout" {
		t.Fatalf("timeout não persistido: status=%s reason=%s err=%v",
			persisted.Match.Status, persisted.Match.EndReason, err)
	}
}

func TestSubscriberDisconnectReasonIsNotReportedAsSpectator(t *testing.T) {
	r := &room{subscribers: map[string]*subscriber{}, departed: map[string]error{}}
	slow := &subscriber{id: "slow", mode: TicketPlayer, slot: 0, out: make(chan ServerMessage, 1)}
	r.subscribers[slow.id] = slow
	r.send(slow, ServerMessage{Type: "first"})
	r.send(slow, ServerMessage{Type: "overflow"})
	if _, err := r.writableSubscriber(slow.id); !errors.Is(err, ErrSubscriberSlow) {
		t.Fatalf("assinante lento recebeu diagnóstico %v", err)
	} else if code := ProtocolCode(err); code != "subscriber_too_slow" {
		t.Fatalf("código do assinante lento: %s", code)
	}

	closed := &subscriber{id: "closed", mode: TicketPlayer, slot: 0, out: make(chan ServerMessage)}
	r.subscribers[closed.id] = closed
	r.dropSubscriber(closed, ErrConnectionClosed)
	if _, err := r.writableSubscriber(closed.id); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("conexão encerrada recebeu diagnóstico %v", err)
	} else if code := ProtocolCode(err); code != "connection_closed" {
		t.Fatalf("código da conexão encerrada: %s", code)
	}

	spectator := &subscriber{id: "spectator", mode: TicketSpectator, slot: -1, out: make(chan ServerMessage)}
	r.subscribers[spectator.id] = spectator
	if _, err := r.writableSubscriber(spectator.id); !errors.Is(err, ErrSpectatorWrite) {
		t.Fatalf("espectador recebeu diagnóstico %v", err)
	}
}

func TestViewAndEventsHidePrivateCards(t *testing.T) {
	config := engine.Config{RulesetVersion: engine.RulesetVersion, Seed: 7, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: "CH-VH-01", Deck: expandDeck(testDeck(t, "u0", "d0", "CH-VH-01"))},
			{ChampionID: "CH-SO-01", Deck: expandDeck(testDeck(t, "u1", "d1", "CH-SO-01"))}}}
	game, err := engine.NewGame(config)
	if err != nil {
		t.Fatal(err)
	}
	view := ViewFor(game, 0, false)
	if len(view.Players[0].Hand) != engine.OpeningHandSize || len(view.Players[1].Hand) != 0 {
		t.Fatalf("mãos redigidas incorretamente: própria=%d rival=%d", len(view.Players[0].Hand), len(view.Players[1].Hand))
	}
	for _, id := range game.State().Players[1].Hand {
		if _, leaked := view.Cards[id]; leaked {
			t.Fatalf("instância rival %s vazou", id)
		}
	}
	draw := engine.Event{Seq: 9, Kind: engine.EvCardDrawn, P: 1, Card: "p1-c01", Def: "VR-001"}
	redacted := RedactEvents([]engine.Event{draw}, 0, false)[0]
	if redacted.Card != "" || redacted.Def != "" {
		t.Fatal("evento de compra rival vazou carta")
	}
}

func waitMessage(t *testing.T, messages <-chan ServerMessage, kind string) ServerMessage {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				t.Fatal("conexão fechada")
			}
			if message.Type == kind {
				return message
			}
		case <-deadline:
			t.Fatalf("timeout aguardando mensagem %s", kind)
		}
	}
}

func waitStatus(t *testing.T, messages <-chan ServerMessage, status Status) ServerMessage {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-messages:
			if message.Status == status {
				return message
			}
		case <-deadline:
			t.Fatalf("timeout aguardando status %s", status)
		}
	}
}

func waitSequence(t *testing.T, messages <-chan ServerMessage, sequence int64, duplicate bool) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-messages:
			if message.ClientSequence == sequence && message.Duplicate == duplicate {
				return
			}
		case <-deadline:
			t.Fatalf("timeout aguardando sequence=%d duplicate=%v", sequence, duplicate)
		}
	}
}

func testDeck(t *testing.T, userID, id, championID string) domain.Deck {
	t.Helper()
	faction := engine.Champions[championID].Faction
	deck := domain.Deck{ID: id, UserID: userID, ChampionID: championID,
		RulesetVersion: engine.RulesetVersion, Name: "Teste"}
	total := 0
	for _, card := range engine.CardList {
		if card.Faction != faction && card.Faction != engine.NeutralFaction {
			continue
		}
		quantity := engine.MaxCopies
		if card.Rarity == engine.RarityLendaria {
			quantity = engine.MaxLegendary
		}
		if total+quantity > engine.DeckSize {
			quantity = engine.DeckSize - total
		}
		if quantity > 0 {
			deck.Cards = append(deck.Cards, domain.DeckCard{CardID: card.ID, Quantity: quantity})
			total += quantity
		}
		if total == engine.DeckSize {
			break
		}
	}
	if err := validateStoredDeck(deck); err != nil {
		t.Fatal(err)
	}
	return deck
}

type memoryBattleStore struct {
	mu      sync.Mutex
	matches map[string]LoadedMatch
	bans    []domain.CardBan
}

func newMemoryBattleStore() *memoryBattleStore {
	return &memoryBattleStore{matches: map[string]LoadedMatch{}}
}

func (s *memoryBattleStore) CreateMatch(_ context.Context, match Match) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches[match.ID] = LoadedMatch{Match: match}
	return nil
}

func (s *memoryBattleStore) MarkReady(_ context.Context, id string, slot int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loaded := s.matches[id]
	loaded.Match.Players[slot].Ready = true
	s.matches[id] = loaded
	return nil
}

func (s *memoryBattleStore) StartMatch(_ context.Context, id string, events []engine.Event, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loaded := s.matches[id]
	loaded.Match.Status = StatusActive
	loaded.Events = cloneEvents(events)
	loaded.Snapshot = cloneSnapshot(&snapshot)
	loaded.Commands = append(loaded.Commands, StoredCommand{Index: 0, PlayerSlot: -1, Origin: "system",
		FirstEventSeq: 0, LastEventSeq: len(events) - 1})
	s.matches[id] = loaded
	return nil
}

func (s *memoryBattleStore) PersistStep(_ context.Context, step Step) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loaded := s.matches[step.MatchID]
	first := -1
	last := -1
	if len(step.Events) > 0 {
		first, last = step.Events[0].Seq, step.Events[len(step.Events)-1].Seq
	}
	loaded.Commands = append(loaded.Commands, StoredCommand{Index: step.CommandIndex, PlayerSlot: step.PlayerSlot,
		ClientSequence: step.ClientSequence, Origin: step.Origin, Command: step.Command,
		FirstEventSeq: first, LastEventSeq: last})
	loaded.Events = append(loaded.Events, cloneEvents(step.Events)...)
	if step.ClientSequence != nil {
		loaded.Match.Players[step.PlayerSlot].LastSequence = *step.ClientSequence
	}
	if step.Snapshot != nil {
		loaded.Snapshot = cloneSnapshot(step.Snapshot)
	}
	if step.Finished {
		loaded.Match.Status, loaded.Match.Winner, loaded.Match.EndReason = StatusFinished, step.Winner, step.EndReason
	}
	s.matches[step.MatchID] = loaded
	return nil
}

func (s *memoryBattleStore) CancelMatch(_ context.Context, id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loaded := s.matches[id]
	loaded.Match.Status, loaded.Match.EndReason = StatusCancelled, reason
	s.matches[id] = loaded
	return nil
}

func (s *memoryBattleStore) LoadMatch(_ context.Context, id string) (LoadedMatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	loaded, ok := s.matches[id]
	if !ok {
		return LoadedMatch{}, domain.ErrNotFound
	}
	raw, _ := json.Marshal(loaded)
	var clone LoadedMatch
	_ = json.Unmarshal(raw, &clone)
	return clone, nil
}

func (s *memoryBattleStore) ActiveBans(ctx context.Context) ([]domain.CardBan, error) {
	return s.bans, nil
}

func (s *memoryBattleStore) ActiveMatchForUser(_ context.Context, userID string) (Match, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, loaded := range s.matches {
		if loaded.Match.Status == StatusFinished || loaded.Match.Status == StatusCancelled {
			continue
		}
		for _, player := range loaded.Match.Players {
			if player.UserID == userID {
				return loaded.Match, player.Slot, nil
			}
		}
	}
	return Match{}, 0, domain.ErrNotFound
}

func cloneEvents(events []engine.Event) []engine.Event {
	return append([]engine.Event{}, events...)
}

func cloneSnapshot(snapshot *Snapshot) *Snapshot {
	if snapshot == nil {
		return nil
	}
	clone := *snapshot
	clone.Data = append([]byte{}, snapshot.Data...)
	return &clone
}

// Fase 7: deck com carta banida não entra na fila; após o lift, entra.
func TestQueueRejectsBannedCards(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.RulesetVersion)
	p0 := domain.Principal{UserID: "00000000-0000-4000-8000-000000000001", Role: domain.RolePlayer}
	deck := testDeck(t, p0.UserID, "deck-ban", "CH-VH-01")

	store.bans = []domain.CardBan{{CardID: deck.Cards[0].CardID, Reason: "loop reportado"}}
	if _, err := manager.Queue(ctx, p0, deck); err == nil ||
		!strings.Contains(err.Error(), "desativada do competitivo") {
		t.Fatalf("fila deveria recusar carta banida; err=%v", err)
	}

	store.bans = nil
	if result, err := manager.Queue(ctx, p0, deck); err != nil || result.Status != "queued" {
		t.Fatalf("fila deveria aceitar após o lift; %+v err=%v", result, err)
	}
}

// Ativação/rollback repontam o matchmaking sem afetar decks da versão antiga.
func TestQueueFollowsActiveRuleset(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(newMemoryBattleStore())
	p0 := domain.Principal{UserID: "00000000-0000-4000-8000-000000000002", Role: domain.RolePlayer}
	deck := testDeck(t, p0.UserID, "deck-vs", "CH-VH-01")

	manager.SetActiveRuleset("alpha-9.9.9")
	if _, err := manager.Queue(ctx, p0, deck); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("deck de versão antiga deveria ser recusado; err=%v", err)
	}
	manager.SetActiveRuleset(engine.RulesetVersion)
	if result, err := manager.Queue(ctx, p0, deck); err != nil || result.Status != "queued" {
		t.Fatalf("rollback deveria reabilitar o deck; %+v err=%v", result, err)
	}
}
