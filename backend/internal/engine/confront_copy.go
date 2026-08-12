package engine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// O texto competitivo nasce da DSL aceita pelo Confronto. Assim, o catálogo
// nunca reaproveita uma promessa de Eclipse/Ressonância que o modo não executa.
func prepareConfrontCardCopy(card *CardDef, fx *CardFx, profile *ConfrontCardProfile, rules ConfrontRulesConfig) {
	keywords := map[string]bool{}
	parts := make([]string, 0, 4)
	switch card.Type {
	case TypeAssalto:
		profile.Role = "IMPACTO"
		power := confrontRangeLabel("Poder", profile.Power, profile.PowerMax)
		parts = append(parts, fmt.Sprintf("Declare este Assalto com %s", power))
		keywords["ASSALTO"] = true
		if !profile.Defendable {
			profile.Role = "FINALIZADOR"
			keywords["IMPLACÁVEL"] = true
			parts = append(parts, "Implacável — este Assalto não abre uma janela de Guarda")
		} else {
			parts = append(parts, "o rival pode responder com uma Guarda")
		}
		if fx != nil && fx.Assault != nil {
			for _, bonus := range fx.Assault.Bonuses {
				if confrontCondAllowed(bonus.If) && !confrontBonusGuaranteed(card.Cost, bonus) {
					parts = append(parts, confrontBonusCopy(bonus, "Poder"))
					keywords["CONDIÇÃO"] = true
				}
			}
			parts = append(parts, confrontOpsCopy(fx.Assault.After, keywords)...)
		}
	case TypeGuarda:
		profile.Role = "RESPOSTA"
		keywords["GUARDA"] = true
		if profile.PreventAll {
			profile.Role = "BLOQUEIO TOTAL"
			keywords["PREVENÇÃO TOTAL"] = true
			parts = append(parts, "Resposta — previna todo o dano do Assalto em confronto")
		} else {
			parts = append(parts, fmt.Sprintf("Resposta — %s do Assalto em confronto",
				strings.ToLower(confrontRangeLabel("Previna", profile.Prevention, profile.PreventionMax))))
		}
		if fx != nil && fx.Guard != nil {
			if fx.Guard.CounterRite {
				profile.Role = "INTERDIÇÃO"
				keywords["SELO DO RITO"] = true
				parts = append(parts, "aplique Selo do Rito ao atacante; a fase de Rito dele após este confronto é pulada")
			}
			for _, bonus := range fx.Guard.Bonuses {
				if confrontCondAllowed(bonus.If) && !confrontBonusGuaranteed(card.Cost, bonus) {
					parts = append(parts, confrontBonusCopy(bonus, "Prevenção"))
					keywords["CONDIÇÃO"] = true
				}
			}
			parts = append(parts, confrontOpsCopy(fx.Guard.OnPlay, keywords)...)
			parts = append(parts, confrontOpsCopy(fx.Guard.After, keywords)...)
		}
	case TypeRito:
		profile.Role = "TÁTICA"
		keywords["RITO"] = true
		if fx != nil && fx.Rite != nil {
			parts = append(parts, confrontOpsCopy(fx.Rite.Steps, keywords)...)
			profile.Role = confrontRiteRole(fx.Rite.Steps)
		}
	}
	profile.TacticalText = confrontSentences(parts)
	if rules.ExposedPowerBonus > 0 {
		profile.TacticalText = strings.ReplaceAll(profile.TacticalText,
			"a Guarda é proibida", fmt.Sprintf("a Guarda é proibida e o Assalto recebe +%d Poder", rules.ExposedPowerBonus))
	}
	profile.Keywords = make([]string, 0, len(keywords))
	for keyword := range keywords {
		profile.Keywords = append(profile.Keywords, keyword)
	}
	sort.Strings(profile.Keywords)
}

func confrontAssaultPowerRange(card *CardDef, fx *AssaultFx, globalBonus int) (int, int) {
	minimum, maximum := confrontBonusRange(card.Cost, fx.Damage, fx.Bonuses)
	instances := max(1, fx.Instances)
	return minimum*instances + globalBonus, maximum*instances + globalBonus
}

