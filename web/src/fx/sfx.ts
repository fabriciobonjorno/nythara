// SFX sintetizados via WebAudio — sem assets, volume discreto, nunca lança.
// O contexto nasce no primeiro uso (sempre após um gesto: a batalha só emite
// sons em resposta a eventos de uma partida iniciada por clique).

let context: AudioContext | null = null;
let master: GainNode | null = null;
let enabled = true;
let ambienceEnabled = true;
let ambientMood: AmbientMood = { ownTurn: false, guard: false, danger: false };
let ambient: {
  bus: GainNode;
  drone: OscillatorNode;
  droneGain: GainNode;
  overtone: OscillatorNode;
  overtoneGain: GainNode;
  filter: BiquadFilterNode;
} | null = null;

export interface AmbientMood {
  ownTurn: boolean;
  guard: boolean;
  danger: boolean;
}

export function setSfxEnabled(value: boolean) {
  enabled = value;
  if (master) master.gain.value = value ? 0.11 : 0;
  if (!value) stopAmbience();
}

export function setAmbienceEnabled(value: boolean) {
  ambienceEnabled = value;
  if (!value) stopAmbience();
}

function ensureContext(): AudioContext | null {
  try {
    if (!context) {
      const Ctor = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!Ctor) return null;
      context = new Ctor();
      master = context.createGain();
      master.gain.value = enabled ? 0.11 : 0;
      master.connect(context.destination);
    }
    if (context.state === "suspended") void context.resume();
    return context;
  } catch {
    return null;
  }
}

function applyAmbientMood() {
  if (!ambient || !context) return;
  const now = context.currentTime;
  const base = ambientMood.danger ? 43.65 : ambientMood.guard ? 61.74 : ambientMood.ownTurn ? 55 : 49;
  const volume = ambientMood.danger ? 0.075 : ambientMood.guard ? 0.062 : ambientMood.ownTurn ? 0.052 : 0.038;
  ambient.drone.frequency.cancelScheduledValues(now);
  ambient.overtone.frequency.cancelScheduledValues(now);
  ambient.bus.gain.cancelScheduledValues(now);
  ambient.drone.frequency.linearRampToValueAtTime(base, now + 0.9);
  ambient.overtone.frequency.linearRampToValueAtTime(base * (ambientMood.guard ? 1.5 : 1.3348), now + 0.9);
  ambient.bus.gain.setValueAtTime(Math.max(0.0001, ambient.bus.gain.value), now);
  ambient.bus.gain.linearRampToValueAtTime(volume, now + 0.8);
}

/** Inicia somente quando chamado por uma ação explícita do jogador. */
export function engageAmbience() {
  if (!enabled || !ambienceEnabled || ambient) return;
  const ctx = ensureContext();
  if (!ctx || !master) return;
  try {
    const bus = ctx.createGain();
    const filter = ctx.createBiquadFilter();
    const drone = ctx.createOscillator();
    const droneGain = ctx.createGain();
    const overtone = ctx.createOscillator();
    const overtoneGain = ctx.createGain();
    bus.gain.value = 0.0001;
    filter.type = "lowpass";
    filter.frequency.value = 420;
    filter.Q.value = 0.55;
    drone.type = "sine";
    overtone.type = "triangle";
    droneGain.gain.value = 0.72;
    overtoneGain.gain.value = 0.16;
    drone.connect(droneGain).connect(filter);
    overtone.connect(overtoneGain).connect(filter);
    filter.connect(bus).connect(master);
    drone.start();
    overtone.start();
    ambient = { bus, drone, droneGain, overtone, overtoneGain, filter };
    applyAmbientMood();
  } catch {
    stopAmbience();
  }
}

export function updateAmbience(mood: AmbientMood) {
  ambientMood = mood;
  applyAmbientMood();
}

export function stopAmbience() {
  if (!ambient) return;
  try {
    ambient.drone.stop();
    ambient.overtone.stop();
    ambient.drone.disconnect();
    ambient.overtone.disconnect();
    ambient.droneGain.disconnect();
    ambient.overtoneGain.disconnect();
    ambient.filter.disconnect();
    ambient.bus.disconnect();
  } catch {
    // limpeza de áudio é sempre best effort
  }
  ambient = null;
}

interface Tone {
  freq: number;
  to?: number;        // glide de frequência
  at?: number;        // atraso em s
  dur?: number;
  type?: OscillatorType;
  gain?: number;
}

function play(tones: Tone[]) {
  if (!enabled) return;
  const ctx = ensureContext();
  if (!ctx || !master) return;
  try {
    for (const tone of tones) {
      const start = ctx.currentTime + (tone.at ?? 0);
      const dur = tone.dur ?? 0.18;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = tone.type ?? "sine";
      osc.frequency.setValueAtTime(tone.freq, start);
      if (tone.to) osc.frequency.exponentialRampToValueAtTime(Math.max(30, tone.to), start + dur);
      gain.gain.setValueAtTime(0.0001, start);
      gain.gain.exponentialRampToValueAtTime(tone.gain ?? 0.5, start + 0.012);
      gain.gain.exponentialRampToValueAtTime(0.0001, start + dur);
      osc.connect(gain).connect(master);
      osc.start(start);
      osc.stop(start + dur + 0.05);
    }
  } catch {
    // áudio jamais derruba a mesa
  }
}

