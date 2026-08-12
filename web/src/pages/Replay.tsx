import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import { useCards, useChampions, useMatchReplay } from "../queries";
import { useSessionStore } from "../store";
import type { BattleEvent, CardDefinition, LastBattle, MatchReplayData } from "../types";
import { Missing } from "./Secondary";

type ReplayPhase = "início" | "compra" | "assalto" | "guarda" | "rito" | "fim";

interface ReplayCardState {
  def: string;
  owner: number;
  stat: number;
  shattered?: boolean;
}

interface ReplayHandCardState {
  instance: string;
  def?: string;
  addedAt: number;
}

interface ReplayFrame {
  event: BattleEvent;
  round: number;
  active: number;
  phase: ReplayPhase;
  vitality: [number, number];
  maximum: [number, number];
  deck: [number, number];
  hand: [number, number];
  handCards: [ReplayHandCardState[], ReplayHandCardState[]];
  assault?: ReplayCardState;
  guard?: ReplayCardState;
  rite?: ReplayCardState;
  damage?: number;
  outcome?: string;
}

interface ReplaySource {
  matchId: string;
  slot: number;
  events: BattleEvent[];
  champions: [string, string];
  maximum: [number, number];
}

const phaseSteps: Array<{ id: ReplayPhase; label: string }> = [
  { id: "compra", label: "Compra" },
  { id: "assalto", label: "Assalto" },
  { id: "guarda", label: "Guarda" },
  { id: "rito", label: "Rito" },
];

