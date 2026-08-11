package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

// liveopsState guarda o lado admin do fakeStore em memória.
type liveopsState struct {
	mu       sync.Mutex
	rulesets []domain.RulesetInfo
	payloads map[string]domain.RulesetPayload
	drafts   map[string]domain.CardDraft
	bans     map[string]domain.CardBan
	seasons  []domain.Season
	audit    []domain.AuditEntry
	rituals     map[string][]domain.RitualState
	progressLog map[string]domain.MatchProgress
}

func newLiveopsState() *liveopsState {
	cards, _ := json.Marshal(engine.CardList)
	var champs []*engine.ChampionDef
	for _, id := range []string{"CH-VH-01", "CH-VH-02", "CH-SO-01", "CH-SO-02", "CH-MI-01",
		"CH-MI-02", "CH-VA-01", "CH-VA-02", "CH-CI-01", "CH-CI-02"} {
		champs = append(champs, engine.Champions[id])
	}
	champsRaw, _ := json.Marshal(champs)
	effects, _ := json.Marshal(engine.Effects)
	return &liveopsState{
		rulesets: []domain.RulesetInfo{{Version: engine.RulesetVersion, Active: true}},
		payloads: map[string]domain.RulesetPayload{
			engine.RulesetVersion: {Version: engine.RulesetVersion, Cards: cards,
				Champions: champsRaw, Effects: effects},
		},
		drafts: map[string]domain.CardDraft{},
		bans:   map[string]domain.CardBan{},
	}
}

func (f *fakeStore) ops() *liveopsState {
	if f.liveops == nil {
		f.liveops = newLiveopsState()
	}
	return f.liveops
}

func (f *fakeStore) ListRulesets(context.Context) ([]domain.RulesetInfo, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.RulesetInfo{}, s.rulesets...), nil
}

func (f *fakeStore) RulesetPayload(_ context.Context, version string) (domain.RulesetPayload, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, ok := s.payloads[version]
	if !ok {
		return domain.RulesetPayload{}, domain.ErrNotFound
	}
	return payload, nil
}

func (f *fakeStore) ListRulesetPayloads(context.Context) ([]domain.RulesetPayload, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.RulesetPayload, 0, len(s.payloads))
	for _, payload := range s.payloads {
		out = append(out, payload)
	}
	return out, nil
}

func (f *fakeStore) PublishRuleset(_ context.Context, payload domain.RulesetPayload,
	draftID string, audit domain.AuditEntry) error {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.payloads[payload.Version]; exists {
		return domain.ErrConflict
	}
	s.payloads[payload.Version] = payload
	s.rulesets = append(s.rulesets, domain.RulesetInfo{Version: payload.Version})
	if draftID != "" {
		draft := s.drafts[draftID]
		draft.Status = domain.DraftPublished
		draft.PublishedVersion = payload.Version
		s.drafts[draftID] = draft
	}
	s.audit = append(s.audit, audit)
	return nil
}

func (f *fakeStore) ActivateRuleset(_ context.Context, version string, audit domain.AuditEntry) error {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.rulesets {
		s.rulesets[i].Active = s.rulesets[i].Version == version
		found = found || s.rulesets[i].Version == version
	}
	if !found {
		return domain.ErrNotFound
	}
	s.audit = append(s.audit, audit)
	return nil
}

func (f *fakeStore) CreateDraft(_ context.Context, draft domain.CardDraft,
	audit domain.AuditEntry) (domain.CardDraft, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	draft.Status = domain.DraftOpen
	s.drafts[draft.ID] = draft
	s.audit = append(s.audit, audit)
	return draft, nil
}

func (f *fakeStore) UpdateDraft(_ context.Context, draft domain.CardDraft,
	audit domain.AuditEntry) (domain.CardDraft, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.drafts[draft.ID]
	if !ok {
		return domain.CardDraft{}, domain.ErrNotFound
	}
	if existing.Status == domain.DraftPublished || existing.Status == domain.DraftDiscarded {
		return domain.CardDraft{}, domain.ErrConflict
	}
	s.drafts[draft.ID] = draft
	s.audit = append(s.audit, audit)
	return draft, nil
}

