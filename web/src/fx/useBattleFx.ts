import { useEffect, useRef, useState } from "react";
import type { BattleEvent } from "../types";
import { setSfxEnabled, sfx } from "./sfx";

// Camada de juice: transforma o log de eventos authoritative em efeitos
// transitórios de apresentação. Nunca toca em lógica de jogo. Reconexão e
// refresh não reanimam o passado: o primeiro lote observado vira baseline.

export interface Floater {
  id: number;
  slot: number;
  text: string;
  tone: "damage" | "cost" | "heal" | "ward" | "prevent" | "sigil";
  icon?: "heart" | "ward" | "reaction" | "sigil";
}

export interface FxBannerData {
  id: number;
  title: string;
  sub?: string;
  tone: "round" | "combo" | "ultimate" | "counter";
}

export interface EclipseCinematicData {
  id: number;
  night: boolean;
  title: string;
}

export interface StanceRevealData {
  id: number;
  mine: string;
  theirs: string;
}

export interface BattleFxState {
  floaters: Floater[];
  banner: FxBannerData | null;
  eclipseCine: EclipseCinematicData | null;
  stanceReveal: StanceRevealData | null;
  shaking: boolean;
  meterFlash: number;
  sigilPulse: { slot: number; index: number; chain: number; key: number } | null;
  burst: { key: number; night: boolean } | null;
}

const FLOATER_LIFE = 1500;
const FLOATER_CAP = 12;

