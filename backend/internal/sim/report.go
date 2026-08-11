package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteJSON(path string, report Report) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeFile(path, raw)
}

func WriteMarkdown(path string, report Report) error {
	var out strings.Builder
	fmt.Fprintf(&out, "# Relatório de balanceamento — %s\n\n", report.Ruleset)
	fmt.Fprintf(&out, "Baseline: **%d partidas**, `%s` vs `%s`, seed base `%d`, replay integral `%t`.\n\n",
		report.Config.Games, report.Config.Bot0, report.Config.Bot1, report.Config.BaseSeed, report.Config.VerifyReplay)
	out.WriteString("## Saúde do gate\n\n")
	fmt.Fprintf(&out, "| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |\n|---:|---:|---:|---:|---:|---:|---:|\n| %d | %d | %d | %d | %d | %d | %d |\n\n",
		report.Health.Completed, report.Health.Crashes, report.Health.Loops, report.Health.DeadStates, report.Health.InvalidStates,
		report.Health.RejectedCommands, report.Health.DeterminismDivergences)
	if len(report.Failures) > 0 {
		out.WriteString("### Falhas\n\n")
		for _, failure := range report.Failures {
			fmt.Fprintf(&out, "- Jogo %d (`%s`): %s\n", failure.Game, failure.Kind, failure.Detail)
		}
		out.WriteString("\n")
	}
	out.WriteString("## Ritmo e iniciativa\n\n")
	fmt.Fprintf(&out, "- Rodadas: média **%.2f**, p50 **%d**, p95 **%d**, máximo **%d**.\n", report.Duration.AverageRounds,
		report.Duration.P50Rounds, report.Duration.P95Rounds, report.Duration.MaxRounds)
	fmt.Fprintf(&out, "- Comandos: média **%.2f**, p50 **%d**, p95 **%d**, máximo **%d**.\n", report.Duration.AverageCommands,
		report.Duration.P50Commands, report.Duration.P95Commands, report.Duration.MaxCommands)
	fmt.Fprintf(&out, "- Primeiro jogador: **%s** (%d/%d).\n\n", percent(report.FirstPlayer.WinRate), report.FirstPlayer.Wins, report.FirstPlayer.Games)
	out.WriteString("### Causa do término\n\n| Causa | Partidas | Participação | Média de rodadas | p95 |\n|---|---:|---:|---:|---:|\n")
	for _, reason := range report.EndReasons {
		fmt.Fprintf(&out, "| %s | %d | %s | %.2f | %d |\n", reason.Reason, reason.Games, percent(reason.Share), reason.AverageRounds, reason.P95Rounds)
	}
	out.WriteString("\n")
	out.WriteString("## Win rate por Campeão\n\n| Campeão | Partidas | Vitórias | Win rate |\n|---|---:|---:|---:|\n")
	for _, champion := range report.Champions {
		fmt.Fprintf(&out, "| %s (`%s`) | %d | %d | %s |\n", champion.Name, champion.ID, champion.Games, champion.Wins, percent(champion.WinRate))
	}
	out.WriteString("\n## Sinais de cartas dominantes\n\n")
	out.WriteString("Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.\n\n")
	out.WriteString("| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |\n|---|---:|---:|---:|---:|---:|---:|:---:|\n")
	for _, card := range report.DominantCards {
		alert := "—"
		if card.DominanceAlert {
			alert = "⚠"
		}
		fmt.Fprintf(&out, "| %s (`%s`) | %d | %s | %s | %d | %.2f | %s | %s |\n", card.Name, card.ID,
			card.PlayedGames, percent(card.PlayedWinRate), percent(card.DrawnWinRate), card.Plays,
			card.AverageRoundPlayed, percent(card.DeadInHandRate), alert)
	}
	if len(report.Warnings) > 0 {
		out.WriteString("\n## Alertas\n\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&out, "- %s\n", warning)
		}
	}
	if report.Gate.Enabled {
		status := "APROVADO"
		if !report.Gate.Passed {
			status = "REPROVADO"
		}
		fmt.Fprintf(&out, "\n## Gate competitivo — %s\n\n", status)
		for _, check := range report.Gate.Checks {
			mark := "✓"
			if !check.Passed {
				mark = "✗"
			}
			fmt.Fprintf(&out, "- %s `%s`: %s\n", mark, check.Name, check.Detail)
		}
	}
	out.WriteString("\n## Interpretação\n\n")
	if report.Config.DeckMode == DeckVaried {
		out.WriteString("Este baseline usa decks variados legais sobre todo o pool do Modo Confronto, com composição mínima 8/8/4. ")
	} else {
		out.WriteString("Este baseline usa os precons oficiais determinísticos. ")
	}
	out.WriteString("Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.\n")
	return writeFile(path, []byte(out.String()))
}

func writeFile(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func percent(value float64) string { return fmt.Sprintf("%.2f%%", value*100) }
