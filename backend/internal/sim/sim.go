package sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"

	"veurubro/backend/internal/engine"
)

type BotKind string

const (
	BotRandom    BotKind = "random"
	BotHeuristic BotKind = "heuristic"
)

type Config struct {
	Games        int     `json:"games"`
	BaseSeed     uint64  `json:"base_seed"`
	Workers      int     `json:"workers"`
	MaxCommands  int     `json:"max_commands"`
	Bot0         BotKind `json:"bot_0"`
	Bot1         BotKind `json:"bot_1"`
	VerifyReplay bool    `json:"verify_replay"`
	// RulesetVersion escolhe a versão registrada na engine (vazio = embutida).
	RulesetVersion string `json:"ruleset_version,omitempty"`
	// DeckMode: "precon" (padrão; a régua do gate de balanceamento) ou
	// "varied" — decks legais sorteados por partida sobre todo o pool do modo,
	// respeitando composição e cópias; é o gate de combinações fora do inicial.
	DeckMode DeckMode `json:"deck_mode,omitempty"`
}

type DeckMode string

const (
	DeckPrecon DeckMode = "precon"
	DeckVaried DeckMode = "varied"
)

// variedDeck sorteia um deck LEGAL determinístico para o Campeão: núcleo
// (facção própria + neutras) garante MinCoreFaction; o resto mistura o
// núcleo com uma facção aliada sorteada. Limites de cópia/lendária valem; o
// ValidateDeck confere o resultado — um deck ilegal aqui é bug, e o sim
// contaria como rejeição.
func variedDeck(rs *engine.Ruleset, championID string, rng *engine.RNG) []string {
	if rs.IsConfront() {
		byType := map[engine.CardType][]string{}
		for _, card := range rs.CardList {
			if card.Confront == nil || !card.Confront.Legal {
				continue
			}
			copies := engine.MaxCopies
			if card.Rarity == engine.RarityLendaria {
				copies = engine.MaxLegendary
			}
			for range copies {
				byType[card.Type] = append(byType[card.Type], card.ID)
			}
		}
		minimums := map[engine.CardType]int{
			engine.TypeAssalto: rs.ConfrontRules.MinAssaults,
			engine.TypeGuarda:  rs.ConfrontRules.MinGuards,
			engine.TypeRito:    rs.ConfrontRules.MinRites,
		}
		deck := make([]string, 0, engine.ConfrontDeckSize)
		var rest []string
		for _, cardType := range []engine.CardType{engine.TypeAssalto, engine.TypeGuarda, engine.TypeRito} {
			pool := byType[cardType]
			rng.Shuffle(len(pool), func(a, b int) { pool[a], pool[b] = pool[b], pool[a] })
			minimum := minimums[cardType]
			deck = append(deck, pool[:minimum]...)
			rest = append(rest, pool[minimum:]...)
		}
		rng.Shuffle(len(rest), func(a, b int) { rest[a], rest[b] = rest[b], rest[a] })
		return append(deck, rest[:engine.ConfrontDeckSize-len(deck)]...)
	}
	champ := rs.Champions[championID]
	var factions []string
	for _, other := range []string{"Casa Vhal", "Ordem Solara", "Conclave Mirr", "Matilha Varka", "Sínodo Cinéreo"} {
		if other != champ.Faction {
			factions = append(factions, other)
		}
	}
	ally := factions[rng.Intn(len(factions))]

	expand := func(match func(*engine.CardDef) bool) []string {
		var pool []string
		for _, card := range rs.CardList {
			if !match(card) {
				continue
			}
			copies := engine.MaxCopies
			if card.Rarity == engine.RarityLendaria {
				copies = engine.MaxLegendary
			}
			for range copies {
				pool = append(pool, card.ID)
			}
		}
		rng.Shuffle(len(pool), func(a, b int) { pool[a], pool[b] = pool[b], pool[a] })
		return pool
	}
	core := expand(func(c *engine.CardDef) bool {
		return c.Faction == champ.Faction || c.Faction == engine.NeutralFaction
	})
	allied := expand(func(c *engine.CardDef) bool { return c.Faction == ally })

	deck := append([]string{}, core[:engine.MinCoreFaction]...)
	rest := append(append([]string{}, core[engine.MinCoreFaction:]...), allied...)
	rng.Shuffle(len(rest), func(a, b int) { rest[a], rest[b] = rest[b], rest[a] })
	alliedUsed := 0
	for _, id := range rest {
		if len(deck) == engine.DeckSize {
			break
		}
		if rs.Cards[id].Faction == ally {
			if alliedUsed == engine.MaxAlliedCards {
				continue
			}
			alliedUsed++
		}
		deck = append(deck, id)
	}
	return deck
}

