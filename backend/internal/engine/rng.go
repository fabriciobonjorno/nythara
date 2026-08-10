package engine

// RNG é um gerador determinístico (SplitMix64) independente de math/rand,
// para que a mesma seed produza a mesma partida em qualquer versão de Go.
type RNG struct {
	State uint64 `json:"state"`
}

func NewRNG(seed uint64) *RNG {
	return &RNG{State: seed}
}

func (r *RNG) next() uint64 {
	r.State += 0x9E3779B97F4A7C15
	z := r.State
	z ^= z >> 30
	z *= 0xBF58476D1CE4E5B9
	z ^= z >> 27
	z *= 0x94D049BB133111EB
	z ^= z >> 31
	return z
}

// Intn retorna um inteiro em [0, n). n deve ser > 0.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		panic("engine: RNG.Intn com n <= 0")
	}
	return int(r.next() % uint64(n))
}

// Shuffle embaralha n elementos via Fisher-Yates.
func (r *RNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
}
