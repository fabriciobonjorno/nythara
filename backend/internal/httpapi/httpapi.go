package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	"veurubro/backend/internal/app"
	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
)

const maxBodyBytes = 1 << 20

type principalKey struct{}

type API struct {
	service *app.Service
	battles *battle.Manager
	logger  *slog.Logger
	ready   func(context.Context) error

	generalLimiter *security.RateLimiter
	authLimiter    *security.RateLimiter
	wsLimiter      *security.RateLimiter
}

func New(service *app.Service, battles *battle.Manager, logger *slog.Logger, ready func(context.Context) error) http.Handler {
	api := &API{service: service, battles: battles, logger: logger, ready: ready,
		generalLimiter: newGeneralLimiter(), authLimiter: newAuthLimiter(),
		wsLimiter: newWSCommandLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /readyz", api.readiness)
	mux.HandleFunc("GET /version", api.version)
	mux.HandleFunc("GET /internal/implementation-report", api.implementationReport)
	mux.Handle("GET /internal/metrics", expvar.Handler())
	mux.HandleFunc("POST /v1/auth/register", api.strictLimit(api.register))
	mux.HandleFunc("POST /v1/auth/login", api.strictLimit(api.login))
	mux.HandleFunc("POST /v1/auth/refresh", api.strictLimit(api.refresh))
	mux.HandleFunc("POST /v1/auth/logout", api.logout)
	mux.HandleFunc("GET /v1/catalog/cards", api.cards)
	mux.HandleFunc("GET /v1/catalog/champions", api.champions)
	mux.HandleFunc("GET /v1/catalog/precons", api.precons)
	mux.HandleFunc("GET /v1/rulesets/current", api.ruleset)
	mux.HandleFunc("GET /v1/seasons/current", api.season)
	mux.Handle("GET /v1/me", api.auth(http.HandlerFunc(api.me)))
	mux.Handle("GET /v1/collection", api.auth(http.HandlerFunc(api.collection)))
	mux.Handle("GET /v1/decks", api.auth(http.HandlerFunc(api.listDecks)))
	mux.Handle("POST /v1/decks", api.auth(http.HandlerFunc(api.createDeck)))
	mux.Handle("POST /v1/decks/precon", api.auth(http.HandlerFunc(api.createPreconDeck)))
	mux.Handle("GET /v1/decks/{id}", api.auth(http.HandlerFunc(api.getDeck)))
	mux.Handle("PUT /v1/decks/{id}", api.auth(http.HandlerFunc(api.updateDeck)))
	mux.Handle("DELETE /v1/decks/{id}", api.auth(http.HandlerFunc(api.deleteDeck)))
	mux.Handle("GET /v1/rewards", api.auth(http.HandlerFunc(api.rewards)))
	mux.Handle("GET /v1/progress", api.auth(http.HandlerFunc(api.progress)))
	mux.Handle("GET /v1/ranked/leaderboard", api.auth(http.HandlerFunc(api.leaderboard)))
	mux.Handle("GET /v1/decks/{id}/code", api.auth(http.HandlerFunc(api.deckCode)))
	mux.Handle("POST /v1/decks/import", api.auth(http.HandlerFunc(api.importDeck)))
	mux.Handle("GET /v1/matches", api.auth(http.HandlerFunc(api.matchHistory)))
	mux.Handle("GET /v1/matches/{id}/replay", api.auth(http.HandlerFunc(api.matchReplay)))
	mux.Handle("POST /v1/feedback", api.auth(http.HandlerFunc(api.submitFeedback)))
	mux.Handle("POST /v1/admin/rewards/grant", api.auth(api.requireRole(domain.RoleAdmin,
		http.HandlerFunc(api.grantReward))))
	api.adminRoutes(mux)
	if battles != nil {
		mux.Handle("GET /v1/matchmaking", api.auth(http.HandlerFunc(api.matchmakingStatus)))
		mux.Handle("POST /v1/matchmaking", api.auth(http.HandlerFunc(api.enqueueMatchmaking)))
		mux.Handle("POST /v1/practice", api.auth(http.HandlerFunc(api.startPractice)))
		mux.Handle("DELETE /v1/matchmaking", api.auth(http.HandlerFunc(api.cancelMatchmaking)))
		mux.Handle("POST /v1/battles/{id}/tickets", api.auth(http.HandlerFunc(api.battleTicket)))
		mux.HandleFunc("GET /v1/battles/{id}/ws", api.battleWebSocket)
	}
	return api.recover(api.harden(mux))
}

func (a *API) matchmakingStatus(w http.ResponseWriter, r *http.Request) {
	result, err := a.battles.QueueStatus(r.Context(), principal(r).UserID)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) enqueueMatchmaking(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeckID string `json:"deck_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	deck, err := a.service.Deck(r.Context(), principal(r), input.DeckID)
	if err != nil {
		a.respondError(w, err)
		return
	}
	result, err := a.battles.Queue(r.Context(), principal(r), deck)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) startPractice(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeckID        string `json:"deck_id"`
		BotChampionID string `json:"bot_champion_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	deck, err := a.service.Deck(r.Context(), principal(r), input.DeckID)
	if err != nil {
		a.respondError(w, err)
		return
	}
	botDeck, err := a.service.BotDeck(r.Context(), input.BotChampionID, deck.RulesetVersion)
	if err != nil {
		a.respondError(w, err)
		return
	}
	result, err := a.battles.StartPractice(r.Context(), principal(r), deck, botDeck)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) progress(w http.ResponseWriter, r *http.Request) {
	summary, err := a.service.ProgressSummary(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) deckCode(w http.ResponseWriter, r *http.Request) {
	code, err := a.service.DeckCode(r.Context(), principal(r), r.PathValue("id"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": code})
}

func (a *API) importDeck(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	deck, created, err := a.service.ImportDeck(r.Context(), principal(r), input.Code,
		input.Name, r.Header.Get("Idempotency-Key"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, deck)
}

func (a *API) matchHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	matches, err := a.service.MatchHistory(r.Context(), principal(r), limit)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"matches": matches})
}

func (a *API) matchReplay(w http.ResponseWriter, r *http.Request) {
	replay, err := a.service.MatchReplay(r.Context(), principal(r), r.PathValue("id"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, replay)
}

// O recado do Alpha é opcional: a resposta é um 204 silencioso, sem prêmio,
// sem contador e sem nada que transforme ajuda voluntária em obrigação.
func (a *API) submitFeedback(w http.ResponseWriter, r *http.Request) {
	var input struct {
		MatchID string `json:"match_id,omitempty"`
		Message string `json:"message"`
	}
	if !decode(w, r, &input) {
		return
	}
	if err := a.service.SubmitFeedback(r.Context(), principal(r), input.MatchID, input.Message); err != nil {
		a.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) leaderboard(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, standing, err := a.service.Leaderboard(r.Context(), principal(r), limit)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "me": standing})
}

func (a *API) cancelMatchmaking(w http.ResponseWriter, r *http.Request) {
	a.battles.CancelQueue(principal(r).UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) battleTicket(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode battle.TicketMode `json:"mode"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Mode == "" {
		input.Mode = battle.TicketPlayer
	}
	if input.Mode != battle.TicketPlayer && input.Mode != battle.TicketSpectator {
		writeError(w, http.StatusBadRequest, "invalid_request", "modo de ticket inválido")
		return
	}
	ticket, err := a.battles.IssueTicket(r.Context(), principal(r), r.PathValue("id"), input.Mode)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ticket)
}

type battleClientMessage struct {
	Type           string         `json:"type"`
	ClientSequence int64          `json:"client_sequence,omitempty"`
	Command        *battle.Intent `json:"command,omitempty"`
}

func (a *API) battleWebSocket(w http.ResponseWriter, r *http.Request) {
	if a.battles == nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "battle server indisponível")
		return
	}
	after := -1
	if raw := r.URL.Query().Get("after_event"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < -1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "after_event inválido")
			return
		}
		after = parsed
	}
	connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return
	}
	defer connection.CloseNow()
	connection.SetReadLimit(64 << 10)
	battleConnection, err := a.battles.Connect(r.Context(), r.URL.Query().Get("ticket"), r.PathValue("id"), after)
	if err != nil {
		_ = connection.Close(websocket.StatusPolicyViolation, err.Error())
		return
	}
	defer battleConnection.Close()

	local := make(chan battle.ServerMessage, 8)
	writerDone := make(chan error, 1)
	go func() {
		for {
			select {
			case message, ok := <-battleConnection.Messages:
				if !ok {
					writerDone <- nil
					return
				}
				if err := writeWSJSON(r.Context(), connection, message); err != nil {
					writerDone <- err
					return
				}
			case message := <-local:
				if err := writeWSJSON(r.Context(), connection, message); err != nil {
					writerDone <- err
					return
				}
			}
		}
	}()

	wsKey := "ws:" + r.URL.Query().Get("ticket")
	for {
		messageType, raw, err := connection.Read(r.Context())
		if err != nil {
			return
		}
		if !a.wsLimiter.Allow(wsKey) {
			metricWSFlood.Add(1)
			_ = connection.Close(websocket.StatusPolicyViolation, "excesso de comandos; reconecte")
			return
		}
		if messageType != websocket.MessageText {
			sendProtocolError(local, "invalid_message", "somente mensagens JSON de texto são aceitas")
			continue
		}
		var message battleClientMessage
		message, err = decodeBattleMessage(raw)
		if err != nil {
			sendProtocolError(local, "invalid_message", "mensagem JSON inválida")
			continue
		}
		switch message.Type {
		case "ready":
			if message.ClientSequence != 0 || message.Command != nil {
				sendProtocolError(local, "invalid_message", "ready não aceita comando ou sequência")
				continue
			} else {
				err = battleConnection.Ready(r.Context())
			}
		case "command":
			if message.Command == nil {
				sendProtocolError(local, "invalid_message", "comando ausente")
				continue
			} else {
				err = battleConnection.Submit(r.Context(), message.ClientSequence, *message.Command)
			}
		default:
			sendProtocolError(local, "invalid_message", "tipo de mensagem desconhecido")
			continue
		}
		if err != nil {
			code, message := battle.ProtocolCode(err), err.Error()
			if code == "battle_error" {
				a.logger.Error("erro interno na sala", "match_id", r.PathValue("id"), "error", err)
				message = "erro interno"
			}
			sendProtocolError(local, code, message)
		}
		select {
		case writerErr := <-writerDone:
			if writerErr != nil {
				a.logger.Debug("WebSocket encerrado", "error", writerErr)
			}
			return
		default:
		}
	}
}

func decodeBattleMessage(raw []byte) (battleClientMessage, error) {
	var message battleClientMessage
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&message); err != nil {
		return message, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return message, fmt.Errorf("mais de um documento JSON")
	}
	return message, nil
}

func writeWSJSON(ctx context.Context, connection *websocket.Conn, value any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return connection.Write(ctx, websocket.MessageText, raw)
}

func sendProtocolError(out chan<- battle.ServerMessage, code, message string) {
	select {
	case out <- battle.ServerMessage{Type: "error", Error: &battle.ProtocolError{Code: code, Message: message}}:
	default:
	}
}

func (a *API) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				metricPanics.Add(1)
				a.logger.Error("panic na API", "error", recovered)
				writeError(w, http.StatusInternalServerError, "internal_error", "erro interno")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "autenticação necessária")
			return
		}
		principal, err := a.service.Authenticate(r.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "token inválido ou expirado")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}

func (a *API) requireRole(role domain.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if principal(r).Role != role {
			writeError(w, http.StatusForbidden, "forbidden", "permissão insuficiente")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *API) readiness(w http.ResponseWriter, r *http.Request) {
	if a.ready != nil && a.ready(r.Context()) != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "dependência indisponível")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (a *API) version(w http.ResponseWriter, _ *http.Request) {
	rs := engine.CompetitiveRuleset()
	writeJSON(w, http.StatusOK, map[string]any{"ruleset": engine.CompetitiveRulesetVersion,
		"cards": len(rs.CardList), "champions": len(rs.Champions)})
}

func (a *API) implementationReport(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, engine.ImplementationReport())
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var input app.RegisterInput
	if !decode(w, r, &input) {
		return
	}
	user, tokens, err := a.service.Register(r.Context(), input)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "tokens": tokens})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &input) {
		return
	}
	user, tokens, err := a.service.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "tokens": tokens})
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(w, r, &input) {
		return
	}
	tokens, err := a.service.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(w, r, &input) {
		return
	}
	if err := a.service.Logout(r.Context(), input.RefreshToken); err != nil {
		a.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) cards(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ruleset_version": engine.CompetitiveRulesetVersion, "cards": engine.CompetitiveRuleset().CardList})
}

func (a *API) champions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ruleset_version": engine.CompetitiveRulesetVersion, "champions": app.ChampionsSorted()})
}

func (a *API) ruleset(w http.ResponseWriter, _ *http.Request) {
	// A Vitalidade inicial viaja com o ruleset: o cliente não deve repetir o
	// número em texto fixo, sob pena de prometer uma regra que mudou.
	rules := engine.CompetitiveRuleset().ConfrontRules
	writeJSON(w, http.StatusOK, map[string]any{"version": engine.CompetitiveRulesetVersion, "active": true,
		"starting_vitality": rules.StartingVitality, "guard_leak_cap": rules.GuardLeakCap,
		"pressure_start_turn": rules.PressureStartTurn, "pressure_base_loss": rules.PressureBaseLoss})
}

func (a *API) season(w http.ResponseWriter, r *http.Request) {
	season, err := a.service.ActiveSeason(r.Context())
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, season)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, principal(r))
}

func (a *API) collection(w http.ResponseWriter, r *http.Request) {
	collection, err := a.service.Collection(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection)
}

func (a *API) listDecks(w http.ResponseWriter, r *http.Request) {
	decks, err := a.service.ListDecks(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decks": decks})
}

func (a *API) getDeck(w http.ResponseWriter, r *http.Request) {
	deck, err := a.service.Deck(r.Context(), principal(r), r.PathValue("id"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deck)
}

type deckInput struct {
	Name       string            `json:"name"`
	ChampionID string            `json:"champion_id"`
	Cards      []domain.DeckCard `json:"cards"`
	Version    int64             `json:"version,omitempty"`
}

func (a *API) createDeck(w http.ResponseWriter, r *http.Request) {
	var input deckInput
	raw, ok := decodeRaw(w, r, &input)
	if !ok {
		return
	}
	deck := domain.Deck{Name: input.Name, ChampionID: input.ChampionID, Cards: input.Cards}
	saved, replayed, err := a.service.SaveDeck(r.Context(), principal(r), deck, nil,
		r.Header.Get("Idempotency-Key"), raw)
	if err != nil {
		a.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, saved)
}

func (a *API) updateDeck(w http.ResponseWriter, r *http.Request) {
	var input deckInput
	raw, ok := decodeRaw(w, r, &input)
	if !ok {
		return
	}
	if input.Version < 1 {
		writeError(w, http.StatusBadRequest, "invalid_request", "version é obrigatória")
		return
	}
	deck := domain.Deck{ID: r.PathValue("id"), Name: input.Name, ChampionID: input.ChampionID, Cards: input.Cards}
	saved, _, err := a.service.SaveDeck(r.Context(), principal(r), deck, &input.Version,
		r.Header.Get("Idempotency-Key"), raw)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (a *API) deleteDeck(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil || version < 1 {
		writeError(w, http.StatusBadRequest, "invalid_request", "version é obrigatória")
		return
	}
	_, err = a.service.DeleteDeck(r.Context(), principal(r), r.PathValue("id"), version,
		r.Header.Get("Idempotency-Key"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) rewards(w http.ResponseWriter, r *http.Request) {
	rewards, err := a.service.ListRewards(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rewards": rewards})
}

func (a *API) grantReward(w http.ResponseWriter, r *http.Request) {
	var reward domain.Reward
	raw, ok := decodeRaw(w, r, &reward)
	if !ok {
		return
	}
	saved, replayed, err := a.service.GrantReward(r.Context(), principal(r), reward,
		r.Header.Get("Idempotency-Key"), raw)
	if err != nil {
		a.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, saved)
}

func (a *API) respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid), errors.Is(err, domain.ErrInvalidCredentials):
		status := http.StatusBadRequest
		code := "invalid_request"
		if errors.Is(err, domain.ErrInvalidCredentials) {
			status, code = http.StatusUnauthorized, "invalid_credentials"
		}
		writeError(w, status, code, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, battle.ErrNotParticipant), errors.Is(err, battle.ErrTicketInvalid):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, battle.ErrAlreadyQueued):
		writeError(w, http.StatusConflict, "already_queued", err.Error())
	default:
		a.logger.Error("erro na API", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "erro interno")
	}
}

func principal(r *http.Request) domain.Principal {
	value, _ := r.Context().Value(principalKey{}).(domain.Principal)
	return value
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	_, ok := decodeRaw(w, r, dst)
	return ok
}

func decodeRaw(w http.ResponseWriter, r *http.Request, dst any) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corpo inválido ou grande demais")
		return nil, false
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", fmt.Sprintf("JSON inválido: %v", err))
		return nil, false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "apenas um documento JSON é permitido")
		return nil, false
	}
	return raw, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
