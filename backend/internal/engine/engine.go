package engine

import (
	"encoding/json"
	"fmt"
	"sort"
)

// PlayerSetup configura um lado da partida.
type PlayerSetup struct {
	ChampionID string   `json:"champion_id"`
	Deck       []string `json:"deck"` // 36 ids de definição (VR-xxx)
}

// Config define uma partida por completo. ruleset + seed + decks + command
// log reproduzem a partida integralmente.
type Config struct {
	RulesetVersion string         `json:"ruleset_version"`
	Seed           uint64         `json:"seed"`
	Players        [2]PlayerSetup `json:"players"`

	// Ganchos de teste/tutorial. Partidas competitivas usam ambos zerados.
	SkipShuffle bool `json:"skip_shuffle,omitempty"` // mantém a ordem dada dos decks
	FirstPlayer int  `json:"first_player,omitempty"` // -1 = sorteio; 0/1 força iniciativa
}

// Game é uma partida em andamento. Não é thread-safe: o chamador serializa
// comandos (uma goroutine por sala no battle server).
type Game struct {
	Cfg        Config
	rs         *Ruleset
	s          *GameState
	Log        []Event
	CommandLog []Command
}

// Ruleset devolve o conjunto de regras sob o qual a partida executa.
func (g *Game) Ruleset() *Ruleset { return g.rs }

// State expõe o estado para leitura. O chamador não deve mutar.
func (g *Game) State() *GameState { return g.s }

// NewGame valida a configuração, embaralha decks, compra mãos iniciais e
// entra na fase de Mulligan.
func NewGame(cfg Config) (*Game, error) {
	rs, err := RulesetByVersion(cfg.RulesetVersion)
	if err != nil {
		return nil, err
	}
	for i, p := range cfg.Players {
		if rs.Champions[p.ChampionID] == nil {
			return nil, fmt.Errorf("jogador %d: campeão desconhecido %q", i, p.ChampionID)
		}
		deckSize := DeckSize
		if rs.IsConfront() {
			deckSize = ConfrontDeckSize
		}
		if len(p.Deck) != deckSize {
			return nil, fmt.Errorf("jogador %d: deck com %d cartas (exigido %d)", i, len(p.Deck), deckSize)
		}
		for _, def := range p.Deck {
			if rs.Cards[def] == nil {
				return nil, fmt.Errorf("jogador %d: carta desconhecida %q", i, def)
			}
		}
	}
	if cfg.FirstPlayer < -1 || cfg.FirstPlayer > 1 {
		return nil, fmt.Errorf("first_player inválido: %d", cfg.FirstPlayer)
	}
	if cfg.RulesetVersion == "" {
		cfg.RulesetVersion = rs.Version
	}

	s := &GameState{
		RulesetVersion: cfg.RulesetVersion,
		Seed:           cfg.Seed,
		Phase:          PhaseMulligan,
		Winner:         -1,
		Cards:          map[string]*CardInstance{},
		RNG:            NewRNG(cfg.Seed),
	}
	g := &Game{Cfg: cfg, rs: rs, s: s}

	if cfg.FirstPlayer >= 0 {
		s.Initiative = cfg.FirstPlayer
	} else {
		s.Initiative = s.RNG.Intn(2)
	}
	s.Active = s.Initiative

	for i := 0; i < 2; i++ {
		champ := g.rs.Champions[cfg.Players[i].ChampionID]
		vitality := champ.Vitality
		if rs.IsConfront() {
			vitality = rs.ConfrontRules.StartingVitality
		}
		p := &PlayerState{
			Champion:    champ.ID,
			Vitality:    vitality,
			MaxVitality: vitality,
		}
		for k, def := range cfg.Players[i].Deck {
			inst := &CardInstance{
				ID:    fmt.Sprintf("p%d-c%02d", i, k),
				Def:   def,
				Owner: i,
				Zone:  ZoneDeck,
			}
			s.Cards[inst.ID] = inst
			p.Deck = append(p.Deck, inst.ID)
		}
		if !cfg.SkipShuffle {
			s.RNG.Shuffle(len(p.Deck), func(a, b int) { p.Deck[a], p.Deck[b] = p.Deck[b], p.Deck[a] })
		}
		s.Players[i] = p
	}

	g.emit(Event{Kind: EvMatchStarted, P: s.Initiative, S: cfg.RulesetVersion})
	for i := 0; i < 2; i++ {
		for k := 0; k < OpeningHandSize; k++ {
			if rs.IsConfront() {
				g.drawConfront(i)
			} else {
				g.drawOne(i)
			}
		}
	}
	if rs.IsConfront() {
		// Compensação de iniciativa: quem não abre a partida começa com cartas
		// extras. Em duelos longos a penalidade de Poder do primeiro turno se
		// dilui, e só a vantagem de cartas segura o first-player perto de 50%.
		if extra := rs.ConfrontRules.SecondPlayerBonusDraw; extra > 0 {
			for k := 0; k < extra; k++ {
				g.drawConfront(1 - s.Initiative)
			}
		}
		g.startConfrontTurn(s.Initiative, true)
		return g, nil
	}
	// Passiva de Oren: vê e reordena as 3 do topo antes do mulligan.
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		ci := championImpls[s.Players[i].Champion]
		if ci != nil && ci.preMulliganScry && len(s.Players[i].Deck) >= 2 {
			n := min(3, len(s.Players[i].Deck))
			g.requestDecision(&Decision{
				Player: i, Kind: DecReorderTop,
				Options: append([]string{}, s.Players[i].Deck[:n]...), N: n,
				Source: s.Players[i].Champion,
			})
		}
	}
	return g, nil
}

