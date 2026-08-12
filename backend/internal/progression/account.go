// Package progression concentra as regras de progressão global da conta.
// O nível é sempre derivado do XP authoritative; nenhum cliente grava nível.
package progression

import "veurubro/backend/internal/domain"

const (
	MinAccountLevel  = 1
	MaxAccountLevel  = 50
	MaxMatchLevelGap = 5

	firstLevelCost = 100
	levelCostStep  = 20

	// MaxAccountXP é a soma dos custos do nível 1 ao 50. XP adicional é
	// descartado para manter o teto também na persistência.
	MaxAccountXP = 28_420
	// MaxAccountXPPerMatch limita a entrada interna mesmo antes do CHECK do
	// banco. O maior crédito possível é uma vitória PvP.
	MaxAccountXPPerMatch = 30
)

// XPForMatch devolve XP somente a partir do resultado authoritative.
func XPForMatch(won, pvp bool) int {
	if !pvp {
		return 0
	}
	xp := 15
	if won {
		xp += 15
	}
	return xp
}

// LevelForXP deriva um nível entre 1 e 50, mesmo diante de valor corrompido.
func LevelForXP(xp int) int {
	return AccountForXP(xp).Level
}

// AccountForXP normaliza o XP e calcula a posição dentro do nível corrente.
func AccountForXP(xp int) domain.AccountProgress {
	if xp < 0 {
		xp = 0
	}
	if xp > MaxAccountXP {
		xp = MaxAccountXP
	}
	account := domain.AccountProgress{XP: xp, Level: MinAccountLevel, MaxLevel: MaxAccountLevel}
	remaining := xp
	for account.Level < MaxAccountLevel {
		cost := levelCost(account.Level)
		if remaining < cost {
			account.LevelXP = remaining
			account.LevelXPRequired = cost
			return account
		}
		remaining -= cost
		account.Level++
	}
	account.LevelXP = 0
	account.LevelXPRequired = 0
	return account
}

func levelCost(level int) int {
	return firstLevelCost + (level-MinAccountLevel)*levelCostStep
}

// LegendaryUnlockLevel define a trilha editorial das oito Lendárias do
// catálogo Alpha. IDs futuros ficam protegidos no nível máximo até receberem
// uma decisão explícita de economia.
func LegendaryUnlockLevel(cardID string) int {
	if level, ok := legendaryUnlockLevels[cardID]; ok {
		return level
	}
	return MaxAccountLevel
}

func LegendaryUnlockLevels() map[string]int {
	out := make(map[string]int, len(legendaryUnlockLevels))
	for id, level := range legendaryUnlockLevels {
		out[id] = level
	}
	return out
}

var legendaryUnlockLevels = map[string]int{
	"VR-012": 10,
	"VR-024": 15,
	"VR-036": 20,
	"VR-048": 25,
	"VR-060": 30,
	"VR-079": 35,
	"VR-080": 40,
	"VR-130": 50,
}

func LevelsCanMatch(left, right int) bool {
	if left < MinAccountLevel || left > MaxAccountLevel ||
		right < MinAccountLevel || right > MaxAccountLevel {
		return false
	}
	difference := left - right
	if difference < 0 {
		difference = -difference
	}
	return difference <= MaxMatchLevelGap
}