type Report struct {
	SchemaVersion string            `json:"schema_version"`
	Ruleset       string            `json:"ruleset_version"`
	Config        Config            `json:"config"`
	Health        HealthMetric      `json:"health"`
	Duration      DurationMetric    `json:"duration"`
	EndReasons    []EndReasonMetric `json:"end_reasons"`
	FirstPlayer   FirstPlayerMetric `json:"first_player"`
	Champions     []ChampionMetric  `json:"champions"`
	Matchups      []MatchupMetric   `json:"matchups"`
	Cards         []CardMetric      `json:"cards"`
	DominantCards []CardMetric      `json:"dominant_cards"`
	Failures      []FailureMetric   `json:"failures,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	Gate          BalanceGate       `json:"balance_gate"`
}

type BalanceGate struct {
	Enabled bool        `json:"enabled"`
	Passed  bool        `json:"passed"`
	Checks  []GateCheck `json:"checks,omitempty"`
}

type GateCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type EndReasonMetric struct {
	Reason        string  `json:"reason"`
	Games         int     `json:"games"`
	Share         float64 `json:"share"`
	AverageRounds float64 `json:"average_rounds"`
	P95Rounds     int     `json:"p95_rounds"`
}

type FailureMetric struct {
	Game   int    `json:"game"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

type HealthMetric struct {
	Requested              int `json:"requested"`
	Completed              int `json:"completed"`
	Crashes                int `json:"crashes"`
	Loops                  int `json:"loops"`
	DeadStates             int `json:"dead_states"`
	InvalidStates          int `json:"invalid_states"`
	RejectedCommands       int `json:"rejected_commands"`
	DeterminismDivergences int `json:"determinism_divergences"`
}

type DurationMetric struct {
	AverageRounds   float64 `json:"average_rounds"`
	P50Rounds       int     `json:"p50_rounds"`
	P95Rounds       int     `json:"p95_rounds"`
	MaxRounds       int     `json:"max_rounds"`
	AverageCommands float64 `json:"average_commands"`
	P50Commands     int     `json:"p50_commands"`
	P95Commands     int     `json:"p95_commands"`
	MaxCommands     int     `json:"max_commands"`
}

type FirstPlayerMetric struct {
	Games   int     `json:"games"`
	Wins    int     `json:"wins"`
	WinRate float64 `json:"win_rate"`
}

type ChampionMetric struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Games   int     `json:"games"`
	Wins    int     `json:"wins"`
	Losses  int     `json:"losses"`
	WinRate float64 `json:"win_rate"`
}

type MatchupMetric struct {
	Champion0     string  `json:"champion_0"`
	Champion1     string  `json:"champion_1"`
	Games         int     `json:"games"`
	Wins0         int     `json:"wins_0"`
	WinRate0      float64 `json:"win_rate_0"`
	AverageRounds float64 `json:"average_rounds"`
	P95Rounds     int     `json:"p95_rounds"`
}

type CardMetric struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	InclusionGames      int     `json:"inclusion_games"`
	InclusionRate       float64 `json:"inclusion_rate"`
	Draws               int     `json:"draws"`
	DrawnGames          int     `json:"drawn_games"`
	DrawnWins           int     `json:"drawn_wins"`
	DrawnWinRate        float64 `json:"drawn_win_rate"`
	Plays               int     `json:"plays"`
	PlayedGames         int     `json:"played_games"`
	PlayedWins          int     `json:"played_wins"`
	PlayedWinRate       float64 `json:"played_win_rate"`
	MulliganSeen        int     `json:"mulligan_seen"`
	MulliganKept        int     `json:"mulligan_kept"`
	MulliganKeepRate    float64 `json:"mulligan_keep_rate"`
	Damage              int     `json:"damage"`
	Prevented           int     `json:"prevented"`
	EclipseDisplacement int     `json:"eclipse_displacement"`
	AverageRoundPlayed  float64 `json:"average_round_played"`
	FinalHandCopies     int     `json:"final_hand_copies"`
	DeadInHandRate      float64 `json:"dead_in_hand_rate"`
	DominanceAlert      bool    `json:"dominance_alert"`
}

type rawChampion struct{ Games, Wins int }
type rawMatchup struct {
	Games, Wins0 int
	Rounds       []int
}
type rawCard struct {
	InclusionGames, Draws, DrawnGames, DrawnWins, Plays, PlayedGames, PlayedWins int
	MulliganSeen, MulliganKept, Damage, Prevented, EclipseDisplacement           int
	RoundPlayed, FinalHandCopies                                                 int
}

type aggregate struct {
	health     HealthMetric
	firstGames int
	firstWins  int
	rounds     []int
	commands   []int
	champions  map[string]*rawChampion
	matchups   map[string]*rawMatchup
	cards      map[string]*rawCard
	endReasons map[string][]int
	failures   []FailureMetric
}

type matchCard struct {
	Inclusion, Draws, Plays, MulliganSeen, Mulliganed int
	Damage, Prevented, EclipseDisplacement            int
	RoundPlayed, FinalHandCopies                      int
	DrawnGame, PlayedGame                             bool
}

type matchResult struct {
	champions  [2]string
	first      int
	winner     int
	endReason  string
	rounds     int
	commands   int
	cards      [2]map[string]*matchCard
	crash      bool
	loop       bool
	dead       bool
	invalid    bool
	rejected   bool
	divergence bool
	index      int
	failure    string
}