// RulesetVersion corrente da engine. Acompanha o version do arquivo de
// efeitos: mudanças de conteúdo/regra exigem bump + regeneração dos goldens.
const (
	// RulesetVersion é o ruleset histórico embutido. Mantido para replay e
	// compatibilidade dos testes do motor legado.
	RulesetVersion                = "alpha-0.8.3"
	ConfrontInitialRulesetVersion = "alpha-0.9.0"
	ConfrontRulesetVersion        = "alpha-0.9.1"
	TacticalInitialRulesetVersion = "alpha-0.10.1"
	TacticalRulesetVersion        = "alpha-0.10.2"
	// LongDuelRulesetVersion é o duelo longo: Guarda com bônus simétrico ao
	// Assalto, Vitalidade maior e relógio de pressão adiado. Ver ADR-035.
	LongDuelRulesetVersion    = "alpha-0.11.0"
	CompetitiveRulesetVersion = LongDuelRulesetVersion
	ConfrontStartingVitality  = 30
	ConfrontPowerBonus        = 4
	ConfrontFirstTurnPenalty  = 2
	ConfrontPressureStartTurn = 25
	ConfrontPressureBaseLoss  = 2
)

// Parâmetros do duelo longo (ADR-035). Calibrados por varredura de simulação
// para levar a partida da faixa de 12 rodadas para a faixa de 40, mantendo a
// Guarda como jogada viável em vez de carta morta na mão.
const (
	LongDuelStartingVitality  = 56
	LongDuelGuardBonus        = 3
	LongDuelPressureStartTurn = 58
	LongDuelPressureBaseLoss  = 2
	// A compensação de iniciativa fica em zero: nesta calibragem o first-player
	// já cai em 49,35% sem ela. A alavanca continua disponível e testada para
	// futuras mudanças de ritmo.
	LongDuelSecondPlayerDraw = 0
	// Faixa de duração prometida pelo formato, verificada pelo gate: o teto
	// impede arrastar, o piso impede voltar ao duelo de 12 rodadas.
	LongDuelTargetP95Rounds    = 60
	LongDuelTargetMinP50Rounds = 30
	// Um terço do baralho responde: é a forma 10/10/10 dos precons, que é a
	// que foi calibrada. Com o mínimo antigo de 8, baralhos montados livremente
	// jogavam uma partida bem mais curta que a prometida.
	LongDuelMinGuards = 10
	// Uma Guarda comprometida nunca deixa passar mais que isto. Corrige o que
	// um bônus fixo de Prevenção não alcança: a curva de Poder vai de 6 a 10,
	// e sem teto a duração da partida passa a depender de quão agressivo é o
	// baralho do rival em vez das decisões da mesa.
	LongDuelGuardLeakCap = 1
)

// DeckSizeForVersion evita reinterpretar snapshots/decks de 36 cartas.
func DeckSizeForVersion(version string) int {
	if IsConfrontVersion(version) {
		return ConfrontDeckSize
	}
	return DeckSize
}

func IsConfrontVersion(version string) bool {
	rs, err := RulesetByVersion(version)
	return err == nil && rs.IsConfront()
}

const (
	VeilFractureStartRound = 25
	VeilFractureBaseLoss   = 1
)

// Apply valida e aplica um comando. Comando ilegal retorna erro e não muta
// nada. Retorna os eventos gerados por este comando.
func (g *Game) Apply(cmd Command) ([]Event, error) {
	s := g.s
	if s.Over {
		return nil, errCmd(ErrMatchOver, "a partida terminou")
	}
	if cmd.Player != 0 && cmd.Player != 1 {
		return nil, errCmd(ErrBadCommand, "jogador inválido: %d", cmd.Player)
	}
	if cmd.Kind == CmdKindConcede {
		reason := "concessao"
		if cmd.Reason == "timeout" {
			reason = "timeout"
		} else if cmd.Reason != "" {
			return nil, errCmd(ErrBadCommand, "motivo de concessão inválido")
		}
		start := len(g.Log)
		g.CommandLog = append(g.CommandLog, cmd)
		g.endMatch(1-cmd.Player, reason)
		return g.Log[start:], nil
	}
	if g.rs.IsConfront() {
		start := len(g.Log)
		if err := g.applyConfront(cmd); err != nil {
			if len(g.Log) != start {
				panic(fmt.Sprintf("engine: comando Confronto rejeitado após emitir eventos: %v", err))
			}
			return nil, err
		}
		g.CommandLog = append(g.CommandLog, cmd)
		return g.Log[start:], nil
	}
	if s.Pending != nil {
		if cmd.Kind != CmdKindChoose {
			return nil, errCmd(ErrPendingChoice, "há uma decisão pendente do jogador %d", s.Pending.Player)
		}
	}

	var err error
	start := len(g.Log)
	switch cmd.Kind {
	case CmdKindMulligan:
		err = g.applyMulligan(cmd)
	case CmdKindStance:
		err = g.applyStance(cmd)
	case CmdKindPlay:
		err = g.applyPlay(cmd)
	case CmdKindChoose:
		err = g.applyChoose(cmd)
	case CmdKindPass:
		err = g.applyPass(cmd)
	case CmdKindActivate:
		err = g.applyActivate(cmd)
	case CmdKindUltimate:
		err = g.applyUltimate(cmd)
	default:
		err = errCmd(ErrBadCommand, "comando desconhecido: %q", cmd.Kind)
	}
	if err != nil {
		// Comandos rejeitados não podem ter emitido eventos.
		if len(g.Log) != start {
			panic(fmt.Sprintf("engine: comando rejeitado após emitir eventos: %v", err))
		}
		return nil, err
	}
	// A cadeia deste comando pode ter mutado zonas DEPOIS de abrir a decisão
	// pendente (ex.: retaliação que compra a carta que uma reordenação de
	// topo segurava — VR-091 × VR-120, ADR-032). Opções apresentadas são
	// sempre as vivas.
	g.settleInvalidPendingDecision()
	g.CommandLog = append(g.CommandLog, cmd)
	return g.Log[start:], nil
}

