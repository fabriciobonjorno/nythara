package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"veurubro/backend/internal/app"
	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

func TestAuthorizationGate(t *testing.T) {
	handler, store := testHandler()

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(
		`{"email":"jogador@example.test","password":"senha-muito-forte","display_name":"Jogador","role":"admin"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("autoelevação no cadastro: status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/collection", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("sem token: status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/admin/rewards/grant", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer player-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("player em rota admin: status=%d body=%s", response.Code, response.Body.String())
	}
	if store.grantCalls != 0 {
		t.Fatal("rota admin chegou à economia antes do RBAC")
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("admin autenticado: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCatalogPublishesLegendaryUnlockLevels(t *testing.T) {
	handler, _ := testHandler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/catalog/cards", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Cards []struct {
			ID          string `json:"id"`
			Rarity      string `json:"rarity"`
			UnlockLevel int    `json:"unlock_level"`
		} `json:"cards"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, card := range payload.Cards {
		if card.ID == "VR-012" {
			found = card.Rarity == "Lendária" && card.UnlockLevel == 10
		}
		if card.Rarity != "Lendária" && card.UnlockLevel != 0 {
			t.Fatalf("carta não-Lendária %s recebeu nível %d", card.ID, card.UnlockLevel)
		}
	}
	if !found {
		t.Fatalf("VR-012 sem marco de nível: %+v", payload.Cards)
	}
}

func TestResendWebhookVerifiesRawBodyAndPersistsMinimalMetadata(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(key)
	store := &fakeStore{tokens: map[string]domain.Principal{}}
	service := app.New(store)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewWithOptions(service, nil, logger, func(context.Context) error { return nil },
		Options{ResendWebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"type":"email.delivered","created_at":"2026-08-12T17:00:00Z","data":{"email_id":"email_123","to":["private@example.test"]}}`)
	id := "msg_webhook_123"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := signWebhookHTTPTest(key, id, timestamp, raw)
	request := httptest.NewRequest(http.MethodPost, "/v1/webhooks/resend", bytes.NewReader(raw))
	request.Header.Set("svix-id", id)
	request.Header.Set("svix-timestamp", timestamp)
	request.Header.Set("svix-signature", "v1,"+signature)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(store.emailEvents) != 1 {
		t.Fatalf("status=%d body=%s eventos=%+v", response.Code, response.Body.String(), store.emailEvents)
	}
	event := store.emailEvents[0]
	if event.ProviderEventID != id || event.ProviderMessageID != "email_123" || event.EventType != "email.delivered" {
		t.Fatalf("metadados inesperados: %+v", event)
	}

	tampered := httptest.NewRequest(http.MethodPost, "/v1/webhooks/resend", bytes.NewReader(append(raw, ' ')))
	tampered.Header = request.Header.Clone()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, tampered)
	if response.Code != http.StatusBadRequest || len(store.emailEvents) != 1 {
		t.Fatalf("payload adulterado: status=%d eventos=%d", response.Code, len(store.emailEvents))
	}
}

func signWebhookHTTPTest(key []byte, id, timestamp string, raw []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(id + "." + timestamp + "." + string(raw)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type noopRecoverySender struct{}

func (noopRecoverySender) SendPasswordReset(context.Context, string, string, string, time.Duration) error {
	return nil
}

func TestPasswordRecoveryHTTPDoesNotEnumerateAndConsumesValidToken(t *testing.T) {
	handler, store := testHandler()
	store.resetErr = domain.ErrNotFound
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"missing@example.test","locale":"pt-BR"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("conta ausente foi enumerada: status=%d body=%s", response.Code, response.Body.String())
	}

	store.resetErr = nil
	token, _, _ := security.NewToken()
	request = httptest.NewRequest(http.MethodPost, "/v1/auth/reset-password",
		bytes.NewBufferString(`{"token":"`+token+`","password":"nova-senha-segura-2026"}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !store.resetPassword {
		t.Fatalf("reset válido: status=%d body=%s consumed=%t", response.Code, response.Body.String(), store.resetPassword)
	}
}

func TestIllegalDeckNeverReachesStore(t *testing.T) {
	handler, store := testHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/decks", bytes.NewBufferString(
		`{"name":"Vazio","champion_id":"CH-VH-01","cards":[]}`))
	request.Header.Set("Authorization", "Bearer player-token")
	request.Header.Set("Idempotency-Key", "deck-vazio-001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("deck ilegal: status=%d body=%s", response.Code, response.Body.String())
	}
	if store.saveDeckCalls != 0 {
		t.Fatal("deck ilegal alcançou a camada de persistência")
	}
}

