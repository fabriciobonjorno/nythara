package main

// Reproduz um jogo do sim variado e encontra o primeiro comando após o qual
// as zonas ficam inválidas (mesma matemática de seed do harness).

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/sim"
)

func main() {
	index := flag.Int("game", 0, "índice do jogo")
	base := flag.Uint64("seed", 1, "seed base")
	cap := flag.Int("cap", 2000, "teto de comandos")
	flag.Parse()

	rs, err := engine.RulesetByVersion(engine.RulesetVersion)
	if err != nil {
		panic(err)
	}
	var champions []string
	for id := range rs.Champions {
		champions = append(champions, id)
	}
	sort.Strings(champions)
	pair := *index % (len(champions) * len(champions))
	repetition := *index / (len(champions) * len(champions))
	c0, c1 := champions[pair/len(champions)], champions[pair%len(champions)]
	first := repetition % 2
	seed := *base + uint64(*index)*0x9e3779b97f4a7c15
	deck0 := sim.VariedDeckFor(rs, c0, engine.NewRNG(seed^0xDEC0DEC0DEC0DEC0))
	deck1 := sim.VariedDeckFor(rs, c1, engine.NewRNG(seed^0x0DECADE0DECADE0D))
	fmt.Println("jogo", *index, c0, "x", c1, "first", first)

	game, err := engine.NewGame(engine.Config{RulesetVersion: engine.RulesetVersion, Seed: seed,
		FirstPlayer: first, Players: [2]engine.PlayerSetup{{ChampionID: c0, Deck: deck0}, {ChampionID: c1, Deck: deck1}}})
	if err != nil {
		panic(err)
	}
	bots := [2]engine.PlayerBot{&engine.HeuristicBot{RNG: engine.NewRNG(seed ^ 0xa5a5a5a5)},
		&engine.HeuristicBot{RNG: engine.NewRNG(seed ^ 0x5a5a5a5a)}}
	for step := 0; step < *cap; step++ {
		if game.State().Over {
			fmt.Println("acabou limpo na rodada", game.State().Round)
			return
		}
		actor, ok := engine.RequiredPlayer(game)
		if !ok {
			fmt.Println("sem ator")
			return
		}
		command, ok := bots[actor].NextFor(game, actor)
		if !ok {
			fmt.Println("bot travou")
			return
		}
		var pend string
		if d := game.State().Pending; d != nil {
			pend = fmt.Sprintf(" [decisão %s de %s opções %v]", d.Kind, d.Source, d.Options)
		}
		if _, err := game.Apply(command); err != nil {
			fmt.Printf("REJEITADO no passo %d rodada %d\ncomando: %+v%s\nerro: %v\n",
				step, game.State().Round, command, pend, err)
			if d := game.State().Pending; d != nil {
				fmt.Printf("pendente: kind=%s id=%d source=%s card=%s n=%d min=%d options=%v\n",
					d.Kind, d.ID, d.Source, d.Card, d.N, d.MinN, d.Options)
			}
			log := game.Log
			for i := max(0, len(log)-20); i < len(log); i++ {
				fmt.Printf("  ev %d %s p%d card=%s def=%s n=%d s=%s\n", log[i].Seq, log[i].Kind,
					log[i].P, log[i].Card, log[i].Def, log[i].N, log[i].S)
			}
			return
		}
		if zerr := sim.ValidateZonesFor(game.State()); zerr != nil {
			fmt.Printf("CORROMPEU no passo %d rodada %d\ncomando: %+v%s\nerro: %v\n",
				step, game.State().Round, command, pend, zerr)
			if inst := game.State().Cards[command.Card]; inst != nil {
				fmt.Println("carta do comando:", inst.Def, game.Ruleset().Cards[inst.Def].Name)
			}
			// últimos eventos para contexto
			log := game.Log
			for i := max(0, len(log)-12); i < len(log); i++ {
				fmt.Printf("  ev %d %s p%d card=%s def=%s n=%d s=%s\n", log[i].Seq, log[i].Kind, log[i].P, log[i].Card, log[i].Def, log[i].N, log[i].S)
			}
			os.Exit(2)
		}
	}
	fmt.Println("teto sem corrupção; rodada", game.State().Round)
}