// --- Mulligan ---

func (g *Game) applyMulligan(cmd Command) error {
	s := g.s
	if s.Phase != PhaseMulligan {
		return errCmd(ErrWrongPhase, "mulligan fora de fase")
	}
	p := s.Players[cmd.Player]
	if p.MulliganDone {
		return errCmd(ErrBadCommand, "mulligan já realizado")
	}
	seen := map[string]bool{}
	for _, id := range cmd.Cards {
		if !containsID(p.Hand, id) || seen[id] {
			return errCmd(ErrInvalidCard, "carta %q não está na mão", id)
		}
		seen[id] = true
	}

	n := len(cmd.Cards)
	for _, id := range cmd.Cards {
		p.Hand, _ = removeID(p.Hand, id)
		p.Deck = append(p.Deck, id)
		s.Cards[id].Zone = ZoneDeck
	}
	if n > 0 {
		s.RNG.Shuffle(len(p.Deck), func(a, b int) { p.Deck[a], p.Deck[b] = p.Deck[b], p.Deck[a] })
		g.emit(Event{Kind: EvDeckReshuffled, P: cmd.Player})
		for i := 0; i < n; i++ {
			g.drawOne(cmd.Player)
		}
	}
	p.MulliganDone = true
	g.emit(Event{Kind: EvMulliganDone, P: cmd.Player, N: n})

	if s.Players[0].MulliganDone && s.Players[1].MulliganDone {
		g.startRound()
	}
	return nil
}

// --- Postura ---

func (g *Game) applyStance(cmd Command) error {
	s := g.s
	if s.Phase != PhaseStance {
		return errCmd(ErrWrongPhase, "postura fora de fase")
	}
	switch cmd.Stance {
	case StancePredacao, StanceVigilia, StanceArcano:
	default:
		return errCmd(ErrBadCommand, "postura inválida: %q", cmd.Stance)
	}
	p := s.Players[cmd.Player]
	if p.StanceCommitted {
		return errCmd(ErrBadCommand, "postura já escolhida")
	}
	p.Stance = cmd.Stance
	p.StanceCommitted = true
	// O evento não revela a postura; a revelação é simultânea.
	g.emit(Event{Kind: EvStanceCommitted, P: cmd.Player})

	if s.Players[0].StanceCommitted && s.Players[1].StanceCommitted {
		g.emit(Event{Kind: EvStancesRevealed, P: -1, S: string(s.Players[0].Stance) + "|" + string(s.Players[1].Stance)})
		for i := 0; i < 2; i++ {
			g.applyStanceMods(i)
		}
		s.Phase = PhaseRite
		s.Active = s.Initiative
		s.Passed = [2]bool{}
		g.emit(Event{Kind: EvWindowOpened, P: s.Active, S: string(PhaseRite)})
	}
	return nil
}

func (g *Game) applyStanceMods(player int) {
	p := g.s.Players[player]
	switch p.Stance {
	case StanceVigilia:
		p.CostMods = append(p.CostMods, CostMod{
			Type: TypeGuarda, Delta: -1, Uses: 1, Round: g.s.Round, Source: "Vigília",
		})
		g.emit(Event{Kind: EvCostModAdded, P: player, N: -1, S: "Vigília"})
	case StanceArcano:
		p.CostMods = append(p.CostMods, CostMod{
			Type: TypeRito, Delta: -1, Uses: 1, Round: g.s.Round, RequireNonDamage: true, Source: "Arcano",
		})
		g.emit(Event{Kind: EvCostModAdded, P: player, N: -1, S: "Arcano"})
	}
	// Predação é bônus de dano no primeiro Assalto; tratado no combate.
}

// --- Passar janela ---

func (g *Game) applyPass(cmd Command) error {
	s := g.s
	switch {
	case s.RiteReact != nil:
		defender := 1 - s.RiteReact.Caster
		if cmd.Player != defender {
			return errCmd(ErrWrongPlayer, "a janela de reação é do jogador %d", defender)
		}
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: "reacao"})
		g.resumeSuspendedRite()
		return nil
	case s.Guard != nil:
		if cmd.Player != s.Guard.Defender {
			return errCmd(ErrWrongPlayer, "a janela de Guarda é do jogador %d", s.Guard.Defender)
		}
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: "guarda"})
		g.resolveAssault()
		return nil
	case s.Extra != nil:
		if cmd.Player != s.Extra.Player {
			return errCmd(ErrWrongPlayer, "a janela extra é do jogador %d", s.Extra.Player)
		}
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: "extra"})
		s.Extra = nil
		g.openExtraOrTwilight()
		return nil
	case s.Phase == PhaseRite, s.Phase == PhaseConfront:
		if cmd.Player != s.Active {
			return errCmd(ErrWrongPlayer, "a janela é do jogador %d", s.Active)
		}
		g.emit(Event{Kind: EvPass, P: cmd.Player, S: string(s.Phase)})
		s.Passed[cmd.Player] = true
		other := 1 - cmd.Player
		if !s.Passed[other] {
			s.Active = other
			g.emit(Event{Kind: EvWindowOpened, P: other, S: string(s.Phase)})
			return nil
		}
		// Ambos passaram: avança de fase.
		if s.Phase == PhaseRite {
			s.Phase = PhaseConfront
			s.Active = s.Initiative
			s.Passed = [2]bool{}
			g.emit(Event{Kind: EvWindowOpened, P: s.Active, S: string(PhaseConfront)})
		} else {
			g.openExtraOrTwilight()
		}
		return nil
	default:
		return errCmd(ErrWrongPhase, "nada a passar na fase %s", s.Phase)
	}
}