func TestBattleProtocolRejectsClientSuppliedResultsAndPlayer(t *testing.T) {
	malicious := []string{
		`{"type":"command","client_sequence":1,"command":{"kind":"play","card":"p0-c01","player":1}}`,
		`{"type":"command","client_sequence":1,"command":{"kind":"play","card":"p0-c01","damage":99}}`,
		`{"type":"command","client_sequence":1,"command":{"kind":"pass"},"winner":0}`,
		`{"type":"command","client_sequence":1,"command":{"kind":"concede","reason":"timeout"}}`,
	}
	for _, raw := range malicious {
		if _, err := decodeBattleMessage([]byte(raw)); err == nil {
			t.Fatalf("payload adulterado foi aceito: %s", raw)
		}
	}
	valid := `{"type":"command","client_sequence":7,"command":{"kind":"play","card":"p0-c01"}}`
	message, err := decodeBattleMessage([]byte(valid))
	if err != nil || message.Command == nil || message.Command.Card != "p0-c01" {
		t.Fatalf("intenção válida rejeitada: message=%+v err=%v", message, err)
	}
}

func TestBattleWebSocketUpgradeAndReadOnlySpectator(t *testing.T) {
	store := &wireBattleStore{loaded: battle.LoadedMatch{Match: battle.Match{
		ID: "00000000-0000-4000-8000-000000000301", Status: battle.StatusWaitingReady,
		Players: [2]battle.Participant{{UserID: "u0", Slot: 0}, {UserID: "u1", Slot: 1}},
	}}}
	manager := battle.NewManager(store)
	admin := domain.Principal{UserID: "admin", Role: domain.RoleAdmin}
	ticket, err := manager.IssueTicket(context.Background(), admin, store.loaded.Match.ID, battle.TicketSpectator)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("ambiente não permite listener local: %v", err)
	}
	server := httptest.NewUnstartedServer(New(nil, manager, logger, nil))
	server.Listener = listener
	server.Start()
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/battles/" + store.loaded.Match.ID + "/ws?ticket=" + ticket.Token
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.CloseNow()
	if message := readBattleServerMessage(t, ctx, connection); message.Type != "sync" {
		t.Fatalf("primeira mensagem=%s; esperado sync", message.Type)
	}
	malicious := []byte(`{"type":"command","client_sequence":1,"command":{"kind":"pass","winner":0}}`)
	if err := connection.Write(ctx, websocket.MessageText, malicious); err != nil {
		t.Fatal(err)
	}
	if message := readBattleServerMessage(t, ctx, connection); message.Error == nil || message.Error.Code != "invalid_message" {
		t.Fatalf("cheat não rejeitado: %+v", message)
	}
	if err := connection.Write(ctx, websocket.MessageText, []byte(`{"type":"ready"}`)); err != nil {
		t.Fatal(err)
	}
	if message := readBattleServerMessage(t, ctx, connection); message.Error == nil || message.Error.Code != "spectator_read_only" {
		t.Fatalf("espectador escreveu: %+v", message)
	}
}

func readBattleServerMessage(t *testing.T, ctx context.Context, connection *websocket.Conn) battle.ServerMessage {
	t.Helper()
	messageType, raw, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageText {
		t.Fatalf("mensagem WebSocket não textual: %v", messageType)
	}
	var message battle.ServerMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		t.Fatal(err)
	}
	return message
}

