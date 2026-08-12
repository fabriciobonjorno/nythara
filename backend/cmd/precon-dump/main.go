package main

import (
	"fmt"

	"veurubro/backend/internal/engine"
)

func main() {
	rs := engine.CompetitiveRuleset()
	deck, err := rs.PreconstructedDeck("CH-SO-01")
	if err != nil {
		panic(err)
	}
	counts := map[string]int{}
	var order []string
	for _, id := range deck {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}
	fmt.Println("== Baralho inicial Confronto ==")
	for _, id := range order {
		c := rs.Cards[id]
		fmt.Printf("%s x%d  c%d %-8s P%d G%d %-5t %s\n", id, counts[id], c.Cost, c.Type,
			c.Confront.Power, c.Confront.Prevention, c.Confront.PreventAll, c.Name)
	}
	fmt.Println("\n== Pool legal fora do inicial ==")
	for _, c := range rs.CardList {
		if c.Confront != nil && c.Confront.Legal && counts[c.ID] == 0 {
			fmt.Printf("%s  c%d %-8s P%d G%d %-5t %s\n", c.ID, c.Cost, c.Type,
				c.Confront.Power, c.Confront.Prevention, c.Confront.PreventAll, c.Name)
		}
	}
}
