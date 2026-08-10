package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

func TestPostgresRejectsIllegalDeckAtCommit(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	deckID, _ := security.NewID()
	championID := ""
	for id := range engine.Champions {
		championID = id
		break
	}
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO decks(id,user_id,name,champion_id,ruleset_version)
		VALUES($1,$2,'Ilegal direto no SQL',$3,$4)`, deckID, userID, championID, engine.RulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
		VALUES($1,$2,$3,1)`, deckID, engine.CardList[0].ID, engine.RulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err == nil {
		t.Fatal("constraint diferida permitiu confirmar deck com uma carta")
	}
}

func TestPostgresSavesLegalDeckIdempotently(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	championID := ""
	for id := range engine.Champions {
		championID = id
		break
	}
	deck := legalIntegrationDeck(t, userID, championID, "Legal")
	mutation := domain.Mutation{Key: "integration-deck-001", Operation: "deck:create", RequestHash: []byte("same-body")}
	saved, replayed, err := db.SaveDeck(ctx, deck, nil, mutation)
	if err != nil || replayed {
		t.Fatalf("salvar deck legal: replay=%v err=%v", replayed, err)
	}
	replayedDeck, replayed, err := db.SaveDeck(ctx, deck, nil, mutation)
	if err != nil || !replayed || replayedDeck.ID != saved.ID {
		t.Fatalf("replay idempotente: deck=%s replay=%v err=%v", replayedDeck.ID, replayed, err)
	}
	mutation.RequestHash = []byte("different-body")
	if _, _, err := db.SaveDeck(ctx, deck, nil, mutation); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("reuso conflitante deveria falhar, recebido: %v", err)
	}
	expected := saved.Version
	deck.Name = "Legal alterado"
	updated, _, err := db.SaveDeck(ctx, deck, &expected, domain.Mutation{
		Key: "integration-deck-update-001", Operation: "deck:update:" + deck.ID, RequestHash: []byte("update-body"),
	})
	if err != nil || updated.Version != expected+1 {
		t.Fatalf("atualizar deck legal: version=%d err=%v", updated.Version, err)
	}
	if _, err := db.DeleteDeck(ctx, userID, deck.ID, updated.Version, domain.Mutation{
		Key: "integration-deck-delete-001", Operation: "deck:delete:" + deck.ID, RequestHash: []byte("delete-body"),
	}); err != nil {
		t.Fatalf("excluir deck legal: %v", err)
	}
}

