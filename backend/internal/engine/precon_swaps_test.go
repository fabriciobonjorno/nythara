package engine

import "testing"

// ADR-031: a cirurgia de precon separa força de deck e de kit — Mara perde a lança
// premium VR-020 por um situacional; Ilyan (mesma facção) fica intacto.
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
	for id, want := range map[string]int{"VR-020": 0, "VR-076": 2} {
		if got := count(mara, id); got != want {
			t.Fatalf("Mara: %s x%d; esperado x%d", id, got, want)
		}
	}
	for id, want := range map[string]int{"VR-020": 2, "VR-076": 0} {
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
