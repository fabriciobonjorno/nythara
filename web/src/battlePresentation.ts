import { useEffect, useRef, useState } from "react";
import { battleTiming, type AnimationPace } from "./battleTiming";
import type { BattleEvent, BattleState, CardDefinition } from "./types";

export type ConfrontPhase = "attack" | "waiting" | "guard" | "impact" | "settled";
export type EffectPhase = "effect" | "impact" | "settled";

export interface ConfrontVisual {
  mode: "confront";
  key: number;
  attackDef: string;
  guardDef?: string;
  attacker: number;
  power: number;
  prevention: number;
  damage: number;
  outcome?: string;
  phase: ConfrontPhase;
  stale?: boolean;
}

export interface EffectVisual {
  mode: "effect";
  key: number;
  cardDef: string;
  player: number;
  target?: number;
  damage: number;
  phase: EffectPhase;
  stale?: boolean;
}

export type ArenaVisual = ConfrontVisual | EffectVisual;

const VITALITY_KINDS = new Set([
  "vitality_spent", "vitality_sacrificed", "damage_dealt", "healed", "fatigue",
]);

interface PendingVitality {
  event: BattleEvent;
  scheduled: boolean;
}

// O snapshot authoritative chega pronto, mas a apresentação não pode pular
// direto para o último número. Esta fila conserva a ordem perceptível:
// carta -> leitura -> impacto -> Vitalidade -> assentamento. Ela não interfere
// na engine nem no relógio do servidor; controla apenas o que é desenhado.
export function useBattlePresentation(
  events: BattleEvent[],
  state: BattleState | null,
  cards: Map<string, CardDefinition>,
  reducedMotion: boolean,
  pace: AnimationPace,
) {
  const [visual, setVisual] = useState<ArenaVisual | null>(null);
  const [displayedVitality, setDisplayedVitality] = useState<[number, number] | null>(null);
  const [queueBusy, setQueueBusy] = useState(false);
  const [consumedSeq, setConsumedSeq] = useState(-1);
  const initialized = useRef(false);
  const lastSeq = useRef(-1);
  const plannedVisual = useRef<ArenaVisual | null>(null);
  const queueUntil = useRef(0);
  const queueToken = useRef(0);
  const queueBusyRef = useRef(false);
  const timers = useRef<Set<number>>(new Set());
  const latestState = useRef(state);
  latestState.current = state;

  useEffect(() => {
    const activeTimers = timers.current;
    return () => {
      activeTimers.forEach((timer) => window.clearTimeout(timer));
      activeTimers.clear();
    };
  }, []);

  useEffect(() => {
    if (!state) return;
    const authoritative: [number, number] = [state.players[0].vitality, state.players[1].vitality];
    if (!initialized.current) {
      initialized.current = true;
      const baseline = events.at(-1)?.seq ?? -1;
      lastSeq.current = baseline;
      setConsumedSeq(baseline);
      setDisplayedVitality(authoritative);
      return;
    }

    const fresh = events.filter((event) => event.seq > lastSeq.current);
    if (!fresh.length) {
      if (!queueBusyRef.current && Date.now() >= queueUntil.current) {
        setDisplayedVitality((current) => current?.[0] === authoritative[0] && current[1] === authoritative[1]
          ? current
          : authoritative);
      }
      return;
    }

    const maxSeq = fresh.at(-1)!.seq;
    lastSeq.current = maxSeq;
    setConsumedSeq(maxSeq);
    const timing = battleTiming(pace, reducedMotion);
    let cursor = Math.max(Date.now() + 16, queueUntil.current);
    const confrontationCards = new Set(fresh
      .filter((event) => event.kind === "confrontation_opened" || event.kind === "guard_committed")
      .map((event) => event.def)
      .filter((definition): definition is string => Boolean(definition)));
    const pendingVitality: PendingVitality[] = fresh
      .filter((event) => VITALITY_KINDS.has(event.kind) && (event.p === 0 || event.p === 1) && event.from !== event.to)
      .map((event) => ({ event, scheduled: false }));
    let effectNeedsSettle = false;
    let effectHits = 0;

    const scheduleAt = (at: number, run: () => void) => {
      const timer = window.setTimeout(() => {
        timers.current.delete(timer);
        run();
      }, Math.max(0, at - Date.now()));
      timers.current.add(timer);
    };
    const planVisual = (next: ArenaVisual | null, at: number) => {
      plannedVisual.current = next;
      scheduleAt(at, () => setVisual(next));
    };
    const scheduleVitality = (pending: PendingVitality, at: number) => {
      if (pending.scheduled) return;
      pending.scheduled = true;
      scheduleAt(at, () => setDisplayedVitality((current) => {
        const next: [number, number] = current
          ? [...current] as [number, number]
          : [latestState.current?.players[0].vitality ?? 0, latestState.current?.players[1].vitality ?? 0];
        next[pending.event.p] = pending.event.to;
        return next;
      }));
    };
    const flushCosts = (player: number, at: number) => {
      pendingVitality
        .filter((pending) => !pending.scheduled && pending.event.p === player &&
          (pending.event.kind === "vitality_spent" || pending.event.kind === "vitality_sacrificed"))
        .forEach((pending, index) => scheduleVitality(pending, at + index * 140));
    };
    const flushDamage = (at: number) => {
      pendingVitality
        .filter((pending) => !pending.scheduled && pending.event.kind === "damage_dealt")
        .forEach((pending, index) => scheduleVitality(pending, at + index * 180));
    };

    queueBusyRef.current = true;
    setQueueBusy(true);

    for (const event of fresh) {
      if (event.kind === "card_played" && event.def) {
        const card = cards.get(event.def);
        // Os eventos semânticos do mesmo lote identificam Assalto/Guarda mesmo
        // se o catálogo ainda estiver carregando. Todo o restante ganha a cena
        // de efeito e a definição/arte entra assim que a consulta terminar.
        if (!confrontationCards.has(event.def) && card?.type !== "Assalto" && card?.type !== "Guarda") {
          const next: EffectVisual = {
            mode: "effect", key: event.seq, cardDef: event.def, player: event.p,
            damage: 0, phase: "effect",
          };
          planVisual(next, cursor);
          flushCosts(event.p, cursor + Math.min(520, Math.round(timing.effectToImpactMs * .42)));
          cursor += timing.effectToImpactMs;
          effectNeedsSettle = true;
          effectHits = 0;
        }
        continue;
      }

      if (event.kind === "confrontation_opened" && event.def) {
        const next: ConfrontVisual = {
          mode: "confront", key: event.seq, attackDef: event.def, attacker: event.p,
          power: event.n, prevention: 0, damage: 0, phase: "attack",
        };
        planVisual(next, cursor);
        flushCosts(event.p, cursor + Math.min(520, Math.round(timing.attackToWaitingMs * .45)));
        cursor += timing.attackToWaitingMs;
        const waiting: ConfrontVisual = { ...next, phase: "waiting" };
        planVisual(waiting, cursor);
        effectNeedsSettle = false;
        continue;
      }

      if (event.kind === "guard_committed" && event.def && plannedVisual.current?.mode === "confront") {
        const guarded: ConfrontVisual = {
          ...plannedVisual.current, guardDef: event.def, prevention: event.n, phase: "guard", stale: false,
        };
        planVisual(guarded, cursor);
        flushCosts(event.p, cursor + Math.min(420, Math.round(timing.guardRevealMs * .48)));
        cursor += timing.guardRevealMs;
        continue;
      }

      if (event.kind === "damage_dealt" && plannedVisual.current?.mode === "effect") {
        const current = plannedVisual.current;
        const impact: EffectVisual = {
          ...current, target: event.p, damage: current.damage + Math.max(0, event.n), phase: "impact", stale: false,
        };
        planVisual(impact, cursor);
        const pending = pendingVitality.find((item) => item.event.seq === event.seq);
        if (pending) scheduleVitality(pending, cursor);
        effectHits += 1;
        cursor += reducedMotion ? 40 : 320;
        effectNeedsSettle = true;
        continue;
      }

      if (event.kind === "healed" && plannedVisual.current?.mode === "effect") {
        const pending = pendingVitality.find((item) => item.event.seq === event.seq);
        if (pending) scheduleVitality(pending, cursor);
        cursor += reducedMotion ? 30 : 240;
        effectNeedsSettle = true;
        continue;
      }

      if (event.kind === "confrontation_resolved") {
        const current = plannedVisual.current?.mode === "confront"
          ? plannedVisual.current
          : {
              mode: "confront" as const, key: event.seq, attackDef: event.def ?? "", attacker: event.p,
              power: event.from, prevention: event.to, damage: 0, phase: "waiting" as const,
            };
        const impactDelay = current.guardDef ? timing.guardedImpactMs : timing.directImpactMs;
        const settleDelay = current.guardDef ? timing.guardedSettleMs : timing.directSettleMs;
        cursor += impactDelay;
        const impact: ConfrontVisual = {
          ...current, power: event.from, prevention: event.to, damage: event.n,
          outcome: event.s, phase: "impact", stale: false,
        };
        planVisual(impact, cursor);
        flushDamage(cursor);
        cursor += Math.max(0, settleDelay - impactDelay);
        planVisual({ ...impact, phase: "settled" }, cursor);
        effectNeedsSettle = false;
        continue;
      }

      if (event.kind === "turn_started" || event.kind === "round_started") {
        const current = plannedVisual.current;
        if (current) {
          const settled = { ...current, phase: "settled", stale: true } as ArenaVisual;
          planVisual(settled, cursor);
        }
      }
    }

    if (plannedVisual.current?.mode === "effect" && effectNeedsSettle) {
      const remaining = Math.max(
        reducedMotion ? 30 : 650,
        timing.effectSettleMs - timing.effectToImpactMs - effectHits * (reducedMotion ? 40 : 320),
      );
      cursor += remaining;
      planVisual({ ...plannedVisual.current, phase: "settled" }, cursor);
    }

    // Pressão, Fadiga e efeitos sem carta associada ainda respeitam uma curta
    // leitura, mas jamais ficam presos esperando uma cena que não existe.
    const unscheduledVitality = pendingVitality.filter((pending) => !pending.scheduled);
    unscheduledVitality.forEach((pending, index) => {
      scheduleVitality(pending, cursor + index * (reducedMotion ? 20 : 180));
    });
    if (unscheduledVitality.length) cursor += reducedMotion ? 40 : 360;

    const token = ++queueToken.current;
    const finishAt = cursor + timing.resultHoldMs;
    queueUntil.current = finishAt;
    scheduleAt(finishAt, () => {
      if (queueToken.current !== token) return;
      queueBusyRef.current = false;
      setQueueBusy(false);
      const latest = latestState.current;
      if (latest) setDisplayedVitality([latest.players[0].vitality, latest.players[1].vitality]);
    });
  }, [cards, events, pace, reducedMotion, state]);

  const authoritative: [number, number] = state
    ? [state.players[0].vitality, state.players[1].vitality]
    : [0, 0];
  const newestSeq = events.at(-1)?.seq ?? -1;
  return {
    visual,
    vitality: displayedVitality ?? authoritative,
    busy: queueBusy || newestSeq > consumedSeq,
  };
}
