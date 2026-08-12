package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"encoding/json"
	"strings"
	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

func TestIdentityMigrationRenamesCaseInsensitiveDuplicatesDeterministically(t *testing.T) {
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
	t.Cleanup(func() { _ = db.Close() })
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE player_profiles (
		user_id uuid PRIMARY KEY,
		display_name text NOT NULL,
		created_at timestamptz NOT NULL,
		updated_at timestamptz NOT NULL
	); SET LOCAL search_path TO pg_temp`); err != nil {
		t.Fatal(err)
	}
	const olderID = "11111111-1111-4111-8111-111111111111"
	const duplicateID = "22222222-2222-4222-8222-222222222222"
	const blockerID = "33333333-3333-4333-8333-333333333333"
	firstCandidate := "u_" + strings.ReplaceAll(duplicateID, "-", "")[:25] + "_0000"
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_profiles(user_id, display_name, created_at, updated_at)
		VALUES ($1, 'Alpha', '2024-01-01', '2024-01-01'),
		       ($2, 'alpha', '2024-02-01', '2024-02-01'),
		       ($3, $4, '2024-03-01', '2024-03-01')`, olderID, duplicateID, blockerID, firstCandidate); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, migration6Up); err != nil {
		t.Fatal(err)
	}

	var olderName, duplicateName, blockerName string
	if err := tx.QueryRowContext(ctx, `SELECT
		max(display_name) FILTER (WHERE user_id=$1),
		max(display_name) FILTER (WHERE user_id=$2),
		max(display_name) FILTER (WHERE user_id=$3)
		FROM player_profiles`, olderID, duplicateID, blockerID).Scan(&olderName, &duplicateName, &blockerName); err != nil {
		t.Fatal(err)
	}
	wantDuplicate := "u_" + strings.ReplaceAll(duplicateID, "-", "")[:25] + "_0001"
	if olderName != "Alpha" || duplicateName != wantDuplicate || blockerName != firstCandidate {
		t.Fatalf("renomeação inesperada: antigo=%q duplicado=%q bloqueador=%q",
			olderName, duplicateName, blockerName)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_profiles(user_id, display_name, created_at, updated_at)
		VALUES ('44444444-4444-4444-8444-444444444444', 'ALPHA', now(), now())`); err == nil {
		t.Fatal("índice case-insensitive aceitou duplicata depois da correção")
	}
}

func TestPostgresRejectsIllegalDeckAtCommit(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	deckID, _ := security.NewID()
	ruleset := engine.CompetitiveRuleset()
	championID := ""
	for id := range ruleset.Champions {
		championID = id
		break
	}
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO decks(id,user_id,name,champion_id,ruleset_version)
		VALUES($1,$2,'Ilegal direto no SQL',$3,$4)`, deckID, userID, championID, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	cardID := ""
	for _, card := range ruleset.CardList {
		if card.Confront != nil && card.Confront.Legal {
			cardID = card.ID
			break
		}
	}
	if cardID == "" {
		t.Fatal("catálogo competitivo sem carta legal")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
		VALUES($1,$2,$3,1)`, deckID, cardID, engine.CompetitiveRulesetVersion)
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
	for id := range engine.CompetitiveRuleset().Champions {
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
		DisplayName: "R-" + user1[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `DELETE FROM decks WHERE user_id=$1`, user1); err != nil {
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
	config := engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 99, FirstPlayer: 0,
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
	command := engine.Command{Player: 0, Kind: engine.CmdKindPass}
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

func TestPasswordResetIsSingleUseAndRevokesSessions(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	now := time.Now().UTC()
	sessionID, _ := security.NewID()
	_, accessHash, _ := security.NewToken()
	_, refreshHash, _ := security.NewToken()
	if err := db.CreateSession(ctx, domain.NewSession{ID: sessionID, UserID: userID,
		AccessHash: accessHash, RefreshHash: refreshHash,
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	plain, tokenHash, _ := security.NewToken()
	resetID, _ := security.NewID()
	if err := db.SavePasswordReset(ctx, domain.PasswordResetToken{ID: resetID, UserID: userID,
		TokenHash: tokenHash, ExpiresAt: now.Add(30 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	newHash, err := security.HashPassword("nova-senha-segura-2026")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ConsumePasswordReset(ctx, security.TokenHash(plain), now.Add(time.Minute), newHash); err != nil {
		t.Fatal(err)
	}
	user, err := db.UserByID(ctx, userID)
	if err != nil || !security.VerifyPassword(user.PasswordHash, "nova-senha-segura-2026") {
		t.Fatalf("senha não foi atualizada: err=%v", err)
	}
	if _, err := db.AccessToken(ctx, accessHash, now.Add(2*time.Minute)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sessão anterior sobreviveu à troca: %v", err)
	}
	if err := db.ConsumePasswordReset(ctx, tokenHash, now.Add(2*time.Minute), newHash); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("token usado foi aceito novamente: %v", err)
	}

	expiredPlain, expiredHash, _ := security.NewToken()
	expiredID, _ := security.NewID()
	if err := db.SavePasswordReset(ctx, domain.PasswordResetToken{ID: expiredID, UserID: userID,
		TokenHash: expiredHash, ExpiresAt: now.Add(5 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := db.ConsumePasswordReset(ctx, security.TokenHash(expiredPlain), now.Add(6*time.Minute), newHash); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("token expirado foi aceito: %v", err)
	}
}

func TestEmailDeliveryEventRetriesAreIdempotent(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	event := domain.EmailDeliveryEvent{ProviderEventID: "msg_retry_123", ProviderMessageID: "email_123",
		EventType: "email.delivered", EventCreatedAt: time.Now().UTC()}
	if err := db.SaveEmailDeliveryEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveEmailDeliveryEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM email_delivery_events
		WHERE provider_event_id=$1`, event.ProviderEventID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("retry criou %d registros", count)
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
	// Identidade (migração 000006): username sem espaços e único.
	_, err = db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@example.test",
		DisplayName: "T-" + userID[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	// A conta Confronto nasce com um baralho oficial. A maioria destes testes
	// exercita criação explícita, então remove apenas esse fixture automático.
	if _, err := db.db.ExecContext(ctx, `DELETE FROM decks WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	return ctx, db, userID
}

func TestAdminInviteAndPlayerBanAreTransactional(t *testing.T) {
	ctx, db, playerID := integrationDB(t)

	privilegedID, _ := security.NewID()
	if _, err := db.CreateUser(ctx, domain.User{ID: privilegedID, Email: privilegedID + "@example.test",
		DisplayName: "NaoPode", Role: domain.RoleAdmin, PasswordHash: "test-only"},
		engine.CompetitiveRulesetVersion); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("CreateUser genérico aceitou admin: %v", err)
	}

	ownerID, _ := security.NewID()
	if _, err := db.db.ExecContext(ctx, `INSERT INTO users(id,email,password_hash,role)
		VALUES($1,$2,'test-only','owner')`, ownerID, ownerID+"@owner.test"); err != nil {
		t.Fatal(err)
	}
	inviteID, _ := security.NewID()
	plain, tokenHash, _ := security.NewToken()
	adminEmail := privilegedID + "@admin.test"
	invite, err := db.CreateAdminInvite(ctx, domain.AdminInvite{ID: inviteID, Email: adminEmail,
		TokenHash: tokenHash, CreatedBy: ownerID, ExpiresAt: time.Now().UTC().Add(time.Hour)},
		domain.AuditEntry{Actor: ownerID, Action: "admin_invite:create", Subject: inviteID})
	if err != nil || invite.ID != inviteID {
		t.Fatalf("convite não criado: invite=%+v err=%v", invite, err)
	}
	duplicateInviteID, _ := security.NewID()
	if _, err := db.CreateAdminInvite(ctx, domain.AdminInvite{ID: duplicateInviteID, Email: adminEmail,
		TokenHash: security.TokenHash("segundo-convite-" + duplicateInviteID), CreatedBy: ownerID,
		ExpiresAt: time.Now().UTC().Add(time.Hour)},
		domain.AuditEntry{Actor: ownerID, Action: "admin_invite:create", Subject: duplicateInviteID}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("e-mail recebeu dois convites ativos: %v", err)
	}

	adminID, _ := security.NewID()
	if _, err := db.CreateInvitedAdmin(ctx, security.TokenHash(plain), time.Now().UTC(), domain.User{
		ID: adminID, Email: "outro@example.test", DisplayName: "GuardiaoErrado", Role: domain.RoleAdmin,
		PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("convite aceitou outro e-mail: %v", err)
	}
	admin, err := db.CreateInvitedAdmin(ctx, security.TokenHash(plain), time.Now().UTC(), domain.User{
		ID: adminID, Email: adminEmail, DisplayName: "Guardiao" + adminID[:6], Role: domain.RoleAdmin,
		PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion)
	if err != nil || admin.Role != domain.RoleAdmin {
		t.Fatalf("convite válido não criou admin: user=%+v err=%v", admin, err)
	}
	secondID, _ := security.NewID()
	if _, err := db.CreateInvitedAdmin(ctx, security.TokenHash(plain), time.Now().UTC(), domain.User{
		ID: secondID, Email: adminEmail, DisplayName: "ReusoNegado", Role: domain.RoleAdmin,
		PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("convite foi reutilizado: %v", err)
	}

	now := time.Now().UTC()
	accessHash := security.TokenHash("access-before-ban-" + playerID)
	sessionID, _ := security.NewID()
	if err := db.CreateSession(ctx, domain.NewSession{ID: sessionID, UserID: playerID,
		AccessHash: accessHash, RefreshHash: security.TokenHash("refresh-before-ban-" + playerID),
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	banID, _ := security.NewID()
	if _, err := db.BanPlayer(ctx, domain.PlayerBan{ID: banID, UserID: playerID, Reason: "abuso confirmado",
		CreatedBy: ownerID}, domain.AuditEntry{Actor: ownerID, Action: "player:ban", Subject: playerID}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AccessToken(ctx, accessHash, now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("token banido continuou válido: %v", err)
	}
	blockedID, _ := security.NewID()
	if err := db.CreateSession(ctx, domain.NewSession{ID: blockedID, UserID: playerID,
		AccessHash: security.TokenHash("blocked-access-" + playerID), RefreshHash: security.TokenHash("blocked-refresh-" + playerID),
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(time.Hour)}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("conta banida criou sessão: %v", err)
	}
	if _, err := db.LiftPlayerBan(ctx, playerID, ownerID,
		domain.AuditEntry{Actor: ownerID, Action: "player:unban", Subject: playerID}); err != nil {
		t.Fatal(err)
	}
	allowedID, _ := security.NewID()
	if err := db.CreateSession(ctx, domain.NewSession{ID: allowedID, UserID: playerID,
		AccessHash: security.TokenHash("allowed-access-" + playerID), RefreshHash: security.TokenHash("allowed-refresh-" + playerID),
		AccessUntil: now.Add(time.Hour), RefreshUntil: now.Add(time.Hour)}); err != nil {
		t.Fatalf("conta liberada não criou sessão: %v", err)
	}
}

func TestProfilePasswordAndOAuthPersistence(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	updated, err := db.UpdateProfileAvatar(ctx, userID, "CH-CI-01")
	if err != nil || updated.AvatarID != "CH-CI-01" {
		t.Fatalf("avatar persistido: user=%+v err=%v", updated, err)
	}
	subject := "subject-profile-" + userID
	if err := db.LinkOAuth(ctx, "google", subject, userID); err != nil {
		t.Fatal(err)
	}
	linked, err := db.UserByOAuth(ctx, "google", subject)
	if err != nil || linked.ID != userID {
		t.Fatalf("identidade federada: user=%+v err=%v", linked, err)
	}

	now := time.Now().UTC()
	ticket := security.TokenHash("oauth-ticket-" + userID)
	if err := db.SaveOAuthTicket(ctx, ticket, userID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if consumed, err := db.ConsumeOAuthTicket(ctx, ticket, now); err != nil || consumed != userID {
		t.Fatalf("ticket OAuth não consumido: user=%s err=%v", consumed, err)
	}
	if _, err := db.ConsumeOAuthTicket(ctx, ticket, now); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("ticket OAuth reutilizado: %v", err)
	}

	sessionID, _ := security.NewID()
	accessHash := security.TokenHash("access-before-password-change-" + userID)
	if err := db.CreateSession(ctx, domain.NewSession{ID: sessionID, UserID: userID, AccessHash: accessHash,
		RefreshHash: security.TokenHash("refresh-before-password-change-" + userID), AccessUntil: now.Add(time.Hour),
		RefreshUntil: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AccessToken(ctx, accessHash, now); err != nil {
		t.Fatalf("sessão fixture inválida: %v", err)
	}
	newHash, _ := security.HashPassword("nova-senha-persistida")
	if err := db.ChangePassword(ctx, userID, newHash, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AccessToken(ctx, accessHash, now.Add(2*time.Second)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("troca de senha não revogou sessão: %v", err)
	}
	changed, err := db.UserByID(ctx, userID)
	if err != nil || !security.VerifyPassword(changed.PasswordHash, "nova-senha-persistida") {
		t.Fatalf("senha nova não persistiu: %v", err)
	}
}

func TestSyncCatalogPropagatesAlphaCompleteGrant(t *testing.T) {
	ctx, db, userID := integrationDB(t)

	if _, err := db.db.ExecContext(ctx, `DELETE FROM player_cards
		WHERE user_id=$1 AND card_id='VR-130' AND ruleset_version=$2`, userID, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `DELETE FROM player_champions
		WHERE user_id=$1 AND champion_id='CH-VH-01' AND ruleset_version=$2`, userID, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}

	if err := db.SyncCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	// A segunda sincronização prova que a propagação é idempotente.
	if err := db.SyncCatalog(ctx); err != nil {
		t.Fatal(err)
	}

	var quantity int
	if err := db.db.QueryRowContext(ctx, `SELECT quantity FROM player_cards
		WHERE user_id=$1 AND card_id='VR-130' AND ruleset_version=$2`, userID, engine.CompetitiveRulesetVersion).Scan(&quantity); err != nil {
		t.Fatal(err)
	}
	if quantity != 1 {
		t.Fatalf("VR-130 deveria voltar como Lendária 1x, recebeu %d", quantity)
	}
	var champions int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM player_champions
		WHERE user_id=$1 AND champion_id='CH-VH-01' AND ruleset_version=$2`, userID, engine.CompetitiveRulesetVersion).Scan(&champions); err != nil {
		t.Fatal(err)
	}
	if champions != 1 {
		t.Fatalf("campeão do grant alpha completo deveria existir uma vez, recebeu %d", champions)
	}
	var grants int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM economy_transactions
		WHERE user_id=$1 AND kind='collection_grant' AND payload->>'grant'='alpha_complete'`, userID).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if grants != 1 {
		t.Fatalf("sincronização não deve duplicar a transação de grant, recebeu %d", grants)
	}
}

func TestSyncCatalogDoesNotDuplicateConfrontStarterDeck(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	userID, _ := security.NewID()
	if _, err := db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@sync.test",
		DisplayName: "S-" + userID[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}
	if err := db.SyncCatalog(ctx); err != nil {
		t.Fatalf("primeira sincronização repetida: %v", err)
	}
	if err := db.SyncCatalog(ctx); err != nil {
		t.Fatalf("segunda sincronização repetida: %v", err)
	}
	var decks int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM decks
		WHERE user_id=$1 AND ruleset_version=$2`, userID, engine.CompetitiveRulesetVersion).Scan(&decks); err != nil {
		t.Fatal(err)
	}
	if decks != 1 {
		t.Fatalf("sincronização deveria preservar um único baralho inicial; recebeu %d", decks)
	}
}

func legalIntegrationDeck(t *testing.T, userID, championID, name string) domain.Deck {
	t.Helper()
	ruleset := engine.CompetitiveRuleset()
	precon, err := ruleset.PreconstructedDeck(championID)
	if err != nil {
		t.Fatalf("montar baralho competitivo: %v", err)
	}
	deck := domain.Deck{UserID: userID, Name: name, ChampionID: championID, RulesetVersion: engine.CompetitiveRulesetVersion}
	deck.ID, _ = security.NewID()
	byID := map[string]int{}
	for _, cardID := range precon {
		index, exists := byID[cardID]
		if !exists {
			deck.Cards = append(deck.Cards, domain.DeckCard{CardID: cardID, Quantity: 1})
			byID[cardID] = len(deck.Cards) - 1
			continue
		}
		deck.Cards[index].Quantity++
	}
	if len(precon) != engine.ConfrontDeckSize {
		t.Fatalf("baralho competitivo com %d cartas; esperado %d", len(precon), engine.ConfrontDeckSize)
	}
	return deck
}

func expandIntegrationDeck(deck domain.Deck) []string {
	result := make([]string, 0, engine.ConfrontDeckSize)
	for _, card := range deck.Cards {
		for range card.Quantity {
			result = append(result, card.CardID)
		}
	}
	return result
}

func TestPostgresConfrontStarterDeckLocksAfterFirstEdit(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	userID, _ := security.NewID()
	_, err := db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@confront.test",
		DisplayName: "C-" + userID[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}

	decks, err := db.ListDecks(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(decks) != 1 {
		t.Fatalf("conta nova deveria receber um único deck ativo, recebeu %d", len(decks))
	}
	starter := decks[0]
	if !starter.Active || !starter.SystemProvided || starter.LockedUntil != nil || starter.RulesetVersion != engine.CompetitiveRulesetVersion {
		t.Fatalf("deck inicial inválido: %+v", starter)
	}
	if got := len(expandIntegrationDeck(starter)); got != engine.ConfrontDeckSize {
		t.Fatalf("deck inicial deveria ter %d cartas, recebeu %d", engine.ConfrontDeckSize, got)
	}
	if err := engine.ValidateDeckForVersion(starter.RulesetVersion, starter.ChampionID, expandIntegrationDeck(starter)); err != nil {
		t.Fatalf("deck inicial ilegal: %v", err)
	}

	lockedUntil := time.Now().UTC().Add(24 * time.Hour)
	starter.Name = "Meu baralho definitivo"
	starter.SystemProvided = false
	starter.LockedUntil = &lockedUntil
	expected := starter.Version
	mutation := domain.Mutation{Key: "confront-first-edit-" + userID, Operation: "deck:update:" + starter.ID,
		RequestHash: []byte("first-edit")}
	saved, replayed, err := db.SaveDeck(ctx, starter, &expected, mutation)
	if err != nil || replayed {
		t.Fatalf("primeira edição do deck inicial: replay=%v err=%v", replayed, err)
	}
	if saved.SystemProvided || saved.LockedUntil == nil || !saved.Active {
		t.Fatalf("deck editado não foi travado/ativado: %+v", saved)
	}

	replayedDeck, replayed, err := db.SaveDeck(ctx, starter, &expected, mutation)
	if err != nil || !replayed || replayedDeck.ID != saved.ID {
		t.Fatalf("retry idempotente durante trava: replay=%v err=%v", replayed, err)
	}
	secondExpected := saved.Version
	saved.Name = "Tentativa antes de 24h"
	if _, _, err := db.SaveDeck(ctx, saved, &secondExpected, domain.Mutation{
		Key: "confront-second-edit-" + userID, Operation: "deck:update:" + saved.ID,
		RequestHash: []byte("second-edit"),
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("segunda edição antes do prazo deveria falhar com conflito, recebeu %v", err)
	}
}

func TestPostgresRejectsConfrontDeckWithoutMinimumComposition(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	userID, _ := security.NewID()
	_, err := db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@composition.test",
		DisplayName: "M-" + userID[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	decks, err := db.ListDecks(ctx, userID)
	if err != nil || len(decks) != 1 {
		t.Fatalf("deck inicial: %d err=%v", len(decks), err)
	}
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM deck_cards dc USING card_definitions cd
		WHERE dc.deck_id=$1 AND cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
		  AND cd.card_type='Rito'`, decks[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO deck_cards(deck_id,card_id,ruleset_version,quantity)
		SELECT $1,cd.id,$2,2 FROM card_definitions cd
		WHERE cd.ruleset_version=$2 AND cd.card_type='Assalto'
		  AND COALESCE((cd.definition->'confront'->>'legal')::boolean,false)
		  AND NOT EXISTS (SELECT 1 FROM deck_cards dc WHERE dc.deck_id=$1 AND dc.card_id=cd.id)
		ORDER BY cd.id LIMIT 5`, decks[0].ID, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err == nil {
		t.Fatal("Postgres confirmou deck de 30 cartas sem nenhum Rito")
	}
}

// Fase 9: publicar → rotacionar clona coleção e decks válidos para a nova
// versão, com idempotência (segunda rotação não duplica).
func TestPostgresRotateToRuleset(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	userID, _ := security.NewID()
	if _, err := db.CreateUser(ctx, domain.User{ID: userID, Email: userID + "@rotate.test",
		DisplayName: "R-" + userID[:8], Role: domain.RolePlayer, PasswordHash: "test-only"}, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}

	// Publica uma versão nova a partir do snapshot do embutido.
	payload, err := db.RulesetPayload(ctx, engine.CompetitiveRulesetVersion)
	if err != nil {
		t.Fatal(err)
	}
	next := "alpha-rotate-" + userID[:8]
	var fx engine.EffectsFile
	if err := json.Unmarshal(payload.Effects, &fx); err != nil {
		t.Fatal(err)
	}
	fx.Version = next
	effects, _ := json.Marshal(&fx)
	newPayload := domain.RulesetPayload{Version: next, Cards: payload.Cards,
		Champions: payload.Champions, Effects: effects}
	audit := domain.AuditEntry{Actor: userID, Action: "test", Subject: next}
	if err := db.PublishRuleset(ctx, newPayload, "", audit); err != nil {
		t.Fatal(err)
	}
	rs, err := engine.CompileRuleset(next, newPayload.Cards, newPayload.Champions, newPayload.Effects)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterRuleset(rs); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engine.UnregisterRuleset(next) })

	// CreateUser semeou o único baralho atual de 30 cartas; ele deve continuar
	// legal e ser clonado na versão publicada.

	granted, cloned, err := db.RotateToRuleset(ctx, next, audit)
	if err != nil {
		t.Fatal(err)
	}
	if granted == 0 || cloned == 0 {
		t.Fatalf("rotação vazia: granted=%d cloned=%d", granted, cloned)
	}
	decks, err := db.ListDecks(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range decks {
		if d.RulesetVersion == next && strings.Contains(d.Name, next) {
			found = true
			if len(d.Cards) == 0 {
				t.Fatal("clone sem cartas")
			}
		}
	}
	if !found {
		t.Fatalf("deck clonado não encontrado: %+v", decks)
	}

	// Idempotência: repetir não duplica decks nem coleção.
	granted2, cloned2, err := db.RotateToRuleset(ctx, next, audit)
	if err != nil {
		t.Fatal(err)
	}
	if granted2 != 0 || cloned2 != 0 {
		t.Fatalf("segunda rotação deveria ser vazia: granted=%d cloned=%d", granted2, cloned2)
	}
}

// P1: gravação de progresso — rituais, carteira auditada, maestria, rating e
// idempotência por partida, contra Postgres real.
func TestPostgresRecordMatchProgress(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	opponentID, _ := security.NewID()
	if _, err := db.CreateUser(ctx, domain.User{ID: opponentID, Email: opponentID + "@ex.test",
		DisplayName: "R-" + opponentID[:8], Role: domain.RolePlayer, PasswordHash: "x"}, engine.CompetitiveRulesetVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `DELETE FROM decks WHERE user_id=$1`, opponentID); err != nil {
		t.Fatal(err)
	}
	season, err := db.ActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	deck0 := legalIntegrationDeck(t, userID, "CH-VH-01", "Progresso 0")
	deck1 := legalIntegrationDeck(t, opponentID, "CH-SO-01", "Progresso 1")
	for i, deck := range []*domain.Deck{&deck0, &deck1} {
		mutation := domain.Mutation{Key: "progress-deck-" + string(rune('1'+i)),
			Operation: "deck:create", RequestHash: []byte{byte(i + 40)}}
		if _, _, err := db.SaveDeck(ctx, *deck, nil, mutation); err != nil {
			t.Fatal(err)
		}
	}
	matchID, _ := security.NewID()
	config := engine.Config{RulesetVersion: engine.CompetitiveRulesetVersion, Seed: 42, FirstPlayer: 0,
		Players: [2]engine.PlayerSetup{{ChampionID: deck0.ChampionID, Deck: expandIntegrationDeck(deck0)},
			{ChampionID: deck1.ChampionID, Deck: expandIntegrationDeck(deck1)}}}
	match := battle.Match{ID: matchID, Mode: "pvp", Config: config,
		Status: battle.StatusWaitingReady, CreatedAt: time.Now().UTC(),
		Players: [2]battle.Participant{{UserID: userID, DeckID: deck0.ID, Slot: 0},
			{UserID: opponentID, DeckID: deck1.ID, Slot: 1}}}
	if err := db.CreateMatch(ctx, match); err != nil {
		t.Fatal(err)
	}

	progress := domain.MatchProgress{MatchID: matchID, Ranked: true, SeasonID: season.ID,
		Players: []domain.PlayerMatchProgress{
			{UserID: userID, ChampionID: "CH-VH-01", Won: true, MasteryXP: 30, AccountXP: 30, RitualDay: "2026-08-10",
				Rituals: []domain.RitualIncrement{{RitualID: "win_pvp_1", Delta: 1, Target: 1, Reward: 60}}},
			{UserID: opponentID, ChampionID: "CH-SO-01", Won: false, MasteryXP: 15, AccountXP: 15, RitualDay: "2026-08-10"},
		}}
	first, err := db.RecordMatchProgress(ctx, progress)
	if err != nil {
		t.Fatal(err)
	}
	if !first {
		t.Fatal("primeira gravação deveria creditar")
	}
	again, err := db.RecordMatchProgress(ctx, progress)
	if err != nil || again {
		t.Fatalf("regressão de idempotência: again=%v err=%v", again, err)
	}

	fragments, err := db.Fragments(ctx, userID)
	if err != nil || fragments != 60 {
		t.Fatalf("fragmentos: %d err=%v; esperado 60", fragments, err)
	}
	rituals, err := db.RitualsFor(ctx, userID, "2026-08-10")
	if err != nil || len(rituals) != 1 || rituals[0].CompletedAt == nil {
		t.Fatalf("ritual deveria estar completo: %+v err=%v", rituals, err)
	}
	winner, _ := db.RankedStandingFor(ctx, userID, season.ID)
	loser, _ := db.RankedStandingFor(ctx, opponentID, season.ID)
	if winner.Rating != 1016 || loser.Rating != 984 {
		t.Fatalf("elo: %d/%d; esperado 1016/984", winner.Rating, loser.Rating)
	}
	board, err := db.Leaderboard(ctx, season.ID, 1000)
	if err != nil || len(board) < 2 {
		t.Fatalf("leaderboard: %+v err=%v", board, err)
	}
	foundWinner := false
	for i, row := range board {
		if i > 0 && board[i-1].Rating < row.Rating {
			t.Fatalf("leaderboard fora de ordem em %d: %+v", i, board)
		}
		if row.UserID == userID && row.Rating == 1016 {
			foundWinner = true
		}
	}
	if !foundWinner {
		t.Fatalf("vencedor não encontrado no leaderboard: %+v", board)
	}
	mastery, err := db.MasteryFor(ctx, userID)
	if err != nil || len(mastery) != 1 || mastery[0].XP != 30 || mastery[0].Wins != 1 {
		t.Fatalf("maestria: %+v err=%v", mastery, err)
	}
	accountXP, err := db.AccountXP(ctx, userID)
	if err != nil || accountXP != 30 {
		t.Fatalf("XP global: %d err=%v; esperado 30", accountXP, err)
	}
}

// Regressão: a telemetria juntava match_players a uma coluna inexistente
// (champion_id); o campeão vem do deck. Pego pelo painel LiveOps.
func TestPostgresMatchTelemetryQueryRuns(t *testing.T) {
	ctx, db, _ := integrationDB(t)
	matchID, _ := security.NewID()
	endedAt := time.Now().UTC()
	startedAt := endedAt.Add(-31 * time.Minute)
	if _, err := db.db.ExecContext(ctx, `INSERT INTO matches
		(id,ruleset_version,seed,config,status,mode,created_at,started_at,ended_at)
		VALUES($1,$2,1,'{}','finished','pvp',$3,$3,$4)`, matchID,
		engine.CompetitiveRulesetVersion, startedAt, endedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `INSERT INTO match_commands
		(match_id,command_index,player_slot,origin,command) VALUES($1,0,-1,'system','{}')`, matchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `INSERT INTO match_events
		(match_id,event_seq,command_index,event) VALUES($1,0,0,'{"round":37}')`, matchID); err != nil {
		t.Fatal(err)
	}
	telemetry, err := db.MatchTelemetry(ctx)
	if err != nil {
		t.Fatalf("telemetria: %v", err)
	}
	if telemetry.TotalMatches < 0 {
		t.Fatal("contagem inválida")
	}
	if telemetry.Rhythm.SampleMatches < 1 || telemetry.Rhythm.AverageDurationSeconds <= 0 ||
		telemetry.Rhythm.AverageRounds <= 0 || telemetry.Rhythm.OverThirtyMinutes < 1 {
		t.Fatalf("ritmo inválido: %+v", telemetry.Rhythm)
	}
}

// ADR-034: fechar temporada concede Fragmentos pela patente final, na mesma
// transação e apenas na transição real (idempotente por construção).
func TestPostgresSeasonCloseGrantsTierRewards(t *testing.T) {
	ctx, db, userID := integrationDB(t)
	season, err := db.ActiveSeason(ctx)
	if err != nil {
		t.Fatalf("temporada ativa: %v", err)
	}
	// Semeia um ranqueado: rating 1210 (Arauto do Eclipse → 130 fragmentos).
	if _, err := db.db.ExecContext(ctx, `INSERT INTO ranked_ratings(user_id,season_id,rating,games,wins)
		VALUES($1,$2,1210,8,5)`, userID, season.ID); err != nil {
		t.Fatalf("seed rating: %v", err)
	}
	before, err := db.Fragments(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}

	newSeasonID, _ := security.NewID()
	_, err = db.CreateSeason(ctx, domain.Season{ID: newSeasonID, Name: "Temporada Prova",
		RulesetVersion: engine.CompetitiveRulesetVersion, StartsAt: time.Now().UTC()},
		domain.AuditEntry{Actor: userID, Action: "season:create", Subject: newSeasonID})
	if err != nil {
		t.Fatalf("virada de temporada: %v", err)
	}

	after, err := db.Fragments(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if after-before != 130 {
		t.Fatalf("fragmentos: %d → %d; esperado +130 (Arauto)", before, after)
	}
	var kind, source string
	if err := db.db.QueryRowContext(ctx, `SELECT kind, source FROM economy_transactions
		WHERE user_id=$1 AND source='season_reward' ORDER BY created_at DESC LIMIT 1`, userID).
		Scan(&kind, &source); err != nil {
		t.Fatalf("trilha econômica ausente: %v", err)
	}
	var audits int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM admin_audit
		WHERE action='season:rewards' AND subject=$1`, season.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("auditoria da concessão: %d entradas; esperado 1", audits)
	}

	// Segunda virada: a temporada antiga já está fechada — nada é reconcedido.
	thirdID, _ := security.NewID()
	if _, err := db.CreateSeason(ctx, domain.Season{ID: thirdID, Name: "Temporada Prova 2",
		RulesetVersion: engine.CompetitiveRulesetVersion, StartsAt: time.Now().UTC().Add(time.Second)},
		domain.AuditEntry{Actor: userID, Action: "season:create", Subject: thirdID}); err != nil {
		t.Fatalf("segunda virada: %v", err)
	}
	final, err := db.Fragments(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if final != after {
		t.Fatalf("reconcessão indevida: %d → %d", after, final)
	}
}