export const sfx = {
  confront() {
    play([{ freq: 330, to: 120, dur: 0.18, type: "sawtooth", gain: 0.22 }, { freq: 520, to: 210, dur: 0.14, at: 0.04, type: "triangle", gain: 0.24 }]);
  },
  shatter() {
    play([{ freq: 920, to: 190, dur: 0.2, type: "square", gain: 0.22 }, { freq: 610, to: 90, dur: 0.28, at: 0.03, type: "sawtooth", gain: 0.18 }, { freq: 1280, to: 320, dur: 0.12, at: 0.08, type: "triangle", gain: 0.16 }]);
  },
  damage(heavy: boolean) {
    play(heavy
      ? [{ freq: 130, to: 55, dur: 0.3, type: "triangle", gain: 0.8 }, { freq: 70, to: 40, dur: 0.4, at: 0.02, type: "sawtooth", gain: 0.3 }]
      : [{ freq: 190, to: 95, dur: 0.14, type: "triangle", gain: 0.55 }]);
  },
  prevented() {
    play([{ freq: 420, dur: 0.09, type: "square", gain: 0.22 }, { freq: 560, dur: 0.1, at: 0.05, type: "square", gain: 0.18 }]);
  },
  heal() {
    play([{ freq: 392, dur: 0.12, gain: 0.3 }, { freq: 523, dur: 0.16, at: 0.07, gain: 0.3 }]);
  },
  ward() {
    play([{ freq: 660, dur: 0.08, type: "triangle", gain: 0.25 }, { freq: 880, dur: 0.12, at: 0.04, type: "triangle", gain: 0.2 }]);
  },
  sigil(chain: number) {
    const base = 440 * Math.pow(1.122, Math.min(chain, 5)); // sobe a cada elo
    play([{ freq: base, dur: 0.1, type: "triangle", gain: 0.32 }, { freq: base * 1.5, dur: 0.14, at: 0.05, type: "sine", gain: 0.22 }]);
  },
  chain() {
    play([{ freq: 523, dur: 0.09, gain: 0.3 }, { freq: 659, dur: 0.09, at: 0.07, gain: 0.3 }, { freq: 784, dur: 0.2, at: 0.14, gain: 0.34 }]);
  },
  eclipseShift() {
    play([{ freq: 240, to: 300, dur: 0.12, type: "sine", gain: 0.2 }]);
  },
  eclipseTotal(night: boolean) {
    play(night
      ? [{ freq: 90, to: 38, dur: 1.1, type: "sawtooth", gain: 0.5 }, { freq: 180, to: 60, dur: 0.9, at: 0.08, type: "triangle", gain: 0.3 }, { freq: 55, dur: 1.3, at: 0.15, type: "sine", gain: 0.45 }]
      : [{ freq: 523, to: 1046, dur: 0.8, type: "sine", gain: 0.32 }, { freq: 659, to: 1318, dur: 0.9, at: 0.1, type: "triangle", gain: 0.22 }, { freq: 392, dur: 1.1, at: 0.05, type: "sine", gain: 0.25 }]);
  },
  stances() {
    play([{ freq: 220, dur: 0.1, type: "triangle", gain: 0.3 }, { freq: 330, dur: 0.16, at: 0.09, type: "triangle", gain: 0.34 }]);
  },
  round() {
    play([{ freq: 262, dur: 0.1, type: "triangle", gain: 0.25 }, { freq: 392, dur: 0.14, at: 0.08, type: "triangle", gain: 0.22 }]);
  },
  ultimate() {
    play([{ freq: 110, to: 220, dur: 0.5, type: "sawtooth", gain: 0.4 }, { freq: 440, to: 880, dur: 0.4, at: 0.12, type: "triangle", gain: 0.25 }]);
  },
  countered() {
    play([{ freq: 700, to: 180, dur: 0.24, type: "square", gain: 0.26 }]);
  },
  ended(won: boolean) {
    play(won
      ? [{ freq: 392, dur: 0.16, gain: 0.32 }, { freq: 523, dur: 0.16, at: 0.14, gain: 0.32 }, { freq: 659, dur: 0.2, at: 0.28, gain: 0.34 }, { freq: 784, dur: 0.42, at: 0.42, gain: 0.4 }]
      : [{ freq: 220, to: 110, dur: 0.7, type: "triangle", gain: 0.35 }, { freq: 165, to: 82, dur: 0.9, at: 0.15, type: "sine", gain: 0.3 }]);
  },
};
