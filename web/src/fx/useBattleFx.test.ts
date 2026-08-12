import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BattleEvent } from "../types";
import { battleTiming } from "../battleTiming";
import { useBattleFx } from "./useBattleFx";

const mocks = vi.hoisted(() => ({
  emitFx: vi.fn(),
  setSfxEnabled: vi.fn(),
  shatter: vi.fn(),
  damage: vi.fn(),
  round: vi.fn(),
  drawCard: vi.fn(),
  ended: vi.fn(),
}));

vi.mock("./bus", () => ({ emitFx: mocks.emitFx }));
vi.mock("./sfx", () => ({
  setSfxEnabled: mocks.setSfxEnabled,
  sfx: new Proxy({}, { get: (_target, property) => {
    if (property === "shatter") return mocks.shatter;
    if (property === "damage") return mocks.damage;
    if (property === "round") return mocks.round;
    if (property === "drawCard") return mocks.drawCard;
    if (property === "ended") return mocks.ended;
    return vi.fn();
  } }),
}));

function battleEvent(seq: number, kind: string): BattleEvent {
  return { seq, round: 1, kind, p: 0, n: 0, from: 0, to: 0 };
}

describe("useBattleFx", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.emitFx.mockClear();
    mocks.shatter.mockClear();
    mocks.damage.mockClear();
    mocks.round.mockClear();
    mocks.drawCard.mockClear();
    mocks.ended.mockClear();
  });

  afterEach(() => vi.useRealTimers());

  it("só dispara o estilhaço depois da pausa cinematográfica", () => {
    const baseline = [battleEvent(0, "match_started")];
    const shattered = { ...battleEvent(1, "card_shattered"), s: "guard" };
    const options = { reducedMotion: false, sound: true, haptics: false, animationPace: "cinematic" as const };
    const { rerender } = renderHook(
      ({ events }: { events: BattleEvent[] }) => useBattleFx(events, 0, options),
      { initialProps: { events: baseline } },
    );

    rerender({ events: [...baseline, shattered] });
    expect(mocks.shatter).not.toHaveBeenCalled();
    expect(mocks.emitFx).not.toHaveBeenCalled();

    const shatterAt = battleTiming("cinematic", false).shatterFxMs;
    act(() => vi.advanceTimersByTime(shatterAt - 1));
    expect(mocks.shatter).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(1));
    expect(mocks.shatter).toHaveBeenCalledOnce();
    expect(mocks.emitFx).toHaveBeenCalledWith({ kind: "shards", target: "slot-assault", power: 0.8 });
  });

  it("sincroniza dano de Rito com o impacto visual em vez de antecipá-lo", () => {
    const baseline = [battleEvent(0, "match_started")];
    const options = { reducedMotion: false, sound: true, haptics: false, animationPace: "normal" as const };
    const { rerender } = renderHook(
      ({ events }: { events: BattleEvent[] }) => useBattleFx(events, 0, options),
      { initialProps: { events: baseline } },
    );
    rerender({ events: [
      ...baseline,
      { ...battleEvent(1, "card_played"), def: "VR-RITO" },
      { ...battleEvent(2, "damage_dealt"), p: 1, n: 4, from: 30, to: 26 },
    ] });

    const impactAt = battleTiming("normal", false).effectToImpactMs;
    act(() => vi.advanceTimersByTime(impactAt - 1));
    expect(mocks.damage).not.toHaveBeenCalled();
    expect(mocks.emitFx).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(1));
    expect(mocks.damage).toHaveBeenCalledOnce();
    expect(mocks.emitFx).toHaveBeenCalledWith({ kind: "sparks", target: "vial-rival", power: 0.45 });
  });

  it("aguarda o assentamento do Rito para anunciar compra e novo turno", () => {
    const baseline = [battleEvent(0, "match_started")];
    const options = { reducedMotion: false, sound: true, haptics: false, animationPace: "normal" as const };
    const { rerender } = renderHook(
      ({ events }: { events: BattleEvent[] }) => useBattleFx(events, 0, options),
      { initialProps: { events: baseline } },
    );
    rerender({ events: [
      ...baseline,
      { ...battleEvent(1, "card_played"), def: "VR-RITO" },
      { ...battleEvent(2, "turn_started"), round: 2 },
      { ...battleEvent(3, "card_drawn"), round: 2 },
    ] });

    const settleAt = battleTiming("normal", false).effectSettleMs;
    act(() => vi.advanceTimersByTime(settleAt - 1));
    expect(mocks.round).not.toHaveBeenCalled();
    expect(mocks.drawCard).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(1));
    expect(mocks.round).toHaveBeenCalledOnce();
    expect(mocks.drawCard).toHaveBeenCalledOnce();
  });

  it("só anuncia o resultado depois do último efeito e do respiro final", () => {
    const baseline = [battleEvent(0, "match_started")];
    const options = { reducedMotion: false, sound: true, haptics: false, animationPace: "normal" as const };
    const { rerender } = renderHook(
      ({ events }: { events: BattleEvent[] }) => useBattleFx(events, 0, options),
      { initialProps: { events: baseline } },
    );
    rerender({ events: [
      ...baseline,
      { ...battleEvent(1, "card_played"), def: "VR-RITO" },
      { ...battleEvent(2, "damage_dealt"), p: 1, n: 30, from: 30, to: 0 },
      { ...battleEvent(3, "match_ended"), p: 0 },
    ] });

    const timing = battleTiming("normal", false);
    const resultAt = timing.effectSettleMs + timing.resultHoldMs;
    act(() => vi.advanceTimersByTime(resultAt - 1));
    expect(mocks.ended).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(1));
    expect(mocks.ended).toHaveBeenCalledWith(true);
  });
});
