package app

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"veurubro/backend/internal/engine"
)

// P1 — rituais diários: 3 objetivos por dia (UTC), sorteio determinístico por
// (dia, jogador), progresso derivado exclusivamente dos eventos authoritative
// das partidas. Rituais também ensinam o jogo (cada um aponta uma mecânica).

// RitualDef define um objetivo do pool.
type RitualDef struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Target      int    `json:"target"`
	Reward      int    `json:"reward"` // fragmentos
	PvPOnly     bool   `json:"pvp_only"`
}

// RitualPool é o pool do alpha. IDs são estáveis (ficam persistidos).
var RitualPool = []RitualDef{
	{ID: "win_pvp_1", Title: "Vença sob o Véu", Description: "Vença 1 partida PvP.", Target: 1, Reward: 60, PvPOnly: true},
	{ID: "play_matches_2", Title: "Dois duelos", Description: "Complete 2 partidas (treino conta).", Target: 2, Reward: 40},
	{ID: "play_ritos_5", Title: "Preparação arcana", Description: "Resolva 5 Ritos.", Target: 5, Reward: 40},
	{ID: "play_guards_3", Title: "Muralha paciente", Description: "Jogue 3 Guardas.", Target: 3, Reward: 40},
	{ID: "deal_damage_25", Title: "Golpes decisivos", Description: "Cause 25 de dano.", Target: 25, Reward: 40},
	{ID: "win_confronts_5", Title: "Domínio do centro", Description: "Vença 5 confrontos entre Assalto e Guarda.", Target: 5, Reward: 50},
	{ID: "shatter_rival_4", Title: "Estilhaços na mesa", Description: "Estilhace 4 cartas rivais.", Target: 4, Reward: 50},
	{ID: "play_assaults_6", Title: "Pressão constante", Description: "Jogue 6 Assaltos.", Target: 6, Reward: 40},
}

const DailyRituals = 3

// ritualsForDay sorteia 3 rituais distintos, determinístico por (dia, user).
func ritualsForDay(day string, userID string) []RitualDef {
	seedBytes := sha256.Sum256([]byte(day + "|" + userID))
	seed := binary.BigEndian.Uint64(seedBytes[:8])
	rng := engine.NewRNG(seed)
	indexes := make([]int, len(RitualPool))
	for i := range indexes {
		indexes[i] = i
	}
	rng.Shuffle(len(indexes), func(a, b int) { indexes[a], indexes[b] = indexes[b], indexes[a] })
	out := make([]RitualDef, 0, DailyRituals)
	for _, index := range indexes[:DailyRituals] {
		out = append(out, RitualPool[index])
	}
	return out
}

func ritualByID(id string) *RitualDef {
	for i := range RitualPool {
		if RitualPool[i].ID == id {
			return &RitualPool[i]
		}
	}
	return nil
}

// utcDay normaliza o dia de referência dos rituais.
func utcDay(now time.Time) string { return now.UTC().Format("2006-01-02") }

// MatchStats agrega, por jogador, o que os eventos da partida comprovam.
type MatchStats struct {
	RitosResolved       int
	GuardsPlayed        int
	AssaultsPlayed      int
	DamageDealt         int
	ConfrontsWon        int
	RivalCardsShattered int
}

// StatsFromEvents deriva as métricas por assento a partir do log
// authoritative. rs resolve tipos de carta da versão da partida.
func StatsFromEvents(rs *engine.Ruleset, events []engine.Event) [2]MatchStats {
	var stats [2]MatchStats
	for _, event := range events {
		switch event.Kind {
		case engine.EvCardPlayed:
			if event.P != 0 && event.P != 1 {
				continue
			}
			def := rs.Cards[event.Def]
			if def == nil {
				continue
			}
			switch def.Type {
			case engine.TypeRito:
				stats[event.P].RitosResolved++
			case engine.TypeGuarda:
				stats[event.P].GuardsPlayed++
			case engine.TypeAssalto:
				stats[event.P].AssaultsPlayed++
			}
		case engine.EvConfrontationResolved:
			if event.P != 0 && event.P != 1 {
				continue
			}
			winner := event.P
			if event.S == "guard" {
				winner = 1 - event.P
			}
			stats[winner].ConfrontsWon++
		case engine.EvCardShattered:
			if event.P == 0 || event.P == 1 {
				stats[1-event.P].RivalCardsShattered++
			}
		case engine.EvDamage:
			// P é a vítima. Só credita o oponente quando o dano tem AUTOR:
			// perdas de sistema (Fadiga, Ruptura do Véu) não são "dano
			// causado", e dano de carta do próprio jogador (Trono de
			// Espinhos, Pacto do Limiar) tampouco — a instância "pN-…"
			// identifica o dono determinístico (auditoria pós-0.8.1).
			if event.P != 0 && event.P != 1 {
				continue
			}
			if event.S == "Fadiga" || event.S == "Ruptura do Véu" || event.S == "Pressão de Nythara" {
				continue
			}
			if strings.HasPrefix(event.Card, fmt.Sprintf("p%d-", event.P)) {
				continue // auto-infligido por carta própria
			}
			stats[1-event.P].DamageDealt += event.N
		}
	}
	return stats
}

// ritualProgressFor traduz as métricas da partida no incremento de um ritual.
func ritualProgressFor(def RitualDef, stats MatchStats, won, pvp bool) int {
	if def.PvPOnly && !pvp {
		return 0
	}
	switch def.ID {
	case "win_pvp_1":
		if won {
			return 1
		}
	case "play_matches_2":
		return 1
	case "play_ritos_5":
		return stats.RitosResolved
	case "play_guards_3":
		return stats.GuardsPlayed
	case "deal_damage_25":
		return stats.DamageDealt
	case "win_confronts_5":
		return stats.ConfrontsWon
	case "shatter_rival_4":
		return stats.RivalCardsShattered
	case "play_assaults_6":
		return stats.AssaultsPlayed
	}
	return 0
}

// MasteryXPFor calcula o XP de maestria de uma partida PvP humana.
// Treino, tutorial, bot e modos desconhecidos falham fechados com zero XP.
func MasteryXPFor(won, pvp bool) int {
	if !pvp {
		return 0
	}
	xp := 15
	if won {
		xp += 15
	}
	return xp
}

// MasteryLevel converte XP em nível (curva quadrática suave, teto 50).
func MasteryLevel(xp int) int {
	level := 1
	for cost := 100; xp >= cost && level < 50; level++ {
		xp -= cost
		cost += 20
	}
	return level
}