func DefaultConfig() Config {
	return Config{Games: 100_000, BaseSeed: 1, Workers: runtime.GOMAXPROCS(0), MaxCommands: 2_000,
		Bot0: BotHeuristic, Bot1: BotHeuristic, VerifyReplay: true}
}

func Run(cfg Config) (Report, error) {
	if err := validateConfig(cfg); err != nil {
		return Report{}, err
	}
	if cfg.RulesetVersion == "" {
		cfg.RulesetVersion = engine.CompetitiveRulesetVersion
	}
	rs, err := engine.RulesetByVersion(cfg.RulesetVersion)
	if err != nil {
		return Report{}, err
	}
	champions := championIDs(rs)
	decks := map[string][]string{}
	for _, id := range champions {
		deck, err := PreconstructedDeckFor(rs, id)
		if err != nil {
			return Report{}, err
		}
		decks[id] = deck
	}
	workers := min(cfg.Workers, cfg.Games)
	jobs := make(chan int)
	partials := make([]*aggregate, workers)
	var wg sync.WaitGroup
	for worker := range workers {
		partials[worker] = newAggregate()
		wg.Add(1)
		go func(out *aggregate) {
			defer wg.Done()
			for index := range jobs {
				out.add(simulateOne(cfg, index, champions, decks))
			}
		}(partials[worker])
	}
	for index := 0; index < cfg.Games; index++ {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	total := newAggregate()
	for _, partial := range partials {
		total.merge(partial)
	}
	report := total.report(cfg)
	if report.Health.Crashes+report.Health.Loops+report.Health.DeadStates+report.Health.InvalidStates+report.Health.RejectedCommands+
		report.Health.DeterminismDivergences > 0 {
		return report, errors.New("simulação encontrou falha de saúde/determinismo")
	}
	if report.Gate.Enabled && !report.Gate.Passed {
		return report, errors.New("simulação reprovou o gate competitivo")
	}
	return report, nil
}

func validateConfig(cfg Config) error {
	switch {
	case cfg.Games < 1:
		return errors.New("games deve ser positivo")
	case cfg.Workers < 1:
		return errors.New("workers deve ser positivo")
	case cfg.MaxCommands < 1:
		return errors.New("max_commands deve ser positivo")
	case cfg.Bot0 != BotRandom && cfg.Bot0 != BotHeuristic:
		return fmt.Errorf("bot_0 desconhecido: %q", cfg.Bot0)
	case cfg.Bot1 != BotRandom && cfg.Bot1 != BotHeuristic:
		return fmt.Errorf("bot_1 desconhecido: %q", cfg.Bot1)
	}
	return nil
}

func championIDs(rs *engine.Ruleset) []string {
	ids := make([]string, 0, len(rs.Champions))
	for id := range rs.Champions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// PreconstructedDeck monta o precon sob o ruleset embutido.
func PreconstructedDeck(championID string) ([]string, error) {
	return engine.Builtin().PreconstructedDeck(championID)
}

// PreconstructedDeckFor monta o precon determinístico sob o ruleset dado.
func PreconstructedDeckFor(rs *engine.Ruleset, championID string) ([]string, error) {
	return rs.PreconstructedDeck(championID)
}

func simulateOne(cfg Config, index int, champions []string, decks map[string][]string) (result matchResult) {
	// declarado antes do defer/recover para aparecer no relatório de crash
	step := -1
	command := engine.Command{}
	pending := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			result.crash = true
			result.failure = fmt.Sprintf("passo %d comando %+v%s: %v", step, command, pending, recovered)
		}
	}()
	result.index = index
	pair := index % (len(champions) * len(champions))
	repetition := index / (len(champions) * len(champions))
	result.champions = [2]string{champions[pair/len(champions)], champions[pair%len(champions)]}
	result.first = repetition % 2
	seed := cfg.BaseSeed + uint64(index)*0x9e3779b97f4a7c15
	deck0, deck1 := decks[result.champions[0]], decks[result.champions[1]]
	if cfg.DeckMode == DeckVaried {
		rs, rsErr := engine.RulesetByVersion(cfg.RulesetVersion)
		if rsErr == nil {
			deck0 = variedDeck(rs, result.champions[0], engine.NewRNG(seed^0xDEC0DEC0DEC0DEC0))
			deck1 = variedDeck(rs, result.champions[1], engine.NewRNG(seed^0x0DECADE0DECADE0D))
		}
	}
	gameCfg := engine.Config{RulesetVersion: cfg.RulesetVersion, Seed: seed, FirstPlayer: result.first,
		Players: [2]engine.PlayerSetup{{ChampionID: result.champions[0], Deck: deck0},
			{ChampionID: result.champions[1], Deck: deck1}}}
	game, err := engine.NewGame(gameCfg)
	if err != nil {
		result.rejected = true
		result.failure = err.Error()
		return result
	}
	result.cards[0], result.cards[1] = map[string]*matchCard{}, map[string]*matchCard{}
	for player := 0; player < 2; player++ {
		seen := map[string]bool{}
		for _, def := range gameCfg.Players[player].Deck {
			if !seen[def] {
				cardResult(result.cards[player], def).Inclusion = 1
				seen[def] = true
			}
		}
		for _, inst := range game.State().Players[player].Hand {
			def := game.State().Cards[inst].Def
			cardResult(result.cards[player], def).MulliganSeen++
		}
	}
	bots := [2]engine.PlayerBot{newBot(cfg.Bot0, seed^0xa5a5a5a5), newBot(cfg.Bot1, seed^0x5a5a5a5a)}
	for step = 0; step < cfg.MaxCommands; step++ {
		if game.State().Over {
			break
		}
		actor, ok := engine.RequiredPlayer(game)
		if !ok {
			result.dead = true
			result.failure = "RequiredPlayer não encontrou ator"
			break
		}
		command, ok = bots[actor].NextFor(game, actor)
		if !ok {
			result.dead = true
			result.failure = fmt.Sprintf("%s não gerou comando para p%d", botFor(cfg, actor), actor)
			break
		}
		if command.Kind == engine.CmdKindMulligan {
			for _, inst := range command.Cards {
				if card := game.State().Cards[inst]; card != nil {
					cardResult(result.cards[actor], card.Def).Mulliganed++
				}
			}
		}
		pending = ""
		if d := game.State().Pending; d != nil {
			optionZones := make([]string, 0, len(d.Options))
			for _, id := range d.Options {
				if card := game.State().Cards[id]; card != nil {
					optionZones = append(optionZones, id+":"+string(card.Zone))
				}
			}
			victim := d.Player
			if d.Kind == engine.DecOppDiscardPick {
				victim = 1 - d.Player
			}
			pending = fmt.Sprintf(" decisão=%s id=%d source=%s card=%s options=%v zonas=%v mão_alvo=%v", d.Kind, d.ID,
				d.Source, d.Card, d.Options, optionZones, game.State().Players[victim].Hand)
		}
		if _, err := game.Apply(command); err != nil {
			result.rejected = true
			result.failure = fmt.Sprintf("passo %d fase=%s ator=%d comando %+v: %v",
				step, game.State().Phase, actor, command, err)
			break
		}
		result.commands++
	}
	if !game.State().Over && !result.dead && !result.invalid && !result.rejected {
		result.loop = true
		result.failure = fmt.Sprintf("teto de %d comandos na rodada %d", cfg.MaxCommands, game.State().Round)
	}
	result.rounds = game.State().Round
	if zoneErr := validateZones(game.State()); zoneErr != nil && !result.invalid {
		result.invalid = true
		result.failure = zoneErr.Error()
	}
	if game.State().Over {
		result.winner = game.State().Winner
		result.endReason = terminalCause(game.Log, game.State().EndReason, game.Ruleset())
		analyzeCards(game, &result)
		if cfg.VerifyReplay {
			replayed, replayErr := engine.Replay(gameCfg, game.CommandLog)
			if replayErr != nil {
				result.divergence = true
				result.failure = replayErr.Error()
			} else {
				leftState, _ := game.SnapshotJSON()
				rightState, _ := replayed.SnapshotJSON()
				leftLog, _ := json.Marshal(game.Log)
				rightLog, _ := json.Marshal(replayed.Log)
				result.divergence = !bytes.Equal(leftState, rightState) || !bytes.Equal(leftLog, rightLog)
				if result.divergence {
					result.failure = "snapshot ou log divergente no replay"
				}
			}
		}
	}
	return result
}

