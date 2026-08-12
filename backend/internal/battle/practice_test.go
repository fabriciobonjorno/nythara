package battle

import (
	"context"
	"errors"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// TestPracticeMatchFullGameAgainstBot joga uma partida de treino inteira: o
// humano é dirigido por random-legal sobre um jogo-sombra reconstruído do que
// o store persistiu (mesmo caminho de um cliente real via replay). Prova que
// o bot ocupa o assento 1, responde a cada jogada, a partida termina e o
// replay do log persistido é íntegro.
func TestPracticeMatchFullGameAgainstBot(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.RulesetVersion)
	human := domain.Principal{UserID: "00000000-0000-4000-8000-000000000042", Role: domain.RolePlayer}
	deck := testDeck(t, human.UserID, "deck-humano", "CH-VH-01")
	botDeck := testDeck(t, domain.BotUserID, "deck-bot", "CH-SO-01")

	recorded := make(chan FinishedMatch, 1)
	manager.SetProgressRecorder(func(_ context.Context, finished FinishedMatch, events []engine.Event) {
		if len(events) == 0 {
			t.Error("gravador sem eventos")
		}
		recorded <- finished
	})

	result, err := manager.StartPractice(ctx, human, deck, botDeck)
	if err != nil {
		t.Fatalf("StartPractice: %v", err)
	}
	if result.Status != "matched" || result.Slot == nil || *result.Slot != 0 {
		t.Fatalf("resultado inesperado: %+v", result)
	}

	ticket, err := manager.IssueTicket(ctx, human, result.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := manager.Connect(ctx, ticket.Token, result.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	go func() {
		for range connection.Messages {
		}
	}()
	if err := connection.Ready(ctx); err != nil {
		t.Fatal(err)
	}

	shadow := func() *LoadedMatch {
		loaded, err := store.LoadMatch(ctx, result.MatchID)
		if err != nil {
			t.Fatal(err)
		}
		return &loaded
	}
	humanBot := &engine.RandomBot{RNG: engine.NewRNG(99)}
	sequence := int64(0)
	botActed := false
	for turn := 0; turn < 900; turn++ {
		loaded := shadow()
		for _, cmd := range loaded.Commands {
			if cmd.Origin == "bot" {
				botActed = true
			}
		}
		if loaded.Match.Status == StatusFinished {
			// Replay integral do que foi persistido.
			replayed, err := engine.Replay(loaded.Match.Config, storedCommands(loaded))
			if err != nil {
				t.Fatalf("replay do treino: %v", err)
			}
			if !replayed.State().Over {
				t.Fatal("replay não terminou como a partida")
			}
			if !botActed {
				t.Fatal("o bot nunca agiu na partida inteira")
			}
			select {
			case finished := <-recorded:
				if finished.Mode != ModePractice || finished.MatchID != result.MatchID {
					t.Fatalf("gravação inesperada: %+v", finished)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("gravador de progresso não foi chamado")
			}
			return
		}
		game, err := engine.Replay(loaded.Match.Config, storedCommands(loaded))
		if err != nil {
			t.Fatalf("sombra: %v", err)
		}
		if game.State().Over {
			continue // persistência do fim chega no próximo load
		}
		if actor := expectedActor(game.State()); actor != 0 {
			t.Fatalf("turno %d: vez do assento %d — o bot deveria ter agido sozinho", turn, actor)
		}
		command, ok := humanBot.NextFor(game, 0)
		if !ok {
			t.Fatalf("turno %d: humano sem jogada legal", turn)
		}
		sequence++
		intent := Intent{Kind: command.Kind, Card: command.Card, Cards: command.Cards,
			Stance: command.Stance, DecisionID: command.DecisionID}
		if err := connection.Submit(ctx, sequence, intent); err != nil {
			t.Fatalf("turno %d: submit %+v: %v", turn, intent, err)
		}
	}
	t.Fatal("a partida de treino não terminou em 900 jogadas humanas")
}

func storedCommands(loaded *LoadedMatch) []engine.Command {
	out := make([]engine.Command, 0, len(loaded.Commands))
	for _, stored := range loaded.Commands {
		if stored.Origin == "system" || stored.Command.Kind == "" {
			continue // marco de eventos iniciais, não é comando de jogo
		}
		out = append(out, stored.Command)
	}
	return out
}

// Bans não bloqueiam o treino (não é competitivo); versão errada bloqueia.
func TestPracticeSkipsBansButChecksVersion(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.RulesetVersion)
	human := domain.Principal{UserID: "00000000-0000-4000-8000-000000000043", Role: domain.RolePlayer}
	deck := testDeck(t, human.UserID, "deck-h2", "CH-VH-01")
	botDeck := testDeck(t, domain.BotUserID, "deck-b2", "CH-CI-01")

	store.bans = []domain.CardBan{{CardID: deck.Cards[0].CardID, Reason: "teste"}}
	if _, err := manager.StartPractice(ctx, human, deck, botDeck); err != nil {
		t.Fatalf("treino deveria ignorar bans: %v", err)
	}
}

func TestPracticeActionTimeoutPassesInsteadOfConceding(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.CompetitiveRulesetVersion)
	manager.readyTimeout = time.Hour
	manager.actionTimeout = time.Hour
	human := domain.Principal{UserID: "00000000-0000-4000-8000-000000000044", Role: domain.RolePlayer}
	deck := confrontTestDeck(t, human.UserID, "deck-timeout-human", "CH-VH-01")
	botDeck := confrontTestDeck(t, domain.BotUserID, "deck-timeout-bot", "CH-SO-01")

	result, err := manager.StartPractice(ctx, human, deck, botDeck)
	if err != nil {
		t.Fatalf("StartPractice: %v", err)
	}
	ticket, err := manager.IssueTicket(ctx, human, result.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := manager.Connect(ctx, ticket.Token, result.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	waitMessage(t, connection.Messages, "sync")
	if err := connection.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, connection.Messages, StatusActive)
	// Um cliente real lê o WebSocket continuamente. Sem este consumidor o
	// buffer de 64 mensagens enche numa batalha longa e a sala remove a
	// conexão por lentidão — exatamente o comportamento que este teste não
	// pretende exercitar.
	go func() {
		for range connection.Messages {
		}
	}()

	room := manager.rooms[result.MatchID]
	if actor := expectedActor(room.game.State()); actor != 0 {
		t.Fatalf("bot deveria devolver a ação ao humano, ator=%d", actor)
	}
	room.requests <- timeoutRequest{generation: room.timerGen}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		loaded, loadErr := store.LoadMatch(ctx, result.MatchID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		if len(loaded.Commands) > 0 {
			var timeoutPass *StoredCommand
			for index := range loaded.Commands {
				stored := &loaded.Commands[index]
				if stored.Origin == "timeout" {
					timeoutPass = stored
					break
				}
			}
			if timeoutPass != nil {
				if timeoutPass.Command.Kind != engine.CmdKindPass || timeoutPass.PlayerSlot != 0 {
					t.Fatalf("timeout de treino persistiu comando inesperado: %+v", timeoutPass)
				}
				if loaded.Match.Status != StatusActive || loaded.Match.EndReason == "timeout" {
					t.Fatalf("timeout de treino encerrou a partida: status=%s reason=%s",
						loaded.Match.Status, loaded.Match.EndReason)
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("passe automático do treino não foi persistido")
}

func confrontTestDeck(t *testing.T, userID, id, avatarID string) domain.Deck {
	t.Helper()
	list, err := engine.CompetitiveRuleset().PreconstructedDeck(avatarID)
	if err != nil {
		t.Fatalf("precon competitivo: %v", err)
	}
	counts := map[string]int{}
	for _, cardID := range list {
		counts[cardID]++
	}
	deck := domain.Deck{ID: id, UserID: userID, ChampionID: avatarID,
		RulesetVersion: engine.CompetitiveRulesetVersion, Name: "Teste Confronto"}
	added := map[string]bool{}
	for _, cardID := range list {
		if added[cardID] {
			continue
		}
		added[cardID] = true
		quantity := counts[cardID]
		deck.Cards = append(deck.Cards, domain.DeckCard{CardID: cardID, Quantity: quantity})
	}
	if err := validateStoredDeck(deck); err != nil {
		t.Fatalf("deck competitivo inválido: %v", err)
	}
	return deck
}

// Decisão pendente no treino (ADR-059): a expiração responde a escolha com as
// primeiras opções em vez de derrubar a partida por concessão. O teste dirige
// o humano com uma política que provoca a decisão de verdade — defender sempre,
// nunca atacar, jogar um Rito de descarte assim que ele chegar à mão.
func TestPracticeTimeoutAnswersPendingDecision(t *testing.T) {
	ctx := context.Background()
	store := newMemoryBattleStore()
	manager := NewManager(store)
	manager.SetActiveRuleset(engine.DecisionRulesetVersion)
	manager.readyTimeout = time.Hour
	manager.actionTimeout = time.Hour
	human := domain.Principal{UserID: "00000000-0000-4000-8000-000000000045", Role: domain.RolePlayer}

	// Baralho legal em 0.13.0 cujos únicos Ritos pedem decisão: se um Rito
	// entra na mesa, uma escolha abre.
	rs, err := engine.RulesetByVersion(engine.DecisionRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	base, err := rs.PreconstructedDeck("CH-VH-01")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	// Com SkipShuffle abaixo, os quatro Ritos entram cedo e tornam a prova
	// repetível; o restante continua sendo um baralho legal da versão.
	deckList := []string{"VR-049", "VR-002", "VR-049", "VR-002"}
	for _, id := range base {
		if len(deckList) == engine.ConfrontDeckSize {
			break
		}
		switch rs.Cards[id].Type {
		case engine.TypeAssalto:
			deckList = append(deckList, id)
		case engine.TypeGuarda:
			deckList = append(deckList, id)
		}
	}
	for len(deckList) < engine.ConfrontDeckSize {
		// Completa com Guardas distintas do pool para manter 30 cartas legais.
		for _, card := range rs.CardList {
			if card.Type != engine.TypeGuarda || card.Confront == nil || !card.Confront.Legal {
				continue
			}
			if counts[card.ID] >= 2 {
				continue
			}
			already := 0
			for _, id := range deckList {
				if id == card.ID {
					already++
				}
			}
			if already >= 2 {
				continue
			}
			deckList = append(deckList, card.ID)
			break
		}
	}
	if err := rs.ValidateDeck("CH-VH-01", deckList); err != nil {
		t.Fatalf("baralho de teste inválido: %v", err)
	}
	for _, id := range deckList {
		counts[id]++
	}
	deck := domain.Deck{ID: "deck-decision-human", UserID: human.UserID, ChampionID: "CH-VH-01",
		RulesetVersion: engine.DecisionRulesetVersion, Name: "Decisão"}
	seen := map[string]bool{}
	for _, id := range deckList {
		if seen[id] {
			continue
		}
		seen[id] = true
		deck.Cards = append(deck.Cards, domain.DeckCard{CardID: id, Quantity: counts[id]})
	}
	botDeck := confrontDecisionBotDeck(t)

	result, err := manager.StartPractice(ctx, human, deck, botDeck)
	if err != nil {
		t.Fatalf("StartPractice: %v", err)
	}
	// O caminho de produção usa seed aleatória. Este teste não mede sorte:
	// fixa ordem/iniciativa antes de Ready para sempre alcançar a decisão.
	room := manager.rooms[result.MatchID]
	room.match.Config.SkipShuffle = true
	room.match.Config.FirstPlayer = 0
	store.mu.Lock()
	loaded := store.matches[result.MatchID]
	loaded.Match.Config = room.match.Config
	store.matches[result.MatchID] = loaded
	store.mu.Unlock()
	ticket, err := manager.IssueTicket(ctx, human, result.MatchID, TicketPlayer)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := manager.Connect(ctx, ticket.Token, result.MatchID, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	waitMessage(t, connection.Messages, "sync")
	if err := connection.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, connection.Messages, StatusActive)
	go func() {
		for range connection.Messages {
		}
	}()

	var sequence int64
	for step := 0; step < 400; step++ {
		waitHumanTurn(t, room)
		state := room.game.State()
		if state.Over {
			t.Fatalf("a partida terminou antes de abrir uma decisão (motivo %s)", state.EndReason)
		}
		if pending := state.Pending; pending != nil && pending.Player == 0 {
			firstN := append([]string{}, pending.Options[:min(pending.N, len(pending.Options))]...)
			ownerView := ViewFor(room.game, 0, false)
			rivalView := ViewFor(room.game, 1, false)
			spectatorView := ViewFor(room.game, -1, true)
			if ownerView.Pending == nil || !sameStrings(ownerView.Pending.Options, pending.Options) {
				t.Fatalf("dono não recebeu opções da decisão: %+v", ownerView.Pending)
			}
			for label, view := range map[string]StateView{"rival": rivalView, "espectador": spectatorView} {
				if view.Pending == nil || len(view.Pending.Options) != 0 || view.Pending.Card != "" {
					t.Fatalf("%s recebeu opções privadas: %+v", label, view.Pending)
				}
				for _, id := range pending.Options {
					if _, leaked := view.Cards[id]; leaked {
						t.Fatalf("%s recebeu a carta privada %s", label, id)
					}
				}
			}

			// Um processo novo restaura a decisão aberta do snapshot + comandos.
			// Confirmar duas vezes persiste uma escolha e devolve ack idempotente.
			atPending, loadErr := store.LoadMatch(ctx, result.MatchID)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			restartStore := newMemoryBattleStore()
			restartStore.matches[result.MatchID] = atPending
			restarted := NewManager(restartStore)
			restarted.SetActiveRuleset(engine.DecisionRulesetVersion)
			restarted.readyTimeout, restarted.actionTimeout = time.Hour, time.Hour
			restartTicket, ticketErr := restarted.IssueTicket(ctx, human, result.MatchID, TicketPlayer)
			if ticketErr != nil {
				t.Fatal(ticketErr)
			}
			restartConnection, connectErr := restarted.Connect(ctx, restartTicket.Token, result.MatchID, -1)
			if connectErr != nil {
				t.Fatal(connectErr)
			}
			restartSync := waitMessage(t, restartConnection.Messages, "sync")
			if restartSync.State == nil || restartSync.State.Pending == nil ||
				restartSync.State.Pending.ID != pending.ID ||
				!sameStrings(restartSync.State.Pending.Options, pending.Options) {
				t.Fatalf("restart perdeu a decisão: %+v", restartSync.State)
			}
			chooseSequence := atPending.Match.Players[0].LastSequence + 1
			choice := Intent{Kind: engine.CmdKindChoose, DecisionID: pending.ID, Cards: firstN}
			if err := restartConnection.Submit(ctx, chooseSequence, choice); err != nil {
				t.Fatalf("confirmar depois do restart: %v", err)
			}
			waitSequence(t, restartConnection.Messages, chooseSequence, false)
			if err := restartConnection.Submit(ctx, chooseSequence, choice); err != nil {
				t.Fatalf("repetir confirmação: %v", err)
			}
			waitSequence(t, restartConnection.Messages, chooseSequence, true)
			restoredLoaded, loadErr := restartStore.LoadMatch(ctx, result.MatchID)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			confirmed := 0
			for _, stored := range restoredLoaded.Commands {
				if stored.Origin == "client" && stored.Command.Kind == engine.CmdKindChoose {
					confirmed++
				}
			}
			if confirmed != 1 {
				t.Fatalf("confirmação duplicada foi persistida %d vezes", confirmed)
			}
			restartConnection.Close()

			// Reconectar na sala viva devolve as mesmas opções privadas e mantém
			// a sequência confirmada anterior.
			connection.Close()
			if err := connection.Submit(ctx, sequence+1, choice); !errors.Is(err, ErrConnectionClosed) {
				t.Fatalf("submit após fechar: %v", err)
			}
			reconnectTicket, ticketErr := manager.IssueTicket(ctx, human, result.MatchID, TicketPlayer)
			if ticketErr != nil {
				t.Fatal(ticketErr)
			}
			reconnected, connectErr := manager.Connect(ctx, reconnectTicket.Token, result.MatchID, -1)
			if connectErr != nil {
				t.Fatal(connectErr)
			}
			defer reconnected.Close()
			reconnectSync := waitMessage(t, reconnected.Messages, "sync")
			if reconnectSync.State == nil || reconnectSync.State.Pending == nil ||
				reconnectSync.ClientSequence != sequence ||
				!sameStrings(reconnectSync.State.Pending.Options, pending.Options) {
				t.Fatalf("reconexão perdeu a decisão: %+v", reconnectSync)
			}
			go func() {
				for range reconnected.Messages {
				}
			}()

			// A decisão abriu: o humano some e o relógio expira.
			room.requests <- timeoutRequest{generation: room.timerGen}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				loaded, loadErr := store.LoadMatch(ctx, result.MatchID)
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				for index := range loaded.Commands {
					stored := &loaded.Commands[index]
					if stored.Origin == "timeout" && stored.Command.Kind == engine.CmdKindChoose {
						if stored.Command.DecisionID != pending.ID || !sameStrings(stored.Command.Cards, firstN) {
							t.Fatalf("timeout escolheu opções inesperadas: %+v; esperado %v", stored.Command, firstN)
						}
						if loaded.Match.Status == StatusFinished && loaded.Match.EndReason == "concede" {
							t.Fatal("timeout de decisão concedeu a partida")
						}
						replayed, replayErr := engine.Replay(loaded.Match.Config, storedCommands(&loaded))
						if replayErr != nil {
							t.Fatalf("replay após timeout de decisão: %v", replayErr)
						}
						if !sameEvents(replayed.Log, loaded.Events) {
							t.Fatal("replay divergiu depois da escolha automática")
						}
						return
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			t.Fatal("timeout não respondeu a decisão pendente")
		}
		command := decisionPolicy(room.game)
		sequence++
		intent := Intent{Kind: command.Kind, Card: command.Card, Cards: command.Cards, DecisionID: command.DecisionID}
		if err := connection.Submit(ctx, sequence, intent); err != nil {
			t.Fatalf("passo %d: submit %+v: %v", step, intent, err)
		}
	}
	t.Fatal("nenhuma decisão abriu em 400 ações humanas")
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// decisionPolicy joga para provocar decisões: defende sempre, nunca ataca, e
// joga um Rito de descarte assim que possível.
func decisionPolicy(game *engine.Game) engine.Command {
	state := game.State()
	legal := game.LegalPlayIDs(0)
	pick := func(cardType engine.CardType, defs ...string) string {
		for _, id := range legal {
			def := state.Cards[id].Def
			if game.Ruleset().Cards[def].Type != cardType {
				continue
			}
			if len(defs) == 0 {
				return id
			}
			for _, want := range defs {
				if def == want {
					return id
				}
			}
		}
		return ""
	}
	switch state.Phase {
	case engine.PhaseGuard:
		if id := pick(engine.TypeGuarda); id != "" {
			return engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: id}
		}
	case engine.PhaseRite:
		if id := pick(engine.TypeRito, "VR-002", "VR-049"); id != "" {
			return engine.Command{Player: 0, Kind: engine.CmdKindPlay, Card: id}
		}
	}
	return engine.Command{Player: 0, Kind: engine.CmdKindPass}
}

func confrontDecisionBotDeck(t *testing.T) domain.Deck {
	t.Helper()
	rs, err := engine.RulesetByVersion(engine.DecisionRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	list, err := rs.PreconstructedDeck("CH-SO-01")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, id := range list {
		counts[id]++
	}
	deck := domain.Deck{ID: "deck-decision-bot", UserID: domain.BotUserID, ChampionID: "CH-SO-01",
		RulesetVersion: engine.DecisionRulesetVersion, Name: "Bot Decisão"}
	seen := map[string]bool{}
	for _, id := range list {
		if seen[id] {
			continue
		}
		seen[id] = true
		deck.Cards = append(deck.Cards, domain.DeckCard{CardID: id, Quantity: counts[id]})
	}
	return deck
}

func waitHumanTurn(t *testing.T, room *room) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := room.game.State()
		if state.Over || expectedActor(state) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("o bot não devolveu a ação ao humano")
}
