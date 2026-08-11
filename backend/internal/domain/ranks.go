package domain

// Patentes ranqueadas e títulos de maestria — derivados puros de valores já
// persistidos (rating Elo, XP). Nada aqui cria estado novo: mudar faixas ou
// nomes é ajuste de apresentação competitiva, sem migração.

// RankTier é a patente da temporada derivada do rating.
type RankTier struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	MinRating int    `json:"min_rating"`
	// NextAt é o rating que promove à próxima patente; 0 no topo.
	NextAt int `json:"next_at,omitempty"`
}

// rankTiers em ordem crescente. Nomes 100% originais do universo do jogo.
var rankTiers = []RankTier{
	{Key: "errante", Name: "Errante do Crepúsculo", MinRating: 0},
	{Key: "iniciado", Name: "Iniciado do Véu", MinRating: 900},
	{Key: "lamina", Name: "Lâmina Velada", MinRating: 1000},
	{Key: "guardiao", Name: "Guardião do Limiar", MinRating: 1100},
	{Key: "arauto", Name: "Arauto do Eclipse", MinRating: 1200},
	{Key: "soberano", Name: "Soberano Crepuscular", MinRating: 1350},
	{Key: "voz", Name: "Voz do Eclipse", MinRating: 1500},
}

// TierForRating devolve a patente do rating, com o limiar da próxima.
func TierForRating(rating int) RankTier {
	tier := rankTiers[0]
	for _, candidate := range rankTiers {
		if rating >= candidate.MinRating {
			tier = candidate
		}
	}
	if tier.MinRating < rankTiers[len(rankTiers)-1].MinRating {
		for _, candidate := range rankTiers {
			if candidate.MinRating > tier.MinRating {
				tier.NextAt = candidate.MinRating
				break
			}
		}
	}
	return tier
}

// seasonRewards são os Fragmentos do Véu concedidos pela patente FINAL ao
// fechar uma temporada (ADR-034). Derivado puro, como as patentes: ajustar
// valores é decisão de economia, sem migração.
var seasonRewards = map[string]int{
	"errante": 20, "iniciado": 40, "lamina": 60, "guardiao": 90,
	"arauto": 130, "soberano": 180, "voz": 250,
}

// SeasonRewardForRating devolve a patente final e a recompensa do rating.
func SeasonRewardForRating(rating int) (RankTier, int) {
	tier := TierForRating(rating)
	return tier, seasonRewards[tier.Key]
}

// MasteryTitle é o título exibido pelo nível de maestria com um Campeão.
func MasteryTitle(level int) string {
	switch {
	case level >= 50:
		return "Encarnação do Véu"
	case level >= 35:
		return "Avatar"
	case level >= 20:
		return "Mestre"
	case level >= 10:
		return "Discípulo"
	case level >= 5:
		return "Adepto"
	default:
		return "Aprendiz"
	}
}