func terminalCause(log []engine.Event, fallback string, rs *engine.Ruleset) string {
	for i := len(log) - 1; i >= 0; i-- {
		event := log[i]
		if event.Kind == engine.EvSacrifice && event.To <= 0 {
			return "sacrificio"
		}
		if event.Kind != engine.EvDamage || event.To > 0 {
			continue
		}
		switch event.S {
		case "Fadiga":
			return "fadiga"
		case "Sangramento":
			return "sangramento"
		case "Maldição":
			return "maldicao"
		case "Ruptura do Véu":
			return "ruptura_do_veu"
		case "Pressão de Nythara":
			return "pressao_de_nythara"
		}
		if rs != nil {
			if card := rs.Cards[event.Def]; card != nil {
				switch card.Type {
				case engine.TypeAssalto:
					return "assalto"
				case engine.TypeRito:
					return "rito"
				case engine.TypeGuarda:
					return "guarda"
				}
			}
			if card := rs.Cards[event.S]; card != nil {
				switch card.Type {
				case engine.TypeAssalto:
					return "assalto"
				case engine.TypeRito:
					return "rito"
				case engine.TypeGuarda:
					return "guarda"
				}
			}
		}
		if event.Def != "" {
			return "assalto"
		}
		if strings.HasPrefix(event.S, "VR-") {
			return "efeito_de_carta"
		}
		if strings.HasPrefix(event.S, "CH-") {
			return "habilidade_de_campeao"
		}
		return "vitalidade"
	}
	if fallback == "" {
		return "desconhecido"
	}
	return fallback
}

