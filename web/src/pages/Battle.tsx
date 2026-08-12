import { useEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { CardTile } from "../components/CardTile";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { DecisionSheet } from "../components/DecisionSheet";
import { DuelCard } from "../components/DuelCard";
import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import { FloaterLayer, FxBanner } from "../fx/BattleFx";
import { ParticleStage } from "../fx/ParticleStage";
import { engageAmbience, setAmbienceEnabled, stopAmbience, updateAmbience } from "../fx/sfx";
import { buildGuidedProgress, guidedLesson, type GuidedLesson } from "../guidedTraining";
import { cardBrief, cardRole, cardStat } from "../cardText";
import { useBattleFx } from "../fx/useBattleFx";
import { useCards, useChampions, useRuleset } from "../queries";
import { usePreferencesStore, useSessionStore } from "../store";
import type { BattleEvent, BattleState, CardDefinition, Champion, PlayerView } from "../types";
import { useBattleSocket } from "../useBattleSocket";
import { translateText } from "../i18n";
import { paceScale, type AnimationPace } from "../battleTiming";
import { useBattlePresentation, type ArenaVisual, type ConfrontVisual } from "../battlePresentation";

// Mesa de duelo. A regra continua inteiramente na engine: esta tela só desenha
// o estado publicado. A permanência das cartas na arena é apresentação — nada
// aqui altera, adia ou antecipa o que o servidor já resolveu.

export function BattlePage() {
  const { matchId = "" } = useParams();
  const navigate = useNavigate();
  const { data: cardData } = useCards();
  const { data: championData } = useChampions();
  const animationPace = usePreferencesStore((store) => store.animationPace);
  const reducedMotion = usePreferencesStore((store) => store.reducedMotion);
  const sound = usePreferencesStore((store) => store.sound);
  const ambience = usePreferencesStore((store) => store.ambience);
  const haptics = usePreferencesStore((store) => store.haptics);
  const setPreference = usePreferencesStore((store) => store.set);
  const guidedMatchId = useSessionStore((store) => store.guidedMatchId);
  const setGuidedMatch = useSessionStore((store) => store.setGuidedMatch);
  const rulesetData = useRuleset().data;
  const leakCap = rulesetData?.guard_leak_cap ?? 0;
  const pressureClock = useMemo(() => ({
    startTurn: rulesetData?.pressure_start_turn ?? 0,
    baseLoss: rulesetData?.pressure_base_loss ?? 0,
  }), [rulesetData?.pressure_base_loss, rulesetData?.pressure_start_turn]);
  const battle = useBattleSocket(matchId);
  const [zoomed, setZoomed] = useState<CardDefinition | null>(null);
  const [logOpen, setLogOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [hoverId, setHoverId] = useState<string | null>(null);
  const [draggingCardId, setDraggingCardId] = useState<string | null>(null);
  const cardsById = useMemo(() => new Map(cardData?.cards.map((card) => [card.id, card]) ?? []), [cardData]);
  const presentation = useBattlePresentation(battle.events, battle.state, cardsById, reducedMotion, animationPace);
  const fx = useBattleFx(battle.events, battle.slot, { reducedMotion, sound, haptics, animationPace });
  const avatars = useMemo(() => new Map(championData?.champions.map((champion) => [champion.id, champion]) ?? []), [championData]);
  const liveState = battle.state;
  const liveSlot = battle.slot;
  const hand = useMemo(() => {
    if (!liveState || liveSlot === null) return [];
    return (liveState.players[liveSlot].hand ?? [])
      .map((instanceId) => ({ instanceId, card: cardsById.get(liveState.cards[instanceId]?.def) }))
      .filter((item): item is HandItem => Boolean(item.card));
  }, [cardsById, liveSlot, liveState]);
  const freshHandCards = useFreshHandCards(hand.map((item) => item.instanceId), reducedMotion, animationPace);
  const trainingProgress = useMemo(() => liveSlot === null
    ? { assault: false, guard: false, rite: false, completed: 0 }
    : buildGuidedProgress(battle.events, liveSlot, cardsById), [battle.events, cardsById, liveSlot]);
  const autoPassKey = useRef("");

  const playable = (instanceId: string) => Boolean(liveState?.playable?.includes(instanceId))
    && liveState?.active === liveSlot && battle.status === "connected" && !battle.pending;
  const playCard = (instanceId: string) => { engageAmbience(); setSelectedId(null); battle.send({ kind: "play", card: instanceId }); };
  const passPhase = () => { engageAmbience(); setSelectedId(null); battle.send({ kind: "pass" }); };

  useEffect(() => {
    if (!liveState || liveSlot === null || liveState.over || battle.pending || battle.status !== "connected" ||
      liveState.pending || liveState.active !== liveSlot || !["assalto", "guarda", "rito"].includes(liveState.phase) ||
      (liveState.playable ?? []).length > 0) return;
    const key = `${liveState.round}:${liveState.phase}:${liveState.active}`;
    if (autoPassKey.current === key) return;
    const timer = window.setTimeout(() => {
      autoPassKey.current = key;
      engageAmbience();
      battle.send({ kind: "pass" });
    }, autoPassDelay(reducedMotion));
    return () => window.clearTimeout(timer);
  }, [battle, liveSlot, liveState, reducedMotion]);

  useEffect(() => {
    setAmbienceEnabled(ambience);
    return () => stopAmbience();
  }, [ambience]);

  useEffect(() => {
    if (!liveState || liveSlot === null) return;
    if (liveState.over) {
      stopAmbience();
      return;
    }
    updateAmbience({
      ownTurn: liveState.active === liveSlot,
      guard: liveState.phase === "guarda" && liveState.active === liveSlot,
      danger: liveState.players[liveSlot].vitality <= 8,
    });
  }, [liveSlot, liveState]);

  // A seleção é sempre de uma carta que ainda está na mão.
  useEffect(() => {
    if (selectedId && !hand.some((item) => item.instanceId === selectedId)) setSelectedId(null);
  }, [hand, selectedId]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (event.key === "Escape") {
        if (zoomed) setZoomed(null);
        else if (logOpen || helpOpen) { setLogOpen(false); setHelpOpen(false); }
        else setSelectedId(null);
        return;
      }
      if (target?.closest("input, select, textarea") || !liveState || liveSlot === null || liveState.over) return;
      if (liveState.pending) return;
      if (event.key.toLocaleLowerCase("pt-BR") === "h") {
        event.preventDefault();
        setHelpOpen((open) => !open);
        return;
      }
      const index = Number(event.key) - 1;
      if (Number.isInteger(index) && index >= 0 && index < hand.length) {
        event.preventDefault();
        const item = hand[index];
        // Primeiro toque inspeciona; repetir o mesmo número confirma a jogada.
        if (selectedId === item.instanceId && playable(item.instanceId)) playCard(item.instanceId);
        else setSelectedId(item.instanceId);
        return;
      }
      if (event.key === "Enter" && selectedId && playable(selectedId)) {
        event.preventDefault();
        playCard(selectedId);
        return;
      }
      if (event.code === "Space" && liveState.active === liveSlot && !battle.pending && ["assalto", "guarda", "rito"].includes(liveState.phase)) {
        if (target?.closest("button")) return;
        event.preventDefault();
        passPhase();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  if (!liveState || liveSlot === null) {
    return <main className="battle-loading"><div className="queue-orbit is-searching" aria-hidden="true"><span className="orbit orbit-a" /><span className="orbit orbit-b" /><NytharaMark /></div><p className="eyebrow">SALA DE DUELO</p><h1>{battle.status === "reconnecting" ? "Restaurando a partida…" : "Preparando a mesa…"}</h1><p>{battle.error ?? "Sincronizando o estado e aguardando os duelistas."}</p><Link className="ghost-button" to="/queue">Voltar à Arena</Link></main>;
  }

  const state = liveState;
  const mySlot = liveSlot;
  const opponentSlot = 1 - mySlot;
  // A engine já publicou o snapshot final, mas os medidores acompanham a fila
  // visual. Assim a carta chega à mesa antes de seu custo ou dano aparecer.
  const me = { ...state.players[mySlot], vitality: presentation.vitality[mySlot] };
  const opponent = { ...state.players[opponentSlot], vitality: presentation.vitality[opponentSlot] };
  const myTurn = state.active === mySlot && battle.status === "connected" && !battle.pending;
  // Na janela de Guarda o ativo é o defensor; o dono do turno é o atacante.
  const turnOwner = state.phase === "guarda"
    ? state.confront?.attacker ?? state.guard?.attacker ?? 1 - state.active
    : state.active;
  const visual: ArenaVisual | null = presentation.visual ?? (state.confront ? {
    mode: "confront",
    key: state.round,
    attackDef: state.confront.assault_def,
    guardDef: state.confront.guard_def,
    attacker: state.confront.attacker,
    power: state.confront.power,
    prevention: state.confront.prevention ?? 0,
    damage: 0,
    phase: state.confront.guard_def ? "guard" as const : "waiting" as const,
  } : null);

  // O ponteiro manda na leitura: passar o mouse já mostra a carta inteira no
  // trilho. A carta fixada por clique é só o alvo dos botões do painel.
  const focused = hand.find((item) => item.instanceId === hoverId)
    ?? hand.find((item) => item.instanceId === selectedId);
  const arenaCard = visual
    ? cardsById.get(visual.mode === "effect" ? visual.cardDef : visual.guardDef ?? visual.attackDef)
    : undefined;
  const inspected = focused?.card ?? arenaCard;
  const inspectSource: InspectSource = focused ? "hand" : arenaCard ? "arena" : "none";
  const choice = focused && focused.instanceId === selectedId ? {
    playable: playable(focused.instanceId),
    onPlay: () => playCard(focused.instanceId),
    onCancel: () => setSelectedId(null),
  } : undefined;
  const playableItems = hand.filter((item) => playable(item.instanceId));
  const coachLesson = guidedLesson(state, mySlot, trainingProgress, { guardLeakCap: leakCap });

  return <main className={`duel-room phase-${state.phase} ${myTurn ? "is-my-turn" : "is-rival-turn"} ${fx.shaking ? "is-shaking" : ""} ${draggingCardId ? "is-card-dragging" : ""} ${me.vitality <= 12 && !state.over ? "is-danger" : ""} ${selectedId ? "is-inspecting" : ""}`}>
    <h1 className="sr-only">Duelo em andamento</h1>
    <header className="duel-topbar">
      <Link to="/app" aria-label="Sair para o início"><NytharaMark /></Link>
      <span className="duel-topbar__link"><span className={`connection-dot ${battle.status}`} /><span className="duel-topbar__label">{battle.status === "connected" ? "Conectado" : "Reconectando"}</span></span>
      <strong className="duel-topbar__turn">TURNO {state.round}</strong>
      <Deadline value={battle.deadline} />
      <button type="button" onClick={() => setHelpOpen((open) => !open)} aria-expanded={helpOpen}><UiIcon name="info" /><span>Ajuda</span></button>
      <button type="button" onClick={() => setLogOpen((open) => !open)} aria-expanded={logOpen}><UiIcon name="history" /><span>Histórico</span></button>
    </header>

    {guidedMatchId === matchId && !state.over && <GuidedCoach lesson={coachLesson} progress={trainingProgress} onClose={() => setGuidedMatch(null)} />}

    <div className="duel-frame">
      <section className="duel-board">
        <DuelistBar own={false} player={opponent} avatar={avatars.get(opponent.champion)} active={state.active === opponentSlot} round={state.round} state={state} cards={cardsById} />

        <ArenaTable
          visual={visual}
          cards={cardsById}
          mySlot={mySlot}
          state={state}
          events={battle.events}
          dragging={Boolean(draggingCardId)}
          clock={pressureClock}
        />

        <DuelistBar own player={me} avatar={avatars.get(me.champion)} active={myTurn} round={state.round} state={state} cards={cardsById} />
      </section>

      {/* Scrim só existe no sheet móvel (CSS); tocar fora solta a carta. */}
      {selectedId && <button type="button" className="rail-scrim" aria-label="Fechar a leitura da carta" onClick={() => setSelectedId(null)} />}
      <aside className="duel-rail" aria-label="Leitura da carta">
        <Inspector card={inspected} source={inspectSource} myTurn={myTurn} choice={choice} onZoom={setZoomed} />
      </aside>
    </div>

    <footer className="duel-base">
      <PhaseOrbs phase={state.phase} active={state.active} mySlot={mySlot} turnOwner={turnOwner} />
      <div className="duel-base__hand">
        <ol className={`duel-hand ${draggingCardId ? "is-dragging" : ""}`} aria-label="Sua mão">
          {hand.map((item, index) => <HandSlot
            key={item.instanceId}
            item={item}
            index={index}
            fresh={freshHandCards.has(item.instanceId)}
            selected={selectedId === item.instanceId}
            playable={playable(item.instanceId)}
            tilt={!reducedMotion}
            onSelect={() => setSelectedId((current) => (current === item.instanceId ? null : item.instanceId))}
            onHover={setHoverId}
            onPlay={() => playCard(item.instanceId)}
            onZoom={() => setZoomed(item.card)}
            onCancel={() => setSelectedId(null)}
            onDragChange={(dragging) => setDraggingCardId(dragging ? item.instanceId : null)}
          />)}
          {!hand.length && <li className="duel-hand__empty">Mão vazia — a compra do próximo turno reabastece.</li>}
        </ol>
      </div>
      <ActionPanel state={state} mySlot={mySlot} myTurn={myTurn} connected={!battle.pending} options={playableItems} autoPassMs={autoPassDelay(reducedMotion)} onPass={passPhase} />
    </footer>

    {battle.error && <p className="duel-error" role="alert">{battle.error}</p>}

    {!reducedMotion && <ParticleStage />}
    <FloaterLayer floaters={fx.floaters} mySlot={mySlot} />
    <FxBanner banner={fx.banner} />

    <aside className={`battle-log ${logOpen ? "is-open" : ""}`} aria-label="Histórico de eventos"><header><div><p className="eyebrow">LINHA DO TEMPO</p><h2>Histórico</h2></div><button type="button" onClick={() => setLogOpen(false)} aria-label="Fechar histórico"><UiIcon name="close" /></button></header><div>{battle.events.slice().reverse().map((event) => <EventLine event={event} key={event.seq} cards={cardsById} ownSlot={mySlot} />)}</div></aside>

    <aside className={`battle-help ${helpOpen ? "is-open" : ""}`} aria-label="Ajuda do duelo"><header><div><p className="eyebrow">COMO SE JOGA</p><h2>Controles e regra</h2></div><button type="button" onClick={() => setHelpOpen(false)} aria-label="Fechar ajuda"><UiIcon name="close" /></button></header><ol><li><kbd>1×</kbd><span><strong>Clique para ler</strong><small>A carta sobe e aparece inteira no painel da direita.</small></span></li><li><kbd>↑</kbd><span><strong>Confirme para jogar</strong><small>Use o botão de jogar da carta ou arraste-a para a mesa.</small></span></li><li><kbd>1–7</kbd><span><strong>Teclado</strong><small>O número lê a carta; o mesmo número de novo confirma.</small></span></li><li><kbd>Espaço</kbd><span><strong>Passar a fase</strong><small>Sem carta legal, a mesa avança sozinha.</small></span></li></ol><div className="battle-help__rule"><strong>Assalto → Guarda → Rito</strong><p>No seu turno você declara o <b>Assalto</b> e, depois, pode usar um <b>Rito</b> — duas cartas na mesma vez. A <b>Guarda</b> é a janela do defensor: você só escolhe uma quando o rival ataca. Poder menos Prevenção vira dano{leakCap > 0 ? <>, e <b>uma Guarda nunca deixa passar mais que {leakCap}</b>. Quem não responde leva o golpe inteiro</> : null}. As cartas ficam na mesa até o próximo confronto, com o resultado carimbado. Os medalhões ao lado de cada duelista mostram os Selos ativos o tempo todo.</p></div><label className="battle-help__toggle"><span><strong>Movimento reduzido</strong><small>Corta voo de carta e tremor de impacto.</small></span><input type="checkbox" checked={reducedMotion} onChange={(event) => setPreference("reducedMotion", event.target.checked)} /></label><Link to="/settings">Som, trilha, vibração e acessibilidade</Link><button className="battle-help__concede" type="button" onClick={() => window.confirm("Conceder esta partida?") && battle.send({ kind: "concede" })}>Conceder partida</button></aside>

    {state.pending && state.pending.player === mySlot && <DecisionSheet
      pending={state.pending}
      cards={state.cards}
      byId={cardsById}
      busy={Boolean(battle.pending) || battle.status !== "connected"}
      onConfirm={(picked) => { engageAmbience(); battle.send({ kind: "choose", decision_id: state.pending!.id, cards: picked }); }}
    />}
    {state.over && !presentation.busy && <div className="match-over-overlay" role="dialog" aria-modal="true"><p className="eyebrow">CONFRONTO ENCERRADO</p><h2>{state.winner === mySlot ? "Vitória" : "Derrota"}</h2><p>{endReason(state.end_reason)}</p><button className="primary-button" type="button" onClick={() => navigate("/result")}>Ver resultado</button></div>}
    {zoomed && <div className="modal-backdrop" onMouseDown={() => setZoomed(null)}><div className="battle-card-zoom" onMouseDown={(event) => event.stopPropagation()}><button className="modal-close" onClick={() => setZoomed(null)} aria-label="Fechar"><UiIcon name="close" /></button><CardTile card={zoomed} /></div></div>}
  </main>;
}

interface HandItem { instanceId: string; card: CardDefinition }
type InspectSource = "hand" | "arena" | "none";

/* ------------------------------------------------------------------ mesa */

function ArenaTable({ visual, cards, mySlot, state, events, dragging, clock }: {
  visual: ArenaVisual | null;
  cards: Map<string, CardDefinition>;
  mySlot: number;
  state: BattleState;
  events: BattleEvent[];
  dragging: boolean;
  clock: { startTurn: number; baseLoss: number };
}) {
  const confront = visual?.mode === "confront" ? visual : null;
  const effect = visual?.mode === "effect" ? visual : null;
  const attack = confront ? cards.get(confront.attackDef) : undefined;
  const guard = confront?.guardDef ? cards.get(confront.guardDef) : undefined;
  // O impacto primeiro revela a conta. A rachadura e o carimbo só entram no
  // assentamento, depois da pausa de leitura, para a carta não explodir junto
  // com o primeiro frame do choque.
  const settled = visual?.phase === "settled";
  const attackBroken = settled && confront?.outcome === "guard";
  const guardBroken = settled && confront?.outcome === "assault";
  const attackerIsMe = confront?.attacker === mySlot;
  // O relógio da Pressão é regra do ruleset ativo. Fixá-lo aqui já fez a mesa
  // anunciar pressão em um turno em que a engine não aplicava nada.
  const start = clock.startTurn;
  const warnFrom = start > 0 ? Math.max(1, start - 8) : 0;
  const pressure = start > 0 && state.round >= warnFrom;
  const pressureOn = start > 0 && state.round >= start;

  return <section className={`arena ${visual ? `is-${visual.phase}` : "is-idle"} ${visual?.stale ? "is-stale" : ""}`} aria-label="Zona de confronto">
    <div className="arena__felt" aria-hidden="true" />

    {pressure && <div className={`arena__pressure ${pressureOn ? "is-active" : ""}`} role="status">
      <UiIcon name="warning" />
      <span><strong>{pressureOn ? "Pressão de Nythara ativa" : "A névoa está se fechando"}</strong>
        <small>{pressureOn
          ? `Ambos perdem ${(state.round - start + 1) * clock.baseLoss} de Vitalidade no início do turno.`
          : `No turno ${start}, ambos passam a perder Vitalidade crescente.`}</small></span>
    </div>}

    {visual?.stale && <p className="arena__stale">JOGADA ANTERIOR — permanece na mesa até a próxima carta</p>}

    {effect ? <EffectStage visual={effect} card={cards.get(effect.cardDef)} mySlot={mySlot} /> : <div className="arena__lane">
      <ArenaSlot
        kind="assault"
        card={attack}
        broken={Boolean(attackBroken)}
        side={attackerIsMe ? "own" : "rival"}
        stamp={confront && settled ? (attackBroken ? "ESTILHAÇADA" : "ATRAVESSOU") : undefined}
        idleTitle={confront ? (attackerIsMe ? "SEU ASSALTO" : "ASSALTO DO RIVAL") : "ASSALTO"}
        idleHint="A carta atacante fica aqui"
      />

      <Verdict visual={confront} mySlot={mySlot} state={state} dragging={dragging} />

      <ArenaSlot
        kind="guard"
        card={guard}
        broken={Boolean(guardBroken)}
        side={attackerIsMe ? "rival" : "own"}
        stamp={confront && settled && !guard ? "SEM GUARDA" : confront && settled ? (guardBroken ? "ROMPIDA" : "SEGUROU") : undefined}
        idleTitle={confront ? (attackerIsMe ? "GUARDA DO RIVAL" : "SUA GUARDA") : "GUARDA"}
        idleHint={visual && visual.phase === "waiting" ? "Aguardando a resposta" : "A resposta defensiva fica aqui"}
        waiting={visual?.phase === "waiting"}
      />
    </div>}

    <Chronicle events={events} cards={cards} mySlot={mySlot} />
  </section>;
}

function EffectStage({ visual, card, mySlot }: {
  visual: Extract<ArenaVisual, { mode: "effect" }>;
  card?: CardDefinition;
  mySlot: number;
}) {
  const mine = visual.player === mySlot;
  const resolved = visual.phase === "impact" || visual.phase === "settled";
  const target = visual.target === undefined ? "a mesa" : visual.target === mySlot ? "você" : "o rival";
  return <div className={`arena-effect from-${mine ? "own" : "rival"} is-${visual.phase}`} data-fx="slot-effect">
    <span className="arena-effect__label">{mine ? "SUA JOGADA" : "JOGADA DO RIVAL"}</span>
    <div className="arena-effect__card">
      {card
        ? <DuelCard card={card} size="table" />
        : <div className="arena-slot__ghost"><UiIcon name="sigil" /><small>Materializando a carta…</small></div>}
    </div>
    <div className={`arena-effect__result ${resolved ? "is-resolved" : ""}`} aria-live="polite">
      <UiIcon name={visual.damage > 0 ? "duel" : "sigil"} />
      <strong>{visual.damage > 0 ? `${visual.damage} DE DANO` : resolved ? "EFEITO RESOLVIDO" : "ATIVANDO EFEITO"}</strong>
      <small>{visual.damage > 0 ? `atingiu ${target}` : "A carta permanece visível durante a resolução."}</small>
    </div>
  </div>;
}

function ArenaSlot({ kind, card, broken, side, stamp, idleTitle, idleHint, waiting }: {
  kind: "assault" | "guard";
  card?: CardDefinition;
  broken: boolean;
  side: "own" | "rival";
  stamp?: string;
  idleTitle: string;
  idleHint: string;
  waiting?: boolean;
}) {
  return <div className={`arena-slot is-${kind} from-${side} ${card ? "is-filled" : "is-empty"} ${waiting ? "is-waiting" : ""}`} data-fx={`slot-${kind}`}>
    <span className="arena-slot__label">{idleTitle}</span>
    {card
      ? <div className="arena-slot__card"><DuelCard card={card} size="table" broken={broken} />{stamp && <b className={`arena-slot__stamp ${broken ? "is-broken" : ""}`}>{stamp}</b>}</div>
      : <div className="arena-slot__ghost"><UiIcon name={kind === "assault" ? "duel" : "ward"} /><small>{idleHint}</small>{stamp && <b className="arena-slot__stamp">{stamp}</b>}</div>}
  </div>;
}

function Verdict({ visual, mySlot, state, dragging }: { visual: ConfrontVisual | null; mySlot: number; state: BattleState; dragging: boolean }) {
  if (dragging) {
    return <div className="verdict is-drop" aria-hidden="true"><UiIcon name="duel" /><strong>Solte para jogar</strong><small>A engine confirma a carta no centro.</small></div>;
  }
  if (!visual) {
    const idle = state.phase === "guarda"
      ? ["RESPOSTA", "O confronto abre quando um Assalto for declarado."]
      : ["MESA LIVRE", "Poder − Prevenção = dano."];
    return <div className="verdict is-idle"><UiIcon name="versus" /><strong>{idle[0]}</strong><small>{idle[1]}</small></div>;
  }
  const resolved = visual.phase === "impact" || visual.phase === "settled";
  const damageToMe = resolved && visual.attacker !== mySlot && visual.damage > 0;
  // O teto de vazamento age quando a subtração daria mais do que passou.
  const capped = resolved && Boolean(visual.guardDef) && visual.power - visual.prevention > visual.damage;
  return <div className={`verdict is-live ${resolved ? "is-resolved" : ""} ${damageToMe ? "is-against-me" : ""}`} data-fx="verdict">
    <div className="verdict__math">
      <span><small>PODER</small><b>{visual.power}</b></span>
      <i aria-hidden="true">−</i>
      <span><small>PREVENÇÃO</small><b>{visual.prevention || "0"}</b></span>
    </div>
    {capped && <span className="verdict__cap">TETO DA GUARDA</span>}
    {resolved
      ? <strong className={visual.damage > 0 ? "verdict__result is-damage" : "verdict__result is-blocked"}>
          {visual.damage > 0 ? `${visual.damage} DE DANO` : "DANO BLOQUEADO"}
          {/* Quando o teto age, a conta na tela não fecha sozinha: dizer por
              quê no exato momento é o que ensina a regra. */}
          <small>{capped ? "a Guarda segurou o resto" : visual.attacker === mySlot ? "no rival" : "em você"}</small>
        </strong>
      : <strong className="verdict__result is-pending">
          {visual.phase === "waiting" ? "AGUARDANDO GUARDA" : "CHOQUE"}
          <small>{visual.attacker === mySlot ? "seu Assalto" : "Assalto do rival"}</small>
        </strong>}
  </div>;
}

function Chronicle({ events, cards, mySlot }: { events: BattleEvent[]; cards: Map<string, CardDefinition>; mySlot: number }) {
  const lines = useMemo(() => events.filter((event) => CHRONICLE_KINDS.has(event.kind)).slice(-3), [events]);
  if (!lines.length) return null;
  return <ul className="arena__chronicle" aria-live="polite">
    {lines.map((event) => <li key={event.seq}>&lt; {chronicleLine(event, cards, mySlot)} &gt;</li>)}
  </ul>;
}

const CHRONICLE_KINDS = new Set([
  "card_played", "confrontation_opened", "guard_committed", "confrontation_resolved",
  "damage_dealt", "damage_prevented", "healed", "ward_gained", "card_shattered",
  "status_applied", "status_expired", "fatigue", "turn_started",
]);

function chronicleLine(event: BattleEvent, cards: Map<string, CardDefinition>, mySlot: number) {
  const who = event.p < 0 ? "A mesa" : event.p === mySlot ? "Você" : "O rival";
  const name = event.def ? cards.get(event.def)?.name ?? event.def : "";
  switch (event.kind) {
    case "turn_started": return `${who} assume o turno ${event.round}.`;
    case "card_played": return `${who} jogou ${name}.`;
    case "confrontation_opened": return `${who} declarou ${name} com Poder ${event.n}.`;
    case "guard_committed": return `${who} respondeu com ${name}, prevenindo ${event.n}.`;
    case "confrontation_resolved": return `Confronto resolvido: ${event.n} de dano.`;
    case "damage_dealt": return `${who} sofreu ${event.n} de dano.`;
    case "damage_prevented": return `${who} preveniu ${event.n} de dano.`;
    case "healed": return `${who} recuperou ${event.n} de Vitalidade.`;
    case "ward_gained": return `${who} ganhou ${event.n} de Ward.`;
    case "card_shattered": return `${name || "A carta"} se estilhaçou.`;
    case "status_applied": return `${who} está sob ${event.s ?? "um efeito"}.`;
    case "status_expired": return `${event.s ?? "O efeito"} se encerrou sobre ${event.p === mySlot ? "você" : "o rival"}.`;
    case "fatigue": return `${who} sofreu ${event.n} de Fadiga.`;
    default: return eventTitle(event.kind);
  }
}

/* -------------------------------------------------------------- duelista */

function DuelistBar({ own, player, avatar, active, round, state, cards }: {
  own: boolean;
  player: PlayerView;
  avatar?: Champion;
  active: boolean;
  round: number;
  state: BattleState;
  cards: Map<string, CardDefinition>;
}) {
  const locale = usePreferencesStore((preferences) => preferences.locale);
  const vitality = Math.max(0, player.vitality);
  const maximum = Math.max(1, player.max_vitality || 30);
  const percentage = Math.min(100, Math.round((vitality / maximum) * 100));
  const faction = avatar?.faction.normalize("NFD").replace(/[̀-ͯ]/g, "").replaceAll(" ", "-").toLocaleLowerCase("pt-BR") ?? "errantes";
  const topDiscard = player.discard[player.discard.length - 1];
  const discardCard = topDiscard ? cards.get(state.cards[topDiscard]?.def) : undefined;

  return <section className={`duelist ${own ? "is-own" : "is-rival"} ${active ? "is-active" : ""} ${vitality <= 8 ? "is-in-danger" : ""}`} aria-label={own ? "Seu lado" : "Lado do rival"}>
    <div className="duelist__identity">
      <span className={`duelist__portrait emblem-frame faction-${faction}`}>{avatar ? <ChampionEmblem id={avatar.id} faction={avatar.faction} /> : <UiIcon name="champion" />}</span>
      <div>
        <small>{own ? "VOCÊ" : "RIVAL"}</small>
        <strong>{(avatar?.name ?? (own ? "Seu Avatar" : "Duelista rival")).split(",")[0]}</strong>
        <em title={avatar?.confront_power ? translateText(avatar.confront_power, locale) : undefined}>{avatar?.confront_power ? translateText(avatar.confront_power, locale) : "Avatar · sem efeito de regra"}</em>
      </div>
    </div>

    <div className="vial" data-fx={own ? "vial-own" : "vial-rival"} role="progressbar" aria-label={`Vitalidade ${vitality} de ${maximum}`} aria-valuemin={0} aria-valuemax={maximum} aria-valuenow={vitality}>
      <span className="vial__glass">
        {/* O fantasma desce devagar atrás do fluido: o "pedaço perdido" fica
            visível por um segundo, como em jogo de luta. */}
        <i className="vial__ghost" style={{ height: `${percentage}%` }} />
        <i className="vial__fluid" style={{ height: `${percentage}%` }} />
      </span>
      <span className="vial__read"><b><TickedNumber value={vitality} /></b><small>VITALIDADE</small></span>
    </div>

    <SealMedals player={player} round={round} />

    <div className="duelist__piles">
      {own
        ? <span className="pile pile-hand"><b>{player.hand?.length ?? player.hand_count}</b><small>MÃO</small></span>
        : <div className="hidden-hand" aria-label={`${player.hand_count} cartas ocultas na mão do rival`}>
            {Array.from({ length: Math.min(player.hand_count, 8) }, (_, index) => <i style={{ "--card-index": index } as CSSProperties} key={index} />)}
            <b>{player.hand_count}</b>
          </div>}
      <span className="pile pile-deck" aria-label={`${player.deck_count} cartas no baralho`}><i /><i /><b>{player.deck_count}</b><small>BARALHO</small></span>
      <span className="pile pile-discard" aria-label={`${player.discard.length} cartas no descarte`}>
        {discardCard ? <img src={`/card-art/${discardCard.id.toLocaleLowerCase("en-US")}.webp`} alt="" /> : null}
        <b>{player.discard.length}</b><small>DESCARTE</small>
      </span>
      <span className="pile pile-ward" aria-label={`Ward ${player.ward}`}><UiIcon name="ward" /><b>{player.ward}</b><small>WARD</small></span>
    </div>
  </section>;
}

// Número que corre até o valor novo: a perda é lida como movimento, não como
// troca de dígito. Puro estado local — o valor autoritativo chega pronto.
function TickedNumber({ value }: { value: number }) {
  const [shown, setShown] = useState(value);
  const raf = useRef(0);
  useEffect(() => {
    cancelAnimationFrame(raf.current);
    const from = shown;
    if (from === value) return;
    const start = performance.now();
    const duration = Math.min(700, 160 + Math.abs(value - from) * 60);
    const step = (now: number) => {
      const t = Math.min(1, (now - start) / duration);
      const eased = 1 - Math.pow(1 - t, 3);
      setShown(Math.round(from + (value - from) * eased));
      if (t < 1) raf.current = requestAnimationFrame(step);
    };
    raf.current = requestAnimationFrame(step);
    return () => cancelAnimationFrame(raf.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);
  return <>{shown}</>;
}

function SealMedals({ player, round }: { player: PlayerView; round: number }) {
  const medals = [
    { key: "atacar", label: "PROIBIDO ATACAR", on: Boolean(player.assault_seal_until && player.assault_seal_until >= round) },
    { key: "defender", label: "PROIBIDO DEFENDER", on: Boolean(player.exposto) },
    { key: "rito", label: "RITO SELADO", on: Boolean(player.rite_seal_until && player.rite_seal_until >= round) },
  ];
  return <div className="seals" aria-label="Selos ativos">
    {medals.map((medal) => <span className={`seal seal-${medal.key} ${medal.on ? "is-on" : ""}`} key={medal.key} title={medal.on ? medal.label : `${medal.label} — inativo`}>
      <i aria-hidden="true" />
      <small>{medal.label}</small>
      <span className="sr-only">{medal.on ? `${medal.label}: ativo` : `${medal.label}: inativo`}</span>
    </span>)}
  </div>;
}

/* ---------------------------------------------------------------- rodapé */

// A janela de Guarda pertence ao DEFENSOR: no seu turno quem responde é o
// rival. O painel diz de quem é cada janela para que ninguém procure uma
// Guarda na própria vez de atacar.
function PhaseOrbs({ phase, active, mySlot, turnOwner }: {
  phase: BattleState["phase"];
  active: number;
  mySlot: number;
  turnOwner: number;
}) {
  const steps = [
    ["assalto", "ASSALTO", "ataque"],
    ["guarda", "GUARDA", "resposta"],
    ["rito", "RITO", "efeito"],
  ] as const;
  const current = steps.findIndex(([step]) => step === phase);
  const ownerOf = (step: string) => (step === "guarda" ? 1 - turnOwner : turnOwner);
  return <div className="phase-panel" aria-label={`Fase atual: ${phase}`}>
    <p className="phase-panel__title">FASE</p>
    <div className="phase-panel__orbs">
      {steps.map(([step, label, hint], index) => {
        const mine = ownerOf(step) === mySlot;
        return <span className={`orb ${phase === step ? "is-current" : current > index ? "is-done" : ""} ${mine ? "is-mine" : "is-rival"}`} key={step}>
          <i aria-hidden="true" />
          <b>{label}</b>
          <em>{phase === step ? (mine ? "sua janela" : "janela do rival") : hint}</em>
        </span>;
      })}
    </div>
    <small className={`phase-panel__actor ${active === mySlot ? "is-mine" : ""}`}>{active === mySlot ? "SUA AÇÃO" : "AÇÃO DO RIVAL"}</small>
  </div>;
}

function HandSlot({ item, index, fresh, selected, playable, tilt, onSelect, onHover, onPlay, onZoom, onCancel, onDragChange }: {
  item: HandItem;
  index: number;
  fresh: boolean;
  selected: boolean;
  playable: boolean;
  tilt: boolean;
  onSelect: () => void;
  onHover: (id: string | null) => void;
  onPlay: () => void;
  onZoom: () => void;
  onCancel: () => void;
  onDragChange: (dragging: boolean) => void;
}) {
  const [drag, setDrag] = useState({ x: 0, y: 0, progress: 0, active: false });
  const gesture = useRef<{ pointerId: number; x: number; y: number; active: boolean } | null>(null);
  const suppressClick = useRef(false);
  const slotRef = useRef<HTMLLIElement>(null);

  // Tilt 3D seguindo o ponteiro — imperativo de propósito: um setState por
  // pointermove faria a mão inteira re-renderizar a 60Hz.
  const setTilt = (rx: number, ry: number) => {
    slotRef.current?.style.setProperty("--tilt-rx", `${rx}deg`);
    slotRef.current?.style.setProperty("--tilt-ry", `${ry}deg`);
  };
  const followTilt = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (!tilt || gesture.current?.active) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const nx = (event.clientX - rect.left) / rect.width - 0.5;
    const ny = (event.clientY - rect.top) / rect.height - 0.5;
    setTilt(-ny * 10, nx * 12);
  };

  const resetDrag = () => {
    if (gesture.current?.active) onDragChange(false);
    gesture.current = null;
    setDrag({ x: 0, y: 0, progress: 0, active: false });
  };
  const pointerDown = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (!playable || event.button !== 0) return;
    gesture.current = { pointerId: event.pointerId, x: event.clientX, y: event.clientY, active: false };
    try { event.currentTarget.setPointerCapture(event.pointerId); } catch { /* captura é melhoria, não requisito do gesto */ }
  };
  const pointerMove = (event: ReactPointerEvent<HTMLButtonElement>) => {
    followTilt(event);
    const current = gesture.current;
    if (!current || current.pointerId !== event.pointerId) return;
    const rawX = event.clientX - current.x;
    const rawY = event.clientY - current.y;
    const active = current.active || Math.hypot(rawX, rawY) > 7;
    if (active && !current.active) {
      current.active = true;
      onSelect();
      onDragChange(true);
      setTilt(0, 0);
    }
    if (!active) return;
    event.preventDefault();
    setDrag({
      x: Math.max(-160, Math.min(160, rawX)),
      y: Math.max(-260, Math.min(28, rawY)),
      progress: Math.max(0, Math.min(1, -rawY / 110)),
      active: true,
    });
  };
  const pointerUp = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const current = gesture.current;
    if (!current || current.pointerId !== event.pointerId) return;
    const shouldPlay = current.active && event.clientY - current.y <= -110;
    if (current.active) {
      suppressClick.current = true;
      window.setTimeout(() => { suppressClick.current = false; }, 0);
      event.preventDefault();
    }
    try { if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId); } catch { /* melhor esforço */ }
    resetDrag();
    if (shouldPlay) onPlay();
  };
  const pointerCancel = (event: ReactPointerEvent<HTMLButtonElement>) => {
    try { if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId); } catch { /* melhor esforço */ }
    resetDrag();
  };

  const style = {
    "--card-index": index,
    "--drag-x": `${drag.x}px`,
    "--drag-y": `${drag.y}px`,
    "--drag-tilt": `${drag.x * 0.02}deg`,
    "--drag-scale": 1 + drag.progress * 0.08,
  } as CSSProperties;

  return <li ref={slotRef} className={`hand-slot ${selected ? "is-selected" : ""} ${playable ? "is-playable" : "is-idle"} ${fresh ? "is-new" : ""} ${drag.active ? "is-dragging" : ""} ${drag.progress >= 1 ? "is-armed" : ""}`} style={style}>
    <button
      type="button"
      className="hand-slot__grab"
      onMouseEnter={() => onHover(item.instanceId)}
      onMouseLeave={() => { onHover(null); setTilt(0, 0); }}
      onFocus={() => onHover(item.instanceId)}
      onBlur={() => onHover(null)}
      onPointerDown={pointerDown}
      onPointerMove={pointerMove}
      onPointerUp={pointerUp}
      onPointerCancel={pointerCancel}
      onClick={() => { if (suppressClick.current) { suppressClick.current = false; return; } onSelect(); }}
      aria-pressed={selected}
      aria-keyshortcuts={String(index + 1)}
      aria-label={`Carta ${index + 1}. ${item.card.name}. ${cardBrief(item.card)}${playable ? " Jogável agora." : " Não é jogável nesta fase."}`}
    >
      <kbd className="hand-slot__key" aria-hidden="true">{index + 1}</kbd>
      <DuelCard card={item.card} size="hand" dimmed={!playable} />
    </button>

    {/* Barra de decisão sobre a carta: usar, ver ou desistir da escolha. */}
    <div className="hand-slot__tools" role="group" aria-label={`Ações de ${item.card.name}`}>
      {playable && <button type="button" className="tool tool-play" onClick={onPlay} aria-label={`Usar ${item.card.name}`}><i aria-hidden="true">↑</i><span>USAR</span></button>}
      <button type="button" className="tool tool-zoom" onClick={onZoom} aria-label={`Ampliar ${item.card.name}`}><UiIcon name="info" /><span>VER</span></button>
      {selected && <button type="button" className="tool tool-cancel" onClick={onCancel} aria-label={`Soltar ${item.card.name}`}><UiIcon name="close" /><span>SOLTAR</span></button>}
    </div>
  </li>;
}

// Painel de ação. Quatro decisões de leitura sustentam este bloco: o título é
// sempre um verbo (o que fazer), a opção é nomeada em vez de contada (ninguém
// deveria varrer a mão para descobrir qual carta é), a espera do avanço
// automático tem barra, e o botão de passar é sólido mas nunca dourado — o
// dourado é de jogar carta, e inverter isso convidaria a passar o turno.
function ActionPanel({ state, mySlot, myTurn, connected, options, autoPassMs, onPass }: {
  state: BattleState;
  mySlot: number;
  myTurn: boolean;
  connected: boolean;
  options: HandItem[];
  autoPassMs: number;
  onPass: () => void;
}) {
  if (!myTurn || (state.pending && state.pending.player !== mySlot)) {
    const rivalDeciding = Boolean(state.pending && state.pending.player !== mySlot);
    const rivalGuard = state.phase === "guarda" && state.active !== mySlot;
    return <aside className="action-panel is-waiting" aria-label="Vez do rival">
      <span className="action-panel__glyph"><UiIcon name="clock" /></span>
      <div className="action-panel__copy">
        <strong>{rivalDeciding ? "O rival faz uma escolha" : rivalGuard ? "O rival decide a Guarda" : "Vez do rival"}</strong>
        <small>{rivalDeciding ? "Uma carta dele exige decisão antes de a mesa seguir." : rivalGuard ? "Ele pode bloquear seu Assalto ou deixá-lo passar." : "A próxima decisão aparece no centro da mesa."}</small>
      </div>
      <span className="action-panel__wait" aria-hidden="true"><i /><i /><i /></span>
    </aside>;
  }

  const prompts: Record<string, { title: string; hint: string; pass: string; icon: "duel" | "ward" | "sigil" }> = {
    assalto: { title: "Ataque agora", hint: "Mande um Assalto ao centro da mesa.", pass: "Não atacar neste turno", icon: "duel" },
    guarda: { title: "Defenda-se", hint: "Bloqueie o Assalto ou deixe o golpe passar.", pass: "Deixar passar", icon: "ward" },
    rito: { title: "Feche o turno", hint: "Ainda dá para usar um Rito antes de encerrar.", pass: "Encerrar meu turno", icon: "sigil" },
  };
  const prompt = prompts[state.phase] ?? { title: "A mesa está resolvendo", hint: "Aguarde a próxima ação.", pass: "Continuar", icon: "duel" as const };

  if (!options.length) {
    const reason: Record<string, string> = {
      assalto: "Nenhum Assalto na sua mão",
      guarda: "Nenhuma Guarda na sua mão",
      rito: "Nenhum Rito na sua mão",
    };
    return <aside className="action-panel is-auto" role="status">
      <span className="action-panel__ribbon">SUA VEZ</span>
      <span className="action-panel__glyph"><UiIcon name={prompt.icon} /></span>
      <div className="action-panel__copy">
        <strong>{reason[state.phase] ?? "Nada para jogar agora"}</strong>
        <small>A mesa avança sozinha — você não perdeu nada.</small>
      </div>
      {/* A barra reinicia a cada janela: a chave é a fase daquela rodada. */}
      <span className="action-panel__auto" key={`${state.round}:${state.phase}`} aria-hidden="true">
        <i style={{ animationDuration: `${autoPassMs}ms` }} />
      </span>
    </aside>;
  }

  return <aside className={`action-panel is-live phase-${state.phase}`} aria-label="Sua ação">
    <span className="action-panel__ribbon">SUA VEZ</span>
    <span className="action-panel__glyph"><UiIcon name={prompt.icon} /></span>
    <div className="action-panel__copy">
      <strong>{prompt.title}</strong>
      <small>{prompt.hint}</small>
    </div>
    <p className="action-panel__options">
      {options.length === 1
        ? <>Pronta na mão: <b>{options[0].card.name}</b></>
        : <><b>{options.length}</b> cartas prontas na sua mão</>}
    </p>
    <button className="action-panel__pass" disabled={!connected} type="button" onClick={onPass}>
      <span>{prompt.pass}</span><kbd>ESPAÇO</kbd>
    </button>
  </aside>;
}

/* -------------------------------------------------------------- inspetor */

function Inspector({ card, source, myTurn, choice, onZoom }: {
  card?: CardDefinition;
  source: InspectSource;
  myTurn: boolean;
  choice?: { playable: boolean; onPlay: () => void; onCancel: () => void };
  onZoom: (card: CardDefinition) => void;
}) {
  if (!card) {
    return <div className="inspector is-empty">
      <p className="inspector__eyebrow">LEITURA</p>
      <div className="inspector__blank"><UiIcon name="collection" /><strong>Nenhuma carta em leitura</strong><small>Passe o ponteiro ou clique numa carta da mão para ver o texto completo aqui.</small></div>
      <p className="inspector__rule"><b>Poder − Prevenção = dano.</b> {myTurn ? "É a sua vez de agir." : "Aguarde a ação do rival."}</p>
    </div>;
  }
  const stat = cardStat(card);
  return <div className="inspector">
    <p className="inspector__eyebrow">{source === "arena" ? "NA MESA" : "EM LEITURA"}</p>
    <DuelCard card={card} size="inspect" />
    <dl className="inspector__stats">
      <div><dt>CUSTO</dt><dd>{card.cost} de Vitalidade</dd></div>
      <div><dt>{stat.label.toLocaleUpperCase("pt-BR")}</dt><dd>{stat.value}</dd></div>
      <div><dt>TIPO</dt><dd>{card.type}</dd></div>
    </dl>
    <p className="inspector__brief">{cardBrief(card)}</p>
    <p className="inspector__role">{cardRole(card)}</p>
    {card.confront?.keywords?.length ? <div className="inspector__keywords">{card.confront.keywords.map((keyword) => <span key={keyword}>{keyword}</span>)}</div> : null}
    {choice
      ? <div className="inspector__choice">
          {choice.playable
            ? <button type="button" className="inspector__use" onClick={choice.onPlay}>USAR ESTA CARTA</button>
            : <p className="inspector__locked">Esta carta não é jogável nesta fase.</p>}
          <div className="inspector__choice-minor">
            <button type="button" onClick={() => onZoom(card)}>Ampliar</button>
            <button type="button" onClick={choice.onCancel}>Cancelar</button>
          </div>
        </div>
      : <button type="button" className="inspector__zoom" onClick={() => onZoom(card)}>Ampliar carta</button>}
  </div>;
}

/* ----------------------------------------------------------------- juice */

function GuidedCoach({ lesson, progress, onClose }: { lesson: GuidedLesson; progress: ReturnType<typeof buildGuidedProgress>; onClose: () => void }) {
  const steps = [
    ["Assalto", progress.assault],
    ["Guarda", progress.guard],
    ["Rito", progress.rite],
  ] as const;
  return <aside className={`guided-coach is-${lesson.tone}`} aria-live="polite" aria-label="Treino guiado"><header><span><UiIcon name="guide" /><small>TREINO GUIADO · {progress.completed}/3</small></span><button type="button" onClick={onClose} aria-label="Encerrar treino guiado"><UiIcon name="close" /></button></header><p className="eyebrow">{lesson.eyebrow}</p><strong>{lesson.title}</strong><p>{lesson.copy}</p><div>{steps.map(([label, done]) => <span className={done ? "is-done" : ""} key={label}><i>{done ? "✓" : ""}</i>{label}</span>)}</div></aside>;
}

function useFreshHandCards(ids: string[], reducedMotion: boolean, pace: AnimationPace) {
  const [fresh, setFresh] = useState<Set<string>>(() => new Set());
  const previous = useRef<Set<string> | null>(null);
  const signature = ids.join("|");
  useEffect(() => {
    const next = new Set(ids);
    if (previous.current) {
      const added = ids.filter((id) => !previous.current?.has(id));
      if (added.length && !reducedMotion) {
        setFresh(new Set(added));
        const timer = window.setTimeout(() => setFresh(new Set()), 1100 * paceScale(pace));
        previous.current = next;
        return () => window.clearTimeout(timer);
      }
    }
    previous.current = next;
    setFresh(new Set());
  }, [pace, reducedMotion, signature]);
  return fresh;
}

// A mesa espera este tanto antes de passar sozinha por falta de carta. O
// painel desenha a mesma duração como barra, então o número vive num lugar só.
function autoPassDelay(reducedMotion: boolean) {
  return reducedMotion ? 350 : 1050;
}

function Deadline({ value }: { value: string | null }) {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => { const update = () => setSeconds(value ? Math.max(0, Math.ceil((new Date(value).getTime() - Date.now()) / 1000)) : 0); update(); const timer = window.setInterval(update, 500); return () => window.clearInterval(timer); }, [value]);
  return <span className={`battle-timer ${seconds <= 10 ? "is-low" : ""}`}><UiIcon name="clock" />{seconds || "—"}</span>;
}

