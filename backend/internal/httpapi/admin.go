package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"veurubro/backend/internal/app"
	"veurubro/backend/internal/domain"
)

// Fase 7 — painel Admin/LiveOps. Todas as rotas exigem papel admin (RBAC no
// middleware e revalidado no serviço); mutações são idempotentes por
// construção e auditadas na mesma transação do efeito.

func (a *API) adminRoutes(mux *http.ServeMux) {
	admin := func(h http.HandlerFunc) http.Handler {
		return a.auth(a.requireRole(domain.RoleAdmin, h))
	}
	mux.Handle("GET /v1/admin/rulesets", admin(a.adminListRulesets))
	mux.Handle("POST /v1/admin/rulesets/{version}/activate", admin(a.adminActivateRuleset))
	mux.Handle("POST /v1/admin/rulesets/{version}/rotate", admin(a.adminRotateRuleset))
	mux.Handle("GET /v1/admin/drafts", admin(a.adminListDrafts))
	mux.Handle("POST /v1/admin/drafts", admin(a.adminCreateDraft))
	mux.Handle("GET /v1/admin/drafts/{id}", admin(a.adminGetDraft))
	mux.Handle("PUT /v1/admin/drafts/{id}", admin(a.adminUpdateDraft))
	mux.Handle("POST /v1/admin/drafts/{id}/validate", admin(a.adminValidateDraft))
	mux.Handle("POST /v1/admin/drafts/{id}/simulate", admin(a.adminSimulateDraft))
	mux.Handle("POST /v1/admin/drafts/{id}/publish", admin(a.adminPublishDraft))
	mux.Handle("GET /v1/admin/bans", admin(a.adminListBans))
	mux.Handle("POST /v1/admin/bans", admin(a.adminCreateBan))
	mux.Handle("POST /v1/admin/bans/{card}/lift", admin(a.adminLiftBan))
	mux.Handle("POST /v1/admin/seasons", admin(a.adminCreateSeason))
	mux.Handle("GET /v1/admin/telemetry", admin(a.adminTelemetry))
	mux.Handle("GET /v1/admin/audit", admin(a.adminAudit))
}

func (a *API) adminListRulesets(w http.ResponseWriter, r *http.Request) {
	list, err := a.service.ListRulesets(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) adminActivateRuleset(w http.ResponseWriter, r *http.Request) {
	if err := a.service.ActivateRuleset(r.Context(), principal(r), r.PathValue("version")); err != nil {
		a.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) adminRotateRuleset(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.RotateRuleset(r.Context(), principal(r), r.PathValue("version"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) adminListDrafts(w http.ResponseWriter, r *http.Request) {
	status := domain.DraftStatus(r.URL.Query().Get("status"))
	drafts, err := a.service.ListDrafts(r.Context(), principal(r), status)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, drafts)
}

func (a *API) adminCreateDraft(w http.ResponseWriter, r *http.Request) {
	var input app.DraftInput
	if !decode(w, r, &input) {
		return
	}
	draft, err := a.service.CreateDraft(r.Context(), principal(r), input)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, draft)
}

func (a *API) adminGetDraft(w http.ResponseWriter, r *http.Request) {
	draft, err := a.service.Draft(r.Context(), principal(r), r.PathValue("id"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (a *API) adminUpdateDraft(w http.ResponseWriter, r *http.Request) {
	var input app.DraftInput
	if !decode(w, r, &input) {
		return
	}
	draft, err := a.service.UpdateDraft(r.Context(), principal(r), r.PathValue("id"), input)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (a *API) adminValidateDraft(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.ValidateDraft(r.Context(), principal(r), r.PathValue("id"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) adminSimulateDraft(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Games int `json:"games"`
	}
	if !decode(w, r, &input) {
		return
	}
	result, err := a.service.SimulateDraft(r.Context(), principal(r), r.PathValue("id"), input.Games)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) adminPublishDraft(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Version string `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	info, err := a.service.PublishDraft(r.Context(), principal(r), r.PathValue("id"), input.Version)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

func (a *API) adminListBans(w http.ResponseWriter, r *http.Request) {
	bans, err := a.service.ListBans(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bans)
}

func (a *API) adminCreateBan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CardID string `json:"card_id"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	ban, err := a.service.BanCard(r.Context(), principal(r), input.CardID, input.Reason)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ban)
}

func (a *API) adminLiftBan(w http.ResponseWriter, r *http.Request) {
	ban, err := a.service.LiftBan(r.Context(), principal(r), r.PathValue("card"))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ban)
}

func (a *API) adminCreateSeason(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name           string    `json:"name"`
		RulesetVersion string    `json:"ruleset_version"`
		StartsAt       time.Time `json:"starts_at"`
	}
	if !decode(w, r, &input) {
		return
	}
	season, err := a.service.CreateSeason(r.Context(), principal(r),
		input.Name, input.RulesetVersion, input.StartsAt)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, season)
}

func (a *API) adminTelemetry(w http.ResponseWriter, r *http.Request) {
	telemetry, err := a.service.AdminTelemetry(r.Context(), principal(r))
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, telemetry)
}

func (a *API) adminAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := a.service.ListAudit(r.Context(), principal(r), limit)
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// precons lista os decks oficiais (público: ajuda o onboarding e o builder).
func (a *API) precons(w http.ResponseWriter, r *http.Request) {
	list, err := app.Precons()
	if err != nil {
		a.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) createPreconDeck(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ChampionID string `json:"champion_id"`
	}
	raw, ok := decodeRaw(w, r, &input)
	if !ok {
		return
	}
	deck, replayed, err := a.service.CreatePreconDeck(r.Context(), principal(r),
		input.ChampionID, r.Header.Get("Idempotency-Key"), raw)
	if err != nil {
		a.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, deck)
}
