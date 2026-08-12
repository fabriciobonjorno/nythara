package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
	accountprogress "veurubro/backend/internal/progression"
	"veurubro/backend/internal/security"
)

const (
	AccessTTL        = 15 * time.Minute
	RefreshTTL       = 30 * 24 * time.Hour
	DefaultDeckLock  = 24 * time.Hour
	PasswordResetTTL = 30 * time.Minute
)

type Clock func() time.Time

type Service struct {
	store domain.Store
	now   Clock

	onRulesetActivated  OnRulesetActivated
	passwordResetSender PasswordResetSender
	publicAppURL        *url.URL
	googleOAuth         googleOAuthProvider
}

type PasswordResetSender interface {
	SendPasswordReset(ctx context.Context, to, link, locale string, ttl time.Duration) error
}

func New(store domain.Store) *Service {
	return &Service{store: store, now: time.Now}
}

func NewWithClock(store domain.Store, now Clock) *Service {
	return &Service{store: store, now: now}
}

func (s *Service) ConfigurePasswordRecovery(sender PasswordResetSender, publicAppURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(publicAppURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return errors.New("PUBLIC_APP_URL deve ser uma URL HTTP(S) absoluta")
	}
	if sender == nil {
		return errors.New("serviço de e-mail ausente")
	}
	parsed.RawQuery, parsed.Fragment = "", ""
	s.passwordResetSender, s.publicAppURL = sender, parsed
	return nil
}

