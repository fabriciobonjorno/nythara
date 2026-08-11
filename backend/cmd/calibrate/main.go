// Comando de calibração: varre uma grade de parâmetros do Modo Confronto,
// compila um ruleset por combinação e roda o simulador em cada um. Serve para
// escolher números de balanceamento com evidência em vez de palpite.
//
//	go run ./cmd/calibrate -games 1500 -vitality 44,52,60 -guard-bonus 3,4
//
// Nada aqui altera o produto: as versões geradas são efêmeras (prefixo `cal-`)
// e existem apenas dentro do processo.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/sim"
)

type variant struct {
	name string
	cfg  engine.ConfrontRulesConfig
}

type outcome struct {
	variant
	rounds     float64
	p50        int
	p95        int
	maxRounds  int
	commands   float64
	firstWR    float64
	pressure   float64
	assault    float64
	fatigue    float64
	spreadWR   float64
	guardDead  float64
	guardPlays float64
}

func main() {
	games := flag.Int("games", 1200, "partidas por variante")
	seed := flag.Uint64("seed", 1, "seed base")
	dataDir := flag.String("data", "internal/engine/data", "diretório com cards/champions/effects")
	vitalities := flag.String("vitality", "44", "lista de Vitalidade inicial")
	guardBonuses := flag.String("guard-bonus", "3", "lista de bônus de Prevenção da Guarda")
	pressures := flag.String("pressure", "34", "lista de turnos de início da Pressão")
	firstPenalties := flag.String("first-penalty", "2", "lista de penalidades do primeiro turno")
	bonusDraws := flag.String("second-draw", "0", "lista de cartas extras para quem não abre")
	minGuards := flag.String("min-guards", strconv.Itoa(engine.ConfrontMinGuards), "lista de mínimos de Guarda no baralho")
	minRites := flag.String("min-rites", strconv.Itoa(engine.ConfrontMinRites), "lista de mínimos de Rito no baralho")
	leakCaps := flag.String("leak-cap", "0", "lista de tetos de vazamento da Guarda (0 = subtração pura)")
	deckMode := flag.String("decks", "precon", "decks: precon|varied")
	powerCap := flag.Int("power-cap", 0, "teto de Poder do Assalto (0 = curva original); testa compressão do topo sem alterar o catálogo")
	flag.Parse()

	cards, champions, effects, err := readData(*dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	variants, err := buildVariants(*vitalities, *guardBonuses, *pressures, *firstPenalties, *bonusDraws, *minGuards, *minRites, *leakCaps)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	results := make([]outcome, 0, len(variants))
	for _, v := range variants {
		rs, err := compile(v, cards, champions, effects, *powerCap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "variante %s: %v\n", v.name, err)
			os.Exit(1)
		}
		if err := engine.RegisterRuleset(rs); err != nil {
			fmt.Fprintf(os.Stderr, "registrar %s: %v\n", v.name, err)
			os.Exit(1)
		}
		report, err := sim.Run(sim.Config{
			Games: *games, BaseSeed: *seed, Workers: runtime.GOMAXPROCS(0),
			MaxCommands: sim.DefaultConfig().MaxCommands,
			Bot0:        sim.BotHeuristic, Bot1: sim.BotHeuristic, VerifyReplay: false,
			RulesetVersion: v.name, DeckMode: sim.DeckMode(*deckMode),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "simular %s: %v\n", v.name, err)
			os.Exit(1)
		}
		results = append(results, summarize(v, report, rs))
		engine.UnregisterRuleset(v.name)
	}

	sort.Slice(results, func(a, b int) bool { return results[a].rounds > results[b].rounds })
	fmt.Printf("%-26s %7s %5s %5s %8s %9s %9s %9s %9s\n",
		"variante", "rodadas", "p50", "p95", "comandos", "1º jog.", "assalto", "fadiga", "pressão")
	for _, r := range results {
		fmt.Printf("%-26s %7.2f %5d %5d %8.2f %8.2f%% %8.2f%% %8.2f%% %8.2f%%\n",
			r.name, r.rounds, r.p50, r.p95, r.commands,
			r.firstWR*100, r.assault*100, r.fatigue*100, r.pressure*100)
	}
	fmt.Println("assalto = partidas decididas por dano; fadiga = decididas pelo baralho acabar")
}

