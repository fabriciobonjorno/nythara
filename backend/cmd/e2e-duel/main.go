// Command e2e-duel atravessa a aplicação em execução como dois clientes reais:
// cadastro, deck, matchmaking, WebSocket, fases, informação oculta e rating.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

type client struct {
	base  string
	http  *http.Client
	token string
}

type authEnvelope struct {
	User   domain.User `json:"user"`
	Tokens struct {
		Access string `json:"access_token"`
	} `json:"tokens"`
}

type ranked struct {
	Rating int `json:"rating"`
	Games  int `json:"games"`
}

type progress struct {
	Ranked *ranked `json:"ranked"`
}

type practiceResult struct {
	MatchID        string `json:"match_id"`
	RulesetVersion string `json:"ruleset_version"`
	CardsPlayed    int    `json:"human_cards_played"`
	BotPlayed      bool   `json:"bot_played"`
	Rounds         int    `json:"rounds"`
	EndReason      string `json:"end_reason"`
	Opening        [2]int `json:"opening_vitality"`
	Final          [2]int `json:"final_vitality"`
}

type peer struct {
	name string
	conn *websocket.Conn
	slot int

	mu     sync.RWMutex
	state  *battle.StateView
	events []engine.Event
	seq    int64
	acks   chan error
}

func main() {
	base := flag.String("base-url", "http://127.0.0.1:18080", "URL da API em execução")
	concedeEarly := flag.Bool("concede-early", false, "encerra no segundo turno (somente smoke test)")
	flag.Parse()
	if err := run(strings.TrimRight(*base, "/"), *concedeEarly); err != nil {
		panic(err)
	}
}