// --- Rodada ---

func (g *Game) startRound() {
	s := g.s
	s.Round++
	s.Phase = PhaseStance
	s.Active = s.Initiative
	s.Passed = [2]bool{}
	g.emit(Event{Kind: EvRoundStarted, P: s.Initiative, N: s.Round})

	for i := 0; i < 2; i++ {
		p := s.Players[i]
		p.resetRound()
		p.EssenceCap = min(EssenceStart+s.Round-1, EssenceMax)
		p.Essence = p.EssenceCap
		p.TempEssence = 0
		g.emit(Event{Kind: EvEssenceRecharged, P: i, N: p.Essence})
	}
	s.RoundSigils = s.RoundSigils[:0]
	s.PlayedRound = s.PlayedRound[:0]
	s.EclipseMovedRound = false

	if s.Round >= VeilFractureStartRound {
		loss := VeilFractureBaseLoss + s.Round - VeilFractureStartRound
		for _, player := range []int{s.Initiative, 1 - s.Initiative} {
			g.applyVitalityLossWithoutWinCheck(player, loss, "Ruptura do Véu", "", "")
		}
		g.checkWin()
		if s.Over {
			return
		}
	}

	// Gatilhos de início de rodada de permanentes (ordem: iniciativa primeiro,
	// Relíquias antes de Manifestações, na ordem em que entraram).
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		p := s.Players[i]
		for _, inst := range append(append([]string{}, p.Relics...), p.Manifs...) {
			g.fireRoundStart(inst)
		}
		if s.Over {
			return
		}
	}

	// Compra: iniciativa primeiro.
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		g.drawOne(i)
		if s.Over {
			return
		}
	}
}

// startTwilight resolve o Crepúsculo: Sangramento, Maldições, expirações e
// limite de mão (que pode exigir decisões de descarte).
func (g *Game) startTwilight() {
	s := g.s
	s.Phase = PhaseTwilight
	g.emit(Event{Kind: EvTwilight, P: -1})

	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		p := s.Players[i]

		var restB []TimedN
		for _, b := range p.Bleeds {
			if b.Round <= s.Round {
				g.emit(Event{Kind: EvBleedTriggered, P: i, N: b.N})
				g.loseVitality(i, b.N, "Sangramento")
			} else {
				restB = append(restB, b)
			}
		}
		p.Bleeds = restB

		var restC []TimedN
		for _, c := range p.Curses {
			if c.Round <= s.Round {
				g.emit(Event{Kind: EvCurseTriggered, P: i, N: c.N, Def: c.Kind})
				// VR-053: sofre N se tiver 5+ cartas na mão.
				if c.Kind == "VR-053" {
					if len(p.Hand) >= 5 {
						g.loseVitality(i, c.N, "Maldição")
					}
				} else {
					g.loseVitality(i, c.N, "Maldição")
				}
			} else {
				restC = append(restC, c)
			}
		}
		p.Curses = restC

		if p.VeilRound > 0 && p.VeilRound <= s.Round {
			p.VeilRound = 0
			g.emit(Event{Kind: EvStatusExpired, P: i, S: "Véu"})
		}
		if s.Over {
			return
		}
	}
	// VR-048: Assaltos marcados retornam à mão antes do limite de mão.
	g.processPendingReturns()
	// Gatilhos de Crepúsculo (VR-076), na ordem da iniciativa.
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		g.firePerms(i, func(pi *permImpl, inst *CardInstance) {
			if pi.onTwilight != nil {
				pi.onTwilight(g, inst)
			}
		})
		if s.Over {
			return
		}
	}
	g.continueTwilight()
}

// continueTwilight aplica o limite de mão; pausa em decisões pendentes.
func (g *Game) continueTwilight() {
	s := g.s
	for _, i := range []int{s.Initiative, 1 - s.Initiative} {
		p := s.Players[i]
		if len(p.Hand) > HandLimit {
			g.requestDecision(&Decision{
				Player:  i,
				Kind:    DecDiscardN,
				Options: append([]string{}, p.Hand...),
				N:       len(p.Hand) - HandLimit,
			})
			return
		}
	}
	g.finishRound()
}

func (g *Game) finishRound() {
	s := g.s
	// Eclipse: um estado total dura até o fim da rodada; depois o medidor
	// retorna a 0. Sem estado total, o medidor persiste entre rodadas.
	if s.EclipseState != EclipseNone {
		g.emit(Event{Kind: EvEclipseReset, P: -1, From: s.Eclipse, To: 0, S: string(s.EclipseState)})
		s.EclipseState = EclipseNone
		s.Eclipse = 0
	}
	for i := 0; i < 2; i++ {
		p := s.Players[i]
		p.TempEssence = 0
		// Modificadores de custo restritos à rodada expiram.
		var keep []CostMod
		for _, m := range p.CostMods {
			if m.Round == 0 || m.Round > s.Round {
				keep = append(keep, m)
			}
		}
		p.CostMods = keep
	}
	s.Initiative = 1 - s.Initiative
	g.startRound()
}

// --- Compra, fadiga, vitalidade ---