func buildVariants(vitality, guard, pressure, penalty, draws, guardFloor, riteFloor, leak string) ([]variant, error) {
	vs, err := ints(vitality)
	if err != nil {
		return nil, fmt.Errorf("vitality: %w", err)
	}
	gs, err := ints(guard)
	if err != nil {
		return nil, fmt.Errorf("guard-bonus: %w", err)
	}
	ps, err := ints(pressure)
	if err != nil {
		return nil, fmt.Errorf("pressure: %w", err)
	}
	fs, err := ints(penalty)
	if err != nil {
		return nil, fmt.Errorf("first-penalty: %w", err)
	}
	ds, err := ints(draws)
	if err != nil {
		return nil, fmt.Errorf("second-draw: %w", err)
	}
	ms, err := ints(guardFloor)
	if err != nil {
		return nil, fmt.Errorf("min-guards: %w", err)
	}
	rt, err := ints(riteFloor)
	if err != nil {
		return nil, fmt.Errorf("min-rites: %w", err)
	}
	lc, err := ints(leak)
	if err != nil {
		return nil, fmt.Errorf("leak-cap: %w", err)
	}
	out := make([]variant, 0, len(vs)*len(gs)*len(ps)*len(fs)*len(ds))
	for _, v := range vs {
		for _, g := range gs {
			for _, p := range ps {
				for _, f := range fs {
					for _, d := range ds {
						for _, m := range ms {
							for _, r := range rt {
								for _, l := range lc {
									cfg := engine.ConfrontRulesConfig{
										StartingVitality: v, PowerBonus: engine.ConfrontPowerBonus,
										FirstTurnPenalty: f, ExposedPowerBonus: 2, DrawOnFirstTurn: true,
										PressureStartTurn: p, PressureBaseLoss: engine.ConfrontPressureBaseLoss,
										MinAssaults: engine.ConfrontMinAssaults, MinGuards: m,
										MinRites: r, TacticalSeals: true,
										GuardBonus: g, SecondPlayerBonusDraw: d, GuardLeakCap: l,
									}
									out = append(out, variant{
										name: fmt.Sprintf("cal-v%dg%dm%dr%dl%d", v, g, m, r, l),
										cfg:  cfg,
									})
								}
							}
						}
					}
				}
			}
		}
	}
	return out, nil
}

func compile(v variant, cards, champions, effects []byte, powerCap int) (*engine.Ruleset, error) {
	var file map[string]any
	if err := json.Unmarshal(effects, &file); err != nil {
		return nil, err
	}
	file["version"] = v.name
	file["mode"] = "confront"
	raw, err := json.Marshal(v.cfg)
	if err != nil {
		return nil, err
	}
	var rules map[string]any
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, err
	}
	file["confront_rules"] = rules
	if powerCap > 0 {
		capAssaultPower(file, powerCap, v.cfg.PowerBonus)
	}
	patched, err := json.Marshal(file)
	if err != nil {
		return nil, err
	}
	return engine.CompileRuleset(v.name, cards, champions, patched)
}

// capAssaultPower comprime o topo da curva de Poder direto no JSON de efeitos,
// permitindo medir o impacto de um rebalanceamento de conteúdo antes de mexer
// no catálogo de verdade.
func capAssaultPower(file map[string]any, cap, powerBonus int) {
	cards, ok := file["cards"].(map[string]any)
	if !ok {
		return
	}
	for _, raw := range cards {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		assault, ok := entry["assault"].(map[string]any)
		if !ok {
			continue
		}
		damage, ok := assault["damage"].(float64)
		if !ok {
			continue
		}
		instances := 1.0
		if value, ok := assault["instances"].(float64); ok && value > 1 {
			instances = value
		}
		maxDamage := float64(cap-powerBonus) / instances
		if damage > maxDamage {
			assault["damage"] = maxDamage
		}
	}
}

func summarize(v variant, report sim.Report, rs *engine.Ruleset) outcome {
	out := outcome{variant: v,
		rounds: report.Duration.AverageRounds, p50: report.Duration.P50Rounds,
		p95: report.Duration.P95Rounds, maxRounds: report.Duration.MaxRounds,
		commands: report.Duration.AverageCommands, firstWR: report.FirstPlayer.WinRate}
	for _, reason := range report.EndReasons {
		switch reason.Reason {
		case "pressao_de_nythara":
			out.pressure = reason.Share
		case "assalto":
			out.assault = reason.Share
		case "fadiga":
			out.fatigue = reason.Share
		}
	}
	low, high := 1.0, 0.0
	for _, champion := range report.Champions {
		low = min(low, champion.WinRate)
		high = max(high, champion.WinRate)
	}
	if len(report.Champions) > 0 {
		out.spreadWR = high - low
	}
	dead, guards := 0.0, 0.0
	for _, card := range report.DominantCards {
		def := rs.Cards[card.ID]
		if def == nil || def.Type != engine.TypeGuarda {
			continue
		}
		dead += card.DeadInHandRate
		guards++
	}
	if guards > 0 {
		out.guardDead = dead / guards
		out.guardPlays = guards
	}
	return out
}

func readData(dir string) (cards, champions, effects []byte, err error) {
	read := func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, name)) }
	if cards, err = read("cards_alpha.json"); err != nil {
		return nil, nil, nil, err
	}
	if champions, err = read("champions_alpha.json"); err != nil {
		return nil, nil, nil, err
	}
	if effects, err = read("effects_alpha.json"); err != nil {
		return nil, nil, nil, err
	}
	return cards, champions, effects, nil
}

func ints(list string) ([]int, error) {
	parts := strings.Split(list, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}