func run(base string, concedeEarly bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	suffix := time.Now().UTC().Format("150405000000")
	root := &client{base: base, http: &http.Client{Timeout: 8 * time.Second}}
	if status, _, err := root.request(ctx, http.MethodGet, "/readyz", nil, nil); err != nil || status != http.StatusOK {
		return fmt.Errorf("API não está pronta: status=%d err=%v", status, err)
	}

	// O servidor, e não apenas o HTML, precisa recusar nomes inválidos.
	invalid := map[string]string{"email": "invalid-" + suffix + "@example.test", "password": "senha-e2e-segura", "username": "nome invalido"}
	if status, _, err := root.request(ctx, http.MethodPost, "/v1/auth/register", invalid, nil); err != nil || status != http.StatusBadRequest {
		return fmt.Errorf("username com espaço não foi recusado: status=%d err=%v", status, err)
	}

	a0, c0, err := register(ctx, root, "e2e_a_"+suffix, "e2e-a-"+suffix+"@example.test")
	if err != nil {
		return err
	}
	_, c1, err := register(ctx, root, "e2e_b_"+suffix, "e2e-b-"+suffix+"@example.test")
	if err != nil {
		return err
	}

	duplicateEmail := map[string]string{"email": a0.User.Email, "password": "senha-e2e-segura", "username": "email_dup_" + suffix}
	if status, _, err := root.request(ctx, http.MethodPost, "/v1/auth/register", duplicateEmail, nil); err != nil || status != http.StatusConflict {
		return fmt.Errorf("e-mail duplicado não produziu 409: status=%d err=%v", status, err)
	}
	duplicateUsername := map[string]string{"email": "user-dup-" + suffix + "@example.test", "password": "senha-e2e-segura", "username": strings.ToUpper(a0.User.DisplayName)}
	if status, _, err := root.request(ctx, http.MethodPost, "/v1/auth/register", duplicateUsername, nil); err != nil || status != http.StatusConflict {
		return fmt.Errorf("username duplicado sem diferenciar caixa não produziu 409: status=%d err=%v", status, err)
	}

	var catalog struct {
		Cards []engine.CardDef `json:"cards"`
	}
	if status, _, err := root.request(ctx, http.MethodGet, "/v1/catalog/cards", nil, &catalog); err != nil || status != http.StatusOK || len(catalog.Cards) != len(engine.CardList) {
		return fmt.Errorf("catálogo não entregou as %d cartas do ruleset: status=%d cartas=%d err=%v", len(engine.CardList), status, len(catalog.Cards), err)
	}

	d0, err := createPrecon(ctx, c0, "CH-VH-01", "precon-a-"+suffix)
	if err != nil {
		return err
	}
	d1, err := createPrecon(ctx, c1, "CH-SO-01", "precon-b-"+suffix)
	if err != nil {
		return err
	}
	if totalCards(d0.Cards) != engine.ConfrontDeckSize || totalCards(d1.Cards) != engine.ConfrontDeckSize {
		return fmt.Errorf("deck salvo fora do tamanho legal: %d/%d", totalCards(d0.Cards), totalCards(d1.Cards))
	}

	before0, err := getProgress(ctx, c0)
	if err != nil {
		return err
	}
	before1, err := getProgress(ctx, c1)
	if err != nil {
		return err
	}

	var q0 battle.QueueResult
	if status, _, err := c0.request(ctx, http.MethodPost, "/v1/matchmaking", map[string]string{"deck_id": d0.ID}, &q0); err != nil || status != http.StatusOK || q0.Status != "queued" {
		return fmt.Errorf("primeiro jogador não entrou na fila: status=%d result=%+v err=%v", status, q0, err)
	}
	var q1 battle.QueueResult
	if status, _, err := c1.request(ctx, http.MethodPost, "/v1/matchmaking", map[string]string{"deck_id": d1.ID}, &q1); err != nil || status != http.StatusOK || q1.Status != "matched" {
		return fmt.Errorf("matchmaking não formou partida: status=%d result=%+v err=%v", status, q1, err)
	}
	var q0After battle.QueueResult
	if status, _, err := c0.request(ctx, http.MethodGet, "/v1/matchmaking", nil, &q0After); err != nil || status != http.StatusOK || q0After.MatchID != q1.MatchID {
		return fmt.Errorf("primeiro jogador não recebeu o match: status=%d result=%+v err=%v", status, q0After, err)
	}

	p0, err := connectPeer(ctx, c0, q1.MatchID, "jogador A")
	if err != nil {
		return err
	}
	defer p0.conn.CloseNow()
	p1, err := connectPeer(ctx, c1, q1.MatchID, "jogador B")
	if err != nil {
		return err
	}
	defer p1.conn.CloseNow()
	peers := [2]*peer{}
	peers[p0.slot], peers[p1.slot] = p0, p1

	if err := p0.ready(ctx); err != nil {
		return err
	}
	if err := p1.ready(ctx); err != nil {
		return err
	}
	if err := waitFor(ctx, func() bool {
		return p0.snapshot() != nil && p1.snapshot() != nil && p0.snapshot().Phase == engine.PhaseAssault && p1.snapshot().Phase == engine.PhaseAssault
	}); err != nil {
		return fmt.Errorf("partida não iniciou: %w", err)
	}
	if err := validateHiddenHands(p0, p1); err != nil {
		return err
	}
	opening := [2]int{p0.snapshot().Players[0].Vitality, p0.snapshot().Players[1].Vitality}

	played := 0
	viewPeer := p1
	for commands := 0; commands < 600; commands++ {
		state := viewPeer.snapshot()
		if state == nil {
			return errors.New("estado da batalha desapareceu")
		}
		if state.Over {
			break
		}
		actor := expectedActor(state)
		current := peers[actor]
		targetRound, targetPhase := state.Round, state.Phase
		if err := waitFor(ctx, func() bool {
			own := current.snapshot()
			return own != nil && own.Round == targetRound && own.Phase == targetPhase && expectedActor(own) == actor
		}); err != nil {
			return fmt.Errorf("estado não chegou ao jogador responsável: %w", err)
		}
		// Toda escolha de mão/decisão usa a visão privada do próprio ator.
		state = current.snapshot()
		viewPeer = current
		intent := battle.Intent{Kind: engine.CmdKindPass}

		switch {
		case state.Pending != nil:
			decision := state.Pending
			choices := append([]string{}, decision.Options...)
			n := decision.N
			if n == 0 {
				n = 1
			}
			if len(choices) > n {
				choices = choices[:n]
			}
			intent = battle.Intent{Kind: engine.CmdKindChoose, Cards: choices, DecisionID: decision.ID}
		default:
			if card := playableCard(state, actor); card != "" {
				intent = battle.Intent{Kind: engine.CmdKindPlay, Card: card}
			}
		}
		if err := current.send(ctx, intent); err != nil {
			// Algumas cartas têm pré-condições próprias; a engine recusou sem
			// consumir sequência. Encerrar a janela continua sendo uma ação real.
			if intent.Kind != engine.CmdKindPlay {
				return fmt.Errorf("comando %s recusado: %w", intent.Kind, err)
			}
			if err := current.send(ctx, battle.Intent{Kind: engine.CmdKindPass}); err != nil {
				return fmt.Errorf("pass após carta contextual: %w", err)
			}
		} else if intent.Kind == engine.CmdKindPlay {
			played++
		}
		// Um cliente real não dispara centenas de ações no mesmo instante. Além
		// de reproduzir esse ritmo, a cadência mantém o teste dentro do contrato
		// público de 120 comandos/minuto por conexão.
		if err := wait(ctx, 325*time.Millisecond); err != nil {
			return err
		}
		// A confirmação com client_sequence é entregue ao emissor junto do
		// StateView atualizado. O outro socket pode receber o broadcast alguns
		// microssegundos depois; portanto ele não é usado como relógio global.
		viewPeer = current

		state = viewPeer.snapshot()
		if concedeEarly && played > 0 && state != nil && state.Round >= 2 && state.Pending == nil {
			actor = expectedActor(state)
			if err := peers[actor].send(ctx, battle.Intent{Kind: engine.CmdKindConcede}); err != nil {
				return fmt.Errorf("concessão real: %w", err)
			}
			viewPeer = peers[actor]
			break
		}
	}

	if err := waitFor(ctx, func() bool { return viewPeer.snapshot() != nil && viewPeer.snapshot().Over }); err != nil {
		return fmt.Errorf("partida não terminou: %w", err)
	}
	final := viewPeer.snapshot()
	if played == 0 {
		return errors.New("nenhuma carta pôde ser jogada no duelo E2E")
	}
	if !concedeEarly && final.EndReason == "concede" {
		return errors.New("duelo completo terminou por concessão")
	}
	if !concedeEarly && final.Players[0].Vitality >= opening[0] && final.Players[1].Vitality >= opening[1] {
		return fmt.Errorf("nenhuma Vitalidade foi deduzida: inicial=%v final=[%d %d]", opening,
			final.Players[0].Vitality, final.Players[1].Vitality)
	}
	if !hasDrawAfterOpening(p0.allEvents()) {
		return errors.New("não houve compra de carta numa rodada posterior à abertura")
	}

	var after0, after1 progress
	if err := waitFor(ctx, func() bool {
		var err0, err1 error
		after0, err0 = getProgress(ctx, c0)
		after1, err1 = getProgress(ctx, c1)
		return err0 == nil && err1 == nil && after0.Ranked != nil && after1.Ranked != nil &&
			after0.Ranked.Games == before0.Ranked.Games+1 && after1.Ranked.Games == before1.Ranked.Games+1
	}); err != nil {
		return fmt.Errorf("rating não foi persistido: %w", err)
	}
	loser := after0.Ranked
	winner := after1.Ranked
	if final.Winner == p0.slot {
		winner, loser = after0.Ranked, after1.Ranked
	}
	if winner.Rating <= 1000 || loser.Rating >= 1000 {
		return fmt.Errorf("pontos não foram somados/deduzidos: vencedor=%d perdedor=%d", winner.Rating, loser.Rating)
	}
	practice, err := runPractice(ctx, c0, d0)
	if err != nil {
		return err
	}
	afterPractice, err := getProgress(ctx, c0)
	if err != nil {
		return err
	}
	if afterPractice.Ranked.Games != after0.Ranked.Games || afterPractice.Ranked.Rating != after0.Ranked.Rating {
		return fmt.Errorf("treino alterou rating competitivo: antes=%+v depois=%+v", after0.Ranked, afterPractice.Ranked)
	}
	if !hasConfrontationLifecycle(p0.allEvents()) {
		return errors.New("o duelo não emitiu o ciclo completo abertura → resolução → estilhaço")
	}

	summary := map[string]any{
		"status":                  "ok",
		"match_id":                q1.MatchID,
		"ruleset_version":         final.RulesetVersion,
		"cards_catalog":           len(catalog.Cards),
		"deck_size":               engine.ConfrontDeckSize,
		"cards_played":            played,
		"opening_vitality":        opening,
		"final_vitality":          [2]int{final.Players[0].Vitality, final.Players[1].Vitality},
		"rounds":                  final.Round,
		"end_reason":              final.EndReason,
		"natural_finish":          !concedeEarly,
		"hidden_opponent_hand":    true,
		"central_confrontation":   true,
		"losing_card_shattered":   true,
		"draw_after_opening":      true,
		"winner_rating":           winner.Rating,
		"loser_rating":            loser.Rating,
		"winner_games":            winner.Games,
		"loser_games":             loser.Games,
		"practice":                practice,
		"practice_rating_ignored": true,
		"unique_email":            true,
		"unique_username_ci":      true,
		"username_format":         true,
	}
	out, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(out))
	return nil
}

