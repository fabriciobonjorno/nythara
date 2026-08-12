package engine

// Zonas de carta.
type Zone string

const (
	ZoneDeck    Zone = "deck"
	ZoneHand    Zone = "hand"
	ZoneDiscard Zone = "discard"
	ZoneExile   Zone = "exile"
	ZonePlay    Zone = "play"  // Relíquias e Manifestações ativas
	ZoneClash   Zone = "clash" // cartas expostas no centro do Modo Confronto
)

// Fases visíveis da rodada. Preparação, Compra e a resolução automática do
// Crepúsculo acontecem dentro das transições; PhaseTwilight só existe enquanto
// há descartes obrigatórios pendentes por limite de mão.
type Phase string

const (
	PhaseMulligan Phase = "mulligan"
	PhaseStance   Phase = "postura"
	PhaseRite     Phase = "rito"
	PhaseConfront Phase = "confronto"
	PhaseTwilight Phase = "crepusculo"
	PhaseAssault  Phase = "assalto"
	PhaseGuard    Phase = "guarda"
	PhaseOver     Phase = "fim"
)

// Posturas (reveladas simultaneamente).
type Stance string

const (
	StancePredacao Stance = "Predação"
	StanceVigilia  Stance = "Vigília"
	StanceArcano   Stance = "Arcano"
)

// Estado global do Eclipse quando o medidor atinge um extremo.
type EclipseTotal string

const (
	EclipseNone   EclipseTotal = ""
	EclipseNight  EclipseTotal = "Eclipse Noturno" // +3
	EclipseAurora EclipseTotal = "Aurora Total"    // -3
)

const (
	DeckSize            = 36
	ConfrontDeckSize    = 30
	ConfrontMinAssaults = 8
	ConfrontMinGuards   = 8
	ConfrontMinRites    = 4
	OpeningHandSize     = 5
	HandLimit           = 7
	EssenceStart        = 3
	EssenceMax          = 10 // alpha-0.5.0: teto maior acelera o meio de jogo
	EclipseMin          = -3
	EclipseMax          = 3
	ResonanceCap        = 5
	maxChainDepth       = 16
)

// ConfrontCtx é o confronto visível do alpha-0.9.x. Ele contém somente dados
// públicos: a mão continua oculta, mas as cartas colocadas no centro são
// reveladas para ambos os lados e para espectadores.
type ConfrontCtx struct {
	Attacker    int    `json:"attacker"`
	Defender    int    `json:"defender"`
	AssaultInst string `json:"assault_inst"`
	AssaultDef  string `json:"assault_def"`
	Power       int    `json:"power"`
	Defendable  bool   `json:"defendable"`
	GuardInst   string `json:"guard_inst,omitempty"`
	GuardDef    string `json:"guard_def,omitempty"`
	Prevention  int    `json:"prevention,omitempty"`
}

// CardInstance é uma cópia física de uma carta dentro da partida.
type CardInstance struct {
	ID          string `json:"id"`  // ex.: "p0-c07"
	Def         string `json:"def"` // ex.: "VR-001"
	Owner       int    `json:"owner"`
	Zone        Zone   `json:"zone"`
	UsedRound   int    `json:"used_round,omitempty"`   // gatilhos "uma vez por rodada" de permanentes
	LockedRound int    `json:"locked_round,omitempty"` // VR-018: não pode ser jogada nesta rodada
}

// CostMod é um modificador de custo pendente (postura, passivas, efeitos).
type CostMod struct {
	Type             CardType `json:"type,omitempty"`     // vazio = qualquer tipo
	Faction          string   `json:"faction,omitempty"`  // vazio = qualquer facção
	Instance         string   `json:"instance,omitempty"` // preso a uma instância específica
	Delta            int      `json:"delta"`
	Uses             int      `json:"uses"`
	Round            int      `json:"round,omitempty"`       // se >0, só vale nessa rodada
	UntilRound       int      `json:"until_round,omitempty"` // se >0, vale até essa rodada (inclusive)
	RequireNonDamage bool     `json:"require_non_damage,omitempty"`
	Source           string   `json:"source"`
}

// TimedN representa Sangramento/Maldição com valor e rodada de disparo.
type TimedN struct {
	N     int    `json:"n"`
	Round int    `json:"round"`
	Kind  string `json:"kind,omitempty"` // para Maldições: def da carta que aplicou
}

