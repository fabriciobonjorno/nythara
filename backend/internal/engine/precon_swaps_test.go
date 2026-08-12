package engine

import "testing"

// ADR-031: a cirurgia de precon separa força de deck e de kit — Mara perde a lança
// premium VR-020 por uma Guarda situacional (alpha-0.8.1: o Relógio ficava
// 63% morto na mão); Ilyan (mesma facção) fica intacto.
func TestPreconSwapsDifferentiateTwins(t *testing.T) {
	rs := Builtin()
	mara, err := rs.PreconstructedDeck("CH-SO-01")
	if err != nil {
		t.Fatal(err)
	}
	ilyan, err := rs.PreconstructedDeck("CH-SO-02")
	if err != nil {
		t.Fatal(err)
	}
	count := func(deck []string, id string) int {
		n := 0
		for _, card := range deck {
			if card == id {
				n++
			}
		}
		return n
	}
	for id, want := range map[string]int{"VR-020": 0, "VR-075": 2, "VR-014": 0, "VR-071": 2} {
		if got := count(mara, id); got != want {
			t.Fatalf("Mara: %s x%d; esperado x%d", id, got, want)
		}
	}
	for id, want := range map[string]int{"VR-020": 2, "VR-075": 0, "VR-014": 2, "VR-071": 0} {
		if got := count(ilyan, id); got != want {
			t.Fatalf("Ilyan: %s x%d; esperado x%d", id, got, want)
		}
	}
	if len(mara) != DeckSize {
		t.Fatalf("precon da Mara com %d cartas", len(mara))
	}
	if err := rs.ValidateDeck("CH-SO-01", mara); err != nil {
		t.Fatalf("precon da Mara ilegal após a cirurgia: %v", err)
	}
}

// ADR-043: gêmeos de facção precisam de precons visivelmente distintos —
// cada par difere em pelo menos 6 cópias, e toda lista continua legal.
func TestTwinPreconsAreDistinct(t *testing.T) {
	rs := Builtin()
	twins := [][2]string{
		{"CH-VH-01", "CH-VH-02"}, {"CH-SO-01", "CH-SO-02"},
		{"CH-MI-01", "CH-MI-02"}, {"CH-VA-01", "CH-VA-02"},
		{"CH-CI-01", "CH-CI-02"},
	}
	counts := func(deck []string) map[string]int {
		out := map[string]int{}
		for _, id := range deck {
			out[id]++
		}
		return out
	}
	for _, pair := range twins {
		deckA, err := rs.PreconstructedDeck(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		deckB, err := rs.PreconstructedDeck(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if err := rs.ValidateDeck(pair[0], deckA); err != nil {
			t.Fatalf("%s ilegal: %v", pair[0], err)
		}
		if err := rs.ValidateDeck(pair[1], deckB); err != nil {
			t.Fatalf("%s ilegal: %v", pair[1], err)
		}
		countA, countB := counts(deckA), counts(deckB)
		diff := 0
		seen := map[string]bool{}
		for id, n := range countA {
			seen[id] = true
			if d := n - countB[id]; d > 0 {
				diff += d
			} else {
				diff -= d
			}
		}
		for id, n := range countB {
			if !seen[id] {
				diff += n
			}
		}
		if diff < 6 {
			t.Fatalf("gêmeos %s×%s diferem em só %d cópias; exigido ≥6", pair[0], pair[1], diff)
		}
	}
}