func runPractice(ctx context.Context, c *client, deck domain.Deck) (practiceResult, error) {
	var result practiceResult
	var queued battle.QueueResult
	status, raw, err := c.request(ctx, http.MethodPost, "/v1/practice",
		map[string]string{"deck_id": deck.ID, "bot_champion_id": "CH-SO-01"}, &queued)
	if err != nil || status != http.StatusOK || queued.Status != "matched" {
		return result, fmt.Errorf("iniciar treino: status=%d body=%s result=%+v err=%v", status, raw, queued, err)
	}
	p, err := connectPeer(ctx, c, queued.MatchID, "jogador de treino")
	if err != nil {
		return result, err
	}
	defer p.conn.CloseNow()
	if p.slot != 0 {
		return result, fmt.Errorf("humano entrou no assento %d do treino", p.slot)
	}
	if err := p.ready(ctx); err != nil {
		return result, err
	}
	if err := waitFor(ctx, func() bool { return p.snapshot() != nil }); err != nil {
		return result, fmt.Errorf("treino não iniciou: %w", err)
	}
	initial := p.snapshot()
	if len(initial.Players[0].Hand) == 0 || initial.Players[1].HandCount == 0 || len(initial.Players[1].Hand) != 0 {
		return result, errors.New("treino não preservou a mão privada do bot/humano")
	}
	result.MatchID = queued.MatchID
	result.RulesetVersion = initial.RulesetVersion
	result.Opening = [2]int{initial.Players[0].Vitality, initial.Players[1].Vitality}

	for commands := 0; commands < 600; commands++ {
		state := p.snapshot()
		if state == nil {
			return result, errors.New("estado do treino desapareceu")
		}
		if state.Over {
			break
		}
		if expectedActor(state) != 0 {
			if err := waitFor(ctx, func() bool {
				next := p.snapshot()
				return next != nil && (next.Over || expectedActor(next) == 0)
			}); err != nil {
				return result, fmt.Errorf("bot não devolveu a vez ao humano: %w", err)
			}
			continue
		}
		intent := battle.Intent{Kind: engine.CmdKindPass}
		switch {
		case state.Pending != nil:
			decision := state.Pending
			choices := append([]string{}, decision.Options...)
			n := decision.N
			if n == 0 {
				n = 1
			}
			if len(choices) > n {
				choices = choices[:n]
			}
			intent = battle.Intent{Kind: engine.CmdKindChoose, Cards: choices, DecisionID: decision.ID}
		default:
			if card := playableCard(state, 0); card != "" {
				intent = battle.Intent{Kind: engine.CmdKindPlay, Card: card}
			}
		}
		if err := p.send(ctx, intent); err != nil {
			if intent.Kind != engine.CmdKindPlay {
				return result, fmt.Errorf("comando de treino %s recusado: %w", intent.Kind, err)
			}
			if err := p.send(ctx, battle.Intent{Kind: engine.CmdKindPass}); err != nil {
				return result, fmt.Errorf("pass do treino após carta contextual: %w", err)
			}
		} else if intent.Kind == engine.CmdKindPlay {
			result.CardsPlayed++
		}
		// No treino o bot responde dentro do servidor; o humano pode receber a
		// vez de volta imediatamente. 550 ms mantém também esse caso abaixo dos
		// 120 comandos/minuto contratados pelo WebSocket.
		if err := wait(ctx, 550*time.Millisecond); err != nil {
			return result, err
		}
	}
	if err := waitFor(ctx, func() bool { return p.snapshot() != nil && p.snapshot().Over }); err != nil {
		return result, fmt.Errorf("treino não terminou: %w", err)
	}
	final := p.snapshot()
	result.Rounds = final.Round
	result.EndReason = final.EndReason
	result.Final = [2]int{final.Players[0].Vitality, final.Players[1].Vitality}
	for _, event := range p.allEvents() {
		if event.Kind == engine.EvCardPlayed && event.P == 1 {
			result.BotPlayed = true
			break
		}
	}
	if result.CardsPlayed == 0 || !result.BotPlayed {
		return result, fmt.Errorf("treino sem jogadas reais: humano=%d bot=%t", result.CardsPlayed, result.BotPlayed)
	}
	if final.EndReason == "concede" || (final.Players[0].Vitality >= result.Opening[0] && final.Players[1].Vitality >= result.Opening[1]) {
		return result, fmt.Errorf("treino não terminou naturalmente com dedução de Vitalidade: reason=%s inicial=%v final=%v",
			final.EndReason, result.Opening, result.Final)
	}
	return result, nil
}

