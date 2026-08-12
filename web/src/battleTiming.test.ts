import { describe, expect, it } from "vitest";
import { battleTiming } from "./battleTiming";

describe("battleTiming", () => {
  it("separa o choque do estilhaço por uma pausa legível em todos os ritmos", () => {
    const cinematic = battleTiming("cinematic", false);
    const normal = battleTiming("normal", false);
    const quick = battleTiming("quick", false);

    expect(cinematic.guardedSettleMs - cinematic.guardedImpactMs).toBeGreaterThanOrEqual(1500);
    expect(normal.guardedSettleMs - normal.guardedImpactMs).toBeGreaterThanOrEqual(1200);
    expect(quick.guardedSettleMs - quick.guardedImpactMs).toBeGreaterThanOrEqual(850);
    expect(cinematic.shatterFxMs).toBe(cinematic.guardedSettleMs);
    expect(normal.shatterFxMs).toBe(normal.guardedSettleMs);
    expect(quick.shatterFxMs).toBe(quick.guardedSettleMs);
  });

  it("mantém a ordem cinematográfico, normal e rápido", () => {
    const cinematic = battleTiming("cinematic", false);
    const normal = battleTiming("normal", false);
    const quick = battleTiming("quick", false);

    expect(cinematic.guardedImpactMs).toBeGreaterThan(normal.guardedImpactMs);
    expect(normal.guardedImpactMs).toBeGreaterThan(quick.guardedImpactMs);
    expect(cinematic.guardedSettleMs).toBeGreaterThan(normal.guardedSettleMs);
    expect(normal.guardedSettleMs).toBeGreaterThan(quick.guardedSettleMs);
  });

  it("encurta a sequência sem remover suas etapas com movimento reduzido", () => {
    const reduced = battleTiming("cinematic", true);

    expect(reduced.guardedImpactMs).toBeLessThan(reduced.guardedSettleMs);
    expect(reduced.shatterFxMs).toBe(reduced.guardedSettleMs);
    expect(reduced.guardedSettleMs).toBeLessThan(200);
  });
});
