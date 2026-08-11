package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/security"
	"veurubro/backend/internal/sim"
)

// Fase 7 — Admin/LiveOps. Mutações administrativas são idempotentes por
// construção (publicar versão repetida ou banir carta já banida conflita;
// ativar a versão ativa é no-op) e todas geram trilha de auditoria na mesma
// transação do efeito.

var versionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{2,63}$`)

// OnRulesetActivated permite ao servidor reagir a ativações (ex.: apontar o
// matchmaking para a nova versão) sem acoplar app a battle.
type OnRulesetActivated func(version string)

func (s *Service) SetRulesetActivationHook(hook OnRulesetActivated) { s.onRulesetActivated = hook }

func requireAdmin(principal domain.Principal) error {
	if principal.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	return nil
}

func auditEntry(principal domain.Principal, action, subject string, payload any) domain.AuditEntry {
	raw, _ := json.Marshal(payload)
	return domain.AuditEntry{Actor: principal.UserID, Action: action, Subject: subject, Payload: raw}
}

// RegisterStoredRulesets compila e registra na engine todas as versões
// persistidas (chamado no boot e após publicações). Retorna as versões
// registradas nesta chamada.
func (s *Service) RegisterStoredRulesets(ctx context.Context) ([]string, error) {
	payloads, err := s.store.ListRulesetPayloads(ctx)
	if err != nil {
		return nil, err
	}
	var registered []string
	for _, payload := range payloads {
		if _, lookupErr := engine.RulesetByVersion(payload.Version); lookupErr == nil {
			continue // versões embutidas/registradas não são recompiladas em outro ponteiro
		}
		rs, err := engine.CompileRuleset(payload.Version, payload.Cards, payload.Champions, payload.Effects)
		if err != nil {
			return registered, fmt.Errorf("ruleset %s persistido não compila: %w", payload.Version, err)
		}
		if err := engine.RegisterRuleset(rs); err != nil {
			return registered, err
		}
		registered = append(registered, payload.Version)
	}
	return registered, nil
}

func (s *Service) ListRulesets(ctx context.Context, principal domain.Principal) ([]domain.RulesetInfo, error) {
	if err := requireAdmin(principal); err != nil {
		return nil, err
	}
	return s.store.ListRulesets(ctx)
}

// ActivateRuleset aponta o competitivo para uma versão publicada (rollback é
// ativar uma versão anterior). A versão precisa compilar antes de assumir.
func (s *Service) ActivateRuleset(ctx context.Context, principal domain.Principal, version string) error {
	if err := requireAdmin(principal); err != nil {
		return err
	}
	if _, err := engine.RulesetByVersion(version); err != nil {
		// Tenta compilar do snapshot persistido antes de recusar.
		payload, perr := s.store.RulesetPayload(ctx, version)
		if perr != nil {
			return fmt.Errorf("%w: versão %s sem snapshot compilável", domain.ErrInvalid, version)
		}
		rs, cerr := engine.CompileRuleset(payload.Version, payload.Cards, payload.Champions, payload.Effects)
		if cerr != nil {
			return fmt.Errorf("%w: versão %s não compila: %v", domain.ErrInvalid, version, cerr)
		}
		if rerr := engine.RegisterRuleset(rs); rerr != nil {
			return rerr
		}
	}
	err := s.store.ActivateRuleset(ctx, version,
		auditEntry(principal, "ruleset:activate", version, nil))
	if err != nil {
		return err
	}
	if s.onRulesetActivated != nil {
		s.onRulesetActivated(version)
	}
	return nil
}

// --- Drafts de carta ---

type DraftInput struct {
	CardID  string          `json:"card_id"`
	Note    string          `json:"note"`
	Card    json.RawMessage `json:"card"`
	Effects json.RawMessage `json:"effects"`
}

func (s *Service) CreateDraft(ctx context.Context, principal domain.Principal,
	input DraftInput) (domain.CardDraft, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.CardDraft{}, err
	}
	if err := validateDraftInput(input); err != nil {
		return domain.CardDraft{}, err
	}
	id, err := security.NewID()
	if err != nil {
		return domain.CardDraft{}, err
	}
	draft := domain.CardDraft{
		ID: id, CardID: input.CardID, BaseVersion: engine.CompetitiveRulesetVersion,
		Note: input.Note, Card: input.Card, Effects: input.Effects, CreatedBy: principal.UserID,
	}
	return s.store.CreateDraft(ctx, draft,
		auditEntry(principal, "draft:create", id, map[string]string{"card_id": input.CardID}))
}

func (s *Service) UpdateDraft(ctx context.Context, principal domain.Principal, id string,
	input DraftInput) (domain.CardDraft, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.CardDraft{}, err
	}
	if err := validateDraftInput(input); err != nil {
		return domain.CardDraft{}, err
	}
	draft, err := s.store.Draft(ctx, id)
	if err != nil {
		return domain.CardDraft{}, err
	}
	draft.Card = input.Card
	draft.Effects = input.Effects
	draft.Note = input.Note
	draft.Status = domain.DraftOpen // qualquer edição invalida a validação anterior
	draft.LastValidation = nil
	return s.store.UpdateDraft(ctx, draft,
		auditEntry(principal, "draft:update", id, map[string]string{"card_id": draft.CardID}))
}

func (s *Service) Draft(ctx context.Context, principal domain.Principal, id string) (domain.CardDraft, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.CardDraft{}, err
	}
	return s.store.Draft(ctx, id)
}

func (s *Service) ListDrafts(ctx context.Context, principal domain.Principal,
	status domain.DraftStatus) ([]domain.CardDraft, error) {
	if err := requireAdmin(principal); err != nil {
		return nil, err
	}
	return s.store.ListDrafts(ctx, status)
}

func validateDraftInput(input DraftInput) error {
	if input.CardID == "" {
		return fmt.Errorf("%w: card_id obrigatório", domain.ErrInvalid)
	}
	var card engine.CardDef
	if err := json.Unmarshal(input.Card, &card); err != nil {
		return fmt.Errorf("%w: card não é um CardDef válido: %v", domain.ErrInvalid, err)
	}
	if card.ID != input.CardID {
		return fmt.Errorf("%w: card.id difere de card_id", domain.ErrInvalid)
	}
	var fx engine.CardFx
	if err := json.Unmarshal(input.Effects, &fx); err != nil {
		return fmt.Errorf("%w: effects não é um CardFx válido: %v", domain.ErrInvalid, err)
	}
	return nil
}

// candidatePayload monta o snapshot da versão base com o draft aplicado
// (substitui a carta existente ou acrescenta uma nova).
func (s *Service) candidatePayload(ctx context.Context, draft domain.CardDraft,
	version string) (domain.RulesetPayload, error) {
	base, err := s.store.RulesetPayload(ctx, draft.BaseVersion)
	if err != nil {
		return domain.RulesetPayload{}, fmt.Errorf("snapshot base %s: %w", draft.BaseVersion, err)
	}
	var cards []json.RawMessage
	if err := json.Unmarshal(base.Cards, &cards); err != nil {
		return domain.RulesetPayload{}, err
	}
	replaced := false
	for i, raw := range cards {
		var probe struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return domain.RulesetPayload{}, err
		}
		if probe.ID == draft.CardID {
			cards[i] = draft.Card
			replaced = true
			break
		}
	}
	if !replaced {
		cards = append(cards, draft.Card)
	}
	newCards, err := json.Marshal(cards)
	if err != nil {
		return domain.RulesetPayload{}, err
	}

	var fx engine.EffectsFile
	if err := json.Unmarshal(base.Effects, &fx); err != nil {
		return domain.RulesetPayload{}, err
	}
	var cardFx engine.CardFx
	if err := json.Unmarshal(draft.Effects, &cardFx); err != nil {
		return domain.RulesetPayload{}, fmt.Errorf("%w: effects do draft: %v", domain.ErrInvalid, err)
	}
	fx.Cards[draft.CardID] = &cardFx
	delete(fx.Unsupported, draft.CardID)
	fx.Version = version
	newEffects, err := json.Marshal(&fx)
	if err != nil {
		return domain.RulesetPayload{}, err
	}
	return domain.RulesetPayload{
		Version: version, Cards: newCards, Champions: base.Champions, Effects: newEffects,
	}, nil
}

// ValidationResult é persistido no draft e devolvido ao painel.
type ValidationResult struct {
	OK          bool      `json:"ok"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
	BaseVersion string    `json:"base_version"`
}