export function ReplayPage() {
  const { matchId = "" } = useParams();
  const battle = useSessionStore((state) => state.lastBattle);
  const me = useSessionStore((state) => state.principal);
  const replayQuery = useMatchReplay(matchId);
  const { data: cardData } = useCards();
  const { data: championData } = useChampions();
  const [cursor, setCursor] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const cards = useMemo(() => new Map(cardData?.cards.map((card) => [card.id, card]) ?? []), [cardData]);
  const champions = useMemo(() => new Map(championData?.champions.map((champion) => [champion.id, champion]) ?? []), [championData]);
  const source = useMemo(() => matchId
    ? replayQuery.data ? persistedReplaySource(replayQuery.data, me?.user_id, champions) : undefined
    : battle ? sessionReplaySource(battle) : undefined,
  [battle, champions, matchId, me?.user_id, replayQuery.data]);
  const frames = useMemo(() => source ? projectReplay(source, cards) : [], [cards, source]);
  const maximum = Math.max(0, frames.length - 1);
  const frame = frames[Math.min(cursor, maximum)];

  useEffect(() => {
    if (!playing || !frame) return;
    if (cursor >= maximum) {
      setPlaying(false);
      return;
    }
    const timer = window.setTimeout(() => setCursor((value) => Math.min(maximum, value + 1)), replayDelay(frame.event.kind) / speed);
    return () => window.clearTimeout(timer);
  }, [cursor, frame, maximum, playing, speed]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.closest("input, select, button, a, textarea")) return;
      if (event.key === "ArrowLeft") {
        setPlaying(false);
        setCursor((value) => Math.max(0, value - 1));
      } else if (event.key === "ArrowRight") {
        setPlaying(false);
        setCursor((value) => Math.min(maximum, value + 1));
      } else if (event.code === "Space") {
        event.preventDefault();
        if (cursor >= maximum) setCursor(0);
        setPlaying((value) => !value);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [cursor, maximum]);

  if (matchId && replayQuery.isLoading) return <ReplayLoadState title="Abrindo replay" copy="Reconstruindo a mesa a partir dos eventos autorizados da partida…" />;
  if (matchId && (replayQuery.error || !replayQuery.data)) return <Missing title="Replay indisponível" copy={replayQuery.error instanceof Error ? replayQuery.error.message : "Não foi possível abrir o registro desta partida."} action="Voltar à Arena" to="/arena" />;
  if (!source || !frame) return <Missing title="Replay indisponível" copy="Conclua uma partida ou escolha um duelo no histórico da Arena." action="Abrir Arena" to="/arena" />;
  const mine = source.slot;
  const rival = 1 - mine;
  const currentCard = frame.event.def ? cards.get(frame.event.def) : undefined;
  const title = eventTitle(frame.event, currentCard);
  const copy = eventDetail(frame.event, currentCard, mine);

  const seek = (next: number) => {
    setPlaying(false);
    setCursor(Math.max(0, Math.min(maximum, next)));
  };
  const togglePlayback = () => {
    if (cursor >= maximum) setCursor(0);
    setPlaying((value) => !value);
  };

  return <div className="page replay-page">
    <header className="page-header replay-header"><div><p className="eyebrow">REPRODUÇÃO VISUAL · LOG REDIGIDO</p><h1>{matchId ? "Replay da partida" : "Replay da sessão"}</h1><p>Veja a mesa mudar evento por evento, no mesmo resultado confirmado pelo servidor.</p></div><div className="replay-header__status"><span className={playing ? "is-live" : ""}>{playing ? "REPRODUZINDO" : "PAUSADO"}</span><b>{frames.length} eventos</b><nav aria-label="Outras leituras da partida"><Link to={`/cronica/${source.matchId}`}>Crônica</Link><Link to="/arena">Arena</Link></nav></div></header>

    <div className="replay-layout">
      <section className="replay-player is-rival" aria-label="Estado do rival">
        <ReplayDuelist label="RIVAL" champion={source.champions[rival]} vitality={frame.vitality[rival]} maximum={frame.maximum[rival]} active={frame.active === rival} champions={champions} />
        <ReplayResources hand={frame.hand[rival]} deck={frame.deck[rival]} />
        <ReplayHand label="Mão do rival" items={frame.handCards[rival]} count={frame.hand[rival]} currentSeq={frame.event.seq} cards={cards} />
      </section>

      <section className={`replay-board phase-${frame.phase}`} aria-label={`Mesa do replay, rodada ${frame.round}`}>
        <ReplayPhaseRail phase={frame.phase} active={frame.active === mine ? "SUA AÇÃO" : "AÇÃO DO RIVAL"} />
        <div className="replay-confront" key={`${frame.event.seq}-${cursor}`}>
          <div className="replay-slot is-assault">
            {frame.assault ? <ReplayCard state={frame.assault} card={cards.get(frame.assault.def)} role="ASSALTO" own={frame.assault.owner === mine} /> : <ReplayEmptySlot title="ASSALTO" copy="A carta atacante entra aqui" />}
          </div>
          <div className="replay-verdict" aria-live="polite">
            <NytharaMark />
            {frame.assault ? <><div><span><small>PODER</small><b>{frame.assault.stat}</b></span><i>×</i><span><small>PREVENÇÃO</small><b>{frame.guard?.stat ?? "—"}</b></span></div><strong className={frame.damage ? "is-damage" : frame.outcome ? "is-blocked" : ""}>{frame.outcome ? frame.damage ? `${frame.damage} DE DANO` : "GOLPE BLOQUEADO" : frame.guard ? "COMPARANDO" : "AGUARDANDO GUARDA"}</strong></> : <><strong>ZONA DE CONFRONTO</strong><small>Assalto e Guarda serão comparados no centro.</small></>}
          </div>
          <div className="replay-slot is-guard">
            {frame.guard ? <ReplayCard state={frame.guard} card={cards.get(frame.guard.def)} role="GUARDA" own={frame.guard.owner === mine} /> : <ReplayEmptySlot title="GUARDA" copy="A resposta defensiva entra aqui" />}
          </div>
          {frame.rite && <div className="replay-rite"><span>RITO</span><img src={cardArt(frame.rite.def)} alt="" /><strong>{cards.get(frame.rite.def)?.name ?? frame.rite.def}</strong><small>{frame.rite.owner === mine ? "Você ativou" : "Rival ativou"}</small></div>}
        </div>
        <div className="replay-event" key={`event-${frame.event.seq}`}><span>{String(frame.event.seq + 1).padStart(2, "0")}</span><div><p className="eyebrow">RODADA {frame.round} · {frame.phase.toLocaleUpperCase("pt-BR")}</p><h2>{title}</h2><p>{copy}</p></div></div>
      </section>

      <section className="replay-player is-own" aria-label="Seu estado">
        <ReplayDuelist label="VOCÊ" champion={source.champions[mine]} vitality={frame.vitality[mine]} maximum={frame.maximum[mine]} active={frame.active === mine} champions={champions} />
        <ReplayResources hand={frame.hand[mine]} deck={frame.deck[mine]} />
        <ReplayHand label="Sua mão" items={frame.handCards[mine]} count={frame.hand[mine]} currentSeq={frame.event.seq} cards={cards} />
      </section>

      <section className="replay-transport" aria-label="Controles de reprodução">
        <button type="button" aria-label="Evento anterior" onClick={() => seek(cursor - 1)} disabled={cursor === 0}><UiIcon name="arrow-left" /><span>Anterior</span></button>
        <button className="replay-play" type="button" onClick={togglePlayback} aria-label={playing ? "Pausar replay" : "Reproduzir replay"}><b>{playing ? "Ⅱ" : "▶"}</b><span>{playing ? "Pausar" : cursor >= maximum ? "Recomeçar" : "Reproduzir"}</span></button>
        <button type="button" aria-label="Próximo evento" onClick={() => seek(cursor + 1)} disabled={cursor >= maximum}><span>Próximo</span><UiIcon name="arrow-right" /></button>
        <label><span>Velocidade</span><select value={speed} onChange={(event) => setSpeed(Number(event.target.value))}><option value={0.75}>0,75×</option><option value={1}>1×</option><option value={1.5}>1,5×</option><option value={2}>2×</option></select></label>
        <div className="replay-scrubber"><input aria-label="Posição do replay" type="range" min="0" max={maximum} value={cursor} onChange={(event) => seek(Number(event.target.value))} /><small>Evento {cursor + 1} de {frames.length}</small></div>
        <p><kbd>Espaço</kbd> play/pausa <kbd>←</kbd><kbd>→</kbd> navegar</p>
      </section>

      <aside className="replay-timeline"><header><div><p className="eyebrow">CRÔNICA VISUAL</p><h2>Linha do tempo</h2></div><span>{cursor + 1}/{frames.length}</span></header><div>{frames.map((item, index) => {
        const card = item.event.def ? cards.get(item.event.def) : undefined;
        return <button type="button" className={`${index === cursor ? "is-current" : ""} ${index < cursor ? "is-past" : ""}`} aria-current={index === cursor ? "step" : undefined} onClick={() => seek(index)} key={item.event.seq}><span>{item.event.seq + 1}</span><div><strong>{eventTitle(item.event, card)}</strong><small>Rodada {item.round} · {item.phase}</small></div></button>;
      })}</div></aside>
    </div>
    <p className="honesty-note"><strong>Replay fiel ao registro visível.</strong> A mesa usa somente eventos redigidos recebidos por você durante esta partida; cartas que o servidor manteve ocultas continuam ocultas.</p>
  </div>;
}