func register(ctx context.Context, root *client, username, email string) (authEnvelope, *client, error) {
	var auth authEnvelope
	body := map[string]string{"email": email, "password": "senha-e2e-segura", "username": username}
	status, raw, err := root.request(ctx, http.MethodPost, "/v1/auth/register", body, &auth)
	if err != nil || status != http.StatusCreated {
		return auth, nil, fmt.Errorf("cadastro de %s: status=%d body=%s err=%v", username, status, raw, err)
	}
	return auth, &client{base: root.base, http: root.http, token: auth.Tokens.Access}, nil
}

func createPrecon(ctx context.Context, c *client, champion, key string) (domain.Deck, error) {
	var deck domain.Deck
	headers := map[string]string{"Idempotency-Key": key}
	status, raw, err := c.requestHeaders(ctx, http.MethodPost, "/v1/decks/precon", map[string]string{"champion_id": champion}, headers, &deck)
	if err != nil || status != http.StatusCreated {
		return deck, fmt.Errorf("criar precon %s: status=%d body=%s err=%v", champion, status, raw, err)
	}
	return deck, nil
}

func getProgress(ctx context.Context, c *client) (progress, error) {
	var result progress
	status, raw, err := c.request(ctx, http.MethodGet, "/v1/progress", nil, &result)
	if err != nil || status != http.StatusOK || result.Ranked == nil {
		return result, fmt.Errorf("ler progresso: status=%d body=%s err=%v", status, raw, err)
	}
	return result, nil
}