// ValidateDraft roda schema + validador da DSL + compilação completa do
// catálogo candidato, sem publicar nada.
func (s *Service) ValidateDraft(ctx context.Context, principal domain.Principal,
	id string) (ValidationResult, error) {
	if err := requireAdmin(principal); err != nil {
		return ValidationResult{}, err
	}
	draft, err := s.store.Draft(ctx, id)
	if err != nil {
		return ValidationResult{}, err
	}
	if draft.Status == domain.DraftPublished || draft.Status == domain.DraftDiscarded {
		return ValidationResult{}, fmt.Errorf("%w: draft %s não está aberto", domain.ErrConflict, id)
	}
	result := ValidationResult{OK: true, CheckedAt: s.now().UTC(), BaseVersion: draft.BaseVersion}
	candidate, err := s.candidatePayload(ctx, draft, "draft:"+id)
	if err == nil {
		_, err = engine.CompileRuleset(candidate.Version, candidate.Cards, candidate.Champions, candidate.Effects)
	}
	if err != nil {
		result.OK = false
		result.Error = err.Error()
	}
	raw, _ := json.Marshal(result)
	draft.LastValidation = raw
	if result.OK {
		draft.Status = domain.DraftValidated
	} else {
		draft.Status = domain.DraftOpen
	}
	if _, err := s.store.UpdateDraft(ctx, draft,
		auditEntry(principal, "draft:validate", id, result)); err != nil {
		return ValidationResult{}, err
	}
	return result, nil
}