func (g *Game) drawOne(player int) *CardInstance {
	s := g.s
	p := s.Players[player]
	if len(p.Deck) == 0 {
		// alpha-0.8.0: Fadiga em passos de 6 (6, 12, 18…) — a corrida de
		// atrito era longa o bastante para passivas recorrentes dominarem o
		// meta, e a rodada de sustain do ADR-029 esticou o fim de jogo;
		// antes: passos de 2 (ADR-024). Campeões modulam o passo (Oren: -1),
		// nunca abaixo de 1.
		p.Fatigue += max(1, 6+champFatigueStepDelta(g, player))
		g.emit(Event{Kind: EvFatigue, P: player, N: p.Fatigue})
		g.loseVitality(player, p.Fatigue, "Fadiga")
		if s.Over {
			return nil
		}
		if len(p.Discard) == 0 {
			return nil
		}
		for _, id := range p.Discard {
			s.Cards[id].Zone = ZoneDeck
		}
		p.Deck = append(p.Deck, p.Discard...)
		p.Discard = p.Discard[:0]
		s.RNG.Shuffle(len(p.Deck), func(a, b int) { p.Deck[a], p.Deck[b] = p.Deck[b], p.Deck[a] })
		g.emit(Event{Kind: EvDeckReshuffled, P: player, N: len(p.Deck)})
	}
	id := p.Deck[0]
	p.Deck = p.Deck[1:]
	p.Hand = append(p.Hand, id)
	s.Cards[id].Zone = ZoneHand
	p.DrawsRound++
	g.emit(Event{Kind: EvCardDrawn, P: player, Card: id, Def: s.Cards[id].Def})
	// Forma de Oren: a primeira compra extra da rodada fica mais barata.
	if p.DrawsRound == 2 {
		champOnExtraDraw(g, player, id)
	}
	// Gatilhos do oponente reagindo à compra (VR-073).
	g.firePerms(1-player, func(pi *permImpl, inst *CardInstance) {
		if pi.onOppDraw != nil {
			pi.onOppDraw(g, inst, player)
		}
	})
	return s.Cards[id]
}

// loseVitality aplica perda direta (fadiga, sangramento, maldição). Ward só
// absorve dano de Assalto no alpha. Passa pelo funil central (escudo de
// Saela, Recusa da Morte de Kaedor).
func (g *Game) loseVitality(player, n int, cause string) {
	g.applyVitalityLoss(player, n, cause, "", "")
}

func (g *Game) sacrificeVitality(player, n int, source string) {
	p := g.s.Players[player]
	from := p.Vitality
	p.Vitality -= n
	p.SacrificesRound++
	g.emit(Event{Kind: EvSacrifice, P: player, N: n, From: from, To: p.Vitality, S: source})
	champOnSacrifice(g, player)
	g.checkWin()
}

func (g *Game) heal(player, n int, source string) {
	if n <= 0 || g.s.Over {
		return
	}
	n += champHealBonus(g, player) // forma de Seris: Dreno +1 na Noite
	p := g.s.Players[player]
	from := p.Vitality
	p.Vitality = min(p.Vitality+n, p.MaxVitality)
	if p.Vitality != from {
		g.emit(Event{Kind: EvHealed, P: player, N: p.Vitality - from, From: from, To: p.Vitality, S: source})
		g.firePerms(player, func(pi *permImpl, inst *CardInstance) {
			if pi.onHeal != nil {
				pi.onHeal(g, inst)
			}
		})
	}
}

func (g *Game) checkWin() {
	s := g.s
	if s.Over {
		return
	}
	dead0 := s.Players[0].Vitality <= 0
	dead1 := s.Players[1].Vitality <= 0
	switch {
	case dead0 && dead1:
		// Resolução sequencial determinística: quem chegou a 0 primeiro já
		// teria encerrado; empate simultâneo dá vitória a quem tem mais
		// vitalidade (menos negativa); persistindo, ao jogador sem iniciativa.
		if s.Players[0].Vitality == s.Players[1].Vitality {
			g.endMatch(1-s.Initiative, "duplo_nocaute")
		} else if s.Players[0].Vitality > s.Players[1].Vitality {
			g.endMatch(0, "duplo_nocaute")
		} else {
			g.endMatch(1, "duplo_nocaute")
		}
	case dead0:
		g.endMatch(1, "vitalidade")
	case dead1:
		g.endMatch(0, "vitalidade")
	}
}

func (g *Game) endMatch(winner int, reason string) {
	s := g.s
	if s.Over {
		return
	}
	s.Over = true
	s.Winner = winner
	s.EndReason = reason
	s.Phase = PhaseOver
	s.Pending = nil
	// s.Guard NÃO é limpo aqui: uma morte no meio de uma resolução (ex.:
	// compra da passiva de Nyra matando por Fadiga durante uma Guarda) deixa
	// a resolução em andamento finalizar as zonas com o contexto íntegro.
	g.emit(Event{Kind: EvMatchEnded, P: winner, S: reason})
}

// --- Essência e custos ---

// effectiveCost calcula o custo com modificadores aplicáveis (sem consumi-los).
func (g *Game) effectiveCost(player int, def *CardDef, inst string) int {
	p := g.s.Players[player]
	cost := def.Cost
	for _, m := range p.CostMods {
		if g.costModApplies(m, def, inst) {
			cost += m.Delta
		}
	}
	cost += champCostDelta(g, player, def)
	return max(cost, 0)
}