func TestPostgresPersistsBattleSnapshotAndCatchup(t *testing.T) {
	ctx, db, user0 := integrationDB(t)
	user1, _ := security.NewID()
	_, err := db.CreateUser(ctx, domain.User{ID: user1, Email: user1 + "@example.test",
		DisplayName: "Rival DB", Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.RulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	deck0 := legalIntegrationDeck(t, user0, "CH-VH-01", "Batalha 0")
	deck1 := legalIntegrationDeck(t, user1, "CH-SO-01", "Batalha 1")
	for i, deck := range []*domain.Deck{&deck0, &deck1} {
		mutation := domain.Mutation{Key: "battle-deck-000" + string(rune('1'+i)), Operation: "deck:create",
			RequestHash: []byte{byte(i + 1)}}
		if _, _, err := db.SaveDeck(ctx, *deck, nil, mutation); err != nil {
			t.Fatal(err)
		}
	}
	matchID, _ := security.NewID()
	config := engine.Config{RulesetVersion: engine.RulesetVersion, Seed: 99, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: deck0.ChampionID, Deck: expandIntegrationDeck(deck0)},
			{ChampionID: deck1.ChampionID, Deck: expandIntegrationDeck(deck1)}}}
	match := battle.Match{ID: matchID, Config: config, Status: battle.StatusWaitingReady, CreatedAt: time.Now().UTC(),
		Players: [2]battle.Participant{{UserID: user0, DeckID: deck0.ID, Slot: 0},
			{UserID: user1, DeckID: deck1.ID, Slot: 1}}}
	if err := db.CreateMatch(ctx, match); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkReady(ctx, matchID, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkReady(ctx, matchID, 1); err != nil {
		t.Fatal(err)
	}
	game, err := engine.NewGame(config)
	if err != nil {
		t.Fatal(err)
	}
	snapshotData, _ := game.SnapshotJSON()
	initial := battle.Snapshot{CommandIndex: 0, EventSeq: len(game.Log) - 1, Data: snapshotData}
	if err := db.StartMatch(ctx, matchID, game.Log, initial); err != nil {
		t.Fatal(err)
	}
	command := engine.Command{Player: 0, Kind: engine.CmdKindMulligan}
	events, err := game.Apply(command)
	if err != nil {
		t.Fatal(err)
	}
	sequence := int64(1)
	if err := db.PersistStep(ctx, battle.Step{MatchID: matchID, CommandIndex: 1, PlayerSlot: 0,
		ClientSequence: &sequence, Origin: "client", Command: command, Events: events}); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.LoadMatch(ctx, matchID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Snapshot == nil || len(loaded.Commands) != 2 || len(loaded.Events) != len(game.Log) {
		t.Fatalf("persistência incompleta: snapshot=%v commands=%d events=%d/%d",
			loaded.Snapshot != nil, len(loaded.Commands), len(loaded.Events), len(game.Log))
	}
	active, slot, err := db.ActiveMatchForUser(ctx, user0)
	if err != nil || active.ID != matchID || slot != 0 {
		t.Fatalf("retomar partida ativa: match=%s slot=%d err=%v", active.ID, slot, err)
	}
	timeoutCommand := engine.Command{Player: 1, Kind: engine.CmdKindConcede, Reason: "timeout"}
	timeoutEvents, err := game.Apply(timeoutCommand)
	if err != nil {
		t.Fatal(err)
	}
	finalSnapshot, _ := game.SnapshotJSON()
	winner := game.State().Winner
	if err := db.PersistStep(ctx, battle.Step{MatchID: matchID, CommandIndex: 2, PlayerSlot: 1,
		Origin: "timeout", Command: timeoutCommand, Events: timeoutEvents, Finished: true,
		Winner: &winner, EndReason: "timeout", Snapshot: &battle.Snapshot{
			CommandIndex: 2, EventSeq: len(game.Log) - 1, Data: finalSnapshot,
		}}); err != nil {
		t.Fatal(err)
	}
	finished, err := db.LoadMatch(ctx, matchID)
	if err != nil || finished.Match.Status != battle.StatusFinished || finished.Match.EndReason != "timeout" ||
		finished.Snapshot == nil || finished.Snapshot.CommandIndex != 2 {
		t.Fatalf("fim persistido incorretamente: status=%s reason=%s snapshot=%+v err=%v",
			finished.Match.Status, finished.Match.EndReason, finished.Snapshot, err)
	}
	if _, _, err := db.ActiveMatchForUser(ctx, user0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("partida finalizada ainda aparece ativa: %v", err)
	}
}

func TestRefreshReuseRevokesSession(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	sessionID, _ := security.NewID()
	now := time.Now().UTC()
	_, oldAccess, _ := security.NewToken()
	_, oldRefresh, _ := security.NewToken()
	if err := db.CreateSession(ctx, domain.NewSession{ID: sessionID, UserID: userID,
		AccessHash: oldAccess, RefreshHash: oldRefresh,
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	_, newAccess, _ := security.NewToken()
	_, newRefresh, _ := security.NewToken()
	_, err := db.RotateSession(ctx, oldRefresh, domain.RotatedSession{
		AccessHash: newAccess, RefreshHash: newRefresh,
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(24 * time.Hour)}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.RotateSession(ctx, oldRefresh, domain.RotatedSession{}, now.Add(time.Second)); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("reuso de refresh não foi rejeitado: %v", err)
	}
	if _, err := db.AccessToken(ctx, newAccess, now.Add(2*time.Second)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sessão não foi revogada após reuso: %v", err)
	}
}

func integrationDB(t *testing.T) (context.Context, *Postgres, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.MigrateUp(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.SyncCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	userID, _ := security.NewID()
	_, err = db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@example.test",
		DisplayName: "Teste DB", Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.RulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, db, userID
}

func legalIntegrationDeck(t *testing.T, userID, championID, name string) domain.Deck {
	t.Helper()
	faction := engine.Champions[championID].Faction
	deck := domain.Deck{UserID: userID, Name: name, ChampionID: championID, RulesetVersion: engine.RulesetVersion}
	deck.ID, _ = security.NewID()
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
	if total != engine.DeckSize {
		t.Fatalf("não foi possível montar deck legal: %d cartas", total)
	}
	return deck
}

func expandIntegrationDeck(deck domain.Deck) []string {
	result := make([]string, 0, engine.DeckSize)
	for _, card := range deck.Cards {
		for range card.Quantity {
			result = append(result, card.CardID)
		}
	}
	return result
}