// PlayerState é o estado completo de um jogador.
type PlayerState struct {
	Champion    string `json:"champion"`
	Vitality    int    `json:"vitality"`
	MaxVitality int    `json:"max_vitality"`

	Essence     int `json:"essence"`
	EssenceCap  int `json:"essence_cap"`
	TempEssence int `json:"temp_essence"`

	Deck    []string `json:"deck"` // índice 0 = topo
	Hand    []string `json:"hand"`
	Discard []string `json:"discard"`
	Exile   []string `json:"exile"`
	Relics  []string `json:"relics"`
	Manifs  []string `json:"manifs"`

	Stance          Stance `json:"stance,omitempty"`
	StanceCommitted bool   `json:"stance_committed"`
	MulliganDone    bool   `json:"mulligan_done"`

	Fatigue int `json:"fatigue"`

	Trail     []Sigil `json:"trail"` // Trilha de Ressonância da rodada (própria)
	Ward      int     `json:"ward"`
	Exposto   bool    `json:"exposto"`
	VeilRound int     `json:"veil_round,omitempty"` // última rodada (inclusive) em que o Véu está ativo
	// Selos de Fase do alpha-0.10.x. O número é a última rodada global em
	// que a janela deve ser pulada; zero significa que não há Selo ativo.
	AssaultSealUntil int      `json:"assault_seal_until,omitempty"`
	RiteSealUntil    int      `json:"rite_seal_until,omitempty"`
	Bleeds           []TimedN `json:"bleeds,omitempty"`
	Curses           []TimedN `json:"curses,omitempty"`

	CostMods []CostMod `json:"cost_mods,omitempty"`

	// Contadores da rodada (zerados na Preparação).
	AssaultsRound    int  `json:"assaults_round"`
	GuardsRound      int  `json:"guards_round"`
	RitesRound       int  `json:"rites_round"`
	SacrificesRound  int  `json:"sacrifices_round"`
	DamageTakenRound int  `json:"damage_taken_round"`
	ExiledRound      bool `json:"exiled_round"`
	DrawsRound       int  `json:"draws_round"`
	RecoversRound    int  `json:"recovers_round"`
	UltimateUsed     bool `json:"ultimate_used"`

	// Flags de efeitos com escopo de rodada.
	LuaNoSangue           bool  `json:"lua_no_sangue,omitempty"`           // VR-045
	ExtraNeutralSigil     Sigil `json:"extra_neutral_sigil,omitempty"`     // VR-077
	RelicDestroyedRound   bool  `json:"relic_destroyed_round,omitempty"`   // VR-071
	ManifsSuppressedUntil int   `json:"manifs_suppressed_until,omitempty"` // VR-021 (rodada limite)

	// Estado para cartas avançadas e Campeões.
	LastDamageDealt  int      `json:"last_damage_dealt,omitempty"`  // último Assalto: dano líquido (rodada)
	NightShiftsRound int      `json:"night_shifts_round,omitempty"` // deslocamentos p/ Noite causados por este jogador
	NullifyOppShift  bool     `json:"nullify_opp_shift,omitempty"`  // ultimate de Ilyan armada
	SaelaShield      bool     `json:"saela_shield,omitempty"`       // ultimate de Saela: próximo dano → 0
	NextAssaultBonus int      `json:"next_assault_bonus,omitempty"` // ultimate de Saela: +2
	CinzaMarkers     int      `json:"cinza_markers,omitempty"`      // passiva de Edda (máx. 3)
	PendingReturns   []string `json:"pending_returns,omitempty"`    // VR-048: Assaltos que voltam à mão
	MirroredRelic    string   `json:"mirrored_relic,omitempty"`     // VR-032: def espelhada
	MirrorUntil      int      `json:"mirror_until,omitempty"`       // VR-032: última rodada válida
	MirrorUsedRound  int      `json:"mirror_used_round,omitempty"`  // 1x/rodada da cópia espelhada
	LastPlayed       string   `json:"last_played,omitempty"`        // última carta anunciada
	PrevPlayed       string   `json:"prev_played,omitempty"`        // a anterior (VR-030)
}

func veilActive(p *PlayerState, round int) bool {
	return p != nil && p.VeilRound >= round && p.VeilRound > 0
}