func (g *Game) costModApplies(m CostMod, def *CardDef, inst string) bool {
	if m.Uses <= 0 {
		return false
	}
	if m.Type != "" && m.Type != def.Type {
		return false
	}
	if m.Faction != "" && m.Faction != def.Faction {
		return false
	}
	if m.Instance != "" && m.Instance != inst {
		return false
	}
	if m.Round != 0 && m.Round != g.s.Round {
		return false
	}
	if m.UntilRound != 0 && g.s.Round > m.UntilRound {
		return false
	}
	if m.RequireNonDamage && g.rs.cardDealsDamage(def.ID) {
		return false
	}
	return true
}

func (g *Game) consumeCostMods(player int, def *CardDef, inst string) {
	p := g.s.Players[player]
	for i := range p.CostMods {
		if g.costModApplies(p.CostMods[i], def, inst) {
			p.CostMods[i].Uses--
		}
	}
}

// spendEssence debita temporária primeiro (ela expira no fim da rodada).
func (g *Game) spendEssence(player, n int) {
	if n == 0 {
		return
	}
	p := g.s.Players[player]
	fromTemp := min(p.TempEssence, n)
	p.TempEssence -= fromTemp
	p.Essence -= n - fromTemp
	g.emit(Event{Kind: EvEssenceSpent, P: player, N: n, To: p.Essence + p.TempEssence})
}

func (g *Game) canAfford(player, cost int) bool {
	p := g.s.Players[player]
	return p.Essence+p.TempEssence >= cost
}

func (g *Game) gainTempEssence(player, n int, source string) {
	p := g.s.Players[player]
	p.TempEssence += n
	g.emit(Event{Kind: EvTempEssence, P: player, N: n, S: source})
}

// --- Eclipse ---

// shiftEclipse move o medidor n passos (positivo = Noite). by identifica o
// jogador cuja carta/habilidade causou o deslocamento (-1 = sistema); a
// ultimate de Ilyan pode anular deslocamentos adversários, e sua passiva pune
// o segundo empurrão à Noite numa rodada.
func (g *Game) shiftEclipse(n int, cause string, by int) {
	s := g.s
	if n == 0 || s.Over {
		return
	}
	if by >= 0 {
		opp := s.Players[1-by]
		if opp.NullifyOppShift {
			opp.NullifyOppShift = false
			g.emit(Event{Kind: EvShiftNullified, P: 1 - by, N: n, S: cause})
			return
		}
	}
	from := s.Eclipse
	to := max(min(from+n, EclipseMax), EclipseMin)
	if to == from {
		return
	}
	s.Eclipse = to
	s.EclipseMovedRound = true
	g.emit(Event{Kind: EvEclipseShifted, P: by, N: n, From: from, To: to, S: cause})
	if by >= 0 && n > 0 {
		s.Players[by].NightShiftsRound++
		if s.Players[by].NightShiftsRound == 2 {
			champOnOppNightShift(g, 1-by)
		}
	}
	if s.EclipseState == EclipseNone {
		if to == EclipseMax {
			s.EclipseState = EclipseNight
			g.emit(Event{Kind: EvEclipseTriggered, P: -1, S: string(EclipseNight)})
		} else if to == EclipseMin {
			s.EclipseState = EclipseAurora
			g.emit(Event{Kind: EvEclipseTriggered, P: -1, S: string(EclipseAurora)})
		}
	}
}

// --- Ressonância ---

func (g *Game) resonanceCap(player int) int {
	cap := ResonanceCap
	if d := champResonanceCapDelta(g, player); d != 0 {
		cap += d
	}
	return cap
}

// emitSigil adiciona um Sigilo à trilha do jogador e à linha do tempo global
// da rodada. Se a trilha atingiu o limite, nada é emitido.
func (g *Game) emitSigil(player int, sig Sigil, source string, depth int) {
	if depth > maxChainDepth {
		g.emit(Event{Kind: EvChainTruncated, P: player, S: source})
		return
	}
	p := g.s.Players[player]
	if len(p.Trail) >= g.resonanceCap(player) {
		return
	}
	p.Trail = append(p.Trail, sig)
	g.s.RoundSigils = append(g.s.RoundSigils, RoundSigil{P: player, S: sig})
	g.emit(Event{Kind: EvSigilAdded, P: player, S: string(sig), Card: source, N: len(p.Trail)})
	// Passiva de Nyra: scry ao completar 3 Sigilos na rodada.
	if len(p.Trail) == 3 {
		champOn3Sigils(g, player)
	}
	g.fireSigilEmitted(player, sig, depth)
}

// trailEndsWith verifica se a trilha própria termina exatamente na sequência.
func trailEndsWith(trail []Sigil, seq ...Sigil) bool {
	if len(trail) < len(seq) {
		return false
	}
	off := len(trail) - len(seq)
	for i, s := range seq {
		if trail[off+i] != s {
			return false
		}
	}
	return true
}

func distinctSigils(trail []Sigil) int {
	set := map[Sigil]bool{}
	for _, s := range trail {
		set[s] = true
	}
	return len(set)
}

// --- Decisões ---

func (g *Game) requestDecision(d *Decision) {
	g.s.NextDecID++
	d.ID = g.s.NextDecID
	if g.s.Pending != nil {
		g.s.DecQueue = append(g.s.DecQueue, d)
		return
	}
	g.s.Pending = d
	g.emit(Event{Kind: EvDecisionRequested, P: d.Player, N: d.N, S: string(d.Kind), Card: d.Card})
}

