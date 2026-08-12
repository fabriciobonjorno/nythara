export type AnimationPace = "cinematic" | "normal" | "quick";

interface BattleTiming {
  attackToWaitingMs: number;
  guardedImpactMs: number;
  directImpactMs: number;
  guardedSettleMs: number;
  directSettleMs: number;
  shatterFxMs: number;
}

// O servidor entrega resolução, dano e estilhaço no mesmo lote. A mesa abre
// espaço entre esses eventos somente na apresentação: pouso -> choque ->
// leitura do resultado -> estilhaço. Nenhum destes tempos bloqueia comandos.
const TIMING_BY_PACE: Record<AnimationPace, BattleTiming> = {
  cinematic: {
    attackToWaitingMs: 920,
    guardedImpactMs: 1050,
    directImpactMs: 520,
    guardedSettleMs: 2650,
    directSettleMs: 1550,
    shatterFxMs: 2650,
  },
  normal: {
    attackToWaitingMs: 700,
    guardedImpactMs: 820,
    directImpactMs: 420,
    guardedSettleMs: 2100,
    directSettleMs: 1200,
    shatterFxMs: 2100,
  },
  quick: {
    attackToWaitingMs: 480,
    guardedImpactMs: 560,
    directImpactMs: 300,
    guardedSettleMs: 1450,
    directSettleMs: 850,
    shatterFxMs: 1450,
  },
};

const REDUCED_MOTION_TIMING: BattleTiming = {
  attackToWaitingMs: 100,
  guardedImpactMs: 120,
  directImpactMs: 100,
  guardedSettleMs: 180,
  directSettleMs: 160,
  shatterFxMs: 180,
};

export function battleTiming(pace: AnimationPace, reducedMotion: boolean): BattleTiming {
  return reducedMotion ? REDUCED_MOTION_TIMING : TIMING_BY_PACE[pace];
}

export function paceScale(pace: AnimationPace) {
  return pace === "quick" ? .6 : pace === "normal" ? .82 : 1;
}