// DecisionKind enumera decisões pendentes tipadas (serializáveis, sem closures).
type DecisionKind string

const (
	DecDiscardN         DecisionKind = "discard_n"          // descartar N das opções
	DecPickTop2         DecisionKind = "pick_top2"          // 1 para a mão, outra para o fundo
	DecExileSelfHeal    DecisionKind = "exile_self_heal"    // sim/não (VR-051)
	DecReorderTop       DecisionKind = "reorder_top"        // permutação do topo do deck
	DecRecoverPick      DecisionKind = "recover_pick"       // N do descarte para a mão
	DecOppDiscardPick   DecisionKind = "opp_discard_pick"   // oponente escolhe o que você descarta
	DecMillTop          DecisionKind = "mill_top"           // sim/não: topo para o descarte
	DecSwapEclipse      DecisionKind = "swap_eclipse"       // sim/não: inverte o medidor
	DecDestroyRelicPick DecisionKind = "destroy_relic_pick" // escolhe Relíquia adversária
	DecTaxTypePick      DecisionKind = "tax_type_pick"      // escolhe carta no descarte rival
	DecPickSigil        DecisionKind = "pick_sigil"         // escolhe um Sigilo

	DecExilePick     DecisionKind = "exile_pick"     // exila N do próprio descarte
	DecReEmitSigils  DecisionKind = "re_emit_sigils" // VR-034: até N sigilos distintos
	DecRevealTax     DecisionKind = "reveal_tax"     // VR-006: +1 numa revelada
	DecLockAssault   DecisionKind = "lock_assault"   // VR-018: trava um Assalto
	DecDeclareType   DecisionKind = "declare_type"   // VR-029
	DecDirection     DecisionKind = "direction"      // VR-055: noite/aurora
	DecFormulaChoice DecisionKind = "formula_choice" // VR-060: dano/cura/compra
	DecMirrorRelic   DecisionKind = "mirror_relic"   // VR-032
	DecCopyPlayed    DecisionKind = "copy_played"    // VR-036
	DecScryBottom    DecisionKind = "scry_bottom"    // passiva de Nyra
	DecOrenChoice    DecisionKind = "oren_choice"    // ultimate de Oren
	DecVorenChoice   DecisionKind = "voren_choice"   // ultimate de Voren: cura/dano por carta
	DecEddaReturn    DecisionKind = "edda_return"    // ultimate de Edda
)

// Decision é uma escolha obrigatória pendente de um jogador. Then carrega os
// efeitos que continuam após a resolução (dados da DSL — serializável).
type Decision struct {
	ID      int          `json:"id"`
	Player  int          `json:"player"`
	Kind    DecisionKind `json:"kind"`
	Options []string     `json:"options"` // instance ids (ou "yes"/"no", ou rótulos)
	N       int          `json:"n"`
	MinN    int          `json:"min_n,omitempty"`  // 0 => exatamente N; senão intervalo [MinN..N]
	Card    string       `json:"card,omitempty"`   // instância de contexto
	Source  string       `json:"source,omitempty"` // def da carta fonte
	Then    []Op         `json:"then,omitempty"`
}

// GuardCtx é a janela de Guarda aberta por um Assalto ainda não resolvido.
type GuardCtx struct {
	Attacker       int    `json:"attacker"`
	Defender       int    `json:"defender"`
	AssaultInst    string `json:"assault_inst"`
	AssaultDef     string `json:"assault_def"`
	BaseDamage     int    `json:"base_damage"`     // dano por instância
	FirstHitBonus  int    `json:"first_hit_bonus"` // bônus aplicados só à 1ª instância (multi-hit)
	Instances      int    `json:"instances"`
	IgnoreWard     int    `json:"ignore_ward"`
	ResonanceBonus bool   `json:"resonance_bonus"`
	GuardInst      string `json:"guard_inst,omitempty"`
	Prevention     int    `json:"prevention"`
	PreventedAll   bool   `json:"prevented_all"`
	NetTotal       int    `json:"net_total"`
	EndWindow      bool   `json:"end_window,omitempty"`   // VR-078
	Announced      bool   `json:"announced,omitempty"`    // janela de Guarda já aberta
	PickedExile    string `json:"picked_exile,omitempty"` // VR-055
}