function ReplayLoadState({ title, copy }: { title: string; copy: string }) {
  return <div className="page replay-page"><section className="replay-load" role="status"><NytharaMark /><p className="eyebrow">MEMÓRIA DA ARENA</p><h1>{title}</h1><p>{copy}</p><span aria-hidden="true"><i /><i /><i /></span></section></div>;
}

function sessionReplaySource(battle: LastBattle): ReplaySource {
  return {
    matchId: battle.matchId,
    slot: battle.slot,
    events: battle.events,
    champions: [battle.state.players[0].champion, battle.state.players[1].champion],
    maximum: [battle.state.players[0].max_vitality || 30, battle.state.players[1].max_vitality || 30],
  };
}

function persistedReplaySource(replay: MatchReplayData, userId: string | undefined,
  champions: Map<string, { vitality: number }>): ReplaySource {
  const visibleSlot = replay.players.findIndex((player) => player.user_id === userId);
  // A base vem do ruleset da própria partida: a Vitalidade do Campeão é valor
  // legado e o Modo Confronto a sobrescreve, então usá-la aqui desenharia a
  // barra contra uma escala que a partida nunca teve.
  const base = replay.starting_vitality ?? 0;
  const initial: [number, number] = [
    base || champions.get(replay.players[0].champion_id)?.vitality || 30,
    base || champions.get(replay.players[1].champion_id)?.vitality || 30,
  ];
  return {
    matchId: replay.match_id,
    slot: visibleSlot === 1 ? 1 : 0,
    events: replay.events,
    champions: [replay.players[0].champion_id, replay.players[1].champion_id],
    maximum: replayVisibleMaximum(replay.events, initial),
  };
}