func (c *client) request(ctx context.Context, method, path string, body, out any) (int, []byte, error) {
	return c.requestHeaders(ctx, method, path, body, nil, out)
}

func (c *client) requestHeaders(ctx context.Context, method, path string, body any, headers map[string]string, out any) (int, []byte, error) {
	var source io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		source = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, source)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, raw, err
	}
	if out != nil && len(raw) > 0 && response.StatusCode < 300 {
		if err := json.Unmarshal(raw, out); err != nil {
			return response.StatusCode, raw, err
		}
	}
	return response.StatusCode, raw, nil
}

func connectPeer(ctx context.Context, c *client, matchID, name string) (*peer, error) {
	var ticket battle.Ticket
	status, raw, err := c.request(ctx, http.MethodPost, "/v1/battles/"+matchID+"/tickets", map[string]string{"mode": "player"}, &ticket)
	if err != nil || status != http.StatusCreated {
		return nil, fmt.Errorf("ticket %s: status=%d body=%s err=%v", name, status, raw, err)
	}
	wsBase := "ws" + strings.TrimPrefix(c.base, "http")
	conn, _, err := websocket.Dial(ctx, wsBase+"/v1/battles/"+matchID+"/ws?ticket="+ticket.Token+"&after_event=-1", nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket %s: %w", name, err)
	}
	p := &peer{name: name, conn: conn, slot: ticket.Slot, acks: make(chan error, 8)}
	go p.read(ctx)
	return p, nil
}

func (p *peer) read(ctx context.Context) {
	for {
		_, raw, err := p.conn.Read(ctx)
		if err != nil {
			select {
			case p.acks <- fmt.Errorf("WebSocket de %s encerrou: %w", p.name, err):
			default:
			}
			return
		}
		var message battle.ServerMessage
		if json.Unmarshal(raw, &message) != nil {
			continue
		}
		p.mu.Lock()
		if message.State != nil {
			copyState := *message.State
			p.state = &copyState
		}
		p.events = append(p.events, message.Events...)
		p.mu.Unlock()
		if message.Error != nil {
			p.acks <- errors.New(message.Error.Message)
		} else if message.ClientSequence > 0 {
			p.acks <- nil
		}
	}
}

