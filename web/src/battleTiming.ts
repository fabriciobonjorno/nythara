export type AnimationPace = "cinematic" | "normal" | "quick";

export interface BattleTiming {
  attackToWaitingMs: number;
  guardRevealMs: number;
  guardedImpactMs: number;
  directImpactMs: number;
  guardedSettleMs: number;
  directSettleMs: number;
  effectToImpactMs: number;
  effectSettleMs: number;
  resultHoldMs: number;
  shatterFxMs: number;
}

// O servidor entrega resolução, dano e estilhaço no mesmo lote. A mesa abre
// espaço entre esses eventos somente na apresentação: pouso -> choque ->
// leitura do resultado -> estilhaço. Nenhum destes tempos bloqueia comandos.
const TIMING_BY_PACE: Record<AnimationPace, BattleTiming> = {
  cinematic: {
    attackToWaitingMs: 1100,
    guardRevealMs: 900,
    guardedImpactMs: 950,
    directImpactMs: 1050,
    guardedSettleMs: 2900,
    directSettleMs: 2500,
    effectToImpactMs: 1250,
    effectSettleMs: 2800,
    resultHoldMs: 650,
    shatterFxMs: 2900,
  },
  normal: {
    attackToWaitingMs: 850,
    guardRevealMs: 700,
    guardedImpactMs: 750,
    directImpactMs: 800,
    guardedSettleMs: 2300,
    directSettleMs: 1900,
    effectToImpactMs: 950,
    effectSettleMs: 2200,
    resultHoldMs: 500,
    shatterFxMs: 2300,
  },
  quick: {
    attackToWaitingMs: 620,
    guardRevealMs: 480,
    guardedImpactMs: 520,
    directImpactMs: 600,
    guardedSettleMs: 1550,
    directSettleMs: 1350,
    effectToImpactMs: 700,
    effectSettleMs: 1550,
    resultHoldMs: 350,
    shatterFxMs: 1550,
  },
};

const REDUCED_MOTION_TIMING: BattleTiming = {
  attackToWaitingMs: 100,
  guardRevealMs: 80,
  guardedImpactMs: 120,
  directImpactMs: 100,
  guardedSettleMs: 180,
  directSettleMs: 160,
  effectToImpactMs: 100,
  effectSettleMs: 180,
  resultHoldMs: 80,
  shatterFxMs: 180,
};

export function battleTiming(pace: AnimationPace, reducedMotion: boolean): BattleTiming {
  return reducedMotion ? REDUCED_MOTION_TIMING : TIMING_BY_PACE[pace];
}

export function paceScale(pace: AnimationPace) {
  return pace === "quick" ? .6 : pace === "normal" ? .82 : 1;
}