func botFor(cfg Config, player int) BotKind {
	if player == 0 {
		return cfg.Bot0
	}
	return cfg.Bot1
}

func newBot(kind BotKind, seed uint64) engine.PlayerBot {
	if kind == BotRandom {
		return &engine.RandomBot{RNG: engine.NewRNG(seed)}
	}
	return &engine.HeuristicBot{RNG: engine.NewRNG(seed)}
}

func cardResult(cards map[string]*matchCard, id string) *matchCard {
	if cards[id] == nil {
		cards[id] = &matchCard{}
	}
	return cards[id]
}

func analyzeCards(game *engine.Game, result *matchResult) {
	for _, event := range game.Log {
		def := event.Def
		owner := event.P
		if event.Card != "" {
			if inst := game.State().Cards[event.Card]; inst != nil {
				owner = inst.Owner
				if def == "" {
					def = inst.Def
				}
			}
		}
		switch event.Kind {
		case engine.EvCardDrawn:
			if event.P >= 0 && def != "" {
				card := cardResult(result.cards[event.P], def)
				card.Draws++
				card.DrawnGame = true
			}
		case engine.EvCardPlayed:
			if event.P >= 0 && def != "" {
				card := cardResult(result.cards[event.P], def)
				card.Plays++
				card.RoundPlayed += event.Round
				card.PlayedGame = true
			}
		case engine.EvDamage:
			if def != "" && game.Ruleset().Cards[def] != nil && owner >= 0 {
				cardResult(result.cards[owner], def).Damage += event.N
			}
		case engine.EvPrevented:
			if def != "" && game.Ruleset().Cards[def] != nil && owner >= 0 {
				cardResult(result.cards[owner], def).Prevented += event.N
			}
		case engine.EvEclipseShifted:
			if game.Ruleset().Cards[event.S] != nil && event.P >= 0 {
				cardResult(result.cards[event.P], event.S).EclipseDisplacement += abs(event.To - event.From)
			}
		}
	}
	for player := 0; player < 2; player++ {
		for _, inst := range game.State().Players[player].Hand {
			cardResult(result.cards[player], game.State().Cards[inst].Def).FinalHandCopies++
		}
	}
}

func validateZones(state *engine.GameState) error {
	seen := make(map[string]engine.Zone, len(state.Cards))
	add := func(player int, zone engine.Zone, ids []string) error {
		for _, id := range ids {
			card := state.Cards[id]
			if card == nil {
				return fmt.Errorf("p%d: zona %s contém instância ausente %s", player, zone, id)
			}
			if card.Owner != player {
				return fmt.Errorf("p%d: zona %s contém carta %s de p%d", player, zone, id, card.Owner)
			}
			if previous, exists := seen[id]; exists {
				return fmt.Errorf("p%d: carta %s aparece nas zonas %s e %s", player, id, previous, zone)
			}
			if card.Zone != zone {
				return fmt.Errorf("p%d: carta %s aparece em %s com zone=%s", player, id, zone, card.Zone)
			}
			seen[id] = zone
		}
		return nil
	}
	for player, p := range state.Players {
		zones := []struct {
			zone engine.Zone
			ids  []string
		}{{engine.ZoneDeck, p.Deck}, {engine.ZoneHand, p.Hand}, {engine.ZoneDiscard, p.Discard},
			{engine.ZoneExile, p.Exile}, {engine.ZonePlay, append(append([]string{}, p.Relics...), p.Manifs...)}}
		for _, current := range zones {
			if err := add(player, current.zone, current.ids); err != nil {
				return err
			}
		}
	}
	transient := [2][]string{}
	if state.Guard != nil && state.Guard.AssaultInst != "" {
		transient[state.Guard.Attacker] = append(transient[state.Guard.Attacker], state.Guard.AssaultInst)
	}
	if state.RiteReact != nil {
		transient[state.RiteReact.Caster] = append(transient[state.RiteReact.Caster], state.RiteReact.Inst)
	}
	for _, pending := range state.PendingRites {
		if pending.Inst != "" {
			transient[pending.Player] = append(transient[pending.Player], pending.Inst)
		}
	}
	for player, ids := range transient {
		if err := add(player, engine.ZonePlay, ids); err != nil {
			return err
		}
	}
	for id, card := range state.Cards {
		if _, exists := seen[id]; !exists {
			return fmt.Errorf("carta %s com zone=%s não aparece em nenhuma zona", id, card.Zone)
		}
	}
	return nil
}

func newAggregate() *aggregate {
	return &aggregate{champions: map[string]*rawChampion{}, matchups: map[string]*rawMatchup{},
		cards: map[string]*rawCard{}, endReasons: map[string][]int{}}
}

