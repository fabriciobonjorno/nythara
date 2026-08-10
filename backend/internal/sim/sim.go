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
}

type Report struct {
	SchemaVersion string            `json:"schema_version"`
	Ruleset       string            `json:"ruleset_version"`
	Config        Config            `json:"config"`
	Health        HealthMetric      `json:"health"`
	Duration      DurationMetric    `json:"duration"`
	FirstPlayer   FirstPlayerMetric `json:"first_player"`
	Champions     []ChampionMetric  `json:"champions"`
	Matchups      []MatchupMetric   `json:"matchups"`
	Cards         []CardMetric      `json:"cards"`
	DominantCards []CardMetric      `json:"dominant_cards"`
	Failures      []FailureMetric   `json:"failures,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
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
	Champion0 string  `json:"champion_0"`
	Champion1 string  `json:"champion_1"`
	Games     int     `json:"games"`
	Wins0     int     `json:"wins_0"`
	WinRate0  float64 `json:"win_rate_0"`
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
type rawMatchup struct{ Games, Wins0 int }
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
		cfg.RulesetVersion = engine.RulesetVersion
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
	return PreconstructedDeckFor(engine.Builtin(), championID)
}

// PreconstructedDeckFor monta o precon determinístico de um Campeão sob o
// ruleset dado.
func PreconstructedDeckFor(rs *engine.Ruleset, championID string) ([]string, error) {
	champion := rs.Champions[championID]
	if champion == nil {
		return nil, fmt.Errorf("campeão desconhecido: %s", championID)
	}
	cards := append([]*engine.CardDef{}, rs.CardList...)
	sort.Slice(cards, func(i, j int) bool {
		leftCore := cards[i].Faction == champion.Faction
		rightCore := cards[j].Faction == champion.Faction
		if leftCore != rightCore {
			return leftCore
		}
		return cards[i].ID < cards[j].ID
	})
	deck := make([]string, 0, engine.DeckSize)
	for _, card := range cards {
		if card.Faction != champion.Faction && card.Faction != engine.NeutralFaction {
			continue
		}
		copies := engine.MaxCopies
		if card.Rarity == engine.RarityLendaria {
			copies = engine.MaxLegendary
		}
		for range copies {
			if len(deck) == engine.DeckSize {
				break
			}
			deck = append(deck, card.ID)
		}
		if len(deck) == engine.DeckSize {
			break
		}
	}
	if err := rs.ValidateDeck(championID, deck); err != nil {
		return nil, fmt.Errorf("precon %s: %w", championID, err)
	}
	return deck, nil
}

func simulateOne(cfg Config, index int, champions []string, decks map[string][]string) (result matchResult) {
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
	gameCfg := engine.Config{RulesetVersion: cfg.RulesetVersion, Seed: seed, FirstPlayer: result.first,
		Players: [2]engine.PlayerSetup{{ChampionID: result.champions[0], Deck: decks[result.champions[0]]},
			{ChampionID: result.champions[1], Deck: decks[result.champions[1]]}}}
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
	return &aggregate{champions: map[string]*rawChampion{}, matchups: map[string]*rawMatchup{}, cards: map[string]*rawCard{}}
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
	report := Report{SchemaVersion: "balance-report.v1", Ruleset: cfg.RulesetVersion, Config: cfg, Health: a.health}
	sort.Slice(a.failures, func(i, j int) bool { return a.failures[i].Game < a.failures[j].Game })
	report.Failures = append(report.Failures, a.failures...)
	report.FirstPlayer = FirstPlayerMetric{Games: a.firstGames, Wins: a.firstWins, WinRate: rate(a.firstWins, a.firstGames)}
	report.Duration = durationMetric(a.rounds, a.commands)
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
		report.Matchups = append(report.Matchups, MatchupMetric{Champion0: parts[0], Champion1: parts[1], Games: raw.Games,
			Wins0: raw.Wins0, WinRate0: rate(raw.Wins0, raw.Games)})
	}
	playerGames := max(1, a.health.Completed*2)
	minimumDominanceGames := max(20, cfg.Games/200)
	for _, definition := range engine.CardList {
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
	return report
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