func confrontBonusRange(cost, base int, bonuses []Bonus) (int, int) {
	minimum, maximum := base, base
	for _, bonus := range bonuses {
		if !confrontCondAllowed(bonus.If) {
			continue
		}
		guaranteed := confrontBonusGuaranteed(cost, bonus)
		apply := func(value int) int {
			if bonus.Instead != 0 {
				return bonus.Instead
			}
			return value + bonus.Add
		}
		if guaranteed {
			minimum, maximum = apply(minimum), apply(maximum)
			continue
		}
		minimum = min(minimum, apply(minimum))
		maximum = max(maximum, apply(maximum))
	}
	return minimum, maximum
}

func confrontBonusGuaranteed(cost int, bonus Bonus) bool {
	return bonus.If.Cond == "always" || (bonus.If.Cond == "sacrificed_this_round" && cost > 0)
}

func confrontRangeLabel(label string, minimum, maximum int) string {
	if maximum > minimum {
		return fmt.Sprintf("%s %d–%d", label, minimum, maximum)
	}
	return fmt.Sprintf("%s %d", label, minimum)
}

func confrontBonusCopy(bonus Bonus, stat string) string {
	condition := confrontCondCopy(bonus.If)
	if bonus.Instead != 0 {
		return fmt.Sprintf("se %s, %s passa a %d", condition, strings.ToLower(stat), bonus.Instead)
	}
	verb := "recebe"
	value := fmt.Sprintf("+%d", bonus.Add)
	if bonus.Add < 0 {
		verb = "perde"
		value = fmt.Sprintf("%d", abs(bonus.Add))
	}
	return fmt.Sprintf("se %s, %s %s %s", condition, strings.ToLower(stat), verb, value)
}

func confrontCondCopy(cond Cond) string {
	switch cond.Cond {
	case "always":
		return "esta carta resolver"
	case "sacrificed_this_round":
		return "você já pagou Vitalidade neste turno"
	case "took_damage_round":
		return "você sofreu dano neste turno"
	case "hand_less_than_opp":
		return "você tem menos cartas na mão que o rival"
	case "hand_ge":
		return fmt.Sprintf("você tem ao menos %d cartas na mão", cond.N)
	case "opp_hand_ge":
		return fmt.Sprintf("o rival tem ao menos %d cartas na mão", cond.N)
	case "opp_vitality_le":
		return fmt.Sprintf("o rival tem %d ou menos de Vitalidade", cond.N)
	case "prevented_all":
		return "esta Guarda preveniu todo o dano"
	case "opp_veiled":
		return "o rival está sob Véu"
	case "discard_ge":
		return fmt.Sprintf("seu descarte tem ao menos %d cartas", cond.N)
	case "opp_assaults_eq":
		return fmt.Sprintf("o rival declarou exatamente %d Assalto(s) neste turno", cond.N)
	case "own_assaults_eq":
		return fmt.Sprintf("você declarou exatamente %d Assalto(s) neste turno", cond.N)
	case "opp_discard_nonempty":
		return "o descarte rival não está vazio"
	default:
		return "a condição indicada for satisfeita"
	}
}