// refreshQueuedDecision atualiza opções que dependem de zonas mutáveis. Uma
// decisão enfileirada ainda não foi apresentada ao jogador; quando chega sua
// vez, precisa refletir o estado produzido pelas escolhas anteriores.
func (g *Game) refreshQueuedDecision(d *Decision) bool {
	s := g.s
	filterZone := func(options []string, player int, zone Zone) []string {
		result := make([]string, 0, len(options))
		for _, id := range options {
			if card := s.Cards[id]; card != nil && card.Owner == player && card.Zone == zone {
				result = append(result, id)
			}
		}
		return result
	}
	switch d.Kind {
	case DecDiscardN:
		d.Options = append(d.Options[:0], s.Players[d.Player].Hand...)
	case DecOppDiscardPick:
		d.Options = filterZone(d.Options, 1-d.Player, ZoneHand)
	case DecPickTop2:
		deck := s.Players[d.Player].Deck
		d.Options = append(d.Options[:0], deck[:min(2, len(deck))]...)
	case DecReorderTop:
		deck := s.Players[d.Player].Deck
		d.Options = append(d.Options[:0], deck[:min(d.N, len(deck))]...)
		d.N = len(d.Options) // o topo pode ter encolhido no meio da cadeia
	case DecRecoverPick, DecExilePick, DecEddaReturn:
		d.Options = filterZone(d.Options, d.Player, ZoneDiscard)
	case DecDestroyRelicPick, DecMirrorRelic:
		d.Options = filterZone(d.Options, 1-d.Player, ZonePlay)
	case DecTaxTypePick:
		d.Options = filterZone(d.Options, 1-d.Player, ZoneDiscard)
	case DecRevealTax, DecLockAssault:
		d.Options = filterZone(d.Options, 1-d.Player, ZoneHand)
	case DecMillTop:
		if len(s.Players[d.Player].Deck) == 0 {
			return false
		}
	case DecScryBottom:
		deck := s.Players[d.Player].Deck
		if len(deck) == 0 {
			return false
		}
		d.Card = deck[0]
	case DecExileSelfHeal:
		if card := s.Cards[d.Card]; card == nil || card.Zone != ZoneDiscard {
			return false
		}
	}
	if len(d.Options) == 0 {
		return false
	}
	if d.N > len(d.Options) {
		d.N = len(d.Options)
	}
	if d.MinN > d.N {
		d.MinN = d.N
	}
	return d.N > 0
}

// settleInvalidPendingDecision encerra uma decisão que já foi apresentada,
// mas perdeu todas as opções durante a MESMA cadeia que a abriu. Exemplo:
// uma Guarda compra e pede descarte; um gatilho posterior ativa a Recusa da
// Morte de Kaedor e esvazia a mão. Manter a decisão aberta criaria um comando
// impossível (escolher 1 entre zero opções) e travaria a partida.
func (g *Game) settleInvalidPendingDecision() {
	for !g.s.Over && g.s.Pending != nil && !g.refreshQueuedDecision(g.s.Pending) {
		stale := g.s.Pending
		g.s.Pending = nil
		g.emit(Event{Kind: EvDecisionResolved, P: stale.Player, N: stale.ID, S: string(stale.Kind)})
		g.runDecisionThen(stale, "")
		g.promoteQueuedDecision()
		g.finalizePendingRite()
	}
}

func (g *Game) runDecisionThen(d *Decision, picked string) {
	if len(d.Then) == 0 || g.s.Over {
		return
	}
	caster := d.Player
	if d.Kind == DecOppDiscardPick {
		caster = 1 - d.Player
	}
	g.runOps(d.Then, &opCtx{player: caster, source: d.Source, inst: d.Card, picked: picked})
}

// promoteQueuedDecision ativa a próxima decisão ainda aplicável. Opções que
// ficaram vazias são puladas, mas sua continuação declarativa é preservada.
func (g *Game) promoteQueuedDecision() {
	s := g.s
	for !s.Over && s.Pending == nil && len(s.DecQueue) > 0 {
		next := s.DecQueue[0]
		s.DecQueue = s.DecQueue[1:]
		if !g.refreshQueuedDecision(next) {
			g.runDecisionThen(next, "")
			continue
		}
		s.Pending = next
		g.emit(Event{Kind: EvDecisionRequested, P: next.Player, N: next.N, S: string(next.Kind), Card: next.Card})
	}
}

func (g *Game) applyChoose(cmd Command) error {
	s := g.s
	d := s.Pending
	if d == nil {
		return errCmd(ErrBadCommand, "não há decisão pendente")
	}
	if cmd.Player != d.Player {
		return errCmd(ErrWrongPlayer, "a decisão pendente é do jogador %d", d.Player)
	}
	if cmd.DecisionID != d.ID {
		return errCmd(ErrBadCommand, "decisão %d não corresponde à pendente %d", cmd.DecisionID, d.ID)
	}

	picks := cmd.Cards
	switch d.Kind {
	case DecDiscardN, DecRecoverPick, DecExilePick, DecReEmitSigils:
		if d.MinN > 0 {
			if len(picks) < d.MinN || len(picks) > d.N {
				return errCmd(ErrBadCommand, "escolha entre %d e %d", d.MinN, d.N)
			}
		} else if len(picks) != d.N {
			return errCmd(ErrBadCommand, "escolha exatamente %d carta(s)", d.N)
		}
		if err := picksFromOptions(picks, d.Options); err != nil {
			return err
		}
	case DecReorderTop:
		if len(picks) != d.N {
			return errCmd(ErrBadCommand, "forneça a permutação completa de %d cartas", d.N)
		}
		if err := picksFromOptions(picks, d.Options); err != nil {
			return err
		}
	case DecPickTop2, DecOppDiscardPick, DecDestroyRelicPick, DecTaxTypePick, DecPickSigil,
		DecRevealTax, DecLockAssault, DecDeclareType, DecDirection, DecFormulaChoice,
		DecMirrorRelic, DecCopyPlayed, DecOrenChoice, DecVorenChoice, DecEddaReturn:
		if len(picks) != 1 || !containsID(d.Options, picks[0]) {
			return errCmd(ErrBadCommand, "escolha exatamente 1 das opções")
		}
	case DecExileSelfHeal, DecMillTop, DecSwapEclipse, DecScryBottom:
		if len(picks) != 1 || (picks[0] != "yes" && picks[0] != "no") {
			return errCmd(ErrBadCommand, `responda "yes" ou "no"`)
		}
	default:
		return errCmd(ErrBadCommand, "decisão desconhecida: %q", d.Kind)
	}

	s.Pending = nil
	g.emit(Event{Kind: EvDecisionResolved, P: d.Player, N: d.ID, S: string(d.Kind)})
	picked := g.resolveDecisionPicks(d, picks)

	// Continuação da DSL: roda no contexto do dono da carta fonte.
	g.runDecisionThen(d, picked)

	// Ativa a próxima decisão enfileirada, se houver.
	g.promoteQueuedDecision()
	g.finalizePendingRite()
	if s.Over {
		return nil
	}
	if s.Phase == PhaseTwilight && s.Pending == nil {
		g.continueTwilight()
	}
	return nil
}