func testHandler() (http.Handler, *fakeStore) {
	store := &fakeStore{tokens: map[string]domain.Principal{
		string(security.TokenHash("player-token")): {UserID: "00000000-0000-4000-8000-000000000010", Role: domain.RolePlayer},
		string(security.TokenHash("admin-token")):  {UserID: "00000000-0000-4000-8000-000000000020", Role: domain.RoleAdmin},
		string(security.TokenHash("owner-token")):  {UserID: "00000000-0000-4000-8000-000000000030", Role: domain.RoleOwner},
	}}
	service := app.NewWithClock(store, func() time.Time { return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(service, nil, logger, func(context.Context) error { return nil }), store
}

var _ domain.Store = (*fakeStore)(nil)

type fakeStore struct {
	tokens        map[string]domain.Principal
	saveDeckCalls int
	grantCalls    int
	liveops       *liveopsState
	replay        *domain.MatchReplayData
	feedback      []domain.Feedback
	resetUser     domain.User
	resetErr      error
	resetSaved    domain.PasswordResetToken
	resetPassword bool
	emailEvents   []domain.EmailDeliveryEvent
}

func (*fakeStore) CreateUser(context.Context, domain.User, string) (domain.User, error) {
	panic("not used")
}
func (*fakeStore) CreateInvitedAdmin(context.Context, []byte, time.Time, domain.User, string) (domain.User, error) {
	panic("not used")
}
func (f *fakeStore) UserByEmail(context.Context, string) (domain.User, error) {
	return f.resetUser, f.resetErr
}
func (*fakeStore) UserByID(context.Context, string) (domain.User, error) { panic("not used") }
func (*fakeStore) UserByOAuth(context.Context, string, string) (domain.User, error) {
	panic("not used")
}
func (*fakeStore) LinkOAuth(context.Context, string, string, string) error { panic("not used") }
func (*fakeStore) SaveOAuthTicket(context.Context, []byte, string, time.Time) error {
	panic("not used")
}
func (*fakeStore) ConsumeOAuthTicket(context.Context, []byte, time.Time) (string, error) {
	panic("not used")
}
func (*fakeStore) UpdateProfileAvatar(context.Context, string, string) (domain.User, error) {
	panic("not used")
}
func (*fakeStore) ChangePassword(context.Context, string, string, time.Time) error {
	panic("not used")
}
func (*fakeStore) CreateSession(context.Context, domain.NewSession) error { panic("not used") }
func (f *fakeStore) AccessToken(_ context.Context, hash []byte, _ time.Time) (domain.TokenRecord, error) {
	principal, ok := f.tokens[string(hash)]
	if !ok {
		return domain.TokenRecord{}, domain.ErrNotFound
	}
	return domain.TokenRecord{Principal: principal}, nil
}
func (*fakeStore) RotateSession(context.Context, []byte, domain.RotatedSession, time.Time) (string, error) {
	panic("not used")
}
func (*fakeStore) RevokeSession(context.Context, []byte) error { panic("not used") }
func (f *fakeStore) SavePasswordReset(_ context.Context, reset domain.PasswordResetToken) error {
	f.resetSaved = reset
	return nil
}
func (f *fakeStore) ConsumePasswordReset(context.Context, []byte, time.Time, string) error {
	f.resetPassword = true
	return f.resetErr
}
func (f *fakeStore) SaveEmailDeliveryEvent(_ context.Context, event domain.EmailDeliveryEvent) error {
	f.emailEvents = append(f.emailEvents, event)
	return nil
}
func (*fakeStore) Collection(context.Context, string, string) (domain.Collection, error) {
	return domain.Collection{}, nil
}
func (*fakeStore) ListDecks(context.Context, string) ([]domain.Deck, error)  { panic("not used") }
func (*fakeStore) Deck(context.Context, string, string) (domain.Deck, error) { panic("not used") }
func (f *fakeStore) SaveDeck(context.Context, domain.Deck, *int64, domain.Mutation) (domain.Deck, bool, error) {
	f.saveDeckCalls++
	return domain.Deck{}, false, nil
}
func (*fakeStore) DeleteDeck(context.Context, string, string, int64, domain.Mutation) (bool, error) {
	panic("not used")
}
func (*fakeStore) ActiveSeason(context.Context) (domain.Season, error) {
	return domain.Season{ID: "00000000-0000-4000-8000-000000000003", Name: "Alpha 0.5"}, nil
}
func (*fakeStore) ListRewards(context.Context, string) ([]domain.Reward, error) { panic("not used") }
func (f *fakeStore) GrantReward(context.Context, domain.Reward, domain.Mutation) (domain.Reward, bool, error) {
	f.grantCalls++
	return domain.Reward{}, false, nil
}

type wireBattleStore struct{ loaded battle.LoadedMatch }

func (*wireBattleStore) ActiveBans(context.Context) ([]domain.CardBan, error) { return nil, nil }

func (s *wireBattleStore) CreateMatch(context.Context, battle.Match) error { return nil }
func (s *wireBattleStore) MarkReady(context.Context, string, int) error    { return nil }
func (s *wireBattleStore) StartMatch(context.Context, string, []engine.Event, battle.Snapshot) error {
	return nil
}
func (s *wireBattleStore) PersistStep(context.Context, battle.Step) error { return nil }
func (s *wireBattleStore) CancelMatch(context.Context, string, string) error {
	return nil
}
func (s *wireBattleStore) LoadMatch(context.Context, string) (battle.LoadedMatch, error) {
	return s.loaded, nil
}
func (s *wireBattleStore) ActiveMatchForUser(context.Context, string) (battle.Match, int, error) {
	return battle.Match{}, 0, domain.ErrNotFound
}