// RequestPasswordReset sempre mantém a mesma resposta pública. ErrNotFound é
// absorvido para que a rota não revele quais endereços possuem conta.
func (s *Service) RequestPasswordReset(ctx context.Context, email, locale string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	user, err := s.store.UserByEmail(ctx, normalized)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if user.ID == domain.BotUserID {
		return nil
	}
	if s.passwordResetSender == nil || s.publicAppURL == nil {
		return errors.New("recuperação de senha não configurada")
	}
	plain, tokenHash, err := security.NewToken()
	if err != nil {
		return err
	}
	id, err := security.NewID()
	if err != nil {
		return err
	}
	now := s.now().UTC()
	if err := s.store.SavePasswordReset(ctx, domain.PasswordResetToken{ID: id, UserID: user.ID,
		TokenHash: tokenHash, ExpiresAt: now.Add(PasswordResetTTL)}); err != nil {
		return err
	}
	link := *s.publicAppURL
	link.Path = "/reset-password"
	query := link.Query()
	query.Set("token", plain)
	link.RawQuery = query.Encode()
	return s.passwordResetSender.SendPasswordReset(ctx, user.Email, link.String(), locale, PasswordResetTTL)
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if len(token) < 32 || len(token) > 256 {
		return domain.ErrInvalidResetToken
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	return s.store.ConsumePasswordReset(ctx, security.TokenHash(token), s.now().UTC(), passwordHash)
}

func (s *Service) RecordEmailDeliveryEvent(ctx context.Context, event domain.EmailDeliveryEvent) error {
	return s.store.SaveEmailDeliveryEvent(ctx, event)
}

type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (s *Service) Register(ctx context.Context, input RegisterInput) (domain.User, domain.SessionTokens, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	username := input.Username
	if username == "" {
		username = input.DisplayName
	} else if input.DisplayName != "" && input.DisplayName != username {
		return domain.User{}, domain.SessionTokens{}, fmt.Errorf("%w: username e display_name não podem divergir", domain.ErrInvalid)
	}
	if err := validateUsername(username); err != nil {
		return domain.User{}, domain.SessionTokens{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	id, err := security.NewID()
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, err
	}
	user, err := s.store.CreateUser(ctx, domain.User{ID: id, Email: email, DisplayName: username,
		Role: domain.RolePlayer, PasswordHash: passwordHash, PasswordSet: true}, engine.CompetitiveRulesetVersion)
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, err
	}
	tokens, err := s.newSession(ctx, user.ID)
	return user, tokens, err
}

func validateUsername(username string) error {
	if n := len(username); n < 2 || n > 32 {
		return errors.New("nome de usuário deve ter entre 2 e 32 caracteres")
	}
	if !usernamePattern.MatchString(username) {
		return errors.New("nome de usuário aceita apenas letras, números, hífen (-) e sublinhado (_), sem espaços")
	}
	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (domain.User, domain.SessionTokens, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, domain.SessionTokens{}, domain.ErrInvalidCredentials
	}
	user, err := s.store.UserByEmail(ctx, normalized)
	if err != nil || !security.VerifyPassword(user.PasswordHash, password) {
		return domain.User{}, domain.SessionTokens{}, domain.ErrInvalidCredentials
	}
	tokens, err := s.newSession(ctx, user.ID)
	return user, tokens, err
}

func (s *Service) newSession(ctx context.Context, userID string) (domain.SessionTokens, error) {
	access, accessHash, err := security.NewToken()
	if err != nil {
		return domain.SessionTokens{}, err
	}
	refresh, refreshHash, err := security.NewToken()
	if err != nil {
		return domain.SessionTokens{}, err
	}
	id, err := security.NewID()
	if err != nil {
		return domain.SessionTokens{}, err
	}
	now := s.now().UTC()
	tokens := domain.SessionTokens{AccessToken: access, RefreshToken: refresh,
		AccessExpiry: now.Add(AccessTTL), RefreshExpiry: now.Add(RefreshTTL)}
	err = s.store.CreateSession(ctx, domain.NewSession{ID: id, UserID: userID, AccessHash: accessHash,
		RefreshHash: refreshHash, AccessUntil: tokens.AccessExpiry, RefreshUntil: tokens.RefreshExpiry})
	return tokens, err
}

func (s *Service) Authenticate(ctx context.Context, token string) (domain.Principal, error) {
	if token == "" {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	record, err := s.store.AccessToken(ctx, security.TokenHash(token), s.now().UTC())
	if err != nil {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	return record.Principal, nil
}

func (s *Service) Refresh(ctx context.Context, refresh string) (domain.SessionTokens, error) {
	access, accessHash, err := security.NewToken()
	if err != nil {
		return domain.SessionTokens{}, err
	}
	nextRefresh, refreshHash, err := security.NewToken()
	if err != nil {
		return domain.SessionTokens{}, err
	}
	now := s.now().UTC()
	tokens := domain.SessionTokens{AccessToken: access, RefreshToken: nextRefresh,
		AccessExpiry: now.Add(AccessTTL), RefreshExpiry: now.Add(RefreshTTL)}
	_, err = s.store.RotateSession(ctx, security.TokenHash(refresh), domain.RotatedSession{
		AccessHash: accessHash, RefreshHash: refreshHash, AccessUntil: tokens.AccessExpiry,
		RefreshUntil: tokens.RefreshExpiry}, now)
	if err != nil {
		return domain.SessionTokens{}, domain.ErrInvalidCredentials
	}
	return tokens, nil
}

func (s *Service) Logout(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}
	return s.store.RevokeSession(ctx, security.TokenHash(refresh))
}

func (s *Service) Collection(ctx context.Context, principal domain.Principal) (domain.Collection, error) {
	collection, err := s.store.Collection(ctx, principal.UserID, engine.CompetitiveRulesetVersion)
	if err != nil {
		return domain.Collection{}, err
	}
	account, err := s.AccountProgress(ctx, principal)
	if err != nil {
		return domain.Collection{}, err
	}
	collection.Cards = cardsAvailableAtLevel(collection.Cards, account.Level)
	return collection, nil
}

func cardsAvailableAtLevel(cards []domain.CollectionCard, level int) []domain.CollectionCard {
	ruleset := engine.CompetitiveRuleset()
	available := make([]domain.CollectionCard, 0, len(cards))
	for _, owned := range cards {
		card := ruleset.Cards[owned.CardID]
		if card != nil && card.Rarity == engine.RarityLendaria &&
			level < accountprogress.LegendaryUnlockLevel(card.ID) {
			continue
		}
		available = append(available, owned)
	}
	return available
}

// BattleDeck carrega um baralho e revalida a progressão imediatamente antes
// de treino/PvP. Isso bloqueia também composições antigas que contenham uma
// Lendária hoje acima do nível da conta.
func (s *Service) BattleDeck(ctx context.Context, principal domain.Principal, id string) (domain.Deck, int, error) {
	deck, err := s.store.Deck(ctx, principal.UserID, id)
	if err != nil {
		return domain.Deck{}, 0, err
	}
	account, err := s.AccountProgress(ctx, principal)
	if err != nil {
		return domain.Deck{}, 0, err
	}
	collection, err := s.Collection(ctx, principal)
	if err != nil {
		return domain.Deck{}, 0, err
	}
	owned := make(map[string]int, len(collection.Cards))
	for _, card := range collection.Cards {
		owned[card.CardID] = card.Quantity
	}
	for _, selected := range deck.Cards {
		if owned[selected.CardID] >= selected.Quantity {
			continue
		}
		if card := engine.CompetitiveRuleset().Cards[selected.CardID]; card != nil && card.Rarity == engine.RarityLendaria {
			return domain.Deck{}, 0, fmt.Errorf("%w: %s é Lendária e desbloqueia no nível %d",
				domain.ErrInvalid, card.Name, accountprogress.LegendaryUnlockLevel(card.ID))
		}
		return domain.Deck{}, 0, fmt.Errorf("%w: baralho usa carta indisponível %s", domain.ErrInvalid, selected.CardID)
	}
	return deck, account.Level, nil
}

func (s *Service) ListDecks(ctx context.Context, principal domain.Principal) ([]domain.Deck, error) {
	return s.store.ListDecks(ctx, principal.UserID)
}

func (s *Service) Deck(ctx context.Context, principal domain.Principal, id string) (domain.Deck, error) {
	return s.store.Deck(ctx, principal.UserID, id)
}

func (s *Service) SaveDeck(ctx context.Context, principal domain.Principal, deck domain.Deck,
	expected *int64, key string, rawBody []byte) (domain.Deck, bool, error) {
	if err := validateIdempotencyKey(key); err != nil {
		return domain.Deck{}, false, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	deck.UserID = principal.UserID
	deck.RulesetVersion = engine.CompetitiveRulesetVersion
	deck.Active = true
	deck.SystemProvided = false
	lockUntil := s.now().UTC().Add(deckLockDuration())
	deck.LockedUntil = &lockUntil
	deck.Name = strings.TrimSpace(deck.Name)
	if len([]rune(deck.Name)) < 1 || len([]rune(deck.Name)) > 64 {
		return domain.Deck{}, false, fmt.Errorf("%w: nome do deck deve ter entre 1 e 64 caracteres", domain.ErrInvalid)
	}
	if deck.ID == "" {
		id, err := security.NewID()
		if err != nil {
			return domain.Deck{}, false, err
		}
		deck.ID = id
	}
	if err := ValidateDeck(deck); err != nil {
		return domain.Deck{}, false, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	collection, err := s.Collection(ctx, principal)
	if err != nil {
		return domain.Deck{}, false, err
	}
	owned := make(map[string]int, len(collection.Cards))
	for _, card := range collection.Cards {
		owned[card.CardID] = card.Quantity
	}
	championOwned := false
	for _, champion := range collection.Champions {
		championOwned = championOwned || champion == deck.ChampionID
	}
	if !championOwned {
		return domain.Deck{}, false, fmt.Errorf("%w: campeão não pertence à coleção", domain.ErrInvalid)
	}
	for _, card := range deck.Cards {
		if owned[card.CardID] < card.Quantity {
			return domain.Deck{}, false, fmt.Errorf("%w: coleção não possui %d cópia(s) de %s", domain.ErrInvalid, card.Quantity, card.CardID)
		}
	}
	hash := sha256.Sum256(rawBody)
	operation := "deck:create"
	if expected != nil {
		operation = "deck:update:" + deck.ID
	}
	return s.store.SaveDeck(ctx, deck, expected, domain.Mutation{Key: key, Operation: operation, RequestHash: hash[:]})
}

func ValidateDeck(deck domain.Deck) error {
	flat := make([]string, 0, engine.DeckSizeForVersion(deck.RulesetVersion))
	seen := make(map[string]bool, len(deck.Cards))
	for _, card := range deck.Cards {
		if seen[card.CardID] {
			return fmt.Errorf("carta repetida na lista: %s", card.CardID)
		}
		seen[card.CardID] = true
		if card.Quantity < 1 || card.Quantity > engine.MaxCopies {
			return fmt.Errorf("quantidade inválida para %s", card.CardID)
		}
		for range card.Quantity {
			flat = append(flat, card.CardID)
		}
	}
	return engine.ValidateDeckForVersion(deck.RulesetVersion, deck.ChampionID, flat)
}

func (s *Service) DeleteDeck(ctx context.Context, principal domain.Principal, deckID string,
	version int64, key string) (bool, error) {
	if err := validateIdempotencyKey(key); err != nil {
		return false, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if deck, err := s.store.Deck(ctx, principal.UserID, deckID); err == nil && engine.IsConfrontVersion(deck.RulesetVersion) {
		return false, fmt.Errorf("%w: o baralho ativo do Confronto não pode ser excluído; edite-o quando a trava terminar", domain.ErrInvalid)
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", deckID, version)))
	return s.store.DeleteDeck(ctx, principal.UserID, deckID, version,
		domain.Mutation{Key: key, Operation: "deck:delete:" + deckID, RequestHash: hash[:]})
}

func (s *Service) ActiveSeason(ctx context.Context) (domain.Season, error) {
	return s.store.ActiveSeason(ctx)
}

func (s *Service) ListRewards(ctx context.Context, principal domain.Principal) ([]domain.Reward, error) {
	return s.store.ListRewards(ctx, principal.UserID)
}

func (s *Service) GrantReward(ctx context.Context, principal domain.Principal, reward domain.Reward,
	key string, rawBody []byte) (domain.Reward, bool, error) {
	if principal.Role != domain.RoleAdmin {
		return reward, false, domain.ErrForbidden
	}
	if err := validateIdempotencyKey(key); err != nil {
		return reward, false, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if reward.Quantity < 1 || reward.Quantity > 100 || (reward.CardID == "") == (reward.ChampionID == "") {
		return reward, false, fmt.Errorf("%w: recompensa inválida", domain.ErrInvalid)
	}
	if reward.ChampionID != "" && reward.Quantity != 1 {
		return reward, false, fmt.Errorf("%w: recompensa de Campeão deve ter quantidade 1", domain.ErrInvalid)
	}
	if reward.CardID != "" && engine.Cards[reward.CardID] == nil {
		return reward, false, fmt.Errorf("%w: carta desconhecida", domain.ErrInvalid)
	}
	if reward.ChampionID != "" && engine.Champions[reward.ChampionID] == nil {
		return reward, false, fmt.Errorf("%w: campeão desconhecido", domain.ErrInvalid)
	}
	if _, err := s.store.UserByID(ctx, reward.UserID); err != nil {
		return reward, false, err
	}
	id, err := security.NewID()
	if err != nil {
		return reward, false, err
	}
	reward.ID = id
	reward.RulesetVersion = engine.CompetitiveRulesetVersion
	hash := sha256.Sum256(rawBody)
	return s.store.GrantReward(ctx, reward, domain.Mutation{Key: key,
		Operation: "reward:grant", RequestHash: hash[:], ScopeUserID: principal.UserID})
}

func ChampionsSorted() []*engine.ChampionDef {
	rs := engine.CompetitiveRuleset()
	ids := make([]string, 0, len(rs.Champions))
	for id := range rs.Champions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]*engine.ChampionDef, 0, len(ids))
	for _, id := range ids {
		result = append(result, rs.Champions[id])
	}
	return result
}

func deckLockDuration() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("NYTHARA_DECK_LOCK_DURATION")); raw != "" {
		if duration, err := time.ParseDuration(raw); err == nil && duration >= 0 {
			return duration
		}
	}
	return DefaultDeckLock
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || len(value) > 320 {
		return "", fmt.Errorf("e-mail inválido")
	}
	return value, nil
}

func validateIdempotencyKey(key string) error {
	if len(key) < 8 || len(key) > 128 || strings.TrimSpace(key) != key {
		return fmt.Errorf("Idempotency-Key deve ter entre 8 e 128 caracteres")
	}
	return nil
}

func IsClientError(err error) bool {
	return errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrIdempotencyConflict) || errors.Is(err, domain.ErrForbidden) ||
		errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrInvalidCredentials)
}