func (p *peer) ready(ctx context.Context) error {
	return p.conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ready"}`))
}

func (p *peer) send(ctx context.Context, intent battle.Intent) error {
	p.mu.RLock()
	sequence := p.seq + 1
	p.mu.RUnlock()
	payload := map[string]any{"type": "command", "client_sequence": sequence, "command": intent}
	raw, _ := json.Marshal(payload)
	if err := p.conn.Write(ctx, websocket.MessageText, raw); err != nil {
		return err
	}
	select {
	case err := <-p.acks:
		if err == nil {
			p.mu.Lock()
			p.seq = sequence
			p.mu.Unlock()
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *peer) snapshot() *battle.StateView {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.state == nil {
		return nil
	}
	copyState := *p.state
	return &copyState
}

func (p *peer) allEvents() []engine.Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]engine.Event{}, p.events...)
}

func validateHiddenHands(p0, p1 *peer) error {
	for _, p := range []*peer{p0, p1} {
		state := p.snapshot()
		if state == nil || len(state.Players[p.slot].Hand) == 0 || state.Players[1-p.slot].HandCount == 0 {
			return fmt.Errorf("mão inicial ausente para %s", p.name)
		}
		if len(state.Players[1-p.slot].Hand) != 0 {
			return fmt.Errorf("%s recebeu IDs da mão adversária", p.name)
		}
		for _, card := range state.Cards {
			if card.Owner == 1-p.slot && card.Zone == engine.ZoneHand {
				return fmt.Errorf("%s recebeu definição de carta adversária virada", p.name)
			}
		}
	}
	return nil
}

func playableCard(state *battle.StateView, slot int) string {
	ruleset, err := engine.RulesetByVersion(state.RulesetVersion)
	if err != nil {
		return ""
	}
	for _, instance := range state.Players[slot].Hand {
		view, ok := state.Cards[instance]
		if !ok {
			continue
		}
		card := ruleset.Cards[view.Def]
		if card == nil || card.Confront == nil || !card.Confront.Legal {
			continue
		}
		cost := card.Cost
		if card.Type == engine.TypeRito {
			if fx := ruleset.Effects.Cards[card.ID]; fx != nil && fx.Rite != nil {
				cost += fx.Rite.Sacrifice
			}
		}
		if state.Players[slot].Vitality-cost < 1 {
			continue
		}
		if state.Phase == engine.PhaseRite && card.Type == engine.TypeRito {
			return instance
		}
		if state.Phase == engine.PhaseAssault && card.Type == engine.TypeAssalto {
			return instance
		}
		if state.Phase == engine.PhaseGuard && card.Type == engine.TypeGuarda && state.Confront != nil && state.Confront.Defender == slot {
			return instance
		}
	}
	return ""
}

func expectedActor(state *battle.StateView) int {
	if state.Pending != nil {
		return state.Pending.Player
	}
	order := [2]int{state.Initiative, 1 - state.Initiative}
	if state.Phase == engine.PhaseMulligan {
		for _, slot := range order {
			if !state.Players[slot].MulliganDone {
				return slot
			}
		}
	}
	if state.Phase == engine.PhaseStance {
		for _, slot := range order {
			if !state.Players[slot].StanceCommitted {
				return slot
			}
		}
	}
	if state.RiteReact != nil {
		return 1 - state.RiteReact.Caster
	}
	if state.Guard != nil {
		return state.Guard.Defender
	}
	if state.Extra != nil {
		return state.Extra.Player
	}
	return state.Active
}

func hasDrawAfterOpening(events []engine.Event) bool {
	for _, event := range events {
		if event.Kind == engine.EvCardDrawn && event.Round >= 2 {
			return true
		}
	}
	return false
}

func hasConfrontationLifecycle(events []engine.Event) bool {
	opened, resolved, shattered := false, false, false
	for _, event := range events {
		switch event.Kind {
		case engine.EvConfrontationOpened:
			opened = true
		case engine.EvConfrontationResolved:
			resolved = true
		case engine.EvCardShattered:
			shattered = true
		}
	}
	return opened && resolved && shattered
}

func totalCards(cards []domain.DeckCard) int {
	total := 0
	for _, card := range cards {
		total += card.Quantity
	}
	return total
}

func waitFor(ctx context.Context, condition func() bool) error {
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