func (a *aggregate) add(result matchResult) {
	a.health.Requested++
	if result.crash {
		a.health.Crashes++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "crash", Detail: result.failure})
		return
	}
	if result.loop {
		a.health.Loops++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "loop", Detail: result.failure})
	}
	if result.dead {
		a.health.DeadStates++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "dead_state", Detail: result.failure})
	}
	if result.invalid {
		a.health.InvalidStates++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "invalid_state", Detail: result.failure})
	}
	if result.rejected {
		a.health.RejectedCommands++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "rejected_command", Detail: result.failure})
	}
	if result.divergence {
		a.health.DeterminismDivergences++
		a.failures = append(a.failures, FailureMetric{Game: result.index, Kind: "determinism", Detail: result.failure})
	}
	if result.loop || result.dead || result.invalid || result.rejected {
		return
	}
	a.health.Completed++
	a.rounds = append(a.rounds, result.rounds)
	a.commands = append(a.commands, result.commands)
	a.endReasons[result.endReason] = append(a.endReasons[result.endReason], result.rounds)
	a.firstGames++
	if result.winner == result.first {
		a.firstWins++
	}
	for player, id := range result.champions {
		if a.champions[id] == nil {
			a.champions[id] = &rawChampion{}
		}
		a.champions[id].Games++
		if result.winner == player {
			a.champions[id].Wins++
		}
	}
	key := result.champions[0] + "|" + result.champions[1]
	if a.matchups[key] == nil {
		a.matchups[key] = &rawMatchup{}
	}
	a.matchups[key].Games++
	a.matchups[key].Rounds = append(a.matchups[key].Rounds, result.rounds)
	if result.winner == 0 {
		a.matchups[key].Wins0++
	}
	for player := 0; player < 2; player++ {
		won := result.winner == player
		for id, observed := range result.cards[player] {
			if a.cards[id] == nil {
				a.cards[id] = &rawCard{}
			}
			raw := a.cards[id]
			raw.InclusionGames += observed.Inclusion
			raw.Draws += observed.Draws
			raw.Plays += observed.Plays
			raw.MulliganSeen += observed.MulliganSeen
			raw.MulliganKept += observed.MulliganSeen - observed.Mulliganed
			raw.Damage += observed.Damage
			raw.Prevented += observed.Prevented
			raw.EclipseDisplacement += observed.EclipseDisplacement
			raw.RoundPlayed += observed.RoundPlayed
			raw.FinalHandCopies += observed.FinalHandCopies
			if observed.DrawnGame {
				raw.DrawnGames++
				if won {
					raw.DrawnWins++
				}
			}
			if observed.PlayedGame {
				raw.PlayedGames++
				if won {
					raw.PlayedWins++
				}
			}
		}
	}
}

func (a *aggregate) merge(other *aggregate) {
	a.health.Requested += other.health.Requested
	a.health.Completed += other.health.Completed
	a.health.Crashes += other.health.Crashes
	a.health.Loops += other.health.Loops
	a.health.DeadStates += other.health.DeadStates
	a.health.InvalidStates += other.health.InvalidStates
	a.health.RejectedCommands += other.health.RejectedCommands
	a.health.DeterminismDivergences += other.health.DeterminismDivergences
	a.firstGames += other.firstGames
	a.firstWins += other.firstWins
	a.rounds = append(a.rounds, other.rounds...)
	a.commands = append(a.commands, other.commands...)
	a.failures = append(a.failures, other.failures...)
	for reason, rounds := range other.endReasons {
		a.endReasons[reason] = append(a.endReasons[reason], rounds...)
	}
	for id, metric := range other.champions {
		if a.champions[id] == nil {
			a.champions[id] = &rawChampion{}
		}
		a.champions[id].Games += metric.Games
		a.champions[id].Wins += metric.Wins
	}
	for key, metric := range other.matchups {
		if a.matchups[key] == nil {
			a.matchups[key] = &rawMatchup{}
		}
		a.matchups[key].Games += metric.Games
		a.matchups[key].Wins0 += metric.Wins0
		a.matchups[key].Rounds = append(a.matchups[key].Rounds, metric.Rounds...)
	}
	for id, metric := range other.cards {
		if a.cards[id] == nil {
			a.cards[id] = &rawCard{}
		}
		left, right := a.cards[id], metric
		left.InclusionGames += right.InclusionGames
		left.Draws += right.Draws
		left.DrawnGames += right.DrawnGames
		left.DrawnWins += right.DrawnWins
		left.Plays += right.Plays
		left.PlayedGames += right.PlayedGames
		left.PlayedWins += right.PlayedWins
		left.MulliganSeen += right.MulliganSeen
		left.MulliganKept += right.MulliganKept
		left.Damage += right.Damage
		left.Prevented += right.Prevented
		left.EclipseDisplacement += right.EclipseDisplacement
		left.RoundPlayed += right.RoundPlayed
		left.FinalHandCopies += right.FinalHandCopies
	}
}