// DraftSimulation resume a simulação de um draft para o painel.
type DraftSimulation struct {
	Games       int                  `json:"games"`
	Healthy     bool                 `json:"healthy"`
	Error       string               `json:"error,omitempty"`
	FirstPlayer float64              `json:"first_player_win_rate"`
	AvgRounds   float64              `json:"average_rounds"`
	Champions   []sim.ChampionMetric `json:"champions"`
}

// SimulateDraft compila o candidato numa versão efêmera, roda o simulador
// headless com verificação de replay e descarta a versão ao final.
func (s *Service) SimulateDraft(ctx context.Context, principal domain.Principal,
	id string, games int) (DraftSimulation, error) {
	if err := requireAdmin(principal); err != nil {
		return DraftSimulation{}, err
	}
	if games <= 0 {
		games = 400
	}
	if games > 2000 {
		games = 2000
	}
	draft, err := s.store.Draft(ctx, id)
	if err != nil {
		return DraftSimulation{}, err
	}
	version := "draft:" + id + ":sim"
	candidate, err := s.candidatePayload(ctx, draft, version)
	if err != nil {
		return DraftSimulation{}, err
	}
	rs, err := engine.CompileRuleset(version, candidate.Cards, candidate.Champions, candidate.Effects)
	if err != nil {
		return DraftSimulation{}, fmt.Errorf("%w: draft não compila: %v", domain.ErrInvalid, err)
	}
	if err := engine.RegisterRuleset(rs); err != nil {
		return DraftSimulation{}, err
	}
	defer engine.UnregisterRuleset(version)

	report, runErr := sim.Run(sim.Config{
		Games: games, BaseSeed: 1, Workers: 4, MaxCommands: 2000,
		Bot0: sim.BotHeuristic, Bot1: sim.BotHeuristic, VerifyReplay: true,
		RulesetVersion: version,
	})
	out := DraftSimulation{
		Games:       report.Health.Completed,
		Healthy:     runErr == nil,
		FirstPlayer: report.FirstPlayer.WinRate,
		AvgRounds:   report.Duration.AverageRounds,
		Champions:   report.Champions,
	}
	if runErr != nil {
		out.Error = runErr.Error()
	}
	s.appendAudit(ctx, auditEntry(principal, "draft:simulate", id,
		map[string]any{"games": games, "healthy": out.Healthy}))
	return out, nil
}

// PublishDraft cria a nova versão imutável a partir do draft validado e a
// registra na engine. Não ativa: ativação/rollback é um passo explícito.
func (s *Service) PublishDraft(ctx context.Context, principal domain.Principal,
	id, newVersion string) (domain.RulesetInfo, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.RulesetInfo{}, err
	}
	if !versionPattern.MatchString(newVersion) || newVersion == engine.CompetitiveRulesetVersion {
		return domain.RulesetInfo{}, fmt.Errorf("%w: versão inválida %q", domain.ErrInvalid, newVersion)
	}
	if strings.HasPrefix(newVersion, "draft:") {
		return domain.RulesetInfo{}, fmt.Errorf("%w: prefixo draft: é reservado", domain.ErrInvalid)
	}
	draft, err := s.store.Draft(ctx, id)
	if err != nil {
		return domain.RulesetInfo{}, err
	}
	if draft.Status != domain.DraftValidated {
		return domain.RulesetInfo{}, fmt.Errorf("%w: valide o draft antes de publicar", domain.ErrConflict)
	}
	candidate, err := s.candidatePayload(ctx, draft, newVersion)
	if err != nil {
		return domain.RulesetInfo{}, err
	}
	rs, err := engine.CompileRuleset(newVersion, candidate.Cards, candidate.Champions, candidate.Effects)
	if err != nil {
		return domain.RulesetInfo{}, fmt.Errorf("%w: candidato não compila: %v", domain.ErrInvalid, err)
	}
	if err := s.store.PublishRuleset(ctx, candidate, id, auditEntry(principal, "ruleset:publish",
		newVersion, map[string]string{"draft_id": id, "card_id": draft.CardID})); err != nil {
		return domain.RulesetInfo{}, err
	}
	if err := engine.RegisterRuleset(rs); err != nil {
		return domain.RulesetInfo{}, err
	}
	return domain.RulesetInfo{Version: newVersion, Active: false, CreatedAt: s.now().UTC()}, nil
}