function EventLine({ event, cards, ownSlot }: { event: BattleEvent; cards: Map<string, CardDefinition>; ownSlot: number }) {
  return <article><span>{event.seq + 1}</span><div><strong>{eventTitle(event.kind)}</strong><small>{event.p < 0 ? "Mesa" : event.p === ownSlot ? "Você" : "Rival"}{event.def ? ` · ${cards.get(event.def)?.name ?? event.def}` : ""}{event.n ? ` · ${event.n}` : ""}</small></div></article>;
}

const eventLabels: Record<string, string> = {
  match_started: "Partida iniciada", turn_started: "Novo turno", round_started: "Nova rodada",
  card_drawn: "Carta comprada", card_burned: "Compra queimada", card_discarded: "Carta enviada ao descarte",
  card_exiled: "Carta exilada", card_to_hand: "Carta recuperada", card_to_bottom: "Carta devolvida ao baralho",
  window_opened: "Janela de ação", vitality_spent: "Custo de Vitalidade", vitality_sacrificed: "Vitalidade sacrificada",
  card_played: "Carta enviada ao centro", confrontation_opened: "Assalto declarado", guard_committed: "Guarda respondida",
  confrontation_resolved: "Confronto resolvido", card_shattered: "Carta estilhaçada", damage_dealt: "Dano causado",
  damage_prevented: "Dano prevenido", healed: "Vitalidade restaurada", ward_gained: "Ward obtido",
  ward_consumed: "Ward consumido", status_applied: "Efeito aplicado", status_expired: "Efeito encerrado",
  status_fizzled: "Efeito dissipado", bleed_triggered: "Sangramento ativado", curse_triggered: "Maldição ativada",
  decision_requested: "Escolha solicitada", decision_resolved: "Escolha resolvida", fatigue: "Fadiga",
  pass: "Fase encerrada", twilight: "Turno encerrado", match_ended: "Partida encerrada",
};
const eventTitle = (kind: string) => eventLabels[kind] ?? kind.replaceAll("_", " ");
const endReason = (reason?: string) => ({ concede: "A partida terminou por concessão.", timeout: "O tempo de ação acabou.", vitalidade: "A Vitalidade de um duelista chegou a zero.", duplo_nocaute: "Os dois duelistas caíram no mesmo efeito." }[reason ?? ""] ?? "A engine confirmou o resultado final.");