// RoundSigil registra a linha do tempo global de Sigilos da rodada.
type RoundSigil struct {
	P int   `json:"p"`
	S Sigil `json:"s"`
}

// PlayedCard registra as cartas resolvidas na rodada (para efeitos de cópia).
type PlayedCard struct {
	Def    string `json:"def"`
	P      int    `json:"p"`
	IsCopy bool   `json:"is_copy,omitempty"`
}

// ExtraWindow é uma janela adicional de Assalto (VR-047, ultimate de Rauk).
type ExtraWindow struct {
	Player  int    `json:"player"`
	MaxCost int    `json:"max_cost,omitempty"` // 0 = sem limite
	Source  string `json:"source"`
}

// RiteReact é a janela de reação a um Rito direcionado (VR-035). Abre apenas
// quando o defensor tem uma Guarda de counter na mão (ver ADR-014).
type RiteReact struct {
	Caster int    `json:"caster"`
	Inst   string `json:"inst"`
	Def    string `json:"def"`
	Cost   int    `json:"cost"`
}

// RiteFinalize preserva a continuação serializável de um Rito cujo efeito
// abriu uma decisão. Sigilo, deslocamento e descarte só acontecem depois que
// toda a cadeia de decisões do efeito terminar (ADR-021).
type RiteFinalize struct {
	Player    int    `json:"player"`
	Inst      string `json:"inst,omitempty"`
	Def       string `json:"def"`
	CopyDepth int    `json:"copy_depth,omitempty"`
}

// GameState é o estado serializável completo (sem o log de eventos).
type GameState struct {
	RulesetVersion string `json:"ruleset_version"`
	Seed           uint64 `json:"seed"`

	Round      int     `json:"round"`
	Phase      Phase   `json:"phase"`
	Initiative int     `json:"initiative"`
	Active     int     `json:"active"` // dono da janela atual
	Passed     [2]bool `json:"passed"`

	Eclipse           int          `json:"eclipse"`
	EclipseState      EclipseTotal `json:"eclipse_state,omitempty"`
	EclipseMovedRound bool         `json:"eclipse_moved_round,omitempty"` // VR-076

	Players [2]*PlayerState          `json:"players"`
	Cards   map[string]*CardInstance `json:"cards"`

	RoundSigils []RoundSigil `json:"round_sigils"`

	Pending      *Decision      `json:"pending,omitempty"`
	DecQueue     []*Decision    `json:"dec_queue,omitempty"`
	NextDecID    int            `json:"next_dec_id"`
	Guard        *GuardCtx      `json:"guard,omitempty"`
	Confront     *ConfrontCtx   `json:"confront,omitempty"`
	RiteReact    *RiteReact     `json:"rite_react,omitempty"`
	PendingRites []RiteFinalize `json:"pending_rites,omitempty"`
	Extra        *ExtraWindow   `json:"extra,omitempty"`
	QueuedExtra  []ExtraWindow  `json:"queued_extra,omitempty"`

	PlayedRound []PlayedCard `json:"played_round,omitempty"`
	LastRite    [2]string    `json:"last_rite,omitempty"` // último Rito resolvido por jogador

	Over      bool   `json:"over"`
	Winner    int    `json:"winner"` // -1 enquanto em jogo
	EndReason string `json:"end_reason,omitempty"`

	RNG *RNG `json:"rng"`
}

func (p *PlayerState) resetRound() {
	p.Trail = p.Trail[:0]
	p.AssaultsRound = 0
	p.GuardsRound = 0
	p.SacrificesRound = 0
	p.DamageTakenRound = 0
	p.ExiledRound = false
	p.DrawsRound = 0
	p.RecoversRound = 0
	p.StanceCommitted = false
	p.Stance = ""
	p.LuaNoSangue = false
	p.ExtraNeutralSigil = ""
	p.RelicDestroyedRound = false
	p.LastDamageDealt = 0
	p.NightShiftsRound = 0
	// NullifyOppShift (ultimate de Ilyan) persiste até ser consumida.
	p.SaelaShield = false
	p.NextAssaultBonus = 0
}

// removeID tira um id de uma slice de zona preservando a ordem.
func removeID(s []string, id string) ([]string, bool) {
	for i, v := range s {
		if v == id {
			return append(s[:i:i], s[i+1:]...), true
		}
	}
	return s, false
}

func containsID(s []string, id string) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}