// RotationResult resume a rotação de coleção para uma versão publicada.
type RotationResult struct {
	Version      string `json:"version"`
	CardsGranted int    `json:"cards_granted"`
	DecksCloned  int    `json:"decks_cloned"`
}

// RotateRuleset concede a coleção da versão publicada a todos os jogadores e
// clona os decks válidos da versão ativa (passo entre publicar e ativar).
func (s *Service) RotateRuleset(ctx context.Context, principal domain.Principal,
	version string) (RotationResult, error) {
	if err := requireAdmin(principal); err != nil {
		return RotationResult{}, err
	}
	if _, err := engine.RulesetByVersion(version); err != nil {
		return RotationResult{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	granted, cloned, err := s.store.RotateToRuleset(ctx, version,
		auditEntry(principal, "ruleset:rotate", version,
			map[string]string{"version": version}))
	if err != nil {
		return RotationResult{}, err
	}
	return RotationResult{Version: version, CardsGranted: granted, DecksCloned: cloned}, nil
}

// --- Bans emergenciais de ranked ---

func (s *Service) BanCard(ctx context.Context, principal domain.Principal,
	cardID, reason string) (domain.CardBan, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.CardBan{}, err
	}
	if engine.Cards[cardID] == nil {
		return domain.CardBan{}, fmt.Errorf("%w: carta desconhecida %q", domain.ErrInvalid, cardID)
	}
	if strings.TrimSpace(reason) == "" {
		return domain.CardBan{}, fmt.Errorf("%w: motivo obrigatório", domain.ErrInvalid)
	}
	id, err := security.NewID()
	if err != nil {
		return domain.CardBan{}, err
	}
	ban := domain.CardBan{ID: id, CardID: cardID, Reason: reason, CreatedBy: principal.UserID}
	return s.store.CreateBan(ctx, ban,
		auditEntry(principal, "ban:create", cardID, map[string]string{"reason": reason}))
}

func (s *Service) LiftBan(ctx context.Context, principal domain.Principal,
	cardID string) (domain.CardBan, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.CardBan{}, err
	}
	return s.store.LiftBan(ctx, cardID, principal.UserID,
		auditEntry(principal, "ban:lift", cardID, nil))
}

func (s *Service) ListBans(ctx context.Context, principal domain.Principal) ([]domain.CardBan, error) {
	if err := requireAdmin(principal); err != nil {
		return nil, err
	}
	return s.store.ActiveBans(ctx)
}

// --- Temporadas ---

func (s *Service) CreateSeason(ctx context.Context, principal domain.Principal,
	name, rulesetVersion string, startsAt time.Time) (domain.Season, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.Season{}, err
	}
	if strings.TrimSpace(name) == "" {
		return domain.Season{}, fmt.Errorf("%w: nome obrigatório", domain.ErrInvalid)
	}
	if startsAt.IsZero() {
		startsAt = s.now().UTC()
	}
	id, err := security.NewID()
	if err != nil {
		return domain.Season{}, err
	}
	season := domain.Season{ID: id, Name: name, RulesetVersion: rulesetVersion, StartsAt: startsAt}
	return s.store.CreateSeason(ctx, season,
		auditEntry(principal, "season:create", id, map[string]string{
			"name": name, "ruleset_version": rulesetVersion}))
}

// --- Telemetria e auditoria ---

func (s *Service) AdminTelemetry(ctx context.Context, principal domain.Principal) (domain.MatchTelemetry, error) {
	if err := requireAdmin(principal); err != nil {
		return domain.MatchTelemetry{}, err
	}
	return s.store.MatchTelemetry(ctx)
}

func (s *Service) ListAudit(ctx context.Context, principal domain.Principal,
	limit int) ([]domain.AuditEntry, error) {
	if err := requireAdmin(principal); err != nil {
		return nil, err
	}
	return s.store.ListAudit(ctx, limit)
}

// appendAudit registra ações administrativas sem efeito transacional próprio
// (ex.: simulações). Falha de auditoria aqui não interrompe a operação.
func (s *Service) appendAudit(ctx context.Context, entry domain.AuditEntry) {
	if auditor, ok := s.store.(interface {
		AppendAudit(context.Context, domain.AuditEntry) error
	}); ok {
		_ = auditor.AppendAudit(ctx, entry)
	}
}