function replayVisibleMaximum(events: BattleEvent[], initial: [number, number]): [number, number] {
  const maximum: [number, number] = [...initial];
  for (const event of events) {
    if (!validPlayer(event.p)) continue;
    maximum[event.p] = Math.max(maximum[event.p], event.from, event.to);
  }
  return maximum;
}

function ReplayDuelist({ label, champion, vitality, maximum, active, champions }: { label: string; champion: string; vitality: number; maximum: number; active: boolean; champions: Map<string, { id: string; name: string; faction: string }> }) {
  const avatar = champions.get(champion);
  const percent = Math.max(0, Math.min(100, (vitality / Math.max(1, maximum)) * 100));
  return <div className={`replay-duelist ${active ? "is-active" : ""}`}><span className="replay-avatar">{avatar ? <ChampionEmblem id={avatar.id} faction={avatar.faction} /> : <UiIcon name="champion" />}</span><div><small>{label}</small><strong>{avatar?.name?.split(",")[0] ?? "Duelista"}</strong><em>{active ? "DECIDINDO AGORA" : "AGUARDANDO"}</em></div><span className={`replay-vitality ${vitality <= 8 ? "is-danger" : ""}`} role="progressbar" aria-label={`Vitalidade ${vitality} de ${maximum}`} aria-valuemin={0} aria-valuemax={maximum} aria-valuenow={Math.max(0, vitality)}><UiIcon name="heart" /><b>{Math.max(0, vitality)}</b><small>VITALIDADE</small><i><i style={{ width: `${percent}%` }} /></i></span></div>;
}

function ReplayResources({ hand, deck }: { hand: number; deck: number }) {
  return <div className="replay-resources"><span><UiIcon name="deck" /><small>BARALHO</small><b>{deck}</b></span><span><UiIcon name="collection" /><small>MÃO</small><b>{hand}</b></span></div>;
}

function ReplayHand({ label, items, count, currentSeq, cards }: {
  label: string;
  items: ReplayHandCardState[];
  count: number;
  currentSeq: number;
  cards: Map<string, CardDefinition>;
}) {
  const missing = Math.max(0, count - items.length);
  const visible: ReplayHandCardState[] = [
    ...items,
    ...Array.from({ length: missing }, (_, index) => ({ instance: `unknown-${label}-${index}`, addedAt: -1 })),
  ];
  return <div className="replay-hand"><strong>{label}</strong><ol aria-label={`${label}, ${count} ${count === 1 ? "carta" : "cartas"}`}>
    {visible.map((item) => {
      const card = item.def ? cards.get(item.def) : undefined;
      const name = card?.name ?? item.def;
      return <li className={`${item.def ? "is-known" : "is-hidden"} ${item.addedAt === currentSeq ? "is-current" : ""}`} key={item.instance} aria-label={name ?? "Carta oculta"}>
        {item.def ? <><img src={cardArt(item.def)} alt="" loading="lazy" decoding="async" onError={(event) => { event.currentTarget.hidden = true; }} /><span><small>{card?.type ?? "CARTA"}</small><b>{name}</b></span></> : <><NytharaMark /><span><small>CARTA</small><b>OCULTA</b></span></>}
      </li>;
    })}
    {!visible.length && <li className="is-empty">Mão vazia</li>}
  </ol></div>;
}