export function useBattleFx(events: BattleEvent[], mySlot: number | null,
  options: { reducedMotion: boolean; sound: boolean; haptics: boolean }): BattleFxState {
  const [floaters, setFloaters] = useState<Floater[]>([]);
  const [banner, setBanner] = useState<FxBannerData | null>(null);
  const [eclipseCine, setEclipseCine] = useState<EclipseCinematicData | null>(null);
  const [stanceReveal, setStanceReveal] = useState<StanceRevealData | null>(null);
  const [shaking, setShaking] = useState(false);
  const [meterFlash, setMeterFlash] = useState(0);
  const [sigilPulse, setSigilPulse] = useState<BattleFxState["sigilPulse"]>(null);
  const [burst, setBurst] = useState<BattleFxState["burst"]>(null);

  const baselineRef = useRef(false);
  const lastSeqRef = useRef(-1);
  const idRef = useRef(0);
  const timersRef = useRef<Set<number>>(new Set());
  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => { setSfxEnabled(options.sound); }, [options.sound]);

  useEffect(() => {
    const timers = timersRef.current;
    return () => { timers.forEach((timer) => window.clearTimeout(timer)); timers.clear(); };
  }, []);

  useEffect(() => {
    if (mySlot === null || !events.length) return;
    const maxSeq = events[events.length - 1].seq;
    if (!baselineRef.current) {
      // Primeiro contato (entrada ou reconexão): não reanimar o histórico.
      baselineRef.current = true;
      lastSeqRef.current = maxSeq;
      return;
    }
    const fresh = events.filter((event) => event.seq > lastSeqRef.current);
    if (!fresh.length) return;
    lastSeqRef.current = maxSeq;

    const { reducedMotion } = optionsRef.current;
    const after = (ms: number, run: () => void) => {
      const timer = window.setTimeout(() => { timersRef.current.delete(timer); run(); }, ms);
      timersRef.current.add(timer);
    };
    const spawnFloater = (slot: number, text: string, tone: Floater["tone"], icon?: Floater["icon"]) => {
      const id = ++idRef.current;
      setFloaters((current) => [...current.slice(-FLOATER_CAP + 1), { id, slot, text, tone, icon }]);
      after(FLOATER_LIFE, () => setFloaters((current) => current.filter((item) => item.id !== id)));
    };
    const showBanner = (title: string, tone: FxBannerData["tone"], sub?: string, life = 1700) => {
      const id = ++idRef.current;
      setBanner({ id, title, sub, tone });
      after(life, () => setBanner((current) => (current?.id === id ? null : current)));
    };
    const vibrate = (pattern: number | number[]) => {
      if (!optionsRef.current.haptics || typeof navigator.vibrate !== "function") return;
      navigator.vibrate(pattern);
    };

    for (const event of fresh) {
      switch (event.kind) {
        case "turn_started":
          showBanner(`TURNO ${event.round}`, "round",
            event.p === mySlot ? "Sua ação: compre e escolha um Assalto." : "O rival prepara o próximo Assalto.");
          sfx.round();
          if (event.p === mySlot) vibrate(18);
          break;
        case "vitality_spent":
          if (event.n > 0) spawnFloater(event.p, `CUSTO −${event.n}`, "cost", "heart");
          break;
        case "confrontation_opened":
          sfx.confront();
          vibrate(16);
          break;
        case "guard_committed":
          sfx.prevented();
          break;
        case "card_shattered":
          sfx.shatter();
          vibrate([28, 22, 42]);
          break;
        case "damage_dealt": {
          if (event.n <= 0) break;
          const heavy = event.n >= 5;
          spawnFloater(event.p, `-${event.n}`, "damage");
          sfx.damage(heavy);
          vibrate(heavy ? [45, 25, 65] : 28);
          if (heavy && !reducedMotion) {
            setShaking(true);
            after(450, () => setShaking(false));
          }
          break;
        }
        case "damage_prevented":
          if (event.n <= 0) break;
          spawnFloater(event.p, `${event.n}`, "prevent", "reaction");
          sfx.prevented();
          break;
        case "healed":
          if (event.n <= 0) break;
          spawnFloater(event.p, `+${event.n}`, "heal", "heart");
          sfx.heal();
          break;
        case "ward_gained":
          if (event.n <= 0) break;
          spawnFloater(event.p, `+${event.n}`, "ward", "ward");
          sfx.ward();
          break;
        case "status_applied":
          if (event.s === "Selo do Assalto") {
            showBanner("ASSALTO SELADO", "counter", event.p === mySlot ? "Seu próximo Assalto será pulado." : "O próximo Assalto rival foi proibido.", 2100);
            sfx.countered();
          } else if (event.s === "Selo do Rito") {
            showBanner("RITO SELADO", "counter", event.p === mySlot ? "Sua janela de Rito foi proibida." : "O Rito do atacante foi interditado.", 2100);
            sfx.countered();
          } else if (event.s?.toLocaleLowerCase("pt-BR") === "exposto") {
            showBanner("GUARDA PROIBIDA", "counter", event.p === mySlot ? "O próximo Assalto contra você será implacável." : "O rival não poderá usar Guarda no próximo golpe.", 2100);
          }
          break;
        case "status_expired":
          if (event.s === "Selo do Assalto" || event.s === "Selo do Rito") {
            showBanner(event.s === "Selo do Assalto" ? "ASSALTO PULADO" : "RITO PULADO", "counter", "O Selo foi consumido e a mesa avançou.", 1850);
          }
          break;
        case "sigil_added": {
          const id = ++idRef.current;
          setSigilPulse({ slot: event.p, index: Math.max(0, event.n - 1), chain: event.n, key: id });
          if (event.p !== mySlot) spawnFloater(event.p, event.s ?? "Sigilo", "sigil", "sigil");
          sfx.sigil(event.n);
          if (event.n === 3) {
            showBanner("TRÍADE DE SIGILOS", "combo",
              event.p === mySlot ? "Sua cadeia ressoa pelo Véu." : "A cadeia rival ressoa pelo Véu.");
            sfx.chain();
          }
          break;
        }
        case "eclipse_shifted":
          setMeterFlash((current) => current + 1);
          sfx.eclipseShift();
          break;
        case "eclipse_triggered": {
          const night = event.s === "Eclipse Noturno";
          const id = ++idRef.current;
          setEclipseCine({ id, night, title: event.s || "Eclipse Total" });
          setBurst({ key: id, night });
          sfx.eclipseTotal(night);
          after(reducedMotion ? 1400 : 2600, () => setEclipseCine((current) => (current?.id === id ? null : current)));
          break;
        }
        case "stances_revealed": {
          const parts = (event.s ?? "").split("|");
          if (parts.length === 2) {
            const id = ++idRef.current;
            setStanceReveal({ id, mine: parts[mySlot], theirs: parts[1 - mySlot] });
            sfx.stances();
            after(reducedMotion ? 1200 : 2000, () => setStanceReveal((current) => (current?.id === id ? null : current)));
          }
          break;
        }
        case "round_started":
          showBanner(`RODADA ${event.n}`, "round",
            event.p === mySlot ? "A iniciativa é sua." : "O rival tem a iniciativa.");
          sfx.round();
          break;
        case "ultimate":
          showBanner("ULTIMATE DESATADA", "ultimate",
            event.p === mySlot ? "Seu Campeão rompe o limite." : "O Campeão rival rompe o limite.", 2000);
          sfx.ultimate();
          break;
        case "rite_countered":
          showBanner("RITO ANULADO", "counter",
            event.p === mySlot ? "Seu Rito foi interrompido." : "Sua Guarda respondeu no tempo exato.");
          sfx.countered();
          break;
        case "match_ended":
          sfx.ended(event.p === mySlot);
          vibrate(event.p === mySlot ? [30, 30, 45, 30, 80] : [70, 35, 90]);
          break;
        default:
          break;
      }
    }
  }, [events, mySlot]);

  return { floaters, banner, eclipseCine, stanceReveal, shaking, meterFlash, sigilPulse, burst };
}