func (f *fakeStore) Draft(_ context.Context, id string) (domain.CardDraft, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[id]
	if !ok {
		return domain.CardDraft{}, domain.ErrNotFound
	}
	return draft, nil
}

func (f *fakeStore) ListDrafts(_ context.Context, status domain.DraftStatus) ([]domain.CardDraft, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.CardDraft
	for _, draft := range s.drafts {
		if status == "" || draft.Status == status {
			out = append(out, draft)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateBan(_ context.Context, ban domain.CardBan,
	audit domain.AuditEntry) (domain.CardBan, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bans[ban.CardID]; exists {
		return domain.CardBan{}, domain.ErrConflict
	}
	ban.CreatedAt = time.Now().UTC()
	s.bans[ban.CardID] = ban
	s.audit = append(s.audit, audit)
	return ban, nil
}

func (f *fakeStore) LiftBan(_ context.Context, cardID, liftedBy string,
	audit domain.AuditEntry) (domain.CardBan, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	ban, ok := s.bans[cardID]
	if !ok {
		return domain.CardBan{}, domain.ErrNotFound
	}
	delete(s.bans, cardID)
	now := time.Now().UTC()
	ban.LiftedBy = liftedBy
	ban.LiftedAt = &now
	s.audit = append(s.audit, audit)
	return ban, nil
}

func (f *fakeStore) ActiveBans(context.Context) ([]domain.CardBan, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.CardBan
	for _, ban := range s.bans {
		out = append(out, ban)
	}
	return out, nil
}

func (f *fakeStore) CreateSeason(_ context.Context, season domain.Season,
	audit domain.AuditEntry) (domain.Season, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seasons = append(s.seasons, season)
	s.audit = append(s.audit, audit)
	return season, nil
}

func (f *fakeStore) RotateToRuleset(_ context.Context, version string,
	audit domain.AuditEntry) (int, int, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.payloads[version]; !ok {
		return 0, 0, domain.ErrNotFound
	}
	s.audit = append(s.audit, audit)
	return 80, 1, nil
}

func (f *fakeStore) MatchTelemetry(context.Context) (domain.MatchTelemetry, error) {
	return domain.MatchTelemetry{}, nil
}

func (f *fakeStore) ListAudit(_ context.Context, limit int) ([]domain.AuditEntry, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]domain.AuditEntry{}, s.audit...)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// --- Testes da Fase 7 ---

func adminRequest(t *testing.T, handler http.Handler, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestAdminRoutesRequireAdminRole(t *testing.T) {
	handler, _ := testHandler()
	playerToken := "player-token"
	for _, route := range []string{"GET /v1/admin/rulesets", "GET /v1/admin/drafts",
		"GET /v1/admin/bans", "GET /v1/admin/telemetry", "GET /v1/admin/audit"} {
		parts := strings.SplitN(route, " ", 2)
		if got := adminRequest(t, handler, playerToken, parts[0], parts[1], nil).Code; got != http.StatusForbidden {
			t.Fatalf("%s: player recebeu %d; esperado 403", route, got)
		}
		if got := adminRequest(t, handler, "", parts[0], parts[1], nil).Code; got != http.StatusUnauthorized {
			t.Fatalf("%s: anônimo recebeu %d; esperado 401", route, got)
		}
	}
}

func TestDraftLifecycleValidatePublishActivate(t *testing.T) {
	handler, _ := testHandler()
	adminToken := "admin-token"

	// Draft: VR-013 com dano 3 (buff proposital).
	card := *engine.Cards["VR-013"]
	fx := *engine.Effects.Cards["VR-013"]
	assault := *fx.Assault
	assault.Damage = 3
	fx.Assault = &assault
	input := map[string]any{"card_id": "VR-013", "note": "buff de teste", "card": card, "effects": fx}

	resp := adminRequest(t, handler, adminToken, "POST", "/v1/admin/drafts", input)
	if resp.Code != http.StatusCreated {
		t.Fatalf("criar draft: %d %s", resp.Code, resp.Body)
	}
	var draft domain.CardDraft
	if err := json.Unmarshal(resp.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}

	// Publicar sem validar é recusado.
	if got := adminRequest(t, handler, adminToken, "POST",
		"/v1/admin/drafts/"+draft.ID+"/publish", map[string]string{"version": "alpha-0.4.1"}).Code; got != http.StatusConflict {
		t.Fatalf("publicar sem validar: %d; esperado 409", got)
	}

	resp = adminRequest(t, handler, adminToken, "POST", "/v1/admin/drafts/"+draft.ID+"/validate", nil)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"ok":true`) {
		t.Fatalf("validar: %d %s", resp.Code, resp.Body)
	}

	resp = adminRequest(t, handler, adminToken, "POST",
		"/v1/admin/drafts/"+draft.ID+"/publish", map[string]string{"version": "alpha-0.4.1"})
	if resp.Code != http.StatusCreated {
		t.Fatalf("publicar: %d %s", resp.Code, resp.Body)
	}
	t.Cleanup(func() { engine.UnregisterRuleset("alpha-0.4.1") })

	// A versão publicada é executável e diverge do embutido no ponto alterado.
	rs, err := engine.RulesetByVersion("alpha-0.4.1")
	if err != nil {
		t.Fatalf("versão publicada não registrada: %v", err)
	}
	if rs.Effects.Cards["VR-013"].Assault.Damage != 3 {
		t.Fatal("efeito publicado não reflete o draft")
	}
	if engine.Effects.Cards["VR-013"].Assault.Damage != 2 {
		t.Fatal("o embutido foi alterado — versões devem ser imutáveis")
	}

	// Ativação e rollback movem o ponteiro; ambas auditadas.
	if got := adminRequest(t, handler, adminToken, "POST",
		"/v1/admin/rulesets/alpha-0.4.1/activate", nil).Code; got != http.StatusNoContent {
		t.Fatalf("ativar: %d", got)
	}
	if got := adminRequest(t, handler, adminToken, "POST",
		"/v1/admin/rulesets/"+engine.RulesetVersion+"/activate", nil).Code; got != http.StatusNoContent {
		t.Fatalf("rollback: %d", got)
	}
	audit := adminRequest(t, handler, adminToken, "GET", "/v1/admin/audit", nil)
	for _, action := range []string{"draft:create", "draft:validate", "ruleset:publish", "ruleset:activate"} {
		if !strings.Contains(audit.Body.String(), action) {
			t.Fatalf("auditoria sem a ação %s: %s", action, audit.Body)
		}
	}
}

func TestDraftValidationCatchesBrokenDSL(t *testing.T) {
	handler, _ := testHandler()
	adminToken := "admin-token"

	card := *engine.Cards["VR-015"]
	input := map[string]any{"card_id": "VR-015", "card": card,
		"effects": map[string]any{"rite": map[string]any{"steps": []map[string]any{{"op": "explodir_tudo"}}}}}
	resp := adminRequest(t, handler, adminToken, "POST", "/v1/admin/drafts", input)
	if resp.Code != http.StatusCreated {
		t.Fatalf("criar draft: %d %s", resp.Code, resp.Body)
	}
	var draft domain.CardDraft
	_ = json.Unmarshal(resp.Body.Bytes(), &draft)

	resp = adminRequest(t, handler, adminToken, "POST", "/v1/admin/drafts/"+draft.ID+"/validate", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("validar: %d %s", resp.Code, resp.Body)
	}
	if !strings.Contains(resp.Body.String(), `"ok":false`) ||
		!strings.Contains(resp.Body.String(), "explodir_tudo") {
		t.Fatalf("validação deveria apontar a op desconhecida: %s", resp.Body)
	}
}

func TestBanLifecycle(t *testing.T) {
	handler, _ := testHandler()
	adminToken := "admin-token"

	resp := adminRequest(t, handler, adminToken, "POST", "/v1/admin/bans",
		map[string]string{"card_id": "VR-020", "reason": "loop reportado"})
	if resp.Code != http.StatusCreated {
		t.Fatalf("banir: %d %s", resp.Code, resp.Body)
	}
	list := adminRequest(t, handler, adminToken, "GET", "/v1/admin/bans", nil)
	if !strings.Contains(list.Body.String(), "VR-020") {
		t.Fatalf("ban ativo ausente: %s", list.Body)
	}
	if got := adminRequest(t, handler, adminToken, "POST", "/v1/admin/bans",
		map[string]string{"card_id": "VR-999", "reason": "x"}).Code; got != http.StatusBadRequest {
		t.Fatalf("ban de carta inexistente: %d; esperado 400", got)
	}
	if got := adminRequest(t, handler, adminToken, "POST", "/v1/admin/bans/VR-020/lift", nil).Code; got != http.StatusOK {
		t.Fatalf("lift: %d", got)
	}
	list = adminRequest(t, handler, adminToken, "GET", "/v1/admin/bans", nil)
	if strings.Contains(list.Body.String(), "VR-020") {
		t.Fatalf("ban deveria ter sido suspenso: %s", list.Body)
	}
}

// --- Progressão (P1): fake em memória ---

func (f *fakeStore) prog() *liveopsState { return f.ops() }

func (f *fakeStore) RecordMatchProgress(_ context.Context, progress domain.MatchProgress) (bool, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progressLog == nil {
		s.progressLog = map[string]domain.MatchProgress{}
	}
	if _, seen := s.progressLog[progress.MatchID]; seen {
		return false, nil
	}
	s.progressLog[progress.MatchID] = progress
	return true, nil
}

func (f *fakeStore) RitualsFor(_ context.Context, userID, day string) ([]domain.RitualState, error) {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.RitualState{}, s.rituals[userID+"|"+day]...), nil
}

func (f *fakeStore) SaveRitualStates(_ context.Context, userID, day string, states []domain.RitualState) error {
	s := f.ops()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rituals == nil {
		s.rituals = map[string][]domain.RitualState{}
	}
	key := userID + "|" + day
	if len(s.rituals[key]) == 0 {
		s.rituals[key] = append([]domain.RitualState{}, states...)
	}
	return nil
}

func (f *fakeStore) Fragments(context.Context, string) (int, error) { return 120, nil }

func (f *fakeStore) MasteryFor(context.Context, string) ([]domain.ChampionMastery, error) {
	return []domain.ChampionMastery{{ChampionID: "CH-VH-01", XP: 150, Games: 5, Wins: 3}}, nil
}

func (f *fakeStore) RankedStandingFor(_ context.Context, _, seasonID string) (domain.RankedStanding, error) {
	return domain.RankedStanding{SeasonID: seasonID, Rating: 1012, Games: 4, Wins: 2, Position: 7}, nil
}

func (f *fakeStore) Leaderboard(_ context.Context, seasonID string, _ int) ([]domain.LeaderboardEntry, error) {
	return []domain.LeaderboardEntry{{Position: 1, UserID: "u1", DisplayName: "Alva", Rating: 1210}}, nil
}

func (f *fakeStore) MatchHistory(_ context.Context, userID string, _ int) ([]domain.MatchSummary, error) {
	return []domain.MatchSummary{{MatchID: "m1", Mode: "practice", MySlot: 0,
		MyChampion: "CH-CI-01", Opponent: "Treinador do Véu",
		OpponentChampion: "CH-VA-02", Won: true, EndReason: "concessao"}}, nil
}

func (f *fakeStore) MatchReplay(_ context.Context, matchID string) (domain.MatchReplayData, error) {
	if f.replay != nil {
		return *f.replay, nil
	}
	return domain.MatchReplayData{}, domain.ErrNotFound
}