function ReplayPhaseRail({ phase, active }: { phase: ReplayPhase; active: string }) {
  const current = phaseSteps.findIndex((step) => step.id === phase);
  return <div className="replay-phase"><small>{active}</small>{phaseSteps.map((step, index) => <span className={step.id === phase ? "is-current" : current > index ? "is-complete" : ""} key={step.id}><i>{current > index ? "✓" : index + 1}</i><b>{step.label}</b></span>)}</div>;
}

function ReplayEmptySlot({ title, copy }: { title: string; copy: string }) {
  return <div className="replay-empty-card"><UiIcon name={title === "ASSALTO" ? "duel" : "ward"} /><strong>{title}</strong><small>{copy}</small></div>;
}

function ReplayCard({ state, card, role, own }: { state: ReplayCardState; card?: CardDefinition; role: string; own: boolean }) {
  return <article className={`replay-card ${state.shattered ? "is-shattered" : ""}`}><img src={cardArt(state.def)} alt={card?.name ?? state.def} /><span>{role} · {own ? "VOCÊ" : "RIVAL"}</span><strong>{card?.name ?? state.def}</strong><b>{role === "ASSALTO" ? `PODER ${state.stat}` : card?.confront?.prevent_all ? "PREVENÇÃO TOTAL" : `PREVENÇÃO ${state.stat}`}</b>{state.shattered && <em>ESTILHAÇADA</em>}</article>;
}