// picksFromOptions valida que as escolhas são distintas e pertencem às opções.
func picksFromOptions(picks, options []string) error {
	seen := map[string]bool{}
	for _, id := range picks {
		if !containsID(options, id) || seen[id] {
			return errCmd(ErrInvalidCard, "opção inválida: %q", id)
		}
		seen[id] = true
	}
	return nil
}

// --- Movimentação de cartas ---

func (g *Game) discardFromHand(player int, id string) {
	p := g.s.Players[player]
	var ok bool
	p.Hand, ok = removeID(p.Hand, id)
	if !ok {
		panic("engine: descarte de carta fora da mão: " + id)
	}
	p.Discard = append(p.Discard, id)
	g.s.Cards[id].Zone = ZoneDiscard
	g.emit(Event{Kind: EvCardDiscarded, P: player, Card: id, Def: g.s.Cards[id].Def})
}

func (g *Game) exileFromDiscard(player int, id string) {
	p := g.s.Players[player]
	firstExile := !p.ExiledRound
	var ok bool
	p.Discard, ok = removeID(p.Discard, id)
	if !ok {
		return
	}
	p.Exile = append(p.Exile, id)
	p.ExiledRound = true
	g.s.Cards[id].Zone = ZoneExile
	g.emit(Event{Kind: EvCardExiled, P: player, Card: id, Def: g.s.Cards[id].Def})
	g.firePerms(player, func(pi *permImpl, inst *CardInstance) {
		if pi.onExile != nil {
			pi.onExile(g, inst)
		}
	})
	if firstExile {
		champOnExileSelf(g, player)
	}
}

// --- Snapshot determinístico ---

// SnapshotJSON serializa o estado de forma estável (cartas ordenadas por id).
func (g *Game) SnapshotJSON() ([]byte, error) {
	type snap struct {
		State *GameState      `json:"state"`
		Cards []*CardInstance `json:"cards_sorted"`
	}
	ids := make([]string, 0, len(g.s.Cards))
	for id := range g.s.Cards {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	cards := make([]*CardInstance, len(ids))
	for i, id := range ids {
		cards[i] = g.s.Cards[id]
	}
	// Zera o mapa na cópia para não serializar ordem instável.
	clone := *g.s
	clone.Cards = nil
	return json.MarshalIndent(snap{State: &clone, Cards: cards}, "", " ")
}

// RestoreSnapshot restaura uma partida para continuação no battle server.
// Eventos e comandos devem representar exatamente o prefixo persistido junto
// ao snapshot; o catch-up posterior continua passando por Apply.
func RestoreSnapshot(cfg Config, snapshot []byte, events []Event, commands []Command) (*Game, error) {
	rs, err := RulesetByVersion(cfg.RulesetVersion)
	if err != nil {
		return nil, err
	}
	type snap struct {
		State *GameState      `json:"state"`
		Cards []*CardInstance `json:"cards_sorted"`
	}
	var decoded snap
	if err := json.Unmarshal(snapshot, &decoded); err != nil {
		return nil, fmt.Errorf("snapshot inválido: %w", err)
	}
	if decoded.State == nil || decoded.State.RNG == nil || decoded.State.Players[0] == nil || decoded.State.Players[1] == nil {
		return nil, fmt.Errorf("snapshot incompleto")
	}
	if decoded.State.RulesetVersion != cfg.RulesetVersion || decoded.State.Seed != cfg.Seed {
		return nil, fmt.Errorf("snapshot não corresponde à configuração")
	}
	for i, event := range events {
		if event.Seq != i {
			return nil, fmt.Errorf("prefixo de eventos não contíguo em %d (seq=%d)", i, event.Seq)
		}
	}
	cards := make(map[string]*CardInstance, len(decoded.Cards))
	for _, card := range decoded.Cards {
		if card == nil || card.ID == "" || cards[card.ID] != nil || card.Owner < 0 || card.Owner > 1 {
			return nil, fmt.Errorf("snapshot contém instância de carta inválida")
		}
		cards[card.ID] = card
	}
	expectedCards := DeckSizeForVersion(cfg.RulesetVersion) * 2
	if len(cards) != expectedCards {
		return nil, fmt.Errorf("snapshot contém %d cartas; esperado %d", len(cards), expectedCards)
	}
	decoded.State.Cards = cards
	return &Game{
		Cfg:        cfg,
		rs:         rs,
		s:          decoded.State,
		Log:        append([]Event{}, events...),
		CommandLog: append([]Command{}, commands...),
	}, nil
}
