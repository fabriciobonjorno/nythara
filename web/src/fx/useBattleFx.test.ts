import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BattleEvent } from "../types";
import { useBattleFx } from "./useBattleFx";

const mocks = vi.hoisted(() => ({
  emitFx: vi.fn(),
  setSfxEnabled: vi.fn(),
  shatter: vi.fn(),
}));

vi.mock("./bus", () => ({ emitFx: mocks.emitFx }));
vi.mock("./sfx", () => ({
  setSfxEnabled: mocks.setSfxEnabled,
  sfx: new Proxy({}, { get: (_target, property) => property === "shatter" ? mocks.shatter : vi.fn() }),
}));

function battleEvent(seq: number, kind: string): BattleEvent {
  return { seq, round: 1, kind, p: 0, n: 0, from: 0, to: 0 };
}

describe("useBattleFx", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.emitFx.mockClear();
    mocks.shatter.mockClear();
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

    act(() => vi.advanceTimersByTime(2649));
    expect(mocks.shatter).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(1));
    expect(mocks.shatter).toHaveBeenCalledOnce();
    expect(mocks.emitFx).toHaveBeenCalledWith({ kind: "shards", target: "slot-assault", power: 0.8 });
  });
});