function projectReplay(battle: ReplaySource, cards: Map<string, CardDefinition>): ReplayFrame[] {
  const maximum: [number, number] = [...battle.maximum];
  let frame: ReplayFrame | undefined;
  const frames: ReplayFrame[] = [];
  for (const event of battle.events) {
    const next: ReplayFrame = frame ? {
      ...frame,
      event,
      vitality: [...frame.vitality] as [number, number],
      maximum: [...frame.maximum] as [number, number],
      deck: [...frame.deck] as [number, number],
      hand: [...frame.hand] as [number, number],
      handCards: [frame.handCards[0].map((card) => ({ ...card })), frame.handCards[1].map((card) => ({ ...card }))],
      assault: frame.assault ? { ...frame.assault } : undefined,
      guard: frame.guard ? { ...frame.guard } : undefined,
      rite: frame.rite ? { ...frame.rite } : undefined,
    } : { event, round: event.round, active: event.p >= 0 ? event.p : 0, phase: "início", vitality: [...maximum], maximum, deck: [30, 30], hand: [0, 0], handCards: [[], []] };
    next.round = Math.max(next.round, event.round);
    if (event.kind === "turn_started") {
      next.active = event.p;
      next.round = event.round;
      next.phase = "compra";
      next.assault = undefined;
      next.guard = undefined;
      next.rite = undefined;
      next.damage = undefined;
      next.outcome = undefined;
    } else if (event.kind === "window_opened" && isReplayPhase(event.s)) {
      next.active = event.p >= 0 ? event.p : next.active;
      next.phase = event.s;
    } else if (event.kind === "card_drawn" && validPlayer(event.p)) {
      next.deck[event.p] = Math.max(0, next.deck[event.p] - 1);
      next.hand[event.p] += 1;
      addReplayHandCard(next.handCards[event.p], event);
      next.phase = next.phase === "início" ? "compra" : next.phase;
    } else if (event.kind === "card_burned" && validPlayer(event.p)) {
      next.deck[event.p] = Math.max(0, next.deck[event.p] - 1);
    } else if ((event.kind === "card_to_hand" || event.kind === "card_returned") && validPlayer(event.p)) {
      next.hand[event.p] += 1;
      addReplayHandCard(next.handCards[event.p], event);
    } else if (event.kind === "card_played" && validPlayer(event.p)) {
      next.hand[event.p] = Math.max(0, next.hand[event.p] - 1);
      removeReplayHandCard(next.handCards[event.p], event);
      if (event.def && (next.phase === "rito" || cards.get(event.def)?.type === "Rito")) next.rite = { def: event.def, owner: event.p, stat: event.n };
    } else if (["card_discarded", "card_exiled", "card_to_bottom"].includes(event.kind) && validPlayer(event.p)) {
      const leftHand = removeReplayHandCard(next.handCards[event.p], event);
      if (leftHand) next.hand[event.p] = Math.max(0, next.hand[event.p] - 1);
      if (leftHand && event.kind === "card_to_bottom") next.deck[event.p] += 1;
    } else if (event.kind === "hand_revealed" && event.def && validPlayer(event.p)) {
      revealReplayHandCard(next.handCards[1 - event.p], event);
    } else if (event.kind === "card_locked" && event.def && validPlayer(event.p)) {
      revealReplayHandCard(next.handCards[event.p], event);
    } else if (event.kind === "confrontation_opened" && event.def) {
      next.phase = "guarda";
      next.assault = { def: event.def, owner: event.p, stat: event.n };
      next.guard = undefined;
      next.damage = undefined;
      next.outcome = undefined;
    } else if (event.kind === "guard_committed" && event.def) {
      next.guard = { def: event.def, owner: event.p, stat: event.n };
    } else if (event.kind === "confrontation_resolved") {
      if (!next.assault && event.def) next.assault = { def: event.def, owner: event.p, stat: event.from };
      if (next.assault) next.assault.stat = event.from || next.assault.stat;
      if (next.guard) next.guard.stat = event.to;
      next.damage = event.n;
      next.outcome = event.s;
    } else if (event.kind === "card_shattered" && event.def) {
      if (next.assault?.def === event.def) next.assault.shattered = true;
      if (next.guard?.def === event.def) next.guard.shattered = true;
    } else if (event.kind === "match_ended") {
      next.phase = "fim";
    }
    if (["vitality_spent", "vitality_sacrificed", "damage_dealt", "healed"].includes(event.kind) && validPlayer(event.p) && (event.from !== 0 || event.to !== 0)) {
      next.vitality[event.p] = event.to;
    }
    frame = next;
    frames.push(next);
  }
  return frames;
}

function addReplayHandCard(hand: ReplayHandCardState[], event: BattleEvent) {
  const instance = event.card ?? `hidden-${event.p}-${event.seq}`;
  const existing = hand.find((item) => item.instance === instance);
  if (existing) {
    if (event.def) existing.def = event.def;
    existing.addedAt = event.seq;
    return;
  }
  hand.push({ instance, def: event.def, addedAt: event.seq });
}

function removeReplayHandCard(hand: ReplayHandCardState[], event: BattleEvent) {
  let index = event.card ? hand.findIndex((item) => item.instance === event.card) : -1;
  if (index < 0 && event.def) index = hand.findIndex((item) => item.def === event.def);
  if (index < 0) index = hand.findIndex((item) => !item.def);
  if (index < 0) return false;
  hand.splice(index, 1);
  return true;
}

function revealReplayHandCard(hand: ReplayHandCardState[], event: BattleEvent) {
  let item = event.card ? hand.find((candidate) => candidate.instance === event.card) : undefined;
  if (!item) item = hand.find((candidate) => !candidate.def);
  if (!item) return;
  if (event.card) item.instance = event.card;
  item.def = event.def;
}

function validPlayer(player: number): player is 0 | 1 { return player === 0 || player === 1; }
function isReplayPhase(value?: string): value is "assalto" | "guarda" | "rito" { return value === "assalto" || value === "guarda" || value === "rito"; }
function cardArt(def: string) { return `/card-art/${def.toLocaleLowerCase("en-US")}.webp`; }

