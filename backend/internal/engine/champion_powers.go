package engine

import "fmt"

// Poderes de Avatar no Modo Confronto (ADR-057).
//
// Os poderes históricos dependiam de Essência, Eclipse e Ressonância — sistemas
// que o Confronto removeu —, então o modo ficou com dez Avatares puramente
// cosméticos. Aqui eles voltam usando só o que este modo tem: custo em
// Vitalidade, janelas de Assalto/Guarda, Ward, estilhaço e Fadiga.
//
// A avaliação é dirigida por dados (gatilho + efeito + grandeza) e vive na
// configuração do ruleset, então cada versão publica seus próprios poderes e
// nenhuma versão anterior muda de significado. Todo poder que dispara emite
// evento: um efeito invisível é indistinguível de bug.

const championPowerEvent = "Poder de Avatar"

// championPower devolve o poder do Avatar do jogador nesta versão de regras.
func (g *Game) championPower(player int) (ChampionPower, bool) {
	powers := g.rs.ConfrontRules.ChampionPowers
	if len(powers) == 0 {
		return ChampionPower{}, false
	}
	power, ok := powers[g.s.Players[player].Champion]
	return power, ok
}

// championDiscountFor reduz o custo de uma carta quando o gatilho do Avatar
// bate. Recebe o ID porque o pagamento é o ponto único por onde todo custo
// passa, e é lá que o desconto precisa valer.
func (g *Game) championDiscountFor(player int, cardID string) int {
	card := g.rs.Cards[cardID]
	if card == nil {
		return 0
	}
	return g.championDiscount(player, card)
}

func (g *Game) championDiscount(player int, card *CardDef) int {
	power, ok := g.championPower(player)
	if !ok || power.Effect != "discount" || power.N <= 0 {
		return 0
	}
	p := g.s.Players[player]
	switch power.Trigger {
	case "first_play_of_round":
		if p.AssaultsRound+p.GuardsRound+p.RitesRound > 0 {
			return 0
		}
	case "first_guard_of_round":
		if card.Type != TypeGuarda || p.GuardsRound > 0 {
			return 0
		}
	case "low_vitality":
		if card.Type != TypeAssalto || p.Vitality > power.At {
			return 0
		}
	case "after_guard":
		if card.Type != TypeAssalto || p.GuardsRound == 0 {
			return 0
		}
	default:
		return 0
	}
	return min(power.N, card.Cost)
}

// championOnTrigger aplica os poderes que reagem a um acontecimento da mesa.
func (g *Game) championOnTrigger(player int, trigger string) {
	power, ok := g.championPower(player)
	if !ok || power.Trigger != trigger || power.N <= 0 {
		return
	}
	p := g.s.Players[player]
	// O limiar do gatilho de resiliência é verificado depois do dano: o poder
	// existe para segurar quem está caindo, não para quem está inteiro.
	if trigger == "low_vitality" && p.Vitality > power.At {
		return
	}
	switch power.Effect {
	case "ward":
		p.Ward += power.N
		g.emit(Event{Kind: EvWardGained, P: player, N: power.N, S: championPowerEvent})
	case "heal":
		before := p.Vitality
		p.Vitality = min(p.MaxVitality, p.Vitality+power.N)
		if p.Vitality == before {
			return
		}
		g.emit(Event{Kind: EvHealed, P: player, N: p.Vitality - before, From: before, To: p.Vitality, S: championPowerEvent})
	case "draw":
		for range power.N {
			g.drawConfront(player)
		}
	default:
		return
	}
}

// championFatigueRelief amortece o crescimento da Fadiga. O piso de 1 impede
// que um Avatar transforme o esgotamento do baralho em recurso infinito.
func (g *Game) championFatigueRelief(player int) int {
	power, ok := g.championPower(player)
	if !ok || power.Trigger != "deck_out" || power.Effect != "fatigue_relief" {
		return 0
	}
	return max(0, power.N)
}

// championOpeningDraw devolve as cartas extras que o Avatar concede na abertura.
func (g *Game) championOpeningDraw(player int) int {
	power, ok := g.championPower(player)
	if !ok || power.Trigger != "opening" || power.Effect != "draw" {
		return 0
	}
	return max(0, power.N)
}

// validateChampionPowers rejeita vocabulário desconhecido no boot: um poder
// escrito errado seria silenciosamente ignorado e ninguém perceberia.
func validateChampionPowers(rs *Ruleset) error {
	for id, power := range rs.ConfrontRules.ChampionPowers {
		if rs.Champions[id] == nil {
			return fmt.Errorf("poder de Avatar para campeão inexistente: %s", id)
		}
		if !ChampionTriggers[power.Trigger] {
			return fmt.Errorf("%s: gatilho de Avatar desconhecido: %q", id, power.Trigger)
		}
		if !ChampionEffects[power.Effect] {
			return fmt.Errorf("%s: efeito de Avatar desconhecido: %q", id, power.Effect)
		}
		if power.Text == "" {
			return fmt.Errorf("%s: poder de Avatar sem texto para o jogador", id)
		}
	}
	return nil
}
