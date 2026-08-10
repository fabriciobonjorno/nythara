package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"veurubro/backend/internal/sim"
)

func main() {
	defaults := sim.DefaultConfig()
	games := flag.Int("games", defaults.Games, "quantidade de partidas")
	seed := flag.Uint64("seed", defaults.BaseSeed, "seed base")
	workers := flag.Int("workers", runtime.GOMAXPROCS(0), "workers paralelos")
	maxCommands := flag.Int("max-commands", defaults.MaxCommands, "teto de comandos por partida")
	bot0 := flag.String("bot0", string(defaults.Bot0), "bot do slot 0: random|heuristic")
	bot1 := flag.String("bot1", string(defaults.Bot1), "bot do slot 1: random|heuristic")
	verify := flag.Bool("verify-replay", defaults.VerifyReplay, "reproduzir cada partida e comparar log/snapshot")
	jsonPath := flag.String("json", "artifacts/balance-report.json", "saída JSON")
	markdownPath := flag.String("markdown", "artifacts/balance-report.md", "saída Markdown")
	flag.Parse()

	cfg := sim.Config{Games: *games, BaseSeed: *seed, Workers: *workers, MaxCommands: *maxCommands,
		Bot0: sim.BotKind(*bot0), Bot1: sim.BotKind(*bot1), VerifyReplay: *verify}
	report, runErr := sim.Run(cfg)
	if report.SchemaVersion != "" {
		if err := sim.WriteJSON(*jsonPath, report); err != nil {
			fmt.Fprintln(os.Stderr, "escrever JSON:", err)
			os.Exit(1)
		}
		if err := sim.WriteMarkdown(*markdownPath, report); err != nil {
			fmt.Fprintln(os.Stderr, "escrever Markdown:", err)
			os.Exit(1)
		}
		fmt.Printf("%d/%d partidas; first-player %.2f%%; média %.2f rodadas; relatórios: %s, %s\n",
			report.Health.Completed, report.Health.Requested, report.FirstPlayer.WinRate*100,
			report.Duration.AverageRounds, *jsonPath, *markdownPath)
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		os.Exit(1)
	}
}