func (a *aggregate) report(cfg Config) Report {
	rs, rsErr := engine.RulesetByVersion(cfg.RulesetVersion)
	if rsErr != nil {
		rs = engine.Builtin()
	}
	report := Report{SchemaVersion: "balance-report.v2", Ruleset: cfg.RulesetVersion, Config: cfg, Health: a.health}
	sort.Slice(a.failures, func(i, j int) bool { return a.failures[i].Game < a.failures[j].Game })
	report.Failures = append(report.Failures, a.failures...)
	report.FirstPlayer = FirstPlayerMetric{Games: a.firstGames, Wins: a.firstWins, WinRate: rate(a.firstWins, a.firstGames)}
	report.Duration = durationMetric(a.rounds, a.commands)
	endReasonNames := make([]string, 0, len(a.endReasons))
	for reason := range a.endReasons {
		endReasonNames = append(endReasonNames, reason)
	}
	sort.Strings(endReasonNames)
	for _, reason := range endReasonNames {
		rounds := append([]int{}, a.endReasons[reason]...)
		sort.Ints(rounds)
		report.EndReasons = append(report.EndReasons, EndReasonMetric{Reason: reason, Games: len(rounds),
			Share: rate(len(rounds), a.health.Completed), AverageRounds: average(rounds), P95Rounds: percentile(rounds, 0.95)})
	}
	for _, id := range championIDs(rs) {
		raw := a.champions[id]
		if raw == nil {
			raw = &rawChampion{}
		}
		report.Champions = append(report.Champions, ChampionMetric{ID: id, Name: championName(rs, id),
			Games: raw.Games, Wins: raw.Wins, Losses: raw.Games - raw.Wins, WinRate: rate(raw.Wins, raw.Games)})
	}
	keys := make([]string, 0, len(a.matchups))
	for key := range a.matchups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, "|")
		raw := a.matchups[key]
		rounds := append([]int{}, raw.Rounds...)
		sort.Ints(rounds)
		report.Matchups = append(report.Matchups, MatchupMetric{Champion0: parts[0], Champion1: parts[1], Games: raw.Games,
			Wins0: raw.Wins0, WinRate0: rate(raw.Wins0, raw.Games), AverageRounds: average(rounds), P95Rounds: percentile(rounds, 0.95)})
	}
	playerGames := max(1, a.health.Completed*2)
	minimumDominanceGames := max(20, cfg.Games/200)
	for _, definition := range rs.CardList {
		raw := a.cards[definition.ID]
		if raw == nil {
			raw = &rawCard{}
		}
		metric := CardMetric{ID: definition.ID, Name: definition.Name, InclusionGames: raw.InclusionGames,
			InclusionRate: rate(raw.InclusionGames, playerGames), Draws: raw.Draws, DrawnGames: raw.DrawnGames,
			DrawnWins: raw.DrawnWins, DrawnWinRate: rate(raw.DrawnWins, raw.DrawnGames), Plays: raw.Plays,
			PlayedGames: raw.PlayedGames, PlayedWins: raw.PlayedWins, PlayedWinRate: rate(raw.PlayedWins, raw.PlayedGames),
			MulliganSeen: raw.MulliganSeen, MulliganKept: raw.MulliganKept, MulliganKeepRate: rate(raw.MulliganKept, raw.MulliganSeen),
			Damage: raw.Damage, Prevented: raw.Prevented, EclipseDisplacement: raw.EclipseDisplacement,
			AverageRoundPlayed: rate(raw.RoundPlayed, raw.Plays), FinalHandCopies: raw.FinalHandCopies,
			DeadInHandRate: rate(raw.FinalHandCopies, raw.Draws)}
		metric.DominanceAlert = metric.PlayedGames >= minimumDominanceGames && metric.PlayedWinRate > 0.65
		report.Cards = append(report.Cards, metric)
		if metric.PlayedGames >= minimumDominanceGames {
			report.DominantCards = append(report.DominantCards, metric)
		}
	}
	sort.Slice(report.DominantCards, func(i, j int) bool {
		if report.DominantCards[i].PlayedWinRate != report.DominantCards[j].PlayedWinRate {
			return report.DominantCards[i].PlayedWinRate > report.DominantCards[j].PlayedWinRate
		}
		if report.DominantCards[i].PlayedGames != report.DominantCards[j].PlayedGames {
			return report.DominantCards[i].PlayedGames > report.DominantCards[j].PlayedGames
		}
		return report.DominantCards[i].ID < report.DominantCards[j].ID
	})
	if len(report.DominantCards) > 15 {
		report.DominantCards = report.DominantCards[:15]
	}
	if math.Abs(report.FirstPlayer.WinRate-0.5) > 0.05 {
		report.Warnings = append(report.Warnings, "vantagem do primeiro jogador acima de 5 pontos percentuais")
	}
	for _, card := range report.DominantCards {
		if card.DominanceAlert {
			report.Warnings = append(report.Warnings, card.ID+" excedeu 65% de played win rate")
		}
	}
	report.Gate = evaluateBalanceGate(report)
	return report
}