func confrontOpsCopy(ops []Op, keywords map[string]bool) []string {
	parts := make([]string, 0, len(ops))
	for i := range ops {
		op := &ops[i]
		target := "você"
		if op.Who == "opponent" {
			target = "o rival"
		}
		switch op.Op {
		case "damage":
			keywords["DANO"] = true
			parts = append(parts, fmt.Sprintf("cause %d de dano %s", op.N, confrontToTarget(target)))
		case "lose_vitality":
			keywords["PERDA DIRETA"] = true
			parts = append(parts, fmt.Sprintf("%s perde %d de Vitalidade", target, op.N))
		case "heal":
			keywords["CURA"] = true
			parts = append(parts, fmt.Sprintf("%s recupera %d de Vitalidade", target, op.N))
		case "draw":
			keywords["COMPRA"] = true
			parts = append(parts, fmt.Sprintf("%s compra %s", target, confrontCardsLabel(op.N)))
		case "choose_discard":
			keywords["ESCOLHA"] = true
			inner := strings.Join(confrontOpsCopy(op.Then, keywords), "; ")
			line := fmt.Sprintf("descarte %s à sua escolha", confrontCardsLabel(op.N))
			if inner != "" {
				line += "; depois, " + inner
			}
			parts = append(parts, line)
		case "both_draw":
			keywords["COMPRA"] = true
			parts = append(parts, "cada duelista compra 1 carta")
		case "ward":
			keywords["WARD"] = true
			parts = append(parts, fmt.Sprintf("%s recebe %d de Ward", target, op.N))
		case "status":
			parts = append(parts, confrontStatusCopy(op, target, keywords))
		case "remove_own_bleeds":
			keywords["PURIFICAÇÃO"] = true
			parts = append(parts, "remova todos os seus Sangramentos")
		case "remove_own_curse":
			keywords["PURIFICAÇÃO"] = true
			parts = append(parts, "remova uma Maldição sua")
		case "remove_curse_smart":
			keywords["PURIFICAÇÃO"] = true
			parts = append(parts, "remova uma Maldição sua; se você não tiver nenhuma, remova uma do rival e compre 1 carta")
		case "remove_veil_expose":
			keywords["SELO DA GUARDA"] = true
			parts = append(parts, "remova o Véu do rival e aplique Exposto; no próximo Assalto contra ele, a Guarda é proibida")
		case "reveal_lock_assault":
			keywords["SELO DO ASSALTO"] = true
			parts = append(parts, "aplique Selo do Assalto ao rival; a próxima fase de Assalto dele é pulada")
		case "conditional":
			if op.If == nil || !confrontCondAllowed(*op.If) {
				continue
			}
			thenText := strings.Join(confrontOpsCopy(op.Then, keywords), "; ")
			elseText := strings.Join(confrontOpsCopy(op.Else, keywords), "; ")
			if thenText == "" {
				continue
			}
			copy := fmt.Sprintf("se %s, %s", confrontCondCopy(*op.If), thenText)
			if elseText != "" {
				copy += "; caso contrário, " + elseText
			}
			parts = append(parts, copy)
		}
	}
	return parts
}

func confrontStatusCopy(op *Op, target string, keywords map[string]bool) string {
	switch strings.ToLower(op.Status) {
	case "exposto":
		keywords["SELO DA GUARDA"] = true
		return fmt.Sprintf("aplique Exposto %s; no próximo Assalto contra esse duelista, a Guarda é proibida", confrontToTarget(target))
	case "veu":
		keywords["VÉU"] = true
		return fmt.Sprintf("%s recebe Véu até o fim do próximo turno e não pode ser alvo de Ritos rivais", target)
	case "sangramento":
		keywords["SANGRAMENTO"] = true
		return fmt.Sprintf("aplique Sangramento %d %s; ele dispara no início do próximo turno", op.N, confrontToTarget(target))
	case "maldicao":
		keywords["MALDIÇÃO"] = true
		return fmt.Sprintf("aplique Maldição %d %s; ela dispara no início do próximo turno", max(1, op.N), confrontToTarget(target))
	default:
		return fmt.Sprintf("aplique %s %s", op.Status, confrontToTarget(target))
	}
}

func confrontToTarget(target string) string {
	if target == "o rival" {
		return "ao rival"
	}
	return "a você"
}

func confrontCardsLabel(amount int) string {
	if amount == 1 {
		return "1 carta"
	}
	return fmt.Sprintf("%d cartas", amount)
}

func confrontRiteRole(ops []Op) string {
	role := "TÁTICA"
	confrontWalkOps(ops, func(op *Op) {
		switch op.Op {
		case "reveal_lock_assault", "remove_veil_expose":
			role = "CONTROLE"
		case "damage", "lose_vitality":
			if role == "TÁTICA" {
				role = "PRESSÃO"
			}
		case "heal", "ward", "remove_own_bleeds", "remove_own_curse", "remove_curse_smart":
			if role == "TÁTICA" {
				role = "SUSTENTAÇÃO"
			}
		case "draw", "both_draw", "choose_discard":
			if role == "TÁTICA" {
				role = "RECURSO"
			}
		case "status":
			if strings.EqualFold(op.Status, "exposto") {
				role = "CONTROLE"
			}
		}
	})
	return role
}

func confrontHasOp(ops []Op, kind string) bool {
	found := false
	confrontWalkOps(ops, func(op *Op) {
		if op.Op == kind {
			found = true
		}
	})
	return found
}

func confrontWalkOps(ops []Op, visit func(*Op)) {
	for i := range ops {
		visit(&ops[i])
		confrontWalkOps(ops[i].Then, visit)
		confrontWalkOps(ops[i].Else, visit)
	}
}

func confrontSentences(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = capitalize(part)
		part = strings.TrimRight(part, ".") + "."
		out = append(out, part)
	}
	return strings.Join(out, " ")
}

func capitalize(value string) string {
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
