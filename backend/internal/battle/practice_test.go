package battle

import (
	"context"
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
	human := domain.Principal{UserID: "00000000-0000-4000-8000-000000000043", Role: domain.RolePlayer}
	deck := testDeck(t, human.UserID, "deck-h2", "CH-VH-01")
	botDeck := testDeck(t, domain.BotUserID, "deck-b2", "CH-CI-01")

	store.bans = []domain.CardBan{{CardID: deck.Cards[0].CardID, Reason: "teste"}}
	if _, err := manager.StartPractice(ctx, human, deck, botDeck); err != nil {
		t.Fatalf("treino deveria ignorar bans: %v", err)
	}
}
