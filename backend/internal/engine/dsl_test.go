package engine_test

import (
	"encoding/json"
	"strings"
	"testing"

	"veurubro/backend/internal/engine"
)

// O validador da DSL precisa rejeitar definições malformadas no boot.
func TestDSLValidatorRejections(t *testing.T) {
	base := func(cards, unsupported string) []byte {
		// Constrói um arquivo cobrindo as 80 cartas: as citadas em `cards`,
		// o resto vai para unsupported com motivo genérico.
		var sb strings.Builder
		sb.WriteString(`{"version":"test","cards":{`)
		sb.WriteString(cards)
		sb.WriteString(`},"unsupported":{`)
		first := unsupported == ""
		if unsupported != "" {
			sb.WriteString(unsupported)
		}
		var cardsObj map[string]json.RawMessage
		_ = json.Unmarshal([]byte("{"+cards+"}"), &cardsObj)
		var unsupObj map[string]json.RawMessage
		_ = json.Unmarshal([]byte("{"+unsupported+"}"), &unsupObj)
		for _, c := range engine.CardList {
			if _, ok := cardsObj[c.ID]; ok {
				continue
			}
			if _, ok := unsupObj[c.ID]; ok {
				continue
			}
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(`"` + c.ID + `":"motivo de teste"`)
		}
		sb.WriteString("}}")
		return []byte(sb.String())
	}

	cases := []struct {
		name    string
		raw     []byte
		wantErr string
	}{
		{
			"op desconhecida",
			base(`"VR-015":{"rite":{"steps":[{"op":"explodir_tudo"}]}}`, ""),
			"operação desconhecida",
		},
		{
			"cobertura incompleta",
			[]byte(`{"version":"test","cards":{},"unsupported":{}}`),
			"ausente de cards e de unsupported",
		},
		{
			"carta nas duas listas",
			base(`"VR-015":{"rite":{"steps":[]}}`, `"VR-015":"duplicada"`),
			"presente em cards e unsupported",
		},
		{
			"unsupported sem motivo",
			base("", `"VR-015":""`),
			"unsupported sem motivo",
		},
		{
			"seção não bate com o tipo",
			base(`"VR-015":{"guard":{"prevent":2}}`, ""),
			"seção guard em carta Rito",
		},
		{
			"três decisões encadeadas",
			base(`"VR-015":{"rite":{"steps":[{"op":"choose_discard","n":1,"then":[{"op":"choose_discard","n":1,"then":[{"op":"choose_discard","n":1}]}]}]}}`, ""),
			"três decisões encadeadas",
		},
		{
			"gatilho de sigilo que emite sem trava",
			base(`"VR-040":{"permanent":{"triggers":[{"on":"sigil_emitted","do":[{"op":"emit_sigil","sigil":"Garra"}]}]}}`, ""),
			"exige once_per_round",
		},
		{
			"condição desconhecida",
			base(`"VR-013":{"assault":{"damage":2,"bonuses":[{"if":{"cond":"fase_da_lua"},"add":1}]}}`, ""),
			"condição desconhecida",
		},
		{
			"dano negativo",
			base(`"VR-013":{"assault":{"damage":-2}}`, ""),
			"dano fora do intervalo",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := engine.LoadEffectsForTest(tc.raw)
			if err == nil {
				t.Fatalf("definição inválida aceita")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("erro %q não contém %q", err, tc.wantErr)
			}
		})
	}
}

func TestDSLCoversAllCardsWithReasons(t *testing.T) {
	r := engine.ImplementationReport()
	if got := len(r.ImplementedCards); got < 60 {
		t.Fatalf("cartas na DSL: %d; esperado ≥60", got)
	}
	for _, id := range r.MissingCards {
		if r.MissingReasons[id] == "" {
			t.Errorf("%s sem motivo de não-suporte", id)
		}
	}
	t.Logf("DSL %s: %d/80 cartas funcionais; %d não suportadas com motivo",
		r.EffectsVersion, len(r.ImplementedCards), len(r.MissingCards))
}