function replayDelay(kind: string) {
  if (kind === "confrontation_opened") return 1700;
  if (kind === "guard_committed") return 1550;
  if (kind === "confrontation_resolved") return 1850;
  if (kind === "card_shattered") return 1650;
  if (kind === "card_played" || kind === "damage_dealt") return 1300;
  return 900;
}

const eventLabels: Record<string, string> = {
  match_started: "Partida iniciada", turn_started: "Novo turno", card_drawn: "Carta comprada", card_burned: "Compra queimada",
  card_discarded: "Carta enviada ao descarte", card_exiled: "Carta exilada", card_to_hand: "Carta recuperada", window_opened: "Nova decisão",
  vitality_spent: "Vitalidade paga", vitality_sacrificed: "Vitalidade sacrificada", card_played: "Carta jogada", confrontation_opened: "Assalto declarado",
  guard_committed: "Guarda respondida", confrontation_resolved: "Confronto resolvido", card_shattered: "Carta estilhaçada", damage_dealt: "Dano confirmado",
  damage_prevented: "Dano prevenido", healed: "Vitalidade restaurada", ward_gained: "Ward obtido", ward_consumed: "Ward consumido",
  status_applied: "Efeito aplicado", status_expired: "Efeito encerrado", status_fizzled: "Efeito dissipado", bleed_triggered: "Sangramento ativado",
  curse_triggered: "Maldição ativada", fatigue: "Fadiga", pass: "Fase encerrada", twilight: "Turno encerrado", match_ended: "Partida encerrada",
};

function eventTitle(event: BattleEvent, card?: CardDefinition) {
  const label = eventLabels[event.kind] ?? event.kind.replaceAll("_", " ");
  return card ? `${label} · ${card.name}` : label;
}

function eventDetail(event: BattleEvent, card: CardDefinition | undefined, mine: number) {
  const actor = event.p === mine ? "Você" : event.p >= 0 ? "O rival" : "A Arena";
  if (event.kind === "match_started") return `Ruleset ${event.s ?? "competitivo"} carregado; a ordem inicial foi confirmada.`;
  if (event.kind === "confrontation_opened") return `${actor} colocou ${card?.name ?? "um Assalto"} no centro com Poder ${event.n}.`;
  if (event.kind === "guard_committed") return `${actor} respondeu com ${card?.name ?? "uma Guarda"} e ${event.n} de Prevenção.`;
  if (event.kind === "confrontation_resolved") return `Poder ${event.from} contra Prevenção ${event.to}: ${event.n ? `${event.n} de dano atravessou.` : "o golpe foi bloqueado."}`;
  if (event.kind === "vitality_spent") return `${actor} pagou ${event.n} e ficou com ${event.to} de Vitalidade.`;
  if (event.kind === "damage_dealt") return `${actor} sofreu ${event.n} de dano e ficou com ${event.to} de Vitalidade.`;
  if (event.kind === "healed") return `${actor} recuperou ${event.n} e chegou a ${event.to} de Vitalidade.`;
  if (event.kind === "card_drawn") return event.def ? `${actor} comprou ${card?.name ?? event.def}.` : `${actor} comprou uma carta oculta.`;
  if (event.kind === "window_opened") return event.s ? `${actor} pode decidir na fase de ${event.s}.` : "Uma nova janela de ação foi aberta.";
  if (event.kind === "turn_started") return `${actor} recebeu o turno ${event.round}.`;
  if (event.kind === "card_shattered") return `${card?.name ?? "A carta perdedora"} se rompeu após a comparação.`;
  if (event.kind === "match_ended") return "O servidor confirmou o vencedor e encerrou a sessão.";
  if (card) return `${actor}: ${card.name}.`;
  return event.s ? `${actor}: ${event.s}.` : `${actor} avançou a partida.`;
}
