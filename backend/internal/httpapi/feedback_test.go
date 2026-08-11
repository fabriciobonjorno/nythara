package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"veurubro/backend/internal/domain"
)

func postWithToken(t *testing.T, handler http.Handler, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

// O recado é opcional, mas o identificador da partida vem do cliente: ele não
// pode virar ponteiro para a partida de outra pessoa no relatório do Alpha.
func TestFeedbackAcceptsOwnMatchAndRejectsOthers(t *testing.T) {
	handler, store := testHandler()
	store.replay = &domain.MatchReplayData{
		MatchID: "m1", Status: "finished",
		Players: [2]domain.ReplayPlayer{
			{UserID: playerUserID, DisplayName: "Jogador", ChampionID: "CH-CI-01"},
			{UserID: domain.BotUserID, DisplayName: "Treinador do Véu", ChampionID: "CH-VA-02"},
		},
		Events: []json.RawMessage{json.RawMessage(`{"seq":0,"kind":"match_started"}`)},
	}

	response := postWithToken(t, handler, "/v1/feedback", "player-token",
		`{"match_id":"m1","message":"A leitura da mesa ficou clara, mas o texto da carta some rápido."}`)
	if response.Code != http.StatusNoContent {
		t.Fatalf("participante: status=%d body=%s", response.Code, response.Body.String())
	}
	if len(store.feedback) != 1 || store.feedback[0].UserID != playerUserID {
		t.Fatalf("recado não persistido: %+v", store.feedback)
	}
	if store.feedback[0].RulesetVersion == "" {
		t.Fatal("recado sem a versão de regras em que foi escrito")
	}

	// Partida de terceiros é recusada mesmo com sessão válida.
	other := *store.replay
	other.Players[0].UserID = "00000000-0000-4000-8000-0000000000ff"
	store.replay = &other
	if response := postWithToken(t, handler, "/v1/feedback", "player-token",
		`{"match_id":"m1","message":"tentativa"}`); response.Code != http.StatusForbidden {
		t.Fatalf("partida alheia: status=%d", response.Code)
	}

	// Recado vazio não vira linha no banco.
	if response := postWithToken(t, handler, "/v1/feedback", "player-token",
		`{"message":"   "}`); response.Code != http.StatusBadRequest {
		t.Fatalf("recado vazio: status=%d body=%s", response.Code, response.Body.String())
	}
	if len(store.feedback) != 1 {
		t.Fatalf("recado inválido foi persistido: %+v", store.feedback)
	}

	// Sem sessão, nem chega ao serviço.
	request := httptest.NewRequest(http.MethodPost, "/v1/feedback", strings.NewReader(`{"message":"oi"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anônimo: status=%d", response.Code)
	}
}
