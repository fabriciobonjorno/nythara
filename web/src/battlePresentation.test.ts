import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useBattlePresentation } from "./battlePresentation";
import type { BattleEvent, BattleState, CardDefinition, PlayerView } from "./types";

function player(vitality: number): PlayerView {
  return {
    champion: "avatar", vitality, max_vitality: 30, essence: 0, essence_cap: 10, temp_essence: 0,
    deck_count: 20, hand_count: 4, discard: [], exile: [], relics: [], manifs: [], stance_committed: true,
    mulligan_done: true, fatigue: 0, trail: [], ward: 0, exposto: false, ultimate_used: false,
  };
}

function state(vitality: [number, number], over = false): BattleState {
  return {
    ruleset_version: "test", round: 1, phase: over ? "fim" : "rito", initiative: 0, active: 0, eclipse: 0,
    players: [player(vitality[0]), player(vitality[1])], cards: {}, over, winner: over ? 0 : -1,
  };
}

function event(seq: number, kind: string, values: Partial<BattleEvent> = {}): BattleEvent {
  return { seq, round: 1, kind, p: -1, n: 0, from: 0, to: 0, ...values };
}

function card(id: string, type: CardDefinition["type"]): CardDefinition {
  return {
    id, name: id, faction: "Errantes", type, rarity: "Comum", cost: 1, eclipse_shift: 0,
    sigil: "Coroa", rules_text: "Teste", flavor: "", design_role: "teste",
  };
}

describe("useBattlePresentation", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("mostra um Rito antes do efeito e só então atualiza a Vitalidade", () => {
    const cards = new Map([["VR-RITO", card("VR-RITO", "Rito")]]);
    const baseline = [event(0, "match_started")];
    const { result, rerender } = renderHook(
      ({ events, snapshot }) => useBattlePresentation(events, snapshot, cards, false, "normal"),
      { initialProps: { events: baseline, snapshot: state([30, 30]) } },
    );

    const resolved = [
      ...baseline,
      event(1, "vitality_spent", { p: 0, n: 1, from: 30, to: 29 }),
      event(2, "card_played", { p: 0, def: "VR-RITO" }),
      event(3, "damage_dealt", { p: 1, n: 5, from: 30, to: 25 }),
      event(4, "match_ended", { p: 0 }),
    ];
    rerender({ events: resolved, snapshot: state([29, 25], true) });

    expect(result.current.busy).toBe(true);
    expect(result.current.vitality).toEqual([30, 30]);
    act(() => vi.advanceTimersByTime(20));
    expect(result.current.visual).toMatchObject({ mode: "effect", cardDef: "VR-RITO", phase: "effect" });
    expect(result.current.vitality[1]).toBe(30);

    act(() => vi.advanceTimersByTime(950));
    expect(result.current.visual).toMatchObject({ mode: "effect", phase: "impact", damage: 5, target: 1 });
    expect(result.current.vitality).toEqual([29, 25]);

    act(() => vi.advanceTimersByTime(1800));
    expect(result.current.busy).toBe(false);
    expect(result.current.visual).toMatchObject({ mode: "effect", phase: "settled" });
  });

  it("mantém o Assalto e os pontos antigos visíveis até o impacto direto", () => {
    const cards = new Map([["VR-HIT", card("VR-HIT", "Assalto")]]);
    const baseline = [event(0, "match_started")];
    const { result, rerender } = renderHook(
      ({ events, snapshot }) => useBattlePresentation(events, snapshot, cards, false, "normal"),
      { initialProps: { events: baseline, snapshot: state([30, 30]) } },
    );
    rerender({
      events: [
        ...baseline,
        event(1, "vitality_spent", { p: 0, n: 1, from: 30, to: 29 }),
        event(2, "card_played", { p: 0, def: "VR-HIT" }),
        event(3, "confrontation_opened", { p: 0, def: "VR-HIT", n: 6 }),
        event(4, "damage_dealt", { p: 1, n: 6, from: 30, to: 24 }),
        event(5, "confrontation_resolved", { p: 0, def: "VR-HIT", from: 6, to: 0, n: 6, s: "assault" }),
      ],
      snapshot: state([29, 24]),
    });

    act(() => vi.advanceTimersByTime(20));
    expect(result.current.visual).toMatchObject({ mode: "confront", attackDef: "VR-HIT", phase: "attack" });
    expect(result.current.vitality[1]).toBe(30);
    act(() => vi.advanceTimersByTime(800));
    expect(result.current.vitality[1]).toBe(30);
    act(() => vi.advanceTimersByTime(850));
    expect(result.current.visual).toMatchObject({ mode: "confront", phase: "impact", damage: 6 });
    expect(result.current.vitality[1]).toBe(24);
  });
});
