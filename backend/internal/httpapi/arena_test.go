package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

const playerUserID = "00000000-0000-4000-8000-000000000010"

func getWithToken(t *testing.T, handler http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

// A crônica expõe o log completo — a autorização é o teste que importa.
func TestMatchReplayAuthorization(t *testing.T) {
	handler, store := testHandler()
	winner := 0
	replay := domain.MatchReplayData{
		MatchID: "m1", Mode: "practice", Status: "finished", Winner: &winner,
		Players: [2]domain.ReplayPlayer{
			{UserID: playerUserID, DisplayName: "Jogador", ChampionID: "CH-CI-01"},
			{UserID: domain.BotUserID, DisplayName: "Treinador do Véu", ChampionID: "CH-VA-02"},
		},
		RulesetVersion: engine.CompetitiveRulesetVersion,
		Events:         []json.RawMessage{json.RawMessage(`{"seq":0,"kind":"match_started"}`)},
	}
	store.replay = &replay

	// Participante vê a crônica.
	response := getWithToken(t, handler, "/v1/matches/m1/replay", "player-token")
	if response.Code != http.StatusOK {
		t.Fatalf("participante: status=%d body=%s", response.Code, response.Body.String())
	}
	var got domain.MatchReplayData
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || len(got.Events) != 1 {
		t.Fatalf("crônica truncada: %v %s", err, response.Body.String())
	}

	// A escala da Vitalidade viaja com a partida: o cliente não pode cair na
	// Vitalidade legada do Campeão, que o Modo Confronto sobrescreve.
	if got.StartingVitality != engine.CompetitiveRuleset().ConfrontRules.StartingVitality {
		t.Fatalf("crônica sem a base de Vitalidade do ruleset: %d", got.StartingVitality)
	}

	// Partida viva jamais vaza o log, nem para o participante.
	live := replay
	live.Status = "active"
	store.replay = &live
	if response := getWithToken(t, handler, "/v1/matches/m1/replay", "player-token"); response.Code != http.StatusForbidden {
		t.Fatalf("partida viva: status=%d", response.Code)
	}

	// Não participante (mesmo autenticado) é barrado — admin passa.
	other := replay
	other.Players[0].UserID = "00000000-0000-4000-8000-0000000000ff"
	store.replay = &other
	if response := getWithToken(t, handler, "/v1/matches/m1/replay", "player-token"); response.Code != http.StatusForbidden {
		t.Fatalf("intruso: status=%d", response.Code)
	}
	if response := getWithToken(t, handler, "/v1/matches/m1/replay", "admin-token"); response.Code != http.StatusOK {
		t.Fatalf("admin: status=%d", response.Code)
	}
}

func TestMatchHistoryAndTiers(t *testing.T) {
	handler, _ := testHandler()

	response := getWithToken(t, handler, "/v1/matches?limit=5", "player-token")
	if response.Code != http.StatusOK {
		t.Fatalf("histórico: status=%d body=%s", response.Code, response.Body.String())
	}
	var history struct {
		Matches []domain.MatchSummary `json:"matches"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil || len(history.Matches) != 1 {
		t.Fatalf("histórico inesperado: %v %s", err, response.Body.String())
	}
	if !history.Matches[0].Won || history.Matches[0].Opponent == "" {
		t.Fatalf("resumo incompleto: %+v", history.Matches[0])
	}

	// Leaderboard ganha patentes derivadas (rating 1210 → Arauto do Eclipse;
	// solicitante 1012 → Lâmina Velada).
	response = getWithToken(t, handler, "/v1/ranked/leaderboard", "player-token")
	if response.Code != http.StatusOK {
		t.Fatalf("leaderboard: status=%d body=%s", response.Code, response.Body.String())
	}
	var board struct {
		Entries []domain.LeaderboardEntry `json:"entries"`
		Me      *domain.RankedStanding    `json:"me"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &board); err != nil {
		t.Fatal(err)
	}
	if len(board.Entries) == 0 || board.Entries[0].Tier != "Arauto do Eclipse" {
		t.Fatalf("patente do topo: %+v", board.Entries)
	}
	if board.Me == nil || board.Me.Tier == nil || board.Me.Tier.Key != "lamina" || board.Me.Tier.NextAt != 1100 {
		t.Fatalf("patente do solicitante: %+v", board.Me)
	}
}