func evaluateBalanceGate(report Report) BalanceGate {
	gate := BalanceGate{Enabled: report.Config.Games >= 100_000 && report.Config.Bot0 == BotHeuristic && report.Config.Bot1 == BotHeuristic, Passed: true}
	if !gate.Enabled {
		return gate
	}
	add := func(name string, passed bool, detail string) {
		gate.Checks = append(gate.Checks, GateCheck{Name: name, Passed: passed, Detail: detail})
		gate.Passed = gate.Passed && passed
	}
	add("first_player_47_5_52_5", report.FirstPlayer.WinRate >= 0.475 && report.FirstPlayer.WinRate <= 0.525,
		fmt.Sprintf("%.2f%%", report.FirstPlayer.WinRate*100))
	maxP95, minP50 := 40, 0
	if rs, err := engine.RulesetByVersion(report.Ruleset); err == nil && rs.IsConfront() {
		maxP95 = 30
		if target := rs.ConfrontRules.TargetP95Rounds; target > 0 {
			maxP95 = target
		}
		// O piso de duração vale sobre os precons, que são a régua oficial do
		// formato. Decks sorteados existem para caçar combinações doentes do
		// pool inteiro e correm mais rápido por construção; medir ritmo neles
		// reprovaria o gate de expansão por um motivo que não é dele.
		if report.Config.DeckMode == DeckPrecon || report.Config.DeckMode == "" {
			minP50 = rs.ConfrontRules.TargetMinP50Rounds
		}
	}
	add(fmt.Sprintf("p95_rounds_le_%d", maxP95), report.Duration.P95Rounds <= maxP95,
		fmt.Sprintf("%d", report.Duration.P95Rounds))
	if minP50 > 0 {
		add(fmt.Sprintf("p50_rounds_ge_%d", minP50), report.Duration.P50Rounds >= minP50,
			fmt.Sprintf("%d", report.Duration.P50Rounds))
	}
	champMin, champMax := 1.0, 0.0
	for _, champion := range report.Champions {
		champMin, champMax = math.Min(champMin, champion.WinRate), math.Max(champMax, champion.WinRate)
	}
	add("champions_40_60", champMin >= 0.40 && champMax <= 0.60,
		fmt.Sprintf("%.2f%%–%.2f%%", champMin*100, champMax*100))
	matchupLow, matchupHigh := 0.25, 0.75
	if report.Config.DeckMode == DeckPrecon || report.Config.DeckMode == "" {
		matchupLow, matchupHigh = 0.20, 0.80
	}
	matchupMin, matchupMax := 1.0, 0.0
	for _, matchup := range report.Matchups {
		matchupMin, matchupMax = math.Min(matchupMin, matchup.WinRate0), math.Max(matchupMax, matchup.WinRate0)
	}
	add("matchups_in_range", matchupMin >= matchupLow && matchupMax <= matchupHigh,
		fmt.Sprintf("%.2f%%–%.2f%% (limite %.0f%%–%.0f%%)", matchupMin*100, matchupMax*100, matchupLow*100, matchupHigh*100))
	minimumSample := max(500, report.Config.Games/200)
	cardPass, cardFailures := true, 0
	for _, card := range report.Cards {
		if card.DrawnGames >= minimumSample && (card.DrawnWinRate < 0.35 || card.DrawnWinRate > 0.65) {
			cardPass = false
			cardFailures++
		}
	}
	add("sampled_cards_drawn_35_65", cardPass, fmt.Sprintf("%d fora da faixa (amostra mínima %d)", cardFailures, minimumSample))
	if rs, err := engine.RulesetByVersion(report.Ruleset); err == nil && rs.IsConfront() {
		assaultShare := 0.0
		for _, reason := range report.EndReasons {
			if reason.Reason == "assalto" {
				assaultShare = reason.Share
				break
			}
		}
		add("assault_finishes_ge_35", assaultShare >= 0.35, fmt.Sprintf("%.2f%%", assaultShare*100))
	}
	return gate
}

func durationMetric(rounds, commands []int) DurationMetric {
	if len(rounds) == 0 {
		return DurationMetric{}
	}
	sort.Ints(rounds)
	sort.Ints(commands)
	return DurationMetric{AverageRounds: average(rounds), P50Rounds: percentile(rounds, 0.50), P95Rounds: percentile(rounds, 0.95), MaxRounds: rounds[len(rounds)-1],
		AverageCommands: average(commands), P50Commands: percentile(commands, 0.50), P95Commands: percentile(commands, 0.95), MaxCommands: commands[len(commands)-1]}
}

func average(values []int) float64 {
	total := 0
	for _, value := range values {
		total += value
	}
	return rounded(float64(total) / float64(len(values)))
}

func percentile(values []int, quantile float64) int {
	index := int(math.Ceil(float64(len(values))*quantile)) - 1
	return values[max(0, min(index, len(values)-1))]
}

func rate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return rounded(float64(numerator) / float64(denominator))
}

func rounded(value float64) float64 { return math.Round(value*1_000_000) / 1_000_000 }
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func championName(rs *engine.Ruleset, id string) string {
	if champ := rs.Champions[id]; champ != nil {
		return champ.Name
	}
	return id
}

// VariedDeckFor expõe o gerador de decks variados para ferramentas de
// depuração (mesma matemática do harness).
func VariedDeckFor(rs *engine.Ruleset, championID string, rng *engine.RNG) []string {
	return variedDeck(rs, championID, rng)
}

// ValidateZonesFor expõe o verificador de zonas para depuração.
func ValidateZonesFor(state *engine.GameState) error { return validateZones(state) }
